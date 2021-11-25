package app

import (
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Migrate(db *gorm.DB) error {
	db.AutoMigrate(&Distribution{}, &Bucket{}, &Pack{})
	db.AutoMigrate(&Settlement{}, &SettlementCollectible{})
	db.AutoMigrate(&Minting{})
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
		if err := tx.Omit(clause.Associations).Create(d.PackTemplate.Buckets).Error; err != nil {
			return err
		}

		// Store packs in batches
		if err := tx.Omit(clause.Associations).CreateInBatches(d.Packs, 1000).Error; err != nil {
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

// List distributions
func ListDistributions(db *gorm.DB, opt ListOptions) ([]Distribution, error) {
	list := []Distribution{}
	if err := db.Omit(clause.Associations).Order("created_at desc").Limit(opt.Limit).Offset(opt.Offset).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Get distribution
func GetDistributionBig(db *gorm.DB, id uuid.UUID) (*Distribution, error) {
	distribution := Distribution{}
	if err := db.Preload(clause.Associations).First(&distribution, id).Error; err != nil {
		return nil, err
	}
	return &distribution, nil
}

func GetDistributionSmall(db *gorm.DB, id uuid.UUID) (*Distribution, error) {
	distribution := Distribution{}
	if err := db.Omit(clause.Associations).First(&distribution, id).Error; err != nil {
		return nil, err
	}
	return &distribution, nil
}

type BucketSmall struct {
	ID                   uuid.UUID       `gorm:"column:id;primary_key;type:uuid;"`
	CollectibleReference AddressLocation `gorm:"embedded;embeddedPrefix:collectible_ref_"`
}

func GetDistributionBucketsSmall(db *gorm.DB, distributionID uuid.UUID) ([]BucketSmall, error) {
	list := []BucketSmall{}
	if err := db.Omit(clause.Associations).Model(&Bucket{}).Where(&Bucket{DistributionID: distributionID}).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Get pack
func GetPack(db *gorm.DB, id uuid.UUID) (*Pack, error) {
	pack := Pack{}
	if err := db.First(&pack, id).Error; err != nil {
		return nil, err
	}
	return &pack, nil
}

// Get Packs for a Distribution and process in batches of 'batchSize'
func DistributionPacksInBatches(db *gorm.DB, distributionID uuid.UUID, batchSize int, processBatch func(tx *gorm.DB, batchNumber int, batch []Pack) error) error {
	batch := []Pack{}
	return db.
		Omit(clause.Associations).
		Where(&Pack{DistributionID: distributionID}).
		FindInBatches(&batch, batchSize, func(tx *gorm.DB, batchNumber int) error {
			return processBatch(tx, batchNumber, batch)
		}).Error
}

// GetMintingPack returns a pack which has no FlowID by its commitmentHash (therefore it should still be minting)
func GetMintingPack(db *gorm.DB, commitmentHash common.BinaryValue) (*Pack, error) {
	pack := Pack{}
	if err := db.Where(&Pack{CommitmentHash: commitmentHash, FlowID: common.FlowID{Valid: false}}).First(&pack).Error; err != nil {
		return nil, err
	}
	return &pack, nil
}

func GetPackByContractAndFlowID(db *gorm.DB, ref AddressLocation, id common.FlowID) (*Pack, error) {
	pack := Pack{}
	if err := db.Where(&Pack{ContractReference: ref, FlowID: id}).First(&pack).Error; err != nil {
		return nil, err
	}
	return &pack, nil
}

func UpdatePack(db *gorm.DB, d *Pack) error {
	return db.Omit(clause.Associations).Save(d).Error
}

func InsertSettlement(db *gorm.DB, d *Settlement) error {
	return db.Omit(clause.Associations).Create(d).Error
}

func InsertSettlementCollectibles(db *gorm.DB, cc []SettlementCollectible) error {
	return db.Omit(clause.Associations).CreateInBatches(cc, 1000).Error
}

// Delete Settlement
func DeleteSettlementForDistribution(db *gorm.DB, distributionID uuid.UUID) error {
	settlement, err := GetDistributionSettlement(db, distributionID)
	if err != nil {
		return err
	}
	return db.Select("Collectibles").Delete(settlement).Error
}

// Update Settlement
func UpdateSettlement(db *gorm.DB, d *Settlement) error {
	return db.Omit(clause.Associations).Save(d).Error
}

// Update Settlement collectible
func UpdateSettlementCollectible(db *gorm.DB, d *SettlementCollectible) error {
	return db.Omit(clause.Associations).Save(d).Error
}

// Get Settlement
func GetDistributionSettlement(db *gorm.DB, distributionID uuid.UUID) (*Settlement, error) {
	settlement := Settlement{}
	if err := db.Omit(clause.Associations).Where(&Settlement{DistributionID: distributionID}).First(&settlement).Error; err != nil {
		return nil, err
	}
	return &settlement, nil
}

// Get SettlementCollectibles that have not been settled for a Settlement and process in batches of 'batchSize'
func NotSettledCollectiblesInBatches(db *gorm.DB, settlementId uuid.UUID, batchSize int, processBatch func(tx *gorm.DB, batchNumber int, batch SettlementCollectibles) error) error {
	batch := SettlementCollectibles{}
	return db.
		Omit(clause.Associations).
		Where(&SettlementCollectible{SettlementID: settlementId, IsSettled: false}).
		FindInBatches(&batch, batchSize, func(tx *gorm.DB, batchNumber int) error {
			return processBatch(tx, batchNumber, batch)
		}).Error
}

// Get Settlement
func GetCirculatingPackContract(db *gorm.DB, name string, address common.FlowAddress) (*CirculatingPackContract, error) {
	circulatingPackContract := CirculatingPackContract{}
	if err := db.Where(&CirculatingPackContract{Name: name, Address: address}).First(&circulatingPackContract).Error; err != nil {
		return nil, err
	}
	return &circulatingPackContract, nil
}

// Insert CirculatingPackContract
func InsertCirculatingPackContract(db *gorm.DB, d *CirculatingPackContract) error {
	return db.Omit(clause.Associations).Create(d).Error
}

// Update CirculatingPackContracts
func UpdateCirculatingPackContract(db *gorm.DB, d *CirculatingPackContract) error {
	return db.Omit(clause.Associations).Save(d).Error
}

// Insert Minting
func InsertMinting(db *gorm.DB, d *Minting) error {
	return db.Omit(clause.Associations).Create(d).Error
}

// Delete Minting
func DeleteMintingForDistribution(db *gorm.DB, distributionID uuid.UUID) error {
	minting, err := GetDistributionMinting(db, distributionID)
	if err != nil {
		return err
	}
	return db.Delete(minting).Error
}

// Get Minting
func GetDistributionMinting(db *gorm.DB, distributionID uuid.UUID) (*Minting, error) {
	minting := Minting{}
	if err := db.Omit(clause.Associations).Where(&Minting{DistributionID: distributionID}).First(&minting).Error; err != nil {
		return nil, err
	}
	return &minting, nil
}

// Update Minting
func UpdateMinting(db *gorm.DB, d *Minting) error {
	return db.Omit(clause.Associations).Save(d).Error
}
