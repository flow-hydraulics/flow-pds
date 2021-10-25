package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	"github.com/google/uuid"
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
	SETTLE_SCRIPT       = "./cadence-transactions/pds/settle.cdc"
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

func NewContract(cfg *config.Config, logger *log.Logger, flowClient *client.Client) (*Contract, error) {
	pdsAccount := flow_helpers.GetAccount(
		flow.HexToAddress(cfg.AdminAddress),
		cfg.AdminPrivateKey,
		cfg.AdminPrivateKeyType,
		cfg.AdminPrivateKeyIndexes,
	)
	flowAccount, err := flowClient.GetAccount(context.Background(), pdsAccount.Address)
	if err != nil {
		return nil, err
	}
	if len(flowAccount.Keys) < len(pdsAccount.KeyIndexes) {
		return nil, fmt.Errorf("too many key indexes given for admin account")
	}
	return &Contract{cfg, logger, flowClient, pdsAccount}, nil
}

// StartSettlement sets the given distributions state to 'settling' and starts the settlement
// phase onchain.
// It lists all collectible NFTs in the distribution and creates batches
// of 'SETTLE_BATCH_SIZE' from them.
// It then creates and stores the settlement Flow transactions (PDS account withdraw from issuer to escrow) in
// database to be later processed by a poller.
// Batching needs to be done to control the transaction size.
func (c *Contract) StartSettlement(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := c.logger.WithFields(log.Fields{
		"method":     "StartSettlement",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Info("Start settlement")

	// Make sure the distribution is in correct state
	if err := dist.SetSettling(); err != nil {
		return err // rollback
	}

	// Update the distribution in database
	if err := UpdateDistribution(db, dist); err != nil {
		return err // rollback
	}

	latestBlockHeader, err := c.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	packs, err := GetDistributionPacks(db, dist.ID)
	if err != nil {
		return err // rollback
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
		StartAtBlock:   latestBlockHeader.Height - 1,
		EscrowAddress:  common.FlowAddressFromString(c.cfg.PDSAddress),
		Collectibles:   settlementCollectibles,
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err // rollback
	}

	for contract, collectibles := range SettlementCollectibles(settlementCollectibles).GroupByContract() {
		txScript, err := flow_helpers.ParseCadenceTemplate(
			SETTLE_SCRIPT,
			&flow_helpers.CadenceTemplateVars{
				CollectibleNFTName:    contract.Name,
				CollectibleNFTAddress: contract.Address.String(),
			},
		)
		if err != nil {
			return err // rollback
		}

		batchIndex := 0
		for {
			begin := batchIndex * SETTLE_BATCH_SIZE
			end := minInt((batchIndex+1)*SETTLE_BATCH_SIZE, len(collectibles))

			if begin > end {
				break
			}

			batchLogger := logger.WithFields(log.Fields{
				"batchNumber": batchIndex + 1,
				"batchBegin":  begin,
				"batchEnd":    end,
			})

			batchLogger.Debug("Initiating settle transaction")

			batch := collectibles[begin:end]

			flowIDs := make([]cadence.Value, len(batch))
			for i, c := range batch {
				flowIDs[i] = cadence.UInt64(c.FlowID.Int64)
			}

			arguments := []cadence.Value{
				cadence.UInt64(dist.FlowID.Int64),
				cadence.NewArray(flowIDs),
			}

			t, err := transactions.NewTransactionWithDistributionID(SETTLE_SCRIPT, txScript, arguments, dist.ID)
			if err != nil {
				return err // rollback
			}

			if err := t.Save(db); err != nil {
				return err // rollback
			}

			batchLogger.Trace("Settle transaction saved")

			batchIndex++
		}
	}

	logger.Trace("Start settlement complete")

	return nil // commit
}

// StartMinting sets the given distributions state to 'minting' and starts the minting
// phase onchain.
// It creates a CirculatingPackContract to allow onchain monitoring
// (listening for events) of any pack that has been put to circulation.
// It then lists all Pack NFTs in the distribution and creates batches
// of 'MINT_BATCH_SIZE' from them.
// It then creates and stores the minting Flow transactions in database to be
// later processed by a poller.
// Batching needs to be done to control the transaction size.
func (c *Contract) StartMinting(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := c.logger.WithFields(log.Fields{
		"method":     "StartMinting",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Info("Start minting")

	// Make sure the distribution is in correct state
	if err := dist.SetMinting(); err != nil {
		return err // rollback
	}

	// Update the distribution in database
	if err := UpdateDistribution(db, dist); err != nil {
		return err // rollback
	}

	latestBlockHeader, err := c.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	// Init a CirculatingPackContract
	cpc := CirculatingPackContract{
		Name:         dist.PackTemplate.PackReference.Name,
		Address:      dist.PackTemplate.PackReference.Address,
		StartAtBlock: latestBlockHeader.Height - 1,
	}

	// Try to find an existing one (CirculatingPackContract)
	if existing, err := GetCirculatingPackContract(db, cpc.Name, cpc.Address); err != nil {
		// Insert the newly initialized if not found
		if err := InsertCirculatingPackContract(db, &cpc); err != nil {
			return err // rollback
		}
	} else { // err == nil, existing found
		if cpc.StartAtBlock < existing.StartAtBlock {
			// Situation where a new cpc has lower blockheight (LastCheckedBlock) than an old one.
			// Should not happen in production but can happen in tests.

			logger.WithFields(log.Fields{
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
				return err // rollback
			}

			// Use the existing one from now on
			cpc = *existing
		}
	}

	packs, err := GetDistributionPacks(db, dist.ID)
	if err != nil {
		return err // rollback
	}

	minting := Minting{
		DistributionID: dist.ID,
		CurrentCount:   0,
		TotalCount:     uint(len(packs)),
		StartAtBlock:   latestBlockHeader.Height - 1,
	}

	if err := InsertMinting(db, &minting); err != nil {
		return err // rollback
	}

	txScript, err := flow_helpers.ParseCadenceTemplate(
		MINT_SCRIPT,
		&flow_helpers.CadenceTemplateVars{
			PackNFTName:    dist.PackTemplate.PackReference.Name,
			PackNFTAddress: dist.PackTemplate.PackReference.Address.String(),
		},
	)
	if err != nil {
		return err // rollback
	}

	batchIndex := 0
	for {
		begin := batchIndex * MINT_BATCH_SIZE
		end := minInt((batchIndex+1)*MINT_BATCH_SIZE, len(packs))

		if begin > end {
			break
		}

		batchLogger := logger.WithFields(log.Fields{
			"batchNumber": batchIndex + 1,
			"batchBegin":  begin,
			"batchEnd":    end,
		})

		batchLogger.Debug("Initiating mint transaction")

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

		t, err := transactions.NewTransactionWithDistributionID(MINT_SCRIPT, txScript, arguments, dist.ID)
		if err != nil {
			return err // rollback
		}

		if err := t.Save(db); err != nil {
			return err // rollback
		}

		batchLogger.Trace("Mint transaction saved")

		batchIndex++
	}

	logger.Trace("Start minting complete")

	return nil // commit
}

// Abort a distribution
func (c *Contract) Abort(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := c.logger.WithFields(log.Fields{
		"method":     "Abort",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Info("Abort")

	// Make sure the distribution is in correct state
	if err := dist.SetInvalid(); err != nil {
		return err // rollback
	}

	// Update the distribution in database
	if err := UpdateDistribution(db, dist); err != nil {
		return err // rollback
	}

	// Update distribution state onchain

	txScript, err := flow_helpers.ParseCadenceTemplate(UPDATE_STATE_SCRIPT, nil)
	if err != nil {
		return err // rollback
	}

	arguments := []cadence.Value{
		cadence.UInt64(dist.FlowID.Int64),
		cadence.UInt8(1),
	}

	t, err := transactions.NewTransactionWithDistributionID(UPDATE_STATE_SCRIPT, txScript, arguments, dist.ID)
	if err != nil {
		return err // rollback
	}

	if err := t.Save(db); err != nil {
		return err // rollback
	}

	logger.WithFields(log.Fields{
		"state":    1,
		"stateStr": "invalid",
	}).Info("Distribution state update transaction saved")

	return nil // commit
}

// UpdateSettlementStatus polls for 'Deposit' events regarding the given distributions
// collectible NFTs.
// It updates the settelement status in database accordingly.
func (c *Contract) UpdateSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := c.logger.WithFields(log.Fields{
		"method":     "UpdateSettlementStatus",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Trace("Update settlement status")

	settlement, err := GetDistributionSettlement(db, dist.ID)
	if err != nil {
		return err // rollback
	}

	latestBlockHeader, err := c.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	begin := settlement.StartAtBlock + 1
	end := min(latestBlockHeader.Height, begin+MAX_EVENTS_PER_CHECK)

	logger = logger.WithFields(log.Fields{
		"blockBegin": begin,
		"blockEnd":   end,
	})

	if begin > end {
		// Nothing to update
		logger.Trace("No blocks to handle")
		return nil // commit
	}

	// Group missing collectibles by their contract reference
	missing, err := MissingCollectibles(db, settlement.ID)
	if err != nil {
		return err // rollback
	}

	for contract, collectibles := range missing.GroupByContract() {
		arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
			Type:        fmt.Sprintf("%s.Deposit", contract.String()),
			StartHeight: begin,
			EndHeight:   end,
		})
		if err != nil {
			return err // rollback
		}

		for _, be := range arr {
			for _, e := range be.Events {
				eventLogger := logger.WithFields(log.Fields{"eventType": e.Type, "eventID": e.ID()})

				eventLogger.Debug("Handling event")

				evtValueMap := flow_helpers.EventValuesToMap(e)

				collectibleFlowIDCadence, ok := evtValueMap["id"]
				if !ok {
					err := fmt.Errorf("could not read 'id' from event %s", e)
					return err // rollback
				}

				collectibleFlowID, err := common.FlowIDFromCadence(collectibleFlowIDCadence)

				if err != nil {
					return err // rollback
				}

				addressCadence, ok := evtValueMap["to"]
				if !ok {
					err := fmt.Errorf("could not read 'to' from event %s", e)
					return err // rollback
				}

				address, err := common.FlowAddressFromCadence(addressCadence)
				if err != nil {
					return err // rollback
				}

				if address == settlement.EscrowAddress {
					if i, ok := collectibles.ContainsID(collectibleFlowID); ok {
						// Collectible in "collectibles" at index "i"

						// Make sure the collectible is in correct state
						if err := collectibles[i].SetSettled(); err != nil {
							return err // rollback
						}

						// Update the collectible in database
						if err := UpdateSettlementCollectible(db, &collectibles[i]); err != nil {
							return err // rollback
						}

						settlement.IncrementCount()
					}
				}

				eventLogger.Trace("Handling event complete")
			}
		}
	}

	if settlement.IsComplete() {
		// TODO: consider updating the distribution separately

		// Make sure the distribution is in correct state
		if err := dist.SetSettled(); err != nil {
			return err // rollback
		}

		// Update the distribution in database
		if err := UpdateDistribution(db, dist); err != nil {
			return err // rollback
		}

		logger.Info("Settlement complete")
	}

	settlement.StartAtBlock = end

	// Update the settlement status in database
	if err := UpdateSettlement(db, settlement); err != nil {
		return err // rollback
	}

	logger.Trace("Update settlement status complete")

	return nil // commit
}

// UpdateMintingStatus polls for 'Mint' events regarding the given distributions
// Pack NFTs.
// It updates the minting status in database accordingly.
func (c *Contract) UpdateMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := c.logger.WithFields(log.Fields{
		"method":     "UpdateMintingStatus",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Trace("Update minting status")

	minting, err := GetDistributionMinting(db, dist.ID)
	if err != nil {
		return err // rollback
	}

	latestBlockHeader, err := c.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	begin := minting.StartAtBlock + 1
	end := min(latestBlockHeader.Height, begin+MAX_EVENTS_PER_CHECK)

	logger = logger.WithFields(log.Fields{
		"blockBegin": begin,
		"blockEnd":   end,
	})

	if begin > end {
		// Nothing to update
		logger.Trace("No blocks to handle")
		return nil // commit
	}

	reference := dist.PackTemplate.PackReference.String()

	arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
		Type:        fmt.Sprintf("%s.Mint", reference),
		StartHeight: begin,
		EndHeight:   end,
	})
	if err != nil {
		return err // rollback
	}

	for _, be := range arr {
		for _, e := range be.Events {
			eventLogger := logger.WithFields(log.Fields{"eventType": e.Type, "eventID": e.ID()})

			eventLogger.Debug("Handling event")

			evtValueMap := flow_helpers.EventValuesToMap(e)

			packFlowIDCadence, ok := evtValueMap["id"]
			if !ok {
				err := fmt.Errorf("could not read 'id' from event %s", e)
				return err // rollback
			}

			packFlowID, err := common.FlowIDFromCadence(packFlowIDCadence)
			if err != nil {
				return err // rollback
			}

			commitmentHashCadence, ok := evtValueMap["commitHash"]
			if !ok {
				err := fmt.Errorf("could not read 'commitHash' from event %s", e)
				return err // rollback
			}

			commitmentHash, err := common.BinaryValueFromCadence(commitmentHashCadence)
			if err != nil {
				return err // rollback
			}

			pack, err := GetMintingPack(db, commitmentHash)
			if err != nil {
				eventLogger.WithFields(log.Fields{
					"packFlowID":     packFlowID,
					"commitmentHash": commitmentHash,
					"error":          err,
				}).Warn("Error while handling event")
				continue // ignore this commitmenthash, go to next event
			}

			// Set the FlowID of the pack
			// Make sure the pack is in correct state
			if err := pack.Seal(packFlowID); err != nil {
				return err // rollback
			}

			// Update the pack in database
			if err := UpdatePack(db, pack); err != nil {
				return err // rollback
			}

			minting.IncrementCount()

			eventLogger.Trace("Handling event complete")
		}
	}

	if minting.IsComplete() {
		// Distribution is now complete

		// TODO: consider updating the distribution separately

		// Make sure the distribution is in correct state
		if err := dist.SetComplete(); err != nil {
			return err // rollback
		}

		// Update the distribution in database
		if err := UpdateDistribution(db, dist); err != nil {
			return err // rollback
		}

		logger.Info("Minting complete")

		// Update distribution state onchain

		txScript, err := flow_helpers.ParseCadenceTemplate(UPDATE_STATE_SCRIPT, nil)
		if err != nil {
			return err // rollback
		}

		arguments := []cadence.Value{
			cadence.UInt64(dist.FlowID.Int64),
			cadence.UInt8(2),
		}

		t, err := transactions.NewTransactionWithDistributionID(UPDATE_STATE_SCRIPT, txScript, arguments, dist.ID)
		if err != nil {
			return err // rollback
		}

		if err := t.Save(db); err != nil {
			return err // rollback
		}

		logger.WithFields(log.Fields{
			"state":    2,
			"stateStr": "complete",
		}).Info("Distribution state update transaction saved")
	}

	minting.StartAtBlock = end

	// Update the minting status in database
	if err := UpdateMinting(db, minting); err != nil {
		return err // rollback
	}

	logger.Trace("Update minting status complete")

	return nil // commit
}

// UpdateCirculatingPack polls for 'REVEAL_REQUEST', 'REVEALED', 'OPEN_REQUEST' and 'OPENED' events
// regarding the given CirculatingPackContract.
// It handles each the 'REVEAL_REQUEST' and 'OPEN_REQUEST' events by creating
// and storing an appropriate Flow transaction in database to be later processed by a poller.
// 'REVEALED' and 'OPENED' events are used to sync the state of a pack in database with onchain state.
func (c *Contract) UpdateCirculatingPack(ctx context.Context, db *gorm.DB, cpc *CirculatingPackContract) error {
	logger := c.logger.WithFields(log.Fields{
		"method": "UpdateCirculatingPack",
		"cpcID":  cpc.ID,
	})

	logger.Trace("Update circulating pack")

	eventNames := []string{
		REVEAL_REQUEST,
		REVEALED,
		OPEN_REQUEST,
		OPENED,
	}

	latestBlockHeader, err := c.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	begin := cpc.StartAtBlock + 1
	end := min(latestBlockHeader.Height, begin+MAX_EVENTS_PER_CHECK)

	logger = logger.WithFields(log.Fields{
		"blockBegin": begin,
		"blockEnd":   end,
	})

	if begin > end {
		logger.Trace("No blocks to handle")
		return nil // commit
	}

	contractRef := AddressLocation{Name: cpc.Name, Address: cpc.Address}

	for _, eventName := range eventNames {
		arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
			Type:        cpc.EventName(eventName),
			StartHeight: begin,
			EndHeight:   end,
		})
		if err != nil {
			return err // rollback
		}

		for _, be := range arr {
			for _, e := range be.Events {
				eventLogger := logger.WithFields(log.Fields{"eventType": e.Type, "eventID": e.ID()})

				eventLogger.Debug("Handling event")

				evtValueMap := flow_helpers.EventValuesToMap(e)

				packFlowIDCadence, ok := evtValueMap["id"]
				if !ok {
					err := fmt.Errorf("could not read 'id' from event %s", e)
					return err // rollback
				}

				packFlowID, err := common.FlowIDFromCadence(packFlowIDCadence)
				if err != nil {
					return err // rollback
				}

				pack, err := GetPackByContractAndFlowID(db, contractRef, packFlowID)
				if err != nil {
					return err // rollback
				}

				distribution, err := GetDistribution(db, pack.DistributionID)
				if err != nil {
					return err // rollback
				}

				eventLogger = eventLogger.WithFields(log.Fields{
					"distID":     distribution.ID,
					"distFlowID": distribution.FlowID,
					"packID":     pack.ID,
					"packFlowID": pack.FlowID,
				})

				switch eventName {
				// -- REVEAL_REQUEST, Owner has requested to reveal a pack ------------
				case REVEAL_REQUEST:

					// Make sure the pack is in correct state
					if err := pack.RevealRequestHandled(); err != nil {
						err := fmt.Errorf("error while handling %s: %w", eventName, err)
						return err // rollback
					}

					// Update the pack in database
					if err := UpdatePack(db, pack); err != nil {
						return err // rollback
					}

					// Get the owner of the pack from the transaction that emitted the open request event
					tx, err := c.flowClient.GetTransaction(ctx, e.TransactionID)
					if err != nil {
						return err // rollback
					}
					owner := tx.Authorizers[0]

					collectibleCount := len(pack.Collectibles)
					collectibleContractAddresses := make([]cadence.Value, collectibleCount)
					collectibleContractNames := make([]cadence.Value, collectibleCount)
					collectibleIDs := make([]cadence.Value, collectibleCount)

					for i, c := range pack.Collectibles {
						collectibleContractAddresses[i] = cadence.Address(c.ContractReference.Address)
						collectibleContractNames[i] = cadence.String(c.ContractReference.Name)
						collectibleIDs[i] = cadence.UInt64(c.FlowID.Int64)
					}

					openRequestValue, ok := evtValueMap["openRequest"]
					if !ok { // TODO(nanuuki): rollback or use a default value for openRequest?
						err := fmt.Errorf("could not read 'openRequest' from event %s", e)
						return err // rollback
					}

					openRequest := openRequestValue.ToGoValue().(bool)
					eventLogger = eventLogger.WithFields(log.Fields{"openRequest": openRequest})

					arguments := []cadence.Value{
						cadence.UInt64(distribution.FlowID.Int64),
						cadence.UInt64(pack.FlowID.Int64),
						cadence.NewArray(collectibleContractAddresses),
						cadence.NewArray(collectibleContractNames),
						cadence.NewArray(collectibleIDs),
						cadence.String(pack.Salt.String()),
						cadence.Address(owner),
						cadence.NewBool(openRequest),
						cadence.Path{Domain: "private", Identifier: "NFTCollectionProvider"},
					}

					// NOTE: this only handles one collectible contract per pack
					txScript, err := flow_helpers.ParseCadenceTemplate(
						REVEAL_SCRIPT,
						&flow_helpers.CadenceTemplateVars{
							PackNFTName:           pack.ContractReference.Name,
							PackNFTAddress:        pack.ContractReference.Address.String(),
							CollectibleNFTName:    pack.Collectibles[0].ContractReference.Name,
							CollectibleNFTAddress: pack.Collectibles[0].ContractReference.Address.String(),
						},
					)
					if err != nil {
						return err // rollback
					}

					t, err := transactions.NewTransactionWithDistributionID(REVEAL_SCRIPT, txScript, arguments, distribution.ID)
					if err != nil {
						return err // rollback
					}

					if err := t.Save(db); err != nil {
						return err // rollback
					}

					if openRequest { // NOTE: This block should run only if we want to reveal AND open the pack
						// Reset the ID to save a second indentical transaction
						t.ID = uuid.Nil
						if err := t.Save(db); err != nil {
							return err // rollback
						}
					}

					eventLogger.Info("Pack reveal transaction created")

				// -- REVEALED, Pack has been revealed onchain ------------------------
				case REVEALED:

					// Make sure the pack is in correct state
					if err := pack.Reveal(); err != nil {
						err := fmt.Errorf("error while handling %s: %w", eventName, err)
						return err // rollback
					}

					// Update the pack in database
					if err := UpdatePack(db, pack); err != nil {
						return err // rollback
					}

				// -- OPEN_REQUEST, Owner has requested to open a pack ----------------
				case OPEN_REQUEST:

					// Make sure the pack is in correct state
					if err := pack.OpenRequestHandled(); err != nil {
						err := fmt.Errorf("error while handling %s: %w", eventName, err)
						return err // rollback
					}

					// Update the pack in database
					if err := UpdatePack(db, pack); err != nil {
						return err // rollback
					}

					// Get the owner of the pack from the transaction that emitted the open request event
					tx, err := c.flowClient.GetTransaction(ctx, e.TransactionID)
					if err != nil {
						return err // rollback
					}
					owner := tx.Authorizers[0]

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
						cadence.Path{Domain: "private", Identifier: "NFTCollectionProvider"},
					}

					// NOTE: this only handles one collectible contract per pack
					txScript, err := flow_helpers.ParseCadenceTemplate(
						OPEN_SCRIPT,
						&flow_helpers.CadenceTemplateVars{
							CollectibleNFTName:    pack.Collectibles[0].ContractReference.Name,
							CollectibleNFTAddress: pack.Collectibles[0].ContractReference.Address.String(),
						},
					)
					if err != nil {
						return err // rollback
					}

					t, err := transactions.NewTransactionWithDistributionID(OPEN_SCRIPT, txScript, arguments, distribution.ID)
					if err != nil {
						return err // rollback
					}

					if err := t.Save(db); err != nil {
						return err // rollback
					}

					eventLogger.Info("Pack open transaction created")

				// -- OPENED, Pack has been opened onchain ----------------------------
				case OPENED:

					// Make sure the pack is in correct state
					if err := pack.Open(); err != nil {
						err := fmt.Errorf("error while handling %s: %w", eventName, err)
						return err // rollback
					}

					// Update the pack in database
					if err := UpdatePack(db, pack); err != nil {
						return err // rollback
					}
				}

				eventLogger.Trace("Handling event complete")
			}
		}
	}

	cpc.StartAtBlock = end

	// Update the CirculatingPackContract in database
	if err := UpdateCirculatingPackContract(db, cpc); err != nil {
		return err // rollback
	}

	logger.Trace("Update circulating pack complete")

	return nil // commit
}
