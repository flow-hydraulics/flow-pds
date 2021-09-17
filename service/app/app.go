package app

import (
	"context"
	"strings"

	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk/client"
)

type App struct {
	cfg         *config.Config
	db          Store
	flowClient  *client.Client
	pdsContract IPDSContract
}

func New(cfg *config.Config, db Store, flowClient *client.Client) *App {
	pdsContract := NewPDSContract(cfg, db, flowClient)
	return &App{cfg, db, flowClient, pdsContract}
}

func (app *App) CreateDistribution(ctx context.Context, distribution *Distribution) error {
	if err := distribution.Validate(); err != nil {
		return err
	}

	if err := distribution.Resolve(); err != nil {
		return err
	}

	if err := app.db.InsertDistribution(distribution); err != nil {
		return err
	}

	return nil
}

func (app *App) ListDistributions(ctx context.Context, limit, offset int) ([]Distribution, error) {
	opt := ParseListOptions(limit, offset)

	distributions, err := app.db.ListDistributions(opt)
	if err != nil {
		return nil, err
	}

	return distributions, nil
}

func (app *App) GetDistribution(ctx context.Context, id uuid.UUID) (*Distribution, *Settlement, error) {
	distribution, err := app.db.GetDistribution(id)
	if err != nil {
		return nil, nil, err
	}

	settlement, err := app.db.GetSettlement(id)
	if err != nil && !strings.Contains(err.Error(), "record not found") {
		return nil, nil, err
	}

	return distribution, settlement, nil
}

func (app *App) SettleDistribution(ctx context.Context, id uuid.UUID) error {
	distribution, err := app.db.GetDistribution(id)
	if err != nil {
		return err
	}

	if err := distribution.Settle(ctx, app.pdsContract); err != nil {
		return err
	}

	if err := app.db.UpdateDistribution(distribution); err != nil {
		return err
	}

	return nil
}

func (app *App) MintDistribution(ctx context.Context, id uuid.UUID) error {
	distribution, err := app.db.GetDistribution(id)
	if err != nil {
		return err
	}

	if err := distribution.Mint(ctx, app.pdsContract); err != nil {
		return err
	}

	if err := app.db.UpdateDistribution(distribution); err != nil {
		return err
	}

	return nil
}

func (app *App) CancelDistribution(ctx context.Context, id uuid.UUID) error {
	distribution, err := app.db.GetDistribution(id)
	if err != nil {
		return err
	}

	if err := distribution.Cancel(ctx, app.pdsContract); err != nil {
		return err
	}

	if err := app.db.UpdateDistribution(distribution); err != nil {
		return err
	}

	return nil
}
