package app

import (
	"context"

	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk/client"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type App struct {
	cfg        *config.Config
	logger     *log.Logger
	db         *gorm.DB
	flowClient *client.Client
	contract   *Contract
	quit       chan bool // Chan type does not matter as we only use this to 'close'
}

func New(cfg *config.Config, logger *log.Logger, db *gorm.DB, flowClient *client.Client, poll bool) *App {
	if logger == nil {
		panic("no logger")
	}

	contract := NewContract(cfg, logger, flowClient)
	quit := make(chan bool)
	app := &App{cfg, logger, db, flowClient, contract, quit}

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

func (app *App) GetDistribution(ctx context.Context, id uuid.UUID) (*Distribution, error) {
	distribution, err := GetDistribution(app.db, id)
	if err != nil {
		return nil, err
	}

	return distribution, nil
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

func (app *App) GetPack(ctx context.Context, id uuid.UUID) (*Pack, error) {
	pack, err := GetPack(app.db, id)
	if err != nil {
		return nil, err
	}
	return pack, nil
}
