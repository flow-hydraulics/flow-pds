package common

import (
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	dbTypePostgresql = "psql"
	dbTypeMysql      = "mysql"
	dbTypeSqlite     = "sqlite"
)

func NewGormDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.DatabaseType {
	default:
		return nil, fmt.Errorf("database type '%s' not supported", cfg.DatabaseType)
	case dbTypePostgresql:
		dialector = postgres.Open(cfg.DatabaseDSN)
	case dbTypeMysql:
		dialector = mysql.Open(cfg.DatabaseDSN)
	case dbTypeSqlite:
		dialector = sqlite.Open(cfg.DatabaseDSN)
	}

	options := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(dialector, options)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CloseGormDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		panic("unable to close database")
	}

	if err := sqlDB.Close(); err != nil {
		panic("unable to close database")
	}
}
