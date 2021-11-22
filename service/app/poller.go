package app

import (
	"context"
	"errors"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"gorm.io/gorm"
)

// TODO: refactor the db transaction logic

// poller is responsible for the main operation of the service
func poller(app *App) {

	ticker := time.NewTicker(time.Second) // TODO (latenssi): configurable?
	transactionRatelimiter := ratelimit.New(app.cfg.TransactionSendRate)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	for {
		select {
		case <-ticker.C:
			app.logger.Trace("Poll start")

			logPollerRun("handleResolved", handleResolved(ctx, app.db, app.service), app.logger)
			logPollerRun("handleSetup", handleSetup(ctx, app.db, app.service), app.logger)
			logPollerRun("handleSettling", handleSettling(ctx, app.db, app.service), app.logger)
			logPollerRun("handleSettled", handleSettled(ctx, app.db, app.service), app.logger)
			logPollerRun("handleMinting", handleMinting(ctx, app.db, app.service), app.logger)
			logPollerRun("handleComplete", handleComplete(ctx, app.db, app.service), app.logger)

			logPollerRun("pollCirculatingPackContractEvents", pollCirculatingPackContractEvents(ctx, app.db, app.service), app.logger)

			logPollerRun("handleSentTransactions", handleSentTransactions(ctx, app.db, app.service), app.logger)
			logPollerRun("handleSendableTransactions", handleSendableTransactions(ctx, app.db, app.service, transactionRatelimiter), app.logger)

			app.logger.Trace("Poll end")
		case <-app.quit:
			cancel()
			ticker.Stop()
			return
		}
	}
}

func min(x, y uint64) uint64 {
	if x > y {
		return y
	}
	return x
}

func logPollerRun(pollerName string, err error, logger *log.Logger) {
	if err != nil {
		logger.WithFields(log.Fields{
			"pollerName": pollerName,
			"error":      err,
		}).Warn("Error while running poller")
	} else {
		logger.WithFields(log.Fields{
			"pollerName": pollerName,
		}).Trace("Done")
	}
}

func listDistributionsByState(db *gorm.DB, state common.DistributionState) ([]Distribution, error) {
	list := []Distribution{}
	return list, db.
		Where(&Distribution{State: state}).
		Order("updated_at asc").
		Find(&list).Error
}

func listCirculatingPacks(db *gorm.DB) ([]CirculatingPackContract, error) {
	list := []CirculatingPackContract{}
	return list, db.
		Order("updated_at asc").
		Limit(10). // Pick 10 (arbitrary) most least recently updated
		Find(&list).Error
}

func handleResolved(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		resolved, err := listDistributionsByState(tx, common.DistributionStateResolved)
		if err != nil {
			return err
		}

		for _, dist := range resolved {
			if err := contract.SetupDistribution(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleSetup(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		setup, err := listDistributionsByState(tx, common.DistributionStateSetup)
		if err != nil {
			return err
		}

		for _, dist := range setup {
			if err := contract.StartSettlement(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleSettling(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		settling, err := listDistributionsByState(tx, common.DistributionStateSettling)
		if err != nil {
			return err
		}

		for _, dist := range settling {
			if err := contract.UpdateSettlementStatus(ctx, tx, &dist); err != nil {
				return err
			}
		}
		return nil
	})
}

func handleSettled(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		settled, err := listDistributionsByState(tx, common.DistributionStateSettled)
		if err != nil {
			return err
		}

		for _, dist := range settled {
			if err := contract.StartMinting(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleMinting(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		minting, err := listDistributionsByState(tx, common.DistributionStateMinting)
		if err != nil {
			return err
		}

		for _, dist := range minting {
			if err := contract.UpdateMintingStatus(ctx, tx, &dist); err != nil {
				return err
			}
		}
		return nil
	})
}

// handleComplete deletes obsolete Settlement, SettlementCollectible and Minting
// objects from database.
func handleComplete(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		complete, err := listDistributionsByState(tx, common.DistributionStateComplete)
		if err != nil {
			return err
		}

		for _, dist := range complete {
			if err := DeleteSettlementForDistribution(tx, dist.ID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			if err := DeleteMintingForDistribution(tx, dist.ID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
		}

		return nil
	})
}

func pollCirculatingPackContractEvents(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	return db.Transaction(func(tx *gorm.DB) error {
		cc, err := listCirculatingPacks(tx)
		if err != nil {
			return err
		}

		for _, c := range cc {
			if err := contract.UpdateCirculatingPack(ctx, tx, &c); err != nil {
				return err
			}
		}

		return nil
	})
}

// handleSendableTransactions sends all transactions which are sendable (state is init or retry)
// with no regard to account proposal key sequence number
// TODO (latenssi): this is basically brute forcing the sequence numbering
// TODO (latenssi): this will currently iterate over all sendable transactions
// in database while locking the poller from doing other actions (limited by maxHandleCount)
func handleSendableTransactions(ctx context.Context, db *gorm.DB, contract *ContractService, rateLimiter ratelimit.Limiter) error {
	logger := log.WithFields(log.Fields{
		"function": "handleSendableTransactions",
	})

	run := true
	handleCount := 0
	maxHandleCount := 100

	for run && handleCount < maxHandleCount {
		// Rate limit
		rateLimiter.Take()

		// Ignoring error for now, as they are already logged with context
		db.Transaction(func(dbtx *gorm.DB) error {
			t, err := transactions.GetNextSendable(dbtx)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					run = false
				} else {
					logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while getting transaction from database")
				}
				return err
			}

			logger = logger.WithFields(log.Fields{
				"ID":             t.ID,
				"name":           t.Name,
				"distributionID": t.DistributionID,
			})

			tx, err := t.Prepare(ctx, contract.flowClient, contract.account, contract.cfg.TransactionGasLimit)
			if err != nil {
				logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while preparing transaction")
				return err
			}

			// Update TransactionID
			t.TransactionID = tx.ID().Hex()

			// Update state
			t.State = common.TransactionStateSent

			logger = logger.WithFields(log.Fields{
				"transactionID": t.TransactionID,
			})

			// Save early as the database might be locked and not allow us to
			// save after sending. This way we fail before actually sending.
			if err := t.Save(dbtx); err != nil {
				logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while saving transaction")
				return err
			}

			if err := contract.flowClient.SendTransaction(ctx, *tx); err != nil {
				logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while sending transaction")

				t.State = common.TransactionStateFailed
				t.Error = err.Error()

				if err := t.Save(dbtx); err != nil {
					logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while saving transaction")
					return err
				}

				// Cant't return the error here as that would rollback this db transaction
			} else {
				logger.Debug("Transaction sent")
			}

			return nil
		})

		handleCount++
	}

	return nil
}

// handleSentTransactions checks the results of sent transactions and updates
// the state in database accordingly
func handleSentTransactions(ctx context.Context, db *gorm.DB, contract *ContractService) error {
	logger := log.WithFields(log.Fields{
		"function": "handleSentTransactions",
	})

	run := true

	for run {
		// Ignoring error for now, as they are already logged with context
		db.Transaction(func(dbtx *gorm.DB) error {
			t, err := transactions.GetNextSent(dbtx)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					run = false
				} else {
					logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while getting transaction from database")
				}
				return err
			}

			logger = logger.WithFields(log.Fields{
				"ID":             t.ID,
				"name":           t.Name,
				"distributionID": t.DistributionID,
			})

			if err := t.HandleResult(ctx, contract.flowClient); err != nil {
				logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while handling transaction result")
				return err
			}

			if err := t.Save(dbtx); err != nil {
				logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while saving transaction")
				return err
			}

			return nil
		})
	}

	return nil
}
