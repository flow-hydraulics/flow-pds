package app

import (
	"context"
	"fmt"
	"sort"

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

// Onchain eventnames
const (
	REVEAL_REQUEST = "RevealRequest"
	REVEALED       = "Revealed"
	OPEN_REQUEST   = "OpenRequest"
	OPENED         = "Opened"
)

// Going much above these will cause the transactions to use more than 9999 gas
const (
	SETTLE_BATCH_SIZE = 40
	MINT_BATCH_SIZE   = 40
)

const (
	// TODO (latenssi): this only handles ExampleNFTs currently
	SETTLE_SCRIPT       = "./cadence-transactions/pds/settle_exampleNFT.cdc"
	MINT_SCRIPT         = "./cadence-transactions/pds/mint_packNFT.cdc"
	REVEAL_SCRIPT       = "./cadence-transactions/pds/reveal_packNFT.cdc"
	OPEN_SCRIPT         = "./cadence-transactions/pds/open_packNFT.cdc"
	UPDATE_STATE_SCRIPT = "./cadence-transactions/pds/update_dist_state.cdc"
)

const (
	MAX_EVENTS_PER_CHECK = 100
)

// Contract handles all the onchain logic and functions
type Contract struct {
	cfg        *config.Config
	logger     *log.Logger
	flowClient *client.Client
	account    *flow_helpers.Account
}

func minInt(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

func NewContract(cfg *config.Config, logger *log.Logger, flowClient *client.Client) *Contract {
	pdsAccount := flow_helpers.GetAccount(
		flow.HexToAddress(cfg.AdminAddress),
		cfg.AdminPrivateKey,
		cfg.AdminPrivateKeyType,
		cfg.AdminPrivateKeyIndexes,
	)
	return &Contract{cfg, logger, flowClient, pdsAccount}
}

// StartSettlement sets the given distributions state to 'settling' and starts the settlement
// phase onchain.
// It lists all collectible NFTs in the distribution and creates batches
// of 'SETTLE_BATCH_SIZE' from them.
// It then creates and stores the settlement Flow transactions (PDS account withdraw from issuer to escrow) in
// database to be later processed by a poller.
// Batching needs to be done to control the transaction size.
func (c *Contract) StartSettlement(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	c.logger.WithFields(log.Fields{
		"method": "StartSettlement",
		"ID":     dist.ID,
	}).Info("Start settlement")

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
		EscrowAddress:  common.FlowAddressFromString(c.cfg.AdminAddress),
		Collectibles:   settlementCollectibles,
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err
	}

	txScript := util.ParseCadenceTemplate(SETTLE_SCRIPT)

	batchIndex := 0
	for {
		begin := batchIndex * SETTLE_BATCH_SIZE
		end := minInt((batchIndex+1)*SETTLE_BATCH_SIZE, len(settlementCollectibles))

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
			cadence.UInt64(dist.FlowID.Int64),
			cadence.NewArray(flowIDs),
		}

		t, err := transactions.NewTransaction(SETTLE_SCRIPT, txScript, arguments)
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

// StartMinting sets the given distributions state to 'minting' and starts the settlement
// phase onchain.
// It creates a CirculatingPackContract to allow onchain monitoring
// (listening for events) of any pack that has been put to circulation.
// It then lists all Pack NFTs in the distribution and creates batches
// of 'MINT_BATCH_SIZE' from them.
// It then creates and stores the minting Flow transactions in database to be
// later processed by a poller.
// Batching needs to be done to control the transaction size.
func (c *Contract) StartMinting(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	c.logger.WithFields(log.Fields{
		"method": "StartMinting",
		"ID":     dist.ID,
	}).Info("Start minting")

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
			c.logger.WithFields(log.Fields{
				"method":          "StartMinting",
				"ID":              dist.ID,
				"existingID":      existing.ID,
				"existingName":    existing.Name,
				"existingAddress": existing.Address,
				"existingStart":   existing.StartAtBlock,
				"newID":           cpc.ID,
				"newName":         cpc.Name,
				"newAddress":      cpc.Address,
				"newStart":        cpc.StartAtBlock,
			}).Warn("CirculatingPackContract with higher block height found in database, should not happen in production")
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

	txScript := util.ParseCadenceTemplate(MINT_SCRIPT)

	batchIndex := 0
	for {
		begin := batchIndex * MINT_BATCH_SIZE
		end := minInt((batchIndex+1)*MINT_BATCH_SIZE, len(packs))

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
			cadence.UInt64(dist.FlowID.Int64),
			cadence.NewArray(commitmentHashes),
			cadence.Address(dist.Issuer),
		}

		t, err := transactions.NewTransaction(MINT_SCRIPT, txScript, arguments)
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

// Abort a distribution
func (c *Contract) Abort(ctx context.Context, db *gorm.DB, dist *Distribution) error {

	c.logger.WithFields(log.Fields{
		"method": "Abort",
		"ID":     dist.ID,
	}).Info("Abort")

	if err := dist.SetInvalid(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	// Update distribution state onchain
	txScript := util.ParseCadenceTemplate(UPDATE_STATE_SCRIPT)
	arguments := []cadence.Value{
		cadence.UInt64(dist.FlowID.Int64),
		cadence.UInt8(1),
	}
	t, err := transactions.NewTransaction(UPDATE_STATE_SCRIPT, txScript, arguments)
	if err != nil {
		return err
	}

	if err := t.Save(db); err != nil {
		return err
	}

	c.logger.WithFields(log.Fields{
		"method":   "Abort",
		"ID":       dist.ID,
		"state":    1,
		"stateStr": "invalid",
	}).Info("Distribution state update transaction saved")

	return nil
}

// UpdateSettlementStatus polls for 'Deposit' events regarding the given distributions
// collectible NFTs.
// It updates the settelement status in database accordingly.
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
	end := min(latestBlock.Height, start+MAX_EVENTS_PER_CHECK)

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
		// TODO: consider updating the distribution separately
		if err := dist.SetSettled(); err != nil {
			return err
		}
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}

		c.logger.WithFields(log.Fields{
			"method":     "UpdateSettlementStatus",
			"ID":         dist.ID,
			"blockStart": start,
			"blockEnd":   end,
		}).Info("Settlement complete")
	}

	settlement.StartAtBlock = end

	if err := UpdateSettlement(db, settlement); err != nil {
		return err
	}

	return nil
}

// UpdateMintingStatus polls for 'Mint' events regarding the given distributions
// Pack NFTs.
// It updates the minting status in database accordingly.
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
	end := min(latestBlock.Height, start+MAX_EVENTS_PER_CHECK)

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
		// Distribution is now complete

		// TODO: consider updating the distribution separately
		if err := dist.SetComplete(); err != nil {
			return err
		}
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}

		c.logger.WithFields(log.Fields{
			"method":     "UpdateMintingStatus",
			"ID":         dist.ID,
			"blockStart": start,
			"blockEnd":   end,
		}).Info("Minting complete")

		// Update distribution state onchain
		txScript := util.ParseCadenceTemplate(UPDATE_STATE_SCRIPT)
		arguments := []cadence.Value{
			cadence.UInt64(dist.FlowID.Int64),
			cadence.UInt8(2),
		}
		t, err := transactions.NewTransaction(UPDATE_STATE_SCRIPT, txScript, arguments)
		if err != nil {
			return err
		}

		if err := t.Save(db); err != nil {
			return err
		}

		c.logger.WithFields(log.Fields{
			"method":   "UpdateMintingStatus",
			"ID":       dist.ID,
			"state":    2,
			"stateStr": "complete",
		}).Info("Distribution state update transaction saved")
	}

	minting.StartAtBlock = end

	if err := UpdateMinting(db, minting); err != nil {
		return err
	}

	return nil
}

