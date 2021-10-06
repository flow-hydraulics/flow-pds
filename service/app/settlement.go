package app

import (
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TODO (latenssi): these could be removed from database after done

// Settlement represents the settlement status of a distribution.
type Settlement struct {
	gorm.Model
	ID             uuid.UUID    `gorm:"column:id;primary_key;type:uuid;"`
	DistributionID uuid.UUID    `gorm:"unique"`
	Distribution   Distribution `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CurrentCount uint   `gorm:"column:current_count"`
	TotalCount   uint   `gorm:"column:total_count"`
	StartAtBlock uint64 `gorm:"column:start_at_block"`

	EscrowAddress common.FlowAddress      `gorm:"column:escrow_address"`
	Collectibles  []SettlementCollectible `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SettlementCollectible struct {
	gorm.Model
	SettlementID uuid.UUID
	ID           uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	FlowID            common.FlowID   `gorm:"column:flow_id;"`                       // ID of the collectible NFT
	ContractReference AddressLocation `gorm:"embedded;embeddedPrefix:contract_ref_"` // Reference to the collectible NFT contract
	IsSettled         bool            `gorm:"column:is_settled"`
}

type SettlementCollectibles []SettlementCollectible

func (Settlement) TableName() string {
	return "settlements"
}

func (s *Settlement) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return nil
}

func (s *Settlement) IsComplete() bool {
	return s.CurrentCount >= s.TotalCount
}

func (s *Settlement) IncrementCount() {
	s.CurrentCount++
}

func (SettlementCollectible) TableName() string {
	return "settlement_collectibles"
}

func (s *SettlementCollectible) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return nil
}

func (s *SettlementCollectible) SetSettled() (err error) {
	if s.IsSettled {
		return fmt.Errorf("settlement collectible already settled")
	}

	s.IsSettled = true

	return nil
}

func (cc SettlementCollectibles) ContainsID(id common.FlowID) (int, bool) {
	for i, v := range cc {
		if v.FlowID == id {
			return i, true
		}
	}
	return -1, false
}
