package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	"gorm.io/gorm"
)

// TODO: instead of running everything in one transaction, separate them by the
// parent object or something suitable

func poller(app *App) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(time.Second) // TODO (latenssi): proper duration
	for {
		select {
		case <-ticker.C:
			handlePollerError("handleResolved", handleResolved(ctx, app.db, app.contract))
			handlePollerError("handleSettling", handleSettling(ctx, app.db, app.contract))
			handlePollerError("handleSettled", handleSettled(ctx, app.db, app.contract))
			handlePollerError("handleMinting", handleMinting(ctx, app.db, app.contract))
			handlePollerError("pollCirculatingPackContractEvents", pollCirculatingPackContractEvents(ctx, app.db, app.contract))
			handlePollerError("handleSendableTransactions", handleSendableTransactions(ctx, app.db, app.contract))
			handlePollerError("handleSentTransactions", handleSentTransactions(ctx, app.db, app.contract))
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

func handlePollerError(pollerName string, err error) {
	if err != nil {
		// Ignore database locked errors as they are part of the control flow
		if strings.Contains(err.Error(), "database is locked") {
			return
		}
		fmt.Printf("error while running poller \"%s\": %s\n", pollerName, err)
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
			// TODO: separate db transaction?
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
			// TODO: separate db transaction?
			if err := contract.UpdateMintingStatus(ctx, tx, &dist); err != nil {
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
func handleSendableTransactions(ctx context.Context, db *gorm.DB, contract *Contract) error {
	sendableIDs, err := transactions.SendableIDs(db)
	if err != nil {
		return err
	}

	for _, id := range sendableIDs {
		err := db.Transaction(func(dbTx *gorm.DB) error {
			t, err := transactions.GetTransaction(dbTx, id)
			if err != nil {
				return err
			}

			if err := t.Send(ctx, contract.flowClient, contract.account); err != nil {
				return err
			}

			if err := t.Save(dbTx); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// handleSentTransactions checks the results of sent transactions and updates
// the state in database accordingly
func handleSentTransactions(ctx context.Context, db *gorm.DB, contract *Contract) error {
	sentIDs, err := transactions.SentIDs(db)
	if err != nil {
		return err
	}

	for _, id := range sentIDs {
		err := db.Transaction(func(dbTx *gorm.DB) error {
			t, err := transactions.GetTransaction(dbTx, id)
			if err != nil {
				return err
			}

			if err := t.HandleResult(ctx, contract.flowClient); err != nil {
				return err
			}

			if err := t.Save(dbTx); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