// UpdateCirculatingPack polls for 'REVEAL_REQUEST' and 'OPEN_REQUEST' events
// regarding the given CirculatingPackContract.
// It handles each event by creating and storing an appropriate Flow transaction
// in database to be later processed by a poller.
func (c *Contract) UpdateCirculatingPack(ctx context.Context, db *gorm.DB, cpc *CirculatingPackContract) error {
	c.logger.WithFields(log.Fields{
		"method": "UpdateCirculatingPack",
		"cpcID":  cpc.ID,
	}).Trace("Update circulating pack")

	eventNames := []string{
		REVEAL_REQUEST,
		REVEALED,
		OPEN_REQUEST,
		OPENED,
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := cpc.StartAtBlock + 1
	end := min(latestBlock.Height, start+MAX_EVENTS_PER_CHECK)

	if start > end {
		c.logger.WithFields(log.Fields{
			"method":     "UpdateCirculatingPack",
			"cpcID":      cpc.ID,
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
					"cpcID":      cpc.ID,
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
				case REVEAL_REQUEST: // Reveal a pack
					if err := pack.RevealRequestHandled(); err != nil {
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
						cadence.UInt64(distribution.FlowID.Int64),
						cadence.UInt64(pack.FlowID.Int64),
						cadence.NewArray(collectibleContractAddresses),
						cadence.NewArray(collectibleContractNames),
						cadence.NewArray(collectibleIDs),
						cadence.String(pack.Salt.String()),
						cadence.NewOptional(nil),
					}

					txScript := util.ParseCadenceTemplate(REVEAL_SCRIPT)
					t, err := transactions.NewTransaction(REVEAL_SCRIPT, txScript, arguments)
					if err != nil {
						return err
					}

					if err := t.Save(db); err != nil {
						return err
					}

					c.logger.WithFields(log.Fields{
						"method":     "UpdateCirculatingPack",
						"cpcID":      cpc.ID,
						"ID":         distribution.ID,
						"packFlowID": flowID,
					}).Info("Pack reveal transaction created")

				case REVEALED:
					if err := pack.Reveal(); err != nil {
						return err
					}
					if err := UpdatePack(db, pack); err != nil {
						return err
					}
				case OPEN_REQUEST: // Open a pack
					if err := pack.OpenRequestHandled(); err != nil {
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
						cadence.UInt64(distribution.FlowID.Int64),
						cadence.UInt64(pack.FlowID.Int64),
						cadence.NewArray(collectibleContractAddresses),
						cadence.NewArray(collectibleContractNames),
						cadence.NewArray(collectibleIDs),
						cadence.Address(owner),
					}

					txScript := util.ParseCadenceTemplate(OPEN_SCRIPT)
					t, err := transactions.NewTransaction(OPEN_SCRIPT, txScript, arguments)
					if err != nil {
						return err
					}

					if err := t.Save(db); err != nil {
						return err
					}

					c.logger.WithFields(log.Fields{
						"method":     "UpdateCirculatingPack",
						"cpcID":      cpc.ID,
						"ID":         distribution.ID,
						"packFlowID": flowID,
					}).Info("Pack open transaction created")
				case OPENED:
					if err := pack.Open(); err != nil {
						return err
					}
					if err := UpdatePack(db, pack); err != nil {
						return err
					}
				}

				c.logger.WithFields(log.Fields{
					"method":     "UpdateCirculatingPack",
					"cpcID":      cpc.ID,
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
