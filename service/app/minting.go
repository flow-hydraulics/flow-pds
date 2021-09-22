package app

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Minting struct {
	gorm.Model
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`
	DistributionID uuid.UUID `gorm:"unique"`
	Distribution   Distribution

	Minted uint `gorm:"column:minted"`
	Total  uint `gorm:"column:total"`

	LastCheckedBlock uint64 `gorm:"column:last_checked_block"`
}

func (Minting) TableName() string {
	return "mintings"
}

func (s *Minting) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return nil
}

func (m *Minting) IsComplete() bool {
	return m.Minted >= m.Total
}

func (m *Minting) IncrementMinted() {
	m.Minted++
}
