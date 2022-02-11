package app

import (
	"context"
	"fmt"
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

const (
	SET_DIST_CAP_SCRIPT     = "./cadence-transactions/pds/set_pack_issuer_cap.cdc"
	SETUP_COLLECTION_SCRIPT = "./cadence-transactions/collectibleNFT/setup_collection_and_link_provider.cdc"
	SETTLE_SCRIPT           = "./cadence-transactions/pds/settle.cdc"
	MINT_SCRIPT             = "./cadence-transactions/pds/mint_packNFT.cdc"
	REVEAL_SCRIPT           = "./cadence-transactions/pds/reveal_packNFT.cdc"
	OPEN_SCRIPT             = "./cadence-transactions/pds/open_packNFT.cdc"
	UPDATE_STATE_SCRIPT     = "./cadence-transactions/pds/update_dist_state.cdc"
)

// ContractService handles interfacing with the chain
type ContractService struct {
	cfg        *config.Config
	flowClient *client.Client
	account    *flow_helpers.Account
}

func NewContractService(cfg *config.Config, flowClient *client.Client) (*ContractService, error) {
	if cfg.AdminAddress != cfg.PDSAddress {
		return nil, fmt.Errorf("admin (FLOW_PDS_ADMIN_ADDRESS) and pds (PDS_ADDRESS) addresses should equal")
	}

	pdsAccount, err := flow_helpers.GetAccount(
		flow.HexToAddress(cfg.AdminAddress),
		cfg.AdminPrivateKey,
		cfg.AdminPrivateKeyType,
		cfg.AdminPrivateKeyIndexes,
	)

	if err != nil {
		return nil, err
	}

	flowAccount, err := flowClient.GetAccount(context.Background(), pdsAccount.Address)
	if err != nil {
		return nil, err
	}
	if len(flowAccount.Keys) < len(pdsAccount.PKeyIndexes) {
		return nil, fmt.Errorf("too many key indexes given for admin account")
	}
	return &ContractService{cfg, flowClient, pdsAccount}, nil
}

func (svc *ContractService) SetDistCap(ctx context.Context, db *gorm.DB, issuer common.FlowAddress) error {
	logger := log.WithFields(log.Fields{
		"method": "SetDistCap",
		"issuer": issuer,
	})

	logger.Info("Set distribution capability")

	latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err
	}

	txScript, err := flow_helpers.ParseCadenceTemplate(SET_DIST_CAP_SCRIPT, nil)
	if err != nil {
		return err
	}

	tx := flow.NewTransaction().
		SetScript(txScript).
		SetGasLimit(svc.cfg.TransactionGasLimit).
		SetReferenceBlockID(latestBlockHeader.ID)

	if err := tx.AddArgument(cadence.Address(issuer)); err != nil {
		return err
	}

	unlockKey, err := flow_helpers.SignProposeAndPayAs(ctx, svc.flowClient, svc.account, tx)
	defer unlockKey()
	if err != nil {
		return err
	}

	if err := svc.flowClient.SendTransaction(ctx, *tx); err != nil {
		return err
	}

	if _, err := flow_helpers.WaitForSeal(ctx, svc.flowClient, tx.ID(), svc.cfg.TransactionTimeout, svc.cfg.TransactionPollInterval); err != nil {
		return err
	}

	logger.Trace("Set distribution capability complete")

	return nil
}

