package app

import (
	"context"
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/onflow/flow-go-sdk"
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

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	packs, err := GetDistributionPacks(db, dist.ID)
	if err != nil {
		return err
	}

	collectibles := make(Collectibles, 0)
	for _, pack := range packs {
		collectibles = append(collectibles, pack.Collectibles...)
	}
	sort.Sort(collectibles)

	settlementCollectibles := make([]SettlementCollectible, len(collectibles))
	for i, c := range collectibles {
		settlementCollectibles[i] = SettlementCollectible{
			FlowID:            c.FlowID,
			ContractReference: c.ContractReference,
			Settled:           false,
		}
	}

	settlement := Settlement{
		DistributionID:   dist.ID,
		Total:            uint(len(collectibles)),
		EscrowAddress:    common.FlowAddress(flow.HexToAddress("f3fcd2c1a78f5eee")), // TODO (latenssi): proper escrow address
		LastCheckedBlock: latestBlock.Height,
		Collectibles:     settlementCollectibles,
	}

	if err := InsertSettlement(db, &settlement); err != nil {
		return err
	}

	// TODO (latenssi)
	// - Send a request to PDS Contract to start withdrawing of collectible NFTs to Contract account
	// - Timeout? Cancel?

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
	settlement, err := GetSettlementByDistId(db, dist.ID)
	if err != nil {
		return err
	}

	groupedMissing, err := MissingCollectibles(db, settlement.ID)
	if err != nil {
		return err
	}

	latestBlock, err := c.flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	start := settlement.LastCheckedBlock + 1
	end := min(latestBlock.Height, start+100)

	if start > end {
		// Nothing to update
		return nil
	}

	for reference, missing := range groupedMissing {
		arr, err := c.flowClient.GetEventsForHeightRange(ctx, client.EventRangeQuery{
			Type:        fmt.Sprintf("%s.Deposit", reference),
			StartHeight: start,
			EndHeight:   end,
		})
		if err != nil {
			return err
		}

		for _, be := range arr {
			for _, e := range be.Events {
				id, err := common.FlowIDFromStr(e.Value.Fields[0].String())
				if err != nil {
					return err
				}
				address := common.FlowAddress(flow.HexToAddress(e.Value.Fields[1].String()))
				if address == settlement.EscrowAddress {
					if i, ok := missing.ContainsID(id); ok {
						missing[i].Settled = true
						if err := UpdateSettlementCollectible(db, &missing[i]); err != nil {
							return err
						}
						settlement.Settled++
					}
				}
			}
		}
	}

	settlement.LastCheckedBlock = end

	if settlement.Settled >= settlement.Total {
		// Update settlement state
		settlement.State = common.SettlementStateDone
	}

	if err := UpdateSettlement(db, settlement); err != nil {
		return err
	}

	if settlement.State == common.SettlementStateDone && dist.State == common.DistributionStateSettling {
		// Update distribution state
		dist.State = common.DistributionStateSettled
		if err := UpdateDistribution(db, dist); err != nil {
			return err
		}
	}

	return nil
}

func (c *Contract) CheckMintingStatus(ctx context.Context, db *gorm.DB, dist *Distribution) error {
	// TODO (latenssi): poll for minting status of distribution => set "complete" when done
	return nil
}
