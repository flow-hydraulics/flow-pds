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

const (
	REVEAL_REQUEST = "RevealRequest"
	OPEN_REQUEST   = "OpenRequest"
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

func minInt(a int, b int) int {
	if a > b {
		return b
	}
	return a
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

	// TODO (latenssi): clean up batching
	batchSize := 40
	batchIndex := 0
	for {
		begin := batchIndex * batchSize
		end := minInt((batchIndex+1)*batchSize, len(flowIDs))

		if begin > end {
			break
		}

		batch := flowIDs[begin:end]

		arguments := []cadence.Value{
			cadence.UInt64(dist.DistID.Int64),
			cadence.NewArray(batch),
		}

		latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
		if err != nil {
			return err
		}

		// TODO (latenssi): this only handles ExampleNFTs currently
		tx, err := flow_helpers.PrepareTransactionAs(
			ctx,
			c.flowClient,
			c.account,
			latestBlock,
			arguments,
			"./cadence-transactions/pds/settle_exampleNFT.cdc",
		)

		if err != nil {
			return err
		}

		if err := c.flowClient.SendTransaction(ctx, *tx); err != nil {
			return err
		}

		batchIndex++
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
	if existing, err := GetCirculatingPackContract(db, cpc.Name, cpc.Address); err != nil {
		// Insert new if not found
		if err := InsertCirculatingPackContract(db, &cpc); err != nil {
			return err
		}
	} else {
		if cpc.LastCheckedBlock < existing.LastCheckedBlock {
			// Situation where a new cpc has lower blockheight (LastCheckedBlock) than an old one.
			// Should not happen in production but can happen in tests.
			fmt.Println("CirculatingPackContract with higher block height found in database, should not happen in production")
			existing.LastCheckedBlock = cpc.LastCheckedBlock
			if err := UpdateCirculatingPackContract(db, existing); err != nil {
				return err
			}
			cpc = *existing
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

	// TODO (latenssi): clean up batching
	batchSize := 40
	batchIndex := 0
	for {
		begin := batchIndex * batchSize
		end := minInt((batchIndex+1)*batchSize, len(commitmentHashes))

		if begin > end {
			break
		}

		batch := commitmentHashes[begin:end]

		arguments := []cadence.Value{
			cadence.UInt64(dist.DistID.Int64),
			cadence.NewArray(batch),
			cadence.Address(dist.Issuer),
		}

		latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
		if err != nil {
			return err
		}

		tx, err := flow_helpers.PrepareTransactionAs(
			ctx,
			c.flowClient,
			c.account,
			latestBlock,
			arguments,
			"./cadence-transactions/pds/mint_packNFT.cdc",
		)

		if err != nil {
			return err
		}

		if err := c.flowClient.SendTransaction(ctx, *tx); err != nil {
			return err
		}

		batchIndex++
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

			pack, err := GetMintingPack(db, commitmentHash)
			if err != nil {
				fmt.Printf("received event: %s but unable find a pack with a missing flowId and with commitmentHash %v\n", e, commitmentHash.String())
				continue
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
		REVEAL_REQUEST,
		OPEN_REQUEST,
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

	contractRef := AddressLocation{Name: cpc.Name, Address: cpc.Address}

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
				// TODO (latenssi): consider separating this one db transaction ("db")

				flowID, err := common.FlowIDFromCadence(e.Value.Fields[0])
				if err != nil {
					return err
				}

				pack, err := GetPackByContractAndFlowID(db, contractRef, flowID)
				if err != nil {
					return err
				}

				switch eventName {
				case REVEAL_REQUEST:
					if err := pack.Reveal(); err != nil {
						return err
					}

					distribution, err := GetDistribution(db, pack.DistributionID)
					if err != nil {
						return err
					}

					collectibleCount := len(pack.Collectibles)

					collectibleContractAddresses := make([]cadence.Value, collectibleCount)
					collectibleContractNames := make([]cadence.Value, collectibleCount)
					collectibleIDs := make([]cadence.Value, collectibleCount)

					for i, c := range pack.Collectibles {
						collectibleContractAddresses[i] = cadence.Address(c.ContractReference.Address)
						collectibleContractNames[i] = cadence.String(c.ContractReference.Name)
						collectibleIDs[i] = cadence.UInt64(c.FlowID.Int64)
					}

					arguments := []cadence.Value{
						cadence.UInt64(distribution.DistID.Int64),
						cadence.UInt64(pack.FlowID.Int64),
						cadence.NewArray(collectibleContractAddresses),
						cadence.NewArray(collectibleContractNames),
						cadence.NewArray(collectibleIDs),
						cadence.String(pack.Salt.String()),
					}

					tx, err := flow_helpers.PrepareTransactionAs(
						ctx,
						c.flowClient,
						c.account,
						latestBlock,
						arguments,
						"./cadence-transactions/pds/reveal_packNFT.cdc",
					)

					if err != nil {
						return err
					}

					if err := c.flowClient.SendTransaction(ctx, *tx); err != nil {
						return err
					}

					if err := UpdatePack(db, pack); err != nil {
						return err
					}

				case OPEN_REQUEST:
					if err := pack.Open(); err != nil {
						return err
					}

					// Get the owner of the pack from the transaction that emitted the open request event
					t, err := c.flowClient.GetTransaction(ctx, e.TransactionID)
					if err != nil {
						return err
					}
					owner := t.Authorizers[0]

					distribution, err := GetDistribution(db, pack.DistributionID)
					if err != nil {
						return err
					}

					collectibleIDs := make([]cadence.Value, len(pack.Collectibles))

					for i, c := range pack.Collectibles {
						collectibleIDs[i] = cadence.UInt64(c.FlowID.Int64)
					}

					arguments := []cadence.Value{
						cadence.UInt64(distribution.DistID.Int64),
						cadence.UInt64(pack.FlowID.Int64),
						cadence.NewArray(collectibleIDs),
						cadence.Address(owner),
					}

					tx, err := flow_helpers.PrepareTransactionAs(
						ctx,
						c.flowClient,
						c.account,
						latestBlock,
						arguments,
						"./cadence-transactions/pds/open_packNFT.cdc",
					)

					if err != nil {
						return err
					}

					if err := c.flowClient.SendTransaction(ctx, *tx); err != nil {
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
