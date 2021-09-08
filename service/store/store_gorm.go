package store

import (
	"gorm.io/gorm"
)

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db}
}

// Insert distribution
func (s *GormStore) InsertDistribution(*Distribution) error { return nil }

// Update distribution
func (s *GormStore) UpdateDistribution(*Distribution) error { return nil }

// Remove distribution
func (s *GormStore) RemoveDistribution(*Distribution) error { return nil }

// List distributions
func (s *GormStore) ListDistributions(ListOptions) ([]*Distribution, error) { return nil, nil }

// Get distribution
func (s *GormStore) GetDistribution() (*Distribution, error) { return nil, nil }
