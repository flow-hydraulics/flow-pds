package app

import (
	"context"
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/onflow/flow-go-sdk/client"
	"gorm.io/gorm"
)

type IContract interface {
	StartSettlement(context.Context, *gorm.DB, *Distribution) error
	StartMinting(context.Context, *gorm.DB, *Distribution) error
	Cancel(context.Context, *gorm.DB, *Distribution) error
	CheckSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error
	CheckMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error
}

type Contract struct {
	cfg        *config.Config
	flowClient *client.Client
}

func NewContract(cfg *config.Config, flowClient *client.Client) *Contract {
	return &Contract{cfg, flowClient}
}

func (c *Contract) StartSettlement(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	if err := dist.SetSettling(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	settlement := Settlement{
		DistributionID: dist.ID,
		Total:          uint(dist.ResolvedCollection().Len()),
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err
	}

	// TODO (latenssi)
	// - Send a request to PDS Contract to start withdrawing of collectible NFTs to Contract account
	// - Listen for deposit events of collectible NFTs to Contract account
	// - Timeout? Cancel?
	// - Once all have been deposited set state to Settled

	return nil
}

func (c *Contract) StartMinting(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	if err := dist.SetMinting(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	// Add CirculatingPackContracts to database
	for _, b := range dist.PackTemplate.Buckets {
		cpc := CirculatingPackContract{
			Name:             b.CollectibleReference.Name,
			Address:          b.CollectibleReference.Address,
			LastCheckedBlock: latestBlock.Height,
		}
		// TODO (latenssi): get from db instead of failing on duplicate index?
		if err := InsertCirculatingPackContract(db, &cpc); err != nil {
			// TODO (latenssi): once duplicate key error handling is done, return error from here
			fmt.Printf("error while inserting CirculatingPackContract: %s\n", err)
		}
	}

	// TODO (latenssi)

	return nil
}

func (c *Contract) Cancel(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	if err := dist.SetCancelled(); err != nil {
		return err
	}

	if err := UpdateDistribution(db, dist); err != nil {
		return err
	}

	// TODO (latenssi)

	return nil
}

func (c *Contract) CheckSettlementStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	// TODO (latenssi): poll for settlement status of distribution => set "settled" when done
	return nil
}

func (c *Contract) CheckMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	// TODO (latenssi): poll for minting status of distribution => set "complete" when done
	return nil
}