// SetupDistribution will make sure the PDS account has a collection onchain
// for the collectible NFTs in the Distribution.
// It also makes sure the withdraw capability is linked.
func (svc *ContractService) SetupDistribution(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := log.WithFields(log.Fields{
		"method":     "SetupDistribution",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Info("Setup distribution")

	// Make sure the distribution is in correct state
	if err := dist.SetSetup(); err != nil {
		return err // rollback
	}

	// Update the distribution in database
	if err := UpdateDistribution(db, dist); err != nil {
		return err // rollback
	}

	buckets, err := GetDistributionBucketsSmall(db, dist.ID)
	if err != nil {
		return err // rollback
	}

	contracts := make(map[AddressLocation]struct{})
	for _, bucket := range buckets {
		contracts[bucket.CollectibleReference] = struct{}{}
	}

	for contract := range contracts {
		logger.WithFields(log.Fields{
			"contract_name":    contract.Name,
			"contract_address": contract.Address,
		}).Debug("Setting up collection and linking")

		latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
		if err != nil {
			return err // rollback
		}

		txScript, err := flow_helpers.ParseCadenceTemplate(
			SETUP_COLLECTION_SCRIPT,
			&flow_helpers.CadenceTemplateVars{
				CollectibleNFTName:    contract.Name,
				CollectibleNFTAddress: contract.Address.String(),
			},
		)
		if err != nil {
			return err // rollback
		}

		tx := flow.NewTransaction().
			SetScript(txScript).
			SetGasLimit(svc.cfg.TransactionGasLimit).
			SetReferenceBlockID(latestBlockHeader.ID)

		if err := tx.AddArgument(cadence.Path{Domain: "private", Identifier: contract.ProviderPath()}); err != nil {
			return err
		}

		// Use anon function here to allow defer as soon as possible
		err = func() error {
			unlockKey, err := flow_helpers.SignProposeAndPayAs(ctx, svc.flowClient, svc.account, tx)
			defer unlockKey()
			if err != nil {
				return err // rollback
			}

			if err := svc.flowClient.SendTransaction(ctx, *tx); err != nil {
				return err // rollback
			}

			if _, err := flow_helpers.WaitForSeal(ctx, svc.flowClient, tx.ID(), svc.cfg.TransactionTimeout, svc.cfg.TransactionPollInterval); err != nil {
				return err // rollback
			}

			return nil
		}()

		if err != nil {
			return err
		}
	}

	logger.Trace("Setup distribution complete")

	return nil
}

// StartSettlement sets the given distributions state to 'settling' and starts the settlement
// phase onchain.
// It lists all collectible NFTs in the distribution and creates batches
// of 'SETTLE_BATCH_SIZE' from them.
// It then creates and stores the settlement Flow transactions (PDS account withdraw from issuer to escrow) in
// database to be later processed by a poller.
// Batching needs to be done to control the transaction size.
func (svc *ContractService) StartSettlement(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := log.WithFields(log.Fields{
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

	latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	settlement := Settlement{
		DistributionID: dist.ID,
		CurrentCount:   0,
		TotalCount:     0, // Update later
		StartAtBlock:   latestBlockHeader.Height - 1,
		EscrowAddress:  common.FlowAddressFromString(svc.cfg.AdminAddress),
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err // rollback
	}

	totalCollectibleCount := 0

	err = DistributionPacksInBatches(db, dist.ID, svc.cfg.BatchProcessSize, func(tx *gorm.DB, batchNumber int, batch []Pack) error {
		collectibles := make(Collectibles, 0)
		for _, pack := range batch {
			collectibles = append(collectibles, pack.Collectibles...)
		}

		totalCollectibleCount += len(collectibles)

		settlementCollectibles := make([]SettlementCollectible, len(collectibles))
		for i, c := range collectibles {
			settlementCollectibles[i] = SettlementCollectible{
				SettlementID:      settlement.ID,
				FlowID:            c.FlowID,
				ContractReference: c.ContractReference,
				IsSettled:         false,
			}
		}

		if err := InsertSettlementCollectibles(db, settlementCollectibles, svc.cfg.BatchInsertSize); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err // rollback
	}

	settlement.TotalCount = uint(totalCollectibleCount)

	if err := UpdateSettlement(db, &settlement); err != nil {
		return err // rollback
	}

	err = NotSettledCollectiblesInBatches(db, settlement.ID, svc.cfg.SettlementBatchSize, func(tx *gorm.DB, batchNumber int, batch SettlementCollectibles) error {
		for contract, collectibles := range batch.GroupByContract() {
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

			batchLogger := logger.WithFields(log.Fields{
				"batchNumber": batchNumber,
			})

			batchLogger.Debug("Initiating settle transaction")

			flowIDs := make([]cadence.Value, len(collectibles))
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
		}

		return nil
	})

	if err != nil {
		return err // rollback
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
func (svc *ContractService) StartMinting(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := log.WithFields(log.Fields{
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

	latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
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

	minting := Minting{
		DistributionID: dist.ID,
		CurrentCount:   0,
		TotalCount:     0, // Update later
		StartAtBlock:   latestBlockHeader.Height - 1,
	}

	if err := InsertMinting(db, &minting); err != nil {
		return err // rollback
	}

	totalPackCount := 0

	err = DistributionPacksInBatches(db, dist.ID, svc.cfg.MintingBatchSize, func(tx *gorm.DB, batchNumber int, batch []Pack) error {
		totalPackCount += len(batch)

		txScript, err := flow_helpers.ParseCadenceTemplate(
			MINT_SCRIPT,
			&flow_helpers.CadenceTemplateVars{
				PackNFTName:    dist.PackTemplate.PackReference.Name,
				PackNFTAddress: dist.PackTemplate.PackReference.Address.String(),
			},
		)
		if err != nil {
			return err
		}

		batchLogger := logger.WithFields(log.Fields{
			"batchNumber": batchNumber,
		})

		batchLogger.Debug("Initiating mint transaction")

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

		return nil
	})

	if err != nil {
		return err // rollback
	}

	minting.TotalCount = uint(totalPackCount)

	if err := UpdateMinting(db, &minting); err != nil {
		return err // rollback
	}

	logger.Trace("Start minting complete")

	return nil // commit
}

// Abort a distribution
func (svc *ContractService) Abort(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := log.WithFields(log.Fields{
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
func (svc *ContractService) UpdateSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := log.WithFields(log.Fields{
		"method":     "UpdateSettlementStatus",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Trace("Update settlement status")

	settlement, err := GetDistributionSettlement(db, dist.ID)
	if err != nil {
		return err // rollback
	}

	latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	begin := settlement.StartAtBlock + 1
	end := min(latestBlockHeader.Height, begin+svc.cfg.MaxBlocksPerCheck)

	logger = logger.WithFields(log.Fields{
		"blockBegin": begin,
		"blockEnd":   end,
	})

	if begin > end {
		// Nothing to update
		logger.Trace("No blocks to handle")
		return nil // commit
	}

	err = NotSettledCollectiblesInBatches(db, settlement.ID, svc.cfg.BatchProcessSize, func(tx *gorm.DB, batchNumber int, batch SettlementCollectibles) error {
		for contract, collectibles := range batch.GroupByContract() {
			arr, err := svc.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
				Type:        fmt.Sprintf("%s.Deposit", contract.String()),
				StartHeight: begin,
				EndHeight:   end,
			})
			if err != nil {
				return err
			}

			for _, be := range arr {
				for _, e := range be.Events {
					eventLogger := logger.WithFields(log.Fields{"eventType": e.Type, "eventID": e.ID()})

					eventLogger.Trace("Handling event")

					evtValueMap := flow_helpers.EventValuesToMap(e)

					collectibleFlowIDCadence, ok := evtValueMap["id"]
					if !ok {
						err := fmt.Errorf("could not read 'id' from event %s", e)
						return err
					}

					collectibleFlowID, err := common.FlowIDFromCadence(collectibleFlowIDCadence)

					if err != nil {
						return err
					}

					addressCadence, ok := evtValueMap["to"]
					if !ok {
						err := fmt.Errorf("could not read 'to' from event %s", e)
						return err
					}

					address, err := common.FlowAddressFromCadence(addressCadence)
					if err != nil {
						return err
					}

					if address == settlement.EscrowAddress {
						if i, ok := collectibles.ContainsID(collectibleFlowID); ok {
							// Collectible in "collectibles" at index "i"

							// Make sure the collectible is in correct state
							if err := collectibles[i].SetSettled(); err != nil {
								return err
							}

							// Update the collectible in database
							if err := UpdateSettlementCollectible(db, &collectibles[i]); err != nil {
								return err
							}

							settlement.IncrementCount()
						}
					}

					eventLogger.Trace("Handling event complete")
				}
			}
		}

		return nil
	})

	if err != nil {
		return err // rollback
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
func (svc *ContractService) UpdateMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	logger := log.WithFields(log.Fields{
		"method":     "UpdateMintingStatus",
		"distID":     dist.ID,
		"distFlowID": dist.FlowID,
	})

	logger.Trace("Update minting status")

	minting, err := GetDistributionMinting(db, dist.ID)
	if err != nil {
		return err // rollback
	}

	latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	begin := minting.StartAtBlock + 1
	end := min(latestBlockHeader.Height, begin+svc.cfg.MaxBlocksPerCheck)

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

	arr, err := svc.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
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

			eventLogger.Trace("Handling event")

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
				logger.Warn(fmt.Sprintf("pack in wrong state %s packFlowId:%+v", err, packFlowID))
				continue
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

// UpdateCirculatingPackContract polls for 'REVEAL_REQUEST', 'REVEALED', 'OPEN_REQUEST' and 'OPENED' events
// regarding the given CirculatingPackContract.
// It handles each the 'REVEAL_REQUEST' and 'OPEN_REQUEST' events by creating
// and storing an appropriate Flow transaction in database to be later processed by a poller.
// 'REVEALED' and 'OPENED' events are used to sync the state of a pack in database with onchain state.
func (svc *ContractService) UpdateCirculatingPackContract(ctx context.Context, db *gorm.DB, cpc *CirculatingPackContract) error {
	logger := log.WithFields(log.Fields{
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

	latestBlockHeader, err := svc.flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err // rollback
	}

	begin := cpc.StartAtBlock + 1
	end := min(latestBlockHeader.Height, begin+svc.cfg.MaxBlocksPerCheck)

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
		arr, err := svc.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
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

				distribution, err := GetDistributionSmall(db, pack.DistributionID)
				if err != nil {
					return err // rollback
				}

				eventLogger = eventLogger.WithFields(log.Fields{
					"distID":     distribution.ID,
					"distFlowID": distribution.FlowID,
					"packID":     pack.ID,
					"packFlowID": pack.FlowID,
				})

				eventLogger.Info("handling event...")

				switch eventName {
				// -- REVEAL_REQUEST, Owner has requested to reveal a pack ------------
				case REVEAL_REQUEST:

					// Make sure the pack is in correct state
					if err := pack.RevealRequestHandled(); err != nil {
						err := fmt.Errorf("error while handling %s: %w", eventName, err)
						eventLogger.Warn(fmt.Sprintf("distID:%s distFlowID:%s packID:%s packFlowID:%s err:%s", distribution.ID, distribution.FlowID, pack.ID, pack.FlowID, err.Error()))
						continue
					}

					// Update the pack in database
					if err := UpdatePack(db, pack); err != nil {
						return err // rollback
					}

					// Get the owner of the pack from the transaction that emitted the open request event
					tx, err := svc.flowClient.GetTransaction(ctx, e.TransactionID)
					if err != nil {
						return err // rollback
					}
					owner := tx.Authorizers[0]

					// NOTE: this only handles one collectible contract per pack
					contract := pack.Collectibles[0].ContractReference

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
						cadence.Path{Domain: "private", Identifier: contract.ProviderPath()},
					}

					// NOTE: this only handles one collectible contract per pack
					txScript, err := flow_helpers.ParseCadenceTemplate(
						REVEAL_SCRIPT,
						&flow_helpers.CadenceTemplateVars{
							PackNFTName:           pack.ContractReference.Name,
							PackNFTAddress:        pack.ContractReference.Address.String(),
							CollectibleNFTName:    contract.Name,
							CollectibleNFTAddress: contract.Address.String(),
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
						eventLogger.Warn(fmt.Sprintf("distID:%s distFlowID:%s packID:%s packFlowID:%s err:%s", distribution.ID, distribution.FlowID, pack.ID, pack.FlowID, err.Error()))
						continue
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
						eventLogger.Warn(fmt.Sprintf("distID:%s distFlowID:%s packID:%s packFlowID:%s err:%s", distribution.ID, distribution.FlowID, pack.ID, pack.FlowID, err.Error()))
						continue
					}

					// Update the pack in database
					if err := UpdatePack(db, pack); err != nil {
						return err // rollback
					}

					// Get the owner of the pack from the transaction that emitted the open request event
					tx, err := svc.flowClient.GetTransaction(ctx, e.TransactionID)
					if err != nil {
						return err // rollback
					}
					owner := tx.Authorizers[0]

					// NOTE: this only handles one collectible contract per pack
					contract := pack.Collectibles[0].ContractReference

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
						cadence.Path{Domain: "private", Identifier: contract.ProviderPath()},
					}

					txScript, err := flow_helpers.ParseCadenceTemplate(
						OPEN_SCRIPT,
						&flow_helpers.CadenceTemplateVars{
							CollectibleNFTName:    contract.Name,
							CollectibleNFTAddress: contract.Address.String(),
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
						eventLogger.Warn(fmt.Sprintf("distID:%s distFlowID:%s packID:%s packFlowID:%s err:%s", distribution.ID, distribution.FlowID, pack.ID, pack.FlowID, err.Error()))
						continue
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
