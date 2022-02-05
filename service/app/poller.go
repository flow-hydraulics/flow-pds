package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/onflow/flow-go-sdk"
	"sync"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"gorm.io/gorm"
)

// TODO: refactor the db transaction logic

// poller is responsible for the main operation of the service
func poller(app *App) {

	ticker := time.NewTicker(time.Second) // TODO (latenssi): configurable?
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	for {
		select {
		case <-ticker.C:
			log.Trace("Poll start")

			logPollerRun("handleResolved", handleResolved(ctx, app))
			logPollerRun("handleSetup", handleSetup(ctx, app))
			logPollerRun("handleSettling", handleSettling(ctx, app))
			logPollerRun("handleSettled", handleSettled(ctx, app))
			logPollerRun("handleMinting", handleMinting(ctx, app))
			logPollerRun("handleComplete", handleComplete(ctx, app))

			log.Trace("Poll end")
		case <-app.quit:
			cancel()
			ticker.Stop()
			return
		}
	}
}

// packContractEventsPoller is responsible for checking pack contract events from flow blockchain blocks
func packContractEventsPoller(app *App) {
	ticker := time.NewTicker(time.Second) // TODO (latenssi): configurable?
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			log.Info("PackContractEventsPoller start")
			logPollerRun("pollCirculatingPackContractEvents", pollCirculatingPackContractEvents(ctx, app))
			log.WithFields(log.Fields{
				"elapsed": time.Since(start),
			}).Info("PackContractEventsPoller end")
		case <-app.quit:
			cancel()
			ticker.Stop()
			return
		}
	}
}

// sendableTransactionPoller is responsible for checking any sendable to transactions in the transactions table.
func sendableTransactionPoller(app *App) {
	ticker := time.NewTicker(time.Second) // TODO (latenssi): configurable?
	transactionRatelimiter := ratelimit.New(app.cfg.TransactionSendRate)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			log.Info("SendableTransactionPoller poll start")
			logPollerRun("handleSendableTransactions", handleSendableTransactions(ctx, app, transactionRatelimiter))
			log.WithFields(log.Fields{
				"elapsed": time.Since(start),
			}).Info("SendableTransactionPoller poll end")
		case <-app.quit:
			cancel()
			ticker.Stop()
			return
		}
	}
}

