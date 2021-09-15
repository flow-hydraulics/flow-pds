package main

import (
	"os"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/http"
)

func getTestCfg() *config.Config {
	return &config.Config{
		DatabaseDSN:  "test.db",
		DatabaseType: "sqlite",
	}
}

func getTestApp(cfg *config.Config) (*app.App, func()) {
	db, err := app.NewGormDB(cfg)
	if err != nil {
		panic(err)
	}
	store := app.NewGormStore(db)
	clean := func() {
		os.Remove(cfg.DatabaseDSN)
	}
	return app.New(cfg, store, nil), clean
}

func getTestServer(cfg *config.Config) (*http.Server, func()) {
	app, cleanupApp := getTestApp(cfg)
	clean := func() {
		cleanupApp()
	}
	return http.NewServer(cfg, nil, app), clean
}

func makeCollection(size int) []common.FlowID {
	collection := make([]common.FlowID, size)
	for i := range collection {
		collection[i] = common.FlowID(i + 1)
	}
	return collection
}
