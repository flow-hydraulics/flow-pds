package transactions

import (
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Migrate(db *gorm.DB) error {
	db.AutoMigrate(&StorableTransaction{})
	return nil
}

func (StorableTransaction) TableName() string {
	return "transactions"
}

func (t *StorableTransaction) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = uuid.New()
	return nil
}

func (t *StorableTransaction) Save(db *gorm.DB) error {
	return db.Omit(clause.Associations).Save(t).Error
}

func GetTransaction(db *gorm.DB, id uuid.UUID) (*StorableTransaction, error) {
	t := StorableTransaction{}
	return &t, db.First(&t, id).Error
}

func SendableIDs(db *gorm.DB) ([]uuid.UUID, error) {
	list := []StorableTransaction{}
	err := db.Select("id").Order("created_at desc").
		Where(map[string]interface{}{"state": common.TransactionStateInit}).
		Or(map[string]interface{}{"state": common.TransactionStateRetry}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	res := make([]uuid.UUID, len(list))
	for i, t := range list {
		res[i] = t.ID
	}
	return res, nil
}

func SentIDs(db *gorm.DB) ([]uuid.UUID, error) {
	list := []StorableTransaction{}
	err := db.Select("id").Order("created_at desc").
		Where(map[string]interface{}{"state": common.TransactionStateSent}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	res := make([]uuid.UUID, len(list))
	for i, t := range list {
		res[i] = t.ID
	}
	return res, nil
}
