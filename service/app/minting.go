package app

import (
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Minting struct {
	gorm.Model
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`
	DistributionID uuid.UUID `gorm:"unique"`
	Distribution   Distribution

	State  common.MintingState `gorm:"column:state"`
	Minted uint                `gorm:"column:minted"`
	Total  uint                `gorm:"column:total"`

	LastCheckedBlock uint64 `gorm:"column:last_checked_block"`
}

func (Minting) TableName() string {
	return "mintings"
}

func (s *Minting) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return nil
}
