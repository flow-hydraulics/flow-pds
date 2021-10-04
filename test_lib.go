package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	"github.com/onflow/flow-go-sdk/client"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

var testLogger *log.Logger

func cleanTestDatabase(cfg *config.Config, db *gorm.DB) {
	// Only run this if database DSN contains "test"
	if strings.Contains(strings.ToLower(cfg.DatabaseDSN), "test") {
		// "If you perform a batch delete without any conditions, GORM WON’T run it, and will return ErrMissingWhereClause error
		// You have to use some conditions or use raw SQL or enable AllowGlobalUpdate"
		// Unscoped to prevent Soft Delete
		db.Unscoped().Where("1 = 1").Delete(&app.Distribution{})
		db.Unscoped().Where("1 = 1").Delete(&app.Bucket{})
		db.Unscoped().Where("1 = 1").Delete(&app.Pack{})
		db.Unscoped().Where("1 = 1").Delete(&app.Settlement{})
		db.Unscoped().Where("1 = 1").Delete(&app.SettlementCollectible{})
		db.Unscoped().Where("1 = 1").Delete(&app.Minting{})
		db.Unscoped().Where("1 = 1").Delete(&app.CirculatingPackContract{})
		db.Unscoped().Where("1 = 1").Delete(&transactions.StorableTransaction{})
	}
}

func getTestCfg() *config.Config {
	cfg, err := config.ParseConfig(&config.ConfigOptions{EnvFilePath: ".env.test"})
	if err != nil {
		panic(err)
	}

	if !strings.Contains(strings.ToLower(cfg.DatabaseDSN), "test") {
		cfg.DatabaseDSN = "test.db"
		cfg.DatabaseType = "sqlite"
	}

	return cfg
}

func getTestApp(cfg *config.Config, poll bool) (*app.App, func()) {
	if testLogger == nil {
		testLogger = log.New()
	}

	flowClient, err := client.New(cfg.AccessAPIHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	db, err := common.NewGormDB(cfg)
	if err != nil {
		panic(err)
	}

	cleanTestDatabase(cfg, db)

	// Migrate app database
	if err := app.Migrate(db); err != nil {
		panic(err)
	}
	if err := transactions.Migrate(db); err != nil {
		panic(err)
	}

	app := app.New(cfg, testLogger, db, flowClient, poll)

	clean := func() {
		app.Close()
		flowClient.Close()
		cleanTestDatabase(cfg, db)
	}

	return app, clean
}

func getTestServer(cfg *config.Config, poll bool) (*http.Server, func()) {
	if testLogger == nil {
		testLogger = log.New()
	}

	app, cleanupApp := getTestApp(cfg, poll)
	clean := func() {
		cleanupApp()
	}

	return http.NewServer(cfg, testLogger, app), clean
}

func makeTestCollection(size int) []common.FlowID {
	collection := make([]common.FlowID, size)
	for i := range collection {
		collection[i] = common.FlowID{Int64: int64(i + 1), Valid: true}
	}
	return collection
}

func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func AssertNotEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		return
	}
	t.Error("Did not expect to equal")
}
