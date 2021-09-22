package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
)

func getTestCfg() *config.Config {
	cfg, err := config.ParseConfig(&config.ConfigOptions{EnvFilePath: ".env.test"})
	if err != nil {
		panic(err)
	}

	cfg.DatabaseDSN = "test.db"
	cfg.DatabaseType = "sqlite"

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

	// Migrate app database
	if err := app.Migrate(db); err != nil {
		panic(err)
	}

	clean := func() {
		if cfg.DatabaseType == "sqlite" {
			os.Remove(cfg.DatabaseDSN)
		}
		flowClient.Close()
	}

	return app.New(cfg, db, flowClient, poll), clean
}

func getTestServer(cfg *config.Config) (*http.Server, func()) {
	app, cleanupApp := getTestApp(cfg, false)
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
