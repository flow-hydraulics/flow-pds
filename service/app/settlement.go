package app

import (
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Settlement struct {
	gorm.Model
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`
	DistributionID uuid.UUID `gorm:"unique"`
	Distribution   Distribution

	State   common.SettlementState `gorm:"column:state"`
	Settled uint                   `gorm:"column:settled"`
	Total   uint                   `gorm:"column:total"`

	EscrowAddress    common.FlowAddress `gorm:"column:escrow_address"`
	LastCheckedBlock uint64             `gorm:"column:last_checked_block"`
	Collectibles     []SettlementCollectible
}

type SettlementCollectible struct {
	gorm.Model
	SettlementID uuid.UUID
	ID           uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	FlowID            common.FlowID   `gorm:"column:flow_id;"`                       // ID of the collectible NFT
	ContractReference AddressLocation `gorm:"embedded;embeddedPrefix:contract_ref_"` // Reference to the collectible NFT contract
	Settled           bool            `gorm:"column:settled"`
}

type SettlementCollectibles []SettlementCollectible

func (Settlement) TableName() string {
	return "settlements"
}

func (s *Settlement) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return nil
}

func (SettlementCollectible) TableName() string {
	return "settlement_collectibles"
}

func (s *SettlementCollectible) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
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
