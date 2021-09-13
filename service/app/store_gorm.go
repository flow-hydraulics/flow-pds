package app

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	db.AutoMigrate(&Distribution{}, &Bucket{}, &Pack{}, &PackSlot{})
	return &GormStore{db}
}

// Insert distribution
func (s *GormStore) InsertDistribution(d *Distribution) error {
	return s.db.Create(d).Error
}

// Update distribution
// Note: this will not update nested objects (Buckets, Packs)
func (s *GormStore) UpdateDistribution(d *Distribution) error {
	// Omit associations as saving associations (nested objects) was causing
	// duplicates of them to be created on each update.
	return s.db.Omit(clause.Associations).Save(d).Error
}

// Remove distribution
func (s *GormStore) RemoveDistribution(*Distribution) error {
	// TODO (latenssi)
	return nil
}

// List distributions
func (s *GormStore) ListDistributions(opt ListOptions) ([]Distribution, error) {
	list := []Distribution{}
	if err := s.db.Order("created_at desc").Limit(opt.Limit).Offset(opt.Offset).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Get distribution
func (s *GormStore) GetDistribution(id uuid.UUID) (*Distribution, error) {
	distribution := Distribution{}
	if err := s.db.Preload("Packs.Slots").Preload(clause.Associations).First(&distribution, id).Error; err != nil {
		return nil, err
	}
	return &distribution, nil
}
