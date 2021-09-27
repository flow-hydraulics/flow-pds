package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func cleanTestDatabase(cfg *config.Config, db *gorm.DB) {
	// Only run this if database DSN contains "test"
	if strings.Contains(strings.ToLower(cfg.DatabaseDSN), "test") {
		db.Delete(app.Distribution{})
		db.Delete(app.Bucket{})
		db.Delete(app.Pack{})
		db.Delete(app.Settlement{})
		db.Delete(app.SettlementCollectible{})
		db.Delete(app.Minting{})
		db.Delete(app.CirculatingPackContract{})
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

	app := app.New(cfg, db, flowClient, poll)

	clean := func() {
		app.Close()
		flowClient.Close()
		cleanTestDatabase(cfg, db)
	}

	return app, clean
}

func getTestServer(cfg *config.Config, poll bool) (*http.Server, func()) {
	app, cleanupApp := getTestApp(cfg, poll)
	clean := func() {
		cleanupApp()
	}
	return http.NewServer(cfg, nil, app), clean
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
