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

func GetDistributionPacks(db *gorm.DB, distributionID uuid.UUID) ([]Pack, error) {
	list := []Pack{}
	if err := db.Where(&Pack{DistributionID: distributionID}).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func GetMintingPack(db *gorm.DB, distributionID uuid.UUID, h common.BinaryValue) (*Pack, error) {
	pack := Pack{}
	if err := db.Where(&Pack{DistributionID: distributionID, CommitmentHash: h, FlowID: common.FlowID{Valid: false}}).First(&pack).Error; err != nil {
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

// Insert settlement
func InsertSettlement(db *gorm.DB, d *Settlement) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Store settlement
		if err := tx.Omit(clause.Associations).Create(d).Error; err != nil {
			return err
		}

		// Update IDs
		for i := range d.Collectibles {
			d.Collectibles[i].SettlementID = d.ID
		}

		// Store collectibles in batches
		if err := tx.Omit(clause.Associations).CreateInBatches(d.Collectibles, 1000).Error; err != nil {
			return err
		}

		// Commit
		return nil
	})
}

// Update settlement
func UpdateSettlement(db *gorm.DB, d *Settlement) error {
	return db.Omit(clause.Associations).Save(d).Error
}

// Update settlement collectible
func UpdateSettlementCollectible(db *gorm.DB, d *SettlementCollectible) error {
	return db.Omit(clause.Associations).Save(d).Error
}

// Get settlement
func GetDistributionSettlement(db *gorm.DB, distributionID uuid.UUID) (*Settlement, error) {
	settlement := Settlement{}
	if err := db.Omit(clause.Associations).Where(&Settlement{DistributionID: distributionID}).First(&settlement).Error; err != nil {
		return nil, err
	}
	return &settlement, nil
}

// Get missing collectibles for a settlement, grouped by collectible contract reference
func MissingCollectibles(db *gorm.DB, settlementId uuid.UUID) (map[string]SettlementCollectibles, error) {
	missing := []SettlementCollectible{}
	err := db.Omit(clause.Associations).Where(&SettlementCollectible{SettlementID: settlementId, IsSettled: false}).Find(&missing).Error
	if err != nil {
		return nil, err
	}

	res := make(map[string]SettlementCollectibles)
	for _, c := range missing {
		key := c.ContractReference.String()
		if _, ok := res[key]; !ok {
			res[key] = SettlementCollectibles{}
		}
		res[key] = append(res[key], c)
	}

	return res, nil
}

// Get settlement
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
	return db.Save(d).Error
}

// Insert Minting
func InsertMinting(db *gorm.DB, d *Minting) error {
	return db.Omit(clause.Associations).Create(d).Error
}

// Get minting
func GetDistributionMinting(db *gorm.DB, distributionID uuid.UUID) (*Minting, error) {
	minting := Minting{}
	if err := db.Omit(clause.Associations).Where(&Minting{DistributionID: distributionID}).First(&minting).Error; err != nil {
		return nil, err
	}
	return &minting, nil
}

// Update minting
func UpdateMinting(db *gorm.DB, d *Minting) error {
	return db.Omit(clause.Associations).Save(d).Error
}
