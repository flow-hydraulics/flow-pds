package app

import (
	"context"
	"strings"

	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk/client"
	"gorm.io/gorm"
)

type App struct {
	cfg        *config.Config
	db         *gorm.DB
	flowClient *client.Client
	contract   IContract
	quit       chan bool // Chan type does not matter as we only use this to 'close'
}

func New(cfg *config.Config, db *gorm.DB, flowClient *client.Client, poll bool) *App {
	contract := NewContract(cfg, flowClient)
	quit := make(chan bool)
	app := &App{cfg, db, flowClient, contract, quit}

	if poll {
		go poller(app)
	}

	return app
}

func (app *App) Close() {
	close(app.quit)
}

func (app *App) CreateDistribution(ctx context.Context, distribution *Distribution) error {
	if err := distribution.Validate(); err != nil {
		return err
	}

	if err := distribution.Resolve(); err != nil {
		return err
	}

	if err := InsertDistribution(app.db, distribution); err != nil {
		return err
	}

	return nil
}

func (app *App) ListDistributions(ctx context.Context, limit, offset int) ([]Distribution, error) {
	opt := ParseListOptions(limit, offset)

	return ListDistributions(app.db, opt)
}

func (app *App) GetDistribution(ctx context.Context, id uuid.UUID) (*Distribution, *Settlement, error) {
	distribution, err := GetDistribution(app.db, id)
	if err != nil {
		return nil, nil, err
	}

	settlement, err := GetDistributionSettlement(app.db, id)
	if err != nil && !strings.Contains(err.Error(), "record not found") {
		return nil, nil, err
	}

	return distribution, settlement, nil
}

func (app *App) SettleDistribution(ctx context.Context, id uuid.UUID) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		distribution, err := GetDistribution(tx, id)
		if err != nil {
			return err
		}

		if err := app.contract.StartSettlement(ctx, tx, distribution); err != nil {
			return err
		}

		return nil
	})
}

func (app *App) MintDistribution(ctx context.Context, id uuid.UUID) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		distribution, err := GetDistribution(tx, id)
		if err != nil {
			return err
		}

		if err := app.contract.StartMinting(ctx, tx, distribution); err != nil {
			return err
		}

		return nil
	})
}

func (app *App) CancelDistribution(ctx context.Context, id uuid.UUID) error {
	return app.db.Transaction(func(tx *gorm.DB) error {
		distribution, err := GetDistribution(tx, id)
		if err != nil {
			return err
		}

		if err := app.contract.Cancel(ctx, tx, distribution); err != nil {
			return err
		}

		return nil
	})
}
