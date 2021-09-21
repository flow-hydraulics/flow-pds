package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
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
			// This is safe? to run as a separate goroutine since it does not lock the distributions table.
			go func() {
				handlePollerError("pollCirculatingPackContractEvents", pollCirculatingPackContractEvents(ctx, app.db, app.flowClient))
			}()

			// These are _not_ safe to run separately as a goroutine since they lock the distributions table.
			// If run as a goroutine, only the first function (handleResolved) will ever be run
			// as the others will always be blocked and the goroutines will fail.
			go func() {
				handlePollerError("handleResolved", handleResolved(ctx, app.db, app.contract))
				handlePollerError("handleSettling", handleSettling(ctx, app.db, app.contract))
				handlePollerError("handleSettled", handleSettled(ctx, app.db, app.contract))
				handlePollerError("handleMinting", handleMinting(ctx, app.db, app.contract))
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
	// Ignore db lock errors, print others
	if err != nil && !strings.Contains(err.Error(), "could not obtain lock on row") {
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
			if err := contract.CheckSettlementStatus(ctx, tx, &dist); err != nil {
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
			if err := contract.CheckMintingStatus(ctx, tx, &dist); err != nil {
				return err
			}
		}
		return nil
	})
}

func pollCirculatingPackContractEvents(ctx context.Context, db *gorm.DB, flowClient *client.Client) error {
	eventNames := []string{
		"RevealRequest",
		"OpenPackRequest",
		"Mint",
	}

	return db.Transaction(func(tx *gorm.DB) error {
		cc, err := listCirculatingPacks(tx)
		if err != nil {
			return err
		}

		latestBlock, err := flowClient.GetLatestBlock(ctx, true)
		if err != nil {
			return err
		}

		for i, c := range cc {
			start := c.LastCheckedBlock + 1
			end := min(latestBlock.Height, start+100)
			if start > end {
				continue
			}

			for _, eventName := range eventNames {
				arr, err := flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
					Type:        c.EventName(eventName),
					StartHeight: start,
					EndHeight:   end,
				})
				if err != nil {
					return err
				}

				for _, be := range arr {
					for _, e := range be.Events {
						if err := handleCirculatingPackContractEvent(e); err != nil {
							return err
						}
					}
				}
			}

			// Make sure to refer the entry of the slice, not the 'c' in this closure
			cc[i].LastCheckedBlock = end
		}

		if len(cc) > 0 {
			if err := UpdateCirculatingPackContracts(tx, cc); err != nil {
				return err
			}
		}

		return nil
	})
}

func handleCirculatingPackContractEvent(flow.Event) error {
	// TODO (latenssi)
	return nil
}
