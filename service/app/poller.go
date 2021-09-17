package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func handlePollerError(err error) {
	// Ignore db lock errors, print others
	if err != nil && !strings.Contains(err.Error(), "could not obtain lock on row") {
		fmt.Printf("error while listing resolved distributions: %s", err)
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

func handleSettled(ctx context.Context, db *gorm.DB, contract IContract) error {
	return db.Transaction(func(tx *gorm.DB) error {
		settled, err := listDistributionsByState(tx, common.DistributionStateSettled)
		if err != nil {
			return err
		}

		for _, dist := range settled {
			if err := contract.StartMinting(ctx, db, &dist); err != nil {
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
			if err := contract.CheckSettlementStatus(ctx, db, &dist); err != nil {
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
			if err := contract.CheckMintingStatus(ctx, db, &dist); err != nil {
				return err
			}
		}
		return nil
	})
}
