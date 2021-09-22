package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk/client"
	"gorm.io/gorm"
)

type IContract interface {
	StartSettlement(context.Context, *gorm.DB, *Distribution) error
	StartMinting(context.Context, *gorm.DB, *Distribution) error
	Cancel(context.Context, *gorm.DB, *Distribution) error
	UpdateSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error
	UpdateMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error
}

type Contract struct {
	cfg        *config.Config
	flowClient *client.Client
}

func NewContract(cfg *config.Config, flowClient *client.Client) *Contract {
	return &Contract{cfg, flowClient}
}

func (c *Contract) StartSettlement(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	if err := dist.SetSettling(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	packs, err := GetDistributionPacks(db, dist.ID)
	if err != nil {
		return err
	}

	collectibles := make(Collectibles, 0)
	for _, pack := range packs {
		collectibles = append(collectibles, pack.Collectibles...)
	}
	sort.Sort(collectibles)

	settlementCollectibles := make([]SettlementCollectible, len(collectibles))
	for i, c := range collectibles {
		settlementCollectibles[i] = SettlementCollectible{
			FlowID:            c.FlowID,
			ContractReference: c.ContractReference,
			Settled:           false,
		}
	}

	settlement := Settlement{
		DistributionID:   dist.ID,
		Total:            uint(len(collectibles)),
		EscrowAddress:    common.FlowAddressFromString("f3fcd2c1a78f5eee"), // TODO (latenssi): proper escrow address
		LastCheckedBlock: latestBlock.Height - 1,
		Collectibles:     settlementCollectibles,
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err
	}

	// TODO (latenssi)
	// - Send a request to PDS Contract to start withdrawing of collectible NFTs to Contract account
	// - Timeout? Cancel?

	return nil
}

func (c *Contract) StartMinting(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	if err := dist.SetMinting(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	// Add CirculatingPackContracts to database
	for _, b := range dist.PackTemplate.Buckets {
		cpc := CirculatingPackContract{
			Name:             b.CollectibleReference.Name,
			Address:          b.CollectibleReference.Address,
			LastCheckedBlock: latestBlock.Height - 1,
		}
		// TODO (latenssi): get from db instead of failing on duplicate index?
		if err := InsertCirculatingPackContract(db, &cpc); err != nil {
			// TODO (latenssi): once duplicate key error handling is done, return error from here
			fmt.Printf("error while inserting CirculatingPackContract: %s\n", err)
		}
	}

	packs, err := GetDistributionPacks(db, dist.ID)
	if err != nil {
		return err
	}

	minting := Minting{
		DistributionID: dist.ID,
		Distribution:   *dist,

		State:  common.MintingStateStarted,
		Minted: 0,
		Total:  uint(len(packs)),

		LastCheckedBlock: latestBlock.Height - 1,
	}

	if err := InsertMinting(db, &minting); err != nil {
		return err
	}

	commitmentHashes := make([]cadence.Value, len(packs))
	for i, p := range packs {
		commitmentHashes[i] = cadence.NewString(p.CommitmentHash.String())
	}

	commitmentHashesArray := cadence.NewArray(commitmentHashes)

	// TODO (latenssi)
	// - call pds contract to start minting (provide commitmentHashes)
	// - Timeout? Cancel?

	g := gwtf.NewGoWithTheFlow([]string{"./flow.json"}, "emulator", false, 3)

	mintPackNFT := "./cadence-transactions/pds/mint_packNFT.cdc"
	mintPackNFTCode := util.ParseCadenceTemplate(mintPackNFT)
	_, err = g.TransactionFromFile(mintPackNFT, mintPackNFTCode).
		SignProposeAndPayAs("pds").
		UInt64Argument(uint64(dist.DistID.Int64)).
		Argument(commitmentHashesArray).
		AccountArgument("issuer").
		RunE()
	if err != nil {
		return err
	}

	return nil
}

func (c *Contract) Cancel(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	if err := dist.SetCancelled(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	// TODO (latenssi)

	return nil
}

func (c *Contract) UpdateSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	settlement, err := GetDistributionSettlement(db, dist.ID)
	if err != nil {
		return err
	}

	groupedMissing, err := MissingCollectibles(db, settlement.ID)
	if err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := settlement.LastCheckedBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		// Nothing to update
		return nil
	}

	for reference, missing := range groupedMissing {
		arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
			Type:        fmt.Sprintf("%s.Deposit", reference),
			StartHeight: start,
			EndHeight:   end,
		})
		if err != nil {
			return err
		}

		for _, be := range arr {
			for _, e := range be.Events {
				flowID, err := common.FlowIDFromCadence(e.Value.Fields[0])
				if err != nil {
					fmt.Println(err)
					continue
				}

				address, err := common.FlowAddressFromCadence(e.Value.Fields[1])
				if err != nil {
					fmt.Println(err)
					continue
				}

				if address == settlement.EscrowAddress {
					if i, ok := missing.ContainsID(flowID); ok {
						missing[i].Settled = true
						if err := UpdateSettlementCollectible(db, &missing[i]); err != nil {
							return err
						}
						settlement.Settled++
					}
				}
			}
		}
	}

	settlement.LastCheckedBlock = end

	if settlement.Settled >= settlement.Total {
		settlement.State = common.SettlementStateDone
	}

	if err := UpdateSettlement(db, settlement); err != nil {
		return err
	}

	if settlement.State == common.SettlementStateDone && dist.State == common.DistributionStateSettling {
		dist.State = common.DistributionStateSettled
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}
	}

	return nil
}

func (c *Contract) UpdateMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	minting, err := GetDistributionMinting(db, dist.ID)
	if err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := minting.LastCheckedBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		// Nothing to update
		return nil
	}

	reference := dist.PackTemplate.PackReference.String()

	arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
		Type:        fmt.Sprintf("%s.Mint", reference),
		StartHeight: start,
		EndHeight:   end,
	})
	if err != nil {
		return err
	}

	for _, be := range arr {
		for _, e := range be.Events {
			flowID, err := common.FlowIDFromCadence(e.Value.Fields[0])
			if err != nil {
				fmt.Println(err)
				continue
			}

			commitmentHash, err := common.BinaryValueFromCadence(e.Value.Fields[1])
			if err != nil {
				fmt.Println(err)
				continue
			}

			pack, err := GetDistributionPackByCommitmentHash(db, dist.ID, commitmentHash)
			if err != nil {
				fmt.Printf("unable find pack with commitmentHash %v\n", commitmentHash.String())
				continue
			}

			if !pack.FlowID.Valid {
				// FlowID was previously null, so this is a new pack NFT
				pack.FlowID = flowID
				if err := UpdatePack(db, pack); err != nil {
					return err
				}
				minting.Minted++
			}
		}
	}

	minting.LastCheckedBlock = end

	if minting.Minted >= minting.Total {
		minting.State = common.MintingStateDone
	}

	if err := UpdateMinting(db, minting); err != nil {
		return err
	}

	if minting.State == common.MintingStateDone && dist.State == common.DistributionStateMinting {
		dist.State = common.DistributionStateComplete
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}
	}

	return nil
}
