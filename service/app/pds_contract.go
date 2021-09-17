package app

import (
	"context"

	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/onflow/flow-go-sdk/client"
)

type IPDSContract interface {
	StartSettlement(context.Context, *Distribution) error
	StartMinting(context.Context, *Distribution) error
	Cancel(context.Context, *Distribution) error
}

type PDSContract struct {
	cfg        *config.Config
	db         Store
	flowClient *client.Client
}

func NewPDSContract(cfg *config.Config, db Store, flowClient *client.Client) *PDSContract {
	return &PDSContract{cfg, db, flowClient}
}

func (c *PDSContract) StartSettlement(ctx context.Context, dist *Distribution) error {
	settlement := Settlement{
		DistributionID: dist.ID,
		Total:          uint(dist.ResolvedCollection().Len()),
	}

	if err := c.db.InsertSettlement(&settlement); err != nil {
		return err
	}

	// TODO (latenssi)
	// - Send a request to PDS Contract to start withdrawing of collectible NFTs to Contract account
	// - Listen for deposit events of collectible NFTs to Contract account
	// - Timeout? Cancel?
	// - Once all have been deposited set state to Settled

	return nil
}

func (c *PDSContract) StartMinting(context.Context, *Distribution) error {
	// TODO (latenssi)
	return nil
}

func (c *PDSContract) Cancel(context.Context, *Distribution) error {
	// TODO (latenssi)
	return nil
}
