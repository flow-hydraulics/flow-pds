package app

import (
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk/client"
)

type App struct {
	cfg        *config.Config
	db         Store
	flowClient *client.Client
}

func New(cfg *config.Config, db Store, flowClient *client.Client) *App {
	return &App{cfg, db, flowClient}
}

func (app *App) CreateDistribution(distribution Distribution) (string, error) {
	if err := distribution.Validate(); err != nil {
		return "", err
	}

	if err := distribution.Resolve(); err != nil {
		return "", err
	}

	if err := app.db.InsertDistribution(&distribution); err != nil {
		return "", err
	}

	return distribution.ID.String(), nil
}

func (app *App) ListDistributions(limit, offset int) ([]Distribution, error) {
	opt := ParseListOptions(limit, offset)

	distributions, err := app.db.ListDistributions(opt)
	if err != nil {
		return nil, err
	}

	return distributions, nil
}

func (app *App) GetDistribution(id uuid.UUID) (*Distribution, error) {
	distribution, err := app.db.GetDistribution(id)
	if err != nil {
		return nil, err
	}

	return distribution, nil
}

func (app *App) CancelDistribution(id uuid.UUID) error {
	distribution, err := app.db.GetDistribution(id)
	if err != nil {
		return err
	}

	if err := distribution.Cancel(); err != nil {
		return err
	}

	if err := app.db.UpdateDistribution(distribution); err != nil {
		return err
	}

	return nil
}
