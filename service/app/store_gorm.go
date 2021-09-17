package app

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Migrate(db *gorm.DB) error {
	db.AutoMigrate(&Distribution{}, &Bucket{}, &Pack{})
	db.AutoMigrate(&Settlement{})
	db.AutoMigrate(&CirculatingPackContract{})
	return nil
}

// Insert distribution
func InsertDistribution(db *gorm.DB, d *Distribution) error {
	return db.Transaction(func(tx *gorm.DB) error {
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
		if err := tx.CreateInBatches(d.Packs, 1000).Error; err != nil {
			return err
		}

		// Commit
		return nil
	})
}

// Update distribution
// Note: this will not update nested objects (Buckets, Packs)
func UpdateDistribution(db *gorm.DB, d *Distribution) error {
	// Omit associations as saving associations (nested objects) was causing
	// duplicates of them to be created on each update.
	return db.Omit(clause.Associations).Save(d).Error
}

// Remove distribution
func RemoveDistribution(*gorm.DB, *Distribution) error {
	// TODO (latenssi)
	return nil
}

// List distributions
func ListDistributions(db *gorm.DB, opt ListOptions) ([]Distribution, error) {
	list := []Distribution{}
	if err := db.Order("created_at desc").Limit(opt.Limit).Offset(opt.Offset).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Get distribution
func GetDistribution(db *gorm.DB, id uuid.UUID) (*Distribution, error) {
	distribution := Distribution{}
	if err := db.Preload(clause.Associations).First(&distribution, id).Error; err != nil {
		return nil, err
	}
	return &distribution, nil
}

// Insert settlement
func InsertSettlement(db *gorm.DB, d *Settlement) error {
	return db.Create(d).Error
}

// Update settlement
func UpdateSettlement(db *gorm.DB, d *Settlement) error {
	return db.Save(d).Error
}

// Get settlement
func GetSettlement(db *gorm.DB, distributionID uuid.UUID) (*Settlement, error) {
	settlement := Settlement{DistributionID: distributionID}
	if err := db.First(&settlement).Error; err != nil {
		return nil, err
	}
	return &settlement, nil
}

// Insert CirculatingPackContract
func InsertCirculatingPackContract(db *gorm.DB, d *CirculatingPackContract) error {
	return db.Create(d).Error
}

// Update CirculatingPackContracts
func UpdateCirculatingPackContracts(db *gorm.DB, d []CirculatingPackContract) error {
	return db.Save(d).Error
}
