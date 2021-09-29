package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	REVEAL_REQUEST = "RevealRequest"
	OPEN_REQUEST   = "OpenRequest"
)

const TX_TIMEOUT = time.Second * 10

// TODO (latenssi):
// - Timeout for settling and minting?
// - Cancel?

type Contract struct {
	cfg        *config.Config
	flowClient *client.Client
	account    *flow_helpers.Account
	logger     *log.Logger
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
	logger := log.New()
	logger.SetLevel(log.InfoLevel) // TODO
	return &Contract{cfg, flowClient, pdsAccount, logger}
}

func (c *Contract) StartSettlement(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	c.logger.WithFields(log.Fields{
		"method": "StartSettlement",
		"ID":     dist.ID,
	}).Trace("Start settlement")

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
		DistributionID: dist.ID,
		CurrentCount:   0,
		TotalCount:     uint(len(collectibles)),
		StartAtBlock:   latestBlock.Height - 1,
		// TODO (latenssi): Can we assume the admin is always the escrow?
		EscrowAddress: common.FlowAddressFromString(c.cfg.AdminAddress),
		Collectibles:  settlementCollectibles,
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err
	}

	// TODO (latenssi): this only handles ExampleNFTs currently
	txScript := util.ParseCadenceTemplate("./cadence-transactions/pds/settle_exampleNFT.cdc")

	batchSize := 40
	batchIndex := 0
	for {
		begin := batchIndex * batchSize
		end := minInt((batchIndex+1)*batchSize, len(settlementCollectibles))

		if begin > end {
			break
		}

		c.logger.WithFields(log.Fields{
			"method":      "StartSettlement",
			"batchNumber": batchIndex + 1,
			"batchBegin":  begin,
			"batchEnd":    end,
		}).Debug("Initiating settle transaction")

		batch := settlementCollectibles[begin:end]

		flowIDs := make([]cadence.Value, len(batch))
		for i, c := range batch {
			flowIDs[i] = cadence.UInt64(c.FlowID.Int64)
		}

		arguments := []cadence.Value{
			cadence.UInt64(dist.DistID.Int64),
			cadence.NewArray(flowIDs),
		}

		t, err := transactions.NewTransaction(txScript, arguments)
		if err != nil {
			return err
		}

		if err := t.Save(db); err != nil {
			return err
		}

		c.logger.WithFields(log.Fields{
			"method":      "StartMinting",
			"batchNumber": batchIndex + 1,
			"batchBegin":  begin,
			"batchEnd":    end,
		}).Trace("Settle transaction saved")

		batchIndex++
	}

	return nil
}

func (c *Contract) StartMinting(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	c.logger.WithFields(log.Fields{
		"method": "StartMinting",
		"ID":     dist.ID,
	}).Trace("Start minting")

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
		Name:         dist.PackTemplate.PackReference.Name,
		Address:      dist.PackTemplate.PackReference.Address,
		StartAtBlock: latestBlock.Height - 1,
	}

	// Try to find one
	if existing, err := GetCirculatingPackContract(db, cpc.Name, cpc.Address); err != nil {
		// Insert new if not found
		if err := InsertCirculatingPackContract(db, &cpc); err != nil {
			return err
		}
	} else {
		if cpc.StartAtBlock < existing.StartAtBlock {
			// Situation where a new cpc has lower blockheight (LastCheckedBlock) than an old one.
			// Should not happen in production but can happen in tests.
			fmt.Println("CirculatingPackContract with higher block height found in database, should not happen in production")
			existing.StartAtBlock = cpc.StartAtBlock
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
		DistributionID: dist.ID,
		CurrentCount:   0,
		TotalCount:     uint(len(packs)),
		StartAtBlock:   latestBlock.Height - 1,
	}

	if err := InsertMinting(db, &minting); err != nil {
		return err
	}

	txScript := util.ParseCadenceTemplate("./cadence-transactions/pds/mint_packNFT.cdc")

	batchSize := 40
	batchIndex := 0
	for {
		begin := batchIndex * batchSize
		end := minInt((batchIndex+1)*batchSize, len(packs))

		if begin > end {
			break
		}

		c.logger.WithFields(log.Fields{
			"method":      "StartMinting",
			"batchNumber": batchIndex + 1,
			"batchBegin":  begin,
			"batchEnd":    end,
		}).Debug("Initiating mint transaction")

		batch := packs[begin:end]

		commitmentHashes := make([]cadence.Value, len(batch))
		for i, p := range batch {
			commitmentHashes[i] = cadence.NewString(p.CommitmentHash.String())
		}

		arguments := []cadence.Value{
			cadence.UInt64(dist.DistID.Int64),
			cadence.NewArray(commitmentHashes),
			cadence.Address(dist.Issuer),
		}

		t, err := transactions.NewTransaction(txScript, arguments)
		if err != nil {
			return err
		}

		if err := t.Save(db); err != nil {
			return err
		}

		c.logger.WithFields(log.Fields{
			"method":      "StartMinting",
			"batchNumber": batchIndex + 1,
			"batchBegin":  begin,
			"batchEnd":    end,
		}).Trace("Mint transaction saved")

		batchIndex++
	}

	return nil
}