// transactionPoller is responsible for sending flow transactions and check transaction status
func sentTransactionsPoller(app *App) {
	ticker := time.NewTicker(time.Second) // TODO (latenssi): configurable?
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			log.Info("SentTransactionsPoller poll start")
			logPollerRun("handleSentTransactions", handleSentTransactions(ctx, app))
			log.WithFields(log.Fields{
				"elapsed": time.Since(start),
			}).Info("SentTransactionsPoller poll end")
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

func logPollerRun(pollerName string, err error) {
	if err != nil {
		log.WithFields(log.Fields{
			"pollerName": pollerName,
			"error":      err,
		}).Warn("Error while running poller")
	} else {
		log.WithFields(log.Fields{
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

func listCirculatingPackContracts(db *gorm.DB) ([]CirculatingPackContract, error) {
	list := []CirculatingPackContract{}
	return list, db.
		Order("updated_at asc").
		Limit(10). // Pick 10 (arbitrary) most least recently updated
		Find(&list).Error
}

func handleResolved(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		resolved, err := listDistributionsByState(tx, common.DistributionStateResolved)
		if err != nil {
			return err
		}

		for _, dist := range resolved {
			if err := app.service.SetupDistribution(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleSetup(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		setup, err := listDistributionsByState(tx, common.DistributionStateSetup)
		if err != nil {
			return err
		}

		for _, dist := range setup {
			if err := app.service.StartSettlement(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleSettling(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		settling, err := listDistributionsByState(tx, common.DistributionStateSettling)
		if err != nil {
			return err
		}

		for _, dist := range settling {
			if err := app.service.UpdateSettlementStatus(ctx, tx, &dist); err != nil {
				return err
			}
		}
		return nil
	})
}

func handleSettled(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		settled, err := listDistributionsByState(tx, common.DistributionStateSettled)
		if err != nil {
			return err
		}

		for _, dist := range settled {
			if err := app.service.StartMinting(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleMinting(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		minting, err := listDistributionsByState(tx, common.DistributionStateMinting)
		if err != nil {
			return err
		}

		for _, dist := range minting {
			if err := app.service.UpdateMintingStatus(ctx, tx, &dist); err != nil {
				return err
			}
		}
		return nil
	})
}

// handleComplete deletes obsolete Settlement, SettlementCollectible and Minting
// objects from database.
func handleComplete(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
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

func pollCirculatingPackContractEvents(ctx context.Context, app *App) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		cc, err := listCirculatingPackContracts(tx)
		if err != nil {
			return err
		}

		for _, c := range cc {
			if err := app.service.UpdateCirculatingPackContract(ctx, tx, &c); err != nil {
				return err
			}
		}

		return nil
	})
}

// handleSendableTransactions sends all transactions which are sendable (state is init or retry)
// with no regard to account proposal key sequence number
func handleSendableTransactions(ctx context.Context, app *App, rateLimiter ratelimit.Limiter) error {

	return app.db.Transaction(func(dbtx *gorm.DB) error {

		availableKeys := app.service.account.AvailableKeys()

		if availableKeys < 1 {
			return fmt.Errorf("not enough available keys, returning")
		}

		ts, err := transactions.GetNextSendables(dbtx, availableKeys)
		wg := sync.WaitGroup{}

		if err != nil {
			return fmt.Errorf("error while getting transactions from database: %w", err)
		}

		for _, t := range ts {
			wg.Add(1)
			transaction := t
			go func(ctx context.Context, dbtx *gorm.DB, wg *sync.WaitGroup, tx *transactions.StorableTransaction) {
				defer wg.Done()
				rateLimiter.Take()
				logger := log.WithFields(log.Fields{
					"function":       "handleSendableTransactions",
					"ID":             tx.ID,
					"name":           tx.Name,
					"distributionID": tx.DistributionID,
					"transactionID":  tx.TransactionID,
				})
				if err := processSendableTransaction(ctx, app, logger, dbtx, tx); err != nil {
					logger.Warn("error processing storable transaction", err)
				}
			}(ctx, dbtx, &wg, &transaction)
		}

		wg.Wait()
		return nil
	})
}

func processSendableTransaction(ctx context.Context, app *App, logger *log.Entry, dbtx *gorm.DB, t *transactions.StorableTransaction) error {
	tx, unlockKey, err := t.Prepare(ctx, app.service.flowClient, app.service.account, app.service.cfg.TransactionGasLimit)

	defer func() {
		// Make sure to unlock if we had an error to prevent deadlocks
		if err != nil {
			unlockKey()
		}
	}()

	if err != nil {
		return fmt.Errorf("error while preparing transaction: %w", err)
	}

	// Update TransactionID
	t.TransactionID = tx.ID().Hex()

	// Update state
	t.State = common.TransactionStateSent

	// Save early as the database might be locked and not allow us to
	// save after sending. This way we fail before actually sending.
	if err = t.Save(dbtx); err != nil {
		return fmt.Errorf("error while saving transaction: %w", err)
	}

	if err = app.service.flowClient.SendTransaction(ctx, *tx); err != nil {
		err = fmt.Errorf("error while sending transaction: %w", err)

		t.State = common.TransactionStateFailed
		t.Error = err.Error()

		if err = t.Save(dbtx); err != nil {
			return fmt.Errorf("error while saving transaction: %w", err)
		}

		// Cant't return the error here as that would rollback this db transaction
	}

	// Double check
	if err != nil {
		return err
	}

	logger.Debug("Transaction sent")

	// Wait for the transaction to finalize (be included in a block, not yet sealed)
	// in a goroutine to unlock the used key
	go func(ctx context.Context, app *App, unlockKey flow_helpers.UnlockKeyFunc, logger *log.Entry) {
		defer unlockKey()
		if _, err := flow_helpers.WaitForSeal(ctx, app.service.flowClient, flow.HexToID(t.TransactionID), time.Minute*10); err != nil {
			logger.WithFields(log.Fields{"error": err.Error()}).Warn("Error while waiting for transaction to seal")
		}
	}(context.Background(), app, unlockKey, logger)

	return nil
}

// handleSentTransactions checks the results of sent transactions and updates
// the state in database accordingly
func handleSentTransactions(ctx context.Context, app *App) error {
	handleCount := 0

	for handleCount < app.cfg.BatchProcessSize {
		err := app.db.Transaction(func(dbtx *gorm.DB) (err error) {
			t, err := transactions.GetNextSent(dbtx)
			if err != nil {
				err = fmt.Errorf("error while getting transaction from database: %w", err)
				return
			}

			if err = t.HandleResult(ctx, app.service.flowClient); err != nil {
				err = fmt.Errorf("error while handling transaction result: %w", err)
				return
			}

			log.WithFields(log.Fields{
				"function":       "handleSentTransactions",
				"ID":             t.ID,
				"name":           t.Name,
				"distributionID": t.DistributionID,
			}).Trace("Sent transaction handled")

			if err = t.Save(dbtx); err != nil {
				err = fmt.Errorf("error while saving transaction: %w", err)
				return
			}

			return nil
		})

		if err != nil {
			// Ignore ErrRecordNotFound and stop iteration
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return err
		}

		handleCount++
	}

	return nil
}
