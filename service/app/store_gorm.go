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
	db.AutoMigrate(&Distribution{}, &Bucket{}, &Pack{}, &Collectible{})
	return &GormStore{db}
}

// Insert distribution
func (s *GormStore) InsertDistribution(d *Distribution) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Store distribution
		if err := tx.Omit(clause.Associations).Create(d).Error; err != nil {
			return err
		}

		// Update distribution IDs
		for i := range d.PackTemplate.Buckets {
			d.PackTemplate.Buckets[i].DistributionID = d.ID
		}

		for i := range d.Packs {
			d.Packs[i].DistributionID = d.ID
		}

		// Store buckets, assuming we won't have too many buckets per distribution
		if err := tx.Create(d.PackTemplate.Buckets).Error; err != nil {
			return err
		}

		// Store packs in batches
		if err := tx.Omit(clause.Associations).CreateInBatches(d.Packs, 1000).Error; err != nil {
			return err
		}

		// Store pack collectibles, assuming we won't have too many collectibles per pack
		for _, p := range d.Packs {
			for i := range p.Collectibles {
				// Update pack ID
				p.Collectibles[i].PackID = p.ID
			}
			if err := tx.Create(p.Collectibles).Error; err != nil {
				return err
			}
		}

		// Commit
		return nil
	})
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
	if err := s.db.Preload("Packs.Collectibles").Preload(clause.Associations).First(&distribution, id).Error; err != nil {
		return nil, err
	}
	return &distribution, nil
}
