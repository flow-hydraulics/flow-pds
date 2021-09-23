package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"gorm.io/gorm"
)

type IContract interface {
	StartSettlement(context.Context, *gorm.DB, *Distribution) error
	StartMinting(context.Context, *gorm.DB, *Distribution) error
	Cancel(context.Context, *gorm.DB, *Distribution) error
	UpdateSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error
	UpdateMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error
	UpdateCirculatingPack(ctx context.Context, db *gorm.DB, cpc *CirculatingPackContract) error
}

// TODO (latenssi):
// - Timeout for settling and minting?
// - Cancel?

type Contract struct {
	cfg        *config.Config
	flowClient *client.Client
	account    *flow_helpers.Account
}

func NewContract(cfg *config.Config, flowClient *client.Client) *Contract {
	pdsAccount := flow_helpers.GetAccount(
		flow.HexToAddress(cfg.AdminAddress),
		cfg.AdminPrivateKey,
		[]int{0}, // TODO (latenssi): more key indexes
	)
	return &Contract{cfg, flowClient, pdsAccount}
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
			IsSettled:         false,
		}
	}

	settlement := Settlement{
		DistributionID:   dist.ID,
		CurrentCount:     0,
		TotalCount:       uint(len(collectibles)),
		LastCheckedBlock: latestBlock.Height - 1,
		// TODO (latenssi): Can we assume the admin is always the escrow?
		EscrowAddress: common.FlowAddressFromString(c.cfg.AdminAddress),
		Collectibles:  settlementCollectibles,
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err
	}

	flowIDs := make([]cadence.Value, len(collectibles))
	for i, c := range collectibles {
		flowIDs[i] = cadence.UInt64(c.FlowID.Int64)
	}

	arguments := []cadence.Value{
		cadence.UInt64(dist.DistID.Int64),
		cadence.NewArray(flowIDs),
	}

	if err := flow_helpers.SendTransactionAs(
		ctx,
		c.flowClient,
		c.account,
		latestBlock,
		arguments,
		"./cadence-transactions/pds/settle_exampleNFT.cdc",
	); err != nil {
		return err
	}

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

	// Add CirculatingPackContract to database
	cpc := CirculatingPackContract{
		Name:             dist.PackTemplate.PackReference.Name,
		Address:          dist.PackTemplate.PackReference.Address,
		LastCheckedBlock: latestBlock.Height - 1,
	}

	// Try to find one
	if _, err := GetCirculatingPackContract(db, cpc.Name, cpc.Address); err != nil {
		// Insert new if not found
		if err := InsertCirculatingPackContract(db, &cpc); err != nil {
			return err
		}
	}

	packs, err := GetDistributionPacks(db, dist.ID)
	if err != nil {
		return err
	}

	minting := Minting{
		DistributionID:   dist.ID,
		CurrentCount:     0,
		TotalCount:       uint(len(packs)),
		LastCheckedBlock: latestBlock.Height - 1,
	}

	if err := InsertMinting(db, &minting); err != nil {
		return err
	}

	commitmentHashes := make([]cadence.Value, len(packs))
	for i, p := range packs {
		commitmentHashes[i] = cadence.NewString(p.CommitmentHash.String())
	}

	arguments := []cadence.Value{
		cadence.UInt64(dist.DistID.Int64),
		cadence.NewArray(commitmentHashes),
		cadence.Address(dist.Issuer),
	}

	if err := flow_helpers.SendTransactionAs(
		ctx,
		c.flowClient,
		c.account,
		latestBlock,
		arguments,
		"./cadence-transactions/pds/mint_packNFT.cdc",
	); err != nil {
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

	groupedMissing, err := MissingCollectibles(db, settlement.ID)
	if err != nil {
		return err
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
					return err
				}

				address, err := common.FlowAddressFromCadence(e.Value.Fields[1])
				if err != nil {
					return err
				}

				if address == settlement.EscrowAddress {
					if i, ok := missing.ContainsID(flowID); ok {
						if err := missing[i].SetSettled(); err != nil {
							return err
						}
						if err := UpdateSettlementCollectible(db, &missing[i]); err != nil {
							return err
						}
						settlement.IncrementCount()
					}
				}
			}
		}
	}

	if settlement.IsComplete() {
		if err := dist.SetSettled(); err != nil {
			return err
		}
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}
	}

	settlement.LastCheckedBlock = end

	if err := UpdateSettlement(db, settlement); err != nil {
		return err
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
				return err
			}

			commitmentHash, err := common.BinaryValueFromCadence(e.Value.Fields[1])
			if err != nil {
				return err
			}

			pack, err := GetMintingPack(db, dist.ID, commitmentHash)
			if err != nil {
				return fmt.Errorf("unable find a minting pack with commitmentHash %v", commitmentHash.String())
			}

			if err := pack.Seal(flowID); err != nil {
				return err
			}

			if err := UpdatePack(db, pack); err != nil {
				return err
			}

			minting.IncrementCount()
		}
	}

	if minting.IsComplete() {
		if err := dist.SetComplete(); err != nil {
			return err
		}
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}
	}

	minting.LastCheckedBlock = end

	if err := UpdateMinting(db, minting); err != nil {
		return err
	}

	return nil
}

func (c *Contract) UpdateCirculatingPack(ctx context.Context, db *gorm.DB, cpc *CirculatingPackContract) error {
	eventNames := []string{
		"RevealRequest",
		"OpenPackRequest",
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := cpc.LastCheckedBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		return nil
	}

	for _, eventName := range eventNames {
		arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
			Type:        cpc.EventName(eventName),
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
					return err
				}

				pack, err := GetPackByContractAndFlowID(db, AddressLocation{Name: cpc.Name, Address: cpc.Address}, flowID)
				if err != nil {
					return err
				}

				switch eventName {
				case "RevealRequest":
					fmt.Println("Reveal pack:", pack.ID)
					if err := pack.Reveal(); err != nil {
						return err
					}
					if err := UpdatePack(db, pack); err != nil {
						return err
					}
				case "OpenPackRequest":
					fmt.Println("Open pack:", pack.ID)
					if err := pack.Open(); err != nil {
						return err
					}
					if err := UpdatePack(db, pack); err != nil {
						return err
					}
				}
			}
		}
	}

	cpc.LastCheckedBlock = end

	if err := UpdateCirculatingPackContract(db, cpc); err != nil {
		return err
	}

	return nil
}