func (c *Contract) Cancel(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	c.logger.WithFields(log.Fields{
		"method": "Cancel",
		"ID":     dist.ID,
	}).Trace("Cancel")

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
	c.logger.WithFields(log.Fields{
		"method": "UpdateSettlementStatus",
		"ID":     dist.ID,
	}).Trace("Update settlement status")

	settlement, err := GetDistributionSettlement(db, dist.ID)
	if err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := settlement.StartAtBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		// Nothing to update
		c.logger.WithFields(log.Fields{
			"method":     "UpdateSettlementStatus",
			"ID":         dist.ID,
			"blockStart": start,
			"blockEnd":   end,
		}).Trace("No blocks to handle")
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
				c.logger.WithFields(log.Fields{
					"method":     "UpdateSettlementStatus",
					"ID":         dist.ID,
					"blockStart": start,
					"blockEnd":   end,
					"eventType":  e.Type,
				}).Debug("Handling event")

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

				c.logger.WithFields(log.Fields{
					"method":     "UpdateSettlementStatus",
					"ID":         dist.ID,
					"blockStart": start,
					"blockEnd":   end,
					"eventType":  e.Type,
				}).Trace("Handling event complete")
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

	settlement.StartAtBlock = end

	if err := UpdateSettlement(db, settlement); err != nil {
		return err
	}

	return nil
}

func (c *Contract) UpdateMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	c.logger.WithFields(log.Fields{
		"method": "UpdateMintingStatus",
		"ID":     dist.ID,
	}).Trace("Update minting status")

	minting, err := GetDistributionMinting(db, dist.ID)
	if err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := minting.StartAtBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		// Nothing to update
		c.logger.WithFields(log.Fields{
			"method":     "UpdateMintingStatus",
			"ID":         dist.ID,
			"blockStart": start,
			"blockEnd":   end,
		}).Trace("No blocks to handle")
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
			c.logger.WithFields(log.Fields{
				"method":     "UpdateMintingStatus",
				"ID":         dist.ID,
				"blockStart": start,
				"blockEnd":   end,
				"eventType":  e.Type,
			}).Debug("Handling event")

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
				c.logger.WithFields(log.Fields{
					"method":         "UpdateMintingStatus",
					"ID":             dist.ID,
					"blockStart":     start,
					"blockEnd":       end,
					"eventType":      e.Type,
					"flowID":         flowID,
					"commitmentHash": commitmentHash,
					"error":          "unable to find matching pack from database",
				}).Errorf("Error while handling event")
				continue
			}

			if err := pack.Seal(flowID); err != nil {
				return err
			}

			if err := UpdatePack(db, pack); err != nil {
				return err
			}

			minting.IncrementCount()

			c.logger.WithFields(log.Fields{
				"method":     "UpdateMintingStatus",
				"ID":         dist.ID,
				"blockStart": start,
				"blockEnd":   end,
				"eventType":  e.Type,
			}).Trace("Handling event complete")
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

	minting.StartAtBlock = end

	if err := UpdateMinting(db, minting); err != nil {
		return err
	}

	return nil
}

func (c *Contract) UpdateCirculatingPack(ctx context.Context, db *gorm.DB, cpc *CirculatingPackContract) error {
	c.logger.WithFields(log.Fields{
		"method": "UpdateCirculatingPack",
		"ID":     cpc.ID,
	}).Trace("Update circulating pack")

	eventNames := []string{
		REVEAL_REQUEST,
		OPEN_REQUEST,
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := cpc.StartAtBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		c.logger.WithFields(log.Fields{
			"method":     "UpdateCirculatingPack",
			"ID":         cpc.ID,
			"blockStart": start,
			"blockEnd":   end,
		}).Trace("No blocks to handle")
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
				c.logger.WithFields(log.Fields{
					"method":     "UpdateCirculatingPack",
					"ID":         cpc.ID,
					"blockStart": start,
					"blockEnd":   end,
					"eventType":  e.Type,
				}).Debug("Handling event")

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

					if err := UpdatePack(db, pack); err != nil {
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

					txScript := util.ParseCadenceTemplate("./cadence-transactions/pds/reveal_packNFT.cdc")
					t, err := transactions.NewTransaction(txScript, arguments)
					if err != nil {
						return err
					}

					if err := t.Save(db); err != nil {
						return err
					}

				case OPEN_REQUEST:
					if err := pack.Open(); err != nil {
						return err
					}

					if err := UpdatePack(db, pack); err != nil {
						return err
					}

					// Get the owner of the pack from the transaction that emitted the open request event
					tx, err := c.flowClient.GetTransaction(ctx, e.TransactionID)
					if err != nil {
						return err
					}
					owner := tx.Authorizers[0]

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

					txScript := util.ParseCadenceTemplate("./cadence-transactions/pds/open_packNFT.cdc")
					t, err := transactions.NewTransaction(txScript, arguments)
					if err != nil {
						return err
					}

					if err := t.Save(db); err != nil {
						return err
					}
				}

				c.logger.WithFields(log.Fields{
					"method":     "UpdateCirculatingPack",
					"ID":         cpc.ID,
					"blockStart": start,
					"blockEnd":   end,
					"eventType":  e.Type,
				}).Trace("Handling event complete")
			}
		}
	}

	cpc.StartAtBlock = end

	if err := UpdateCirculatingPackContract(db, cpc); err != nil {
		return err
	}

	return nil
}
