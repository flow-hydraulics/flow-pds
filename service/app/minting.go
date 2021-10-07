package app

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Minting represents the minting status of a distribution.
type Minting struct {
	gorm.Model
	ID             uuid.UUID    `gorm:"column:id;primary_key;type:uuid;"`
	DistributionID uuid.UUID    `gorm:"unique"`
	Distribution   Distribution `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CurrentCount uint   `gorm:"column:current_count"`
	TotalCount   uint   `gorm:"column:total_count"`
	StartAtBlock uint64 `gorm:"column:start_at_block"`
}

func (Minting) TableName() string {
	return "mintings"
}

func (m *Minting) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.New()
	return nil
}

func (m *Minting) IsComplete() bool {
	return m.CurrentCount >= m.TotalCount
}

func (m *Minting) IncrementCount() {
	m.CurrentCount++
}
