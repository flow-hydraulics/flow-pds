package transactions

import (
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&StorableTransaction{}); err != nil {
		return err
	}
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

// GetTransaction returns a StorableTransaction from database.
func GetTransaction(db *gorm.DB, id uuid.UUID) (*StorableTransaction, error) {
	t := StorableTransaction{}
	return &t, db.First(&t, id).Error
}

func GetNextSendable(db *gorm.DB) (*StorableTransaction, error) {
	t := StorableTransaction{}
	err := db.Order("updated_at asc").
		Clauses(clause.Locking{Strength: "UPDATE SKIP LOCKED"}).
		Where(map[string]interface{}{"state": common.TransactionStateInit}).
		Or(map[string]interface{}{"state": common.TransactionStateRetry}).
		First(&t).Error
	return &t, err
}

func GetNextSendables(db *gorm.DB, limit int) ([]StorableTransaction, error) {
	var ts []StorableTransaction
	if err := db.Order("updated_at asc").
		Clauses(clause.Locking{Strength: "UPDATE SKIP LOCKED"}).
		Where(map[string]interface{}{"state": common.TransactionStateInit}).
		Or(map[string]interface{}{"state": common.TransactionStateRetry}).
		Limit(limit).
		Find(&ts).Error; err != nil {
		return nil, err
	}

	return ts, nil
}

func GetNextSent(db *gorm.DB) (*StorableTransaction, error) {
	t := StorableTransaction{}
	err := db.Order("updated_at asc").
		Where(map[string]interface{}{"state": common.TransactionStateSent}).
		First(&t).Error
	return &t, err
}

func GetNextSents(db *gorm.DB, limit int) ([]StorableTransaction, error) {
	var ts []StorableTransaction
	if err := db.Order("updated_at asc").
		Clauses(clause.Locking{Strength: "UPDATE SKIP LOCKED"}).
		Where(map[string]interface{}{"state": common.TransactionStateSent}).
		Limit(limit).
		Find(&ts).Error; err != nil {
		return nil, err
	}

	return ts, nil
}
