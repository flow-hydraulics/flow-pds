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
}

func (Settlement) TableName() string {
	return "settlements"
}

func (s *Settlement) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return nil
}
