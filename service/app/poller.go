package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"gorm.io/gorm"
)

// TODO: refactor the db transaction logic

func poller(app *App) {

	ticker := time.NewTicker(time.Second) // TODO (latenssi): configurable?
	transactionRatelimiter := ratelimit.New(app.cfg.SendTransactionRate)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	for {
		app.logger.Trace("Poll start")
		select {
		case <-ticker.C:
			handlePollerError("handleResolved", handleResolved(ctx, app.db, app.contract))
			handlePollerError("handleSettling", handleSettling(ctx, app.db, app.contract))
			handlePollerError("handleSettled", handleSettled(ctx, app.db, app.contract))
			handlePollerError("handleMinting", handleMinting(ctx, app.db, app.contract))
			handlePollerError("handleComplete", handleComplete(ctx, app.db, app.contract))

			handlePollerError("pollCirculatingPackContractEvents", pollCirculatingPackContractEvents(ctx, app.db, app.contract))

			handlePollerError("handleSentTransactions", handleSentTransactions(ctx, app.db, app.contract))
			handlePollerError("handleSendableTransactions", handleSendableTransactions(ctx, app.db, app.contract, app.logger, transactionRatelimiter))
		case <-app.quit:
			cancel()
			ticker.Stop()
			return
		}
		app.logger.Trace("Poll end")
	}
}

func min(x, y uint64) uint64 {
	if x > y {
		return y
	}
	return x
}

func handlePollerError(pollerName string, err error) {
	if err != nil {
		// Ignore database locked errors as they are part of the control flow
		if strings.Contains(err.Error(), "database is locked") {
			return
		}
		fmt.Printf("error while runnig poller \"%s\": %s\n", pollerName, err)
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

func handleResolved(ctx context.Context, db *gorm.DB, contract *Contract) error {
	return db.Transaction(func(tx *gorm.DB) error {
		resolved, err := listDistributionsByState(tx, common.DistributionStateResolved)
		if err != nil {
			return err
		}

		for _, dist := range resolved {
			if err := contract.StartSettlement(ctx, tx, &dist); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleSettling(ctx context.Context, db *gorm.DB, contract *Contract) error {
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

func handleSettled(ctx context.Context, db *gorm.DB, contract *Contract) error {
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

func handleMinting(ctx context.Context, db *gorm.DB, contract *Contract) error {
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
func handleComplete(ctx context.Context, db *gorm.DB, contract *Contract) error {
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

func pollCirculatingPackContractEvents(ctx context.Context, db *gorm.DB, contract *Contract) error {
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
// in database while locking the poller from doing other actions
func handleSendableTransactions(ctx context.Context, db *gorm.DB, contract *Contract, logger *log.Logger, rateLimiter ratelimit.Limiter) error {
	run := true

	for run {
		// Rate limit
		rateLimiter.Take()

		// Ignoring error for now, as they are already logged with context
		db.Transaction(func(dbtx *gorm.DB) error {
			t, err := transactions.GetNextSendable(dbtx)
			if err != nil {
				if strings.Contains(err.Error(), "record not found") {
					run = false
				} else {
					log.WithFields(log.Fields{
						"function": "handleSendableTransactions",
						"ID":       t.ID,
						"error":    err.Error(),
					}).Warn("Error while getting transaction from database")
				}
				return err
			}

			tx, err := t.Prepare(ctx, contract.flowClient, contract.account)
			if err != nil {
				log.WithFields(log.Fields{
					"function": "handleSendableTransactions",
					"ID":       t.ID,
					"error":    err.Error(),
				}).Warn("Error while preparing transaction")
				return err
			}

			// Update TransactionID
			t.TransactionID = tx.ID().Hex()

			// Update state
			t.State = common.TransactionStateSent

			// Save early as the database might be locked and not allow us to
			// save after sending. This way we fail before actually sending.
			if err := t.Save(dbtx); err != nil {
				log.WithFields(log.Fields{
					"function": "handleSendableTransactions",
					"ID":       t.ID,
					"error":    err.Error(),
				}).Warn("Error while saving transaction")
				return err
			}

			if err := contract.flowClient.SendTransaction(ctx, *tx); err != nil {
				log.WithFields(log.Fields{
					"function": "handleSendableTransactions",
					"ID":       t.ID,
					"error":    err.Error(),
				}).Warn("Error while sending transaction")

				t.State = common.TransactionStateFailed
				t.Error = err.Error()

				if err := t.Save(dbtx); err != nil {
					log.WithFields(log.Fields{
						"function": "handleSendableTransactions",
						"ID":       t.ID,
						"error":    err.Error(),
					}).Warn("Error while saving transaction")
					return err
				}

				return err
			}

			return nil
		})
	}

	return nil
}

// handleSentTransactions checks the results of sent transactions and updates
// the state in database accordingly
func handleSentTransactions(ctx context.Context, db *gorm.DB, contract *Contract) error {
	run := true

	for run {
		// Ignoring error for now, as they are already logged with context
		db.Transaction(func(dbtx *gorm.DB) error {
			t, err := transactions.GetNextSent(dbtx)
			if err != nil {
				if strings.Contains(err.Error(), "record not found") {
					run = false
				} else {
					log.WithFields(log.Fields{
						"function": "handleSentTransactions",
						"ID":       t.ID,
						"error":    err.Error(),
					}).Warn("error while getting transaction from database")
				}
				return err
			}

			if err := t.HandleResult(ctx, contract.flowClient); err != nil {
				log.WithFields(log.Fields{
					"function": "handleSentTransactions",
					"ID":       t.ID,
					"error":    err.Error(),
				}).Warn("error while handling transaction result")
				return err
			}

			if err := t.Save(dbtx); err != nil {
				log.WithFields(log.Fields{
					"function": "handleSentTransactions",
					"ID":       t.ID,
					"error":    err.Error(),
				}).Warn("error while saving transaction")
				return err
			}

			return nil
		})
	}

	return nil
}
