package app

import (
	"context"
	"fmt"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func poller(app *App) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(time.Second) // TODO (latenssi): proper duration
	for {
		select {
		case <-ticker.C:
			go func() {
				handlePollerError("handleResolved", handleResolved(ctx, app.db, app.contract))
				handlePollerError("handleSettling", handleSettling(ctx, app.db, app.contract))
				handlePollerError("handleSettled", handleSettled(ctx, app.db, app.contract))
				handlePollerError("handleMinting", handleMinting(ctx, app.db, app.contract))
				handlePollerError("pollCirculatingPackContractEvents", pollCirculatingPackContractEvents(ctx, app.db, app.contract))
			}()
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
		fmt.Printf("error while runnig poller \"%s\": %s\n", pollerName, err)
	}
}

func listDistributionsByState(db *gorm.DB, state common.DistributionState) ([]Distribution, error) {
	list := []Distribution{}
	return list, db.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
		Where(&Distribution{State: state}).
		Order("updated_at asc").
		Find(&list).Error
}

func listCirculatingPacks(db *gorm.DB) ([]CirculatingPackContract, error) {
	list := []CirculatingPackContract{}
	return list, db.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
		Order("updated_at asc").
		Limit(10). // Pick 10 (arbitrary) most least recently updated
		Find(&list).Error
}

func handleResolved(ctx context.Context, db *gorm.DB, contract IContract) error {
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

func handleSettling(ctx context.Context, db *gorm.DB, contract IContract) error {
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

func handleSettled(ctx context.Context, db *gorm.DB, contract IContract) error {
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

func handleMinting(ctx context.Context, db *gorm.DB, contract IContract) error {
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

func pollCirculatingPackContractEvents(ctx context.Context, db *gorm.DB, contract IContract) error {
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
