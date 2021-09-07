package app

import (
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/store"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
)

type App struct {
	cfg        *config.Config
	db         store.Store
	flowClient *client.Client
}

func New(cfg *config.Config, db store.Store, flowClient *client.Client) *App {
	return &App{cfg, db, flowClient}
}

func (app *App) CreateDistribution(Distribution) (string, error) {
	// TODO (latenssi)
	return "todo-new-distribution-id", nil
}

func (app *App) ListDistributions() ([]Distribution, error) {
	// TODO (latenssi)
	return []Distribution{
		{Issuer: flow.HexToAddress("0x1")},
	}, nil
}

func (app *App) GetDistribution(id string) (*Distribution, error) {
	// TODO (latenssi)
	return &Distribution{Issuer: flow.HexToAddress("0x1")}, nil
}

func (app *App) SettleDistribution(id string) error {
	// TODO (latenssi)
	return nil
}

func (app *App) ConfirmDistribution(id string) error {
	// TODO (latenssi)
	return nil
}

func (app *App) CancelDistribution(id string) error {
	// TODO (latenssi)
	return nil
}
