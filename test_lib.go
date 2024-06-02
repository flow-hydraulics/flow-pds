package main

import (
	"context"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

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

func getTestCfg(t *testing.T, b *testing.B) *config.Config {

	cfg, err := config.ParseConfig(&config.ConfigOptions{EnvFilePath: ".env.test"})
	if err != nil {
		panic(err)
	}

	cfg.DatabaseDSN = "postgresql://pds:pds@localhost:5432/pds"
	cfg.DatabaseType = "psql"

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
	if err := transactions.Migrate(db); err != nil {
		panic(err)
	}

	app, err := app.New(cfg, db, flowClient, poll)
	if err != nil {
		panic(err)
	}

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

	return http.NewServer(cfg, app), clean
}

func makeTestCollection(size int) []common.FlowID {
	collection := make([]common.FlowID, size)
	for i := range collection {
		collection[i] = common.FlowID{Int64: int64(i + 1), Valid: true}
	}
	return collection
}

func getExampleNFTBalance(flowClient *client.Client, address flow.Address) (uint64, error) {

	balanceScript, err := flow_helpers.ParseCadenceTemplate(
		"./cadence-scripts/collectibleNFT/balance.cdc",
		&flow_helpers.CadenceTemplateVars{
			NonFungibleToken:      "f8d6e0586b0a20c7",
			CollectibleNFTName:    "ExampleNFT",
			CollectibleNFTAddress: "01cf0e2f2f715450",
		},
	)
	if err != nil {
		return 0, err
	}

	balanceArgs := []cadence.Value{
		cadence.NewAddress(address),
	}

	v, err := flowClient.ExecuteScriptAtLatestBlock(context.Background(), balanceScript, balanceArgs)
	if err != nil {
		return 0, err
	}

	return v.ToGoValue().(*big.Int).Uint64(), err
}

func getExampleNFTIDs(flowClient *client.Client, address flow.Address, balance uint64) (common.FlowIDList, error) {

	idsScript, err := flow_helpers.ParseCadenceTemplate(
		"./cadence-scripts/collectibleNFT/balance_ids.cdc",
		&flow_helpers.CadenceTemplateVars{
			NonFungibleToken:      "f8d6e0586b0a20c7",
			CollectibleNFTName:    "ExampleNFT",
			CollectibleNFTAddress: "01cf0e2f2f715450",
		},
	)
	if err != nil {
		return nil, err
	}

	var res common.FlowIDList

	limit := uint64(10000)

	for offset := uint64(0); offset < balance; offset = offset + limit {
		idsArgs := []cadence.Value{
			cadence.NewAddress(address),
			cadence.NewUInt64(offset),
			cadence.NewUInt64(limit),
		}

		ids, err := flowClient.ExecuteScriptAtLatestBlock(context.Background(), idsScript, idsArgs)
		if err != nil {
			return nil, err
		}

		for _, v := range ids.ToGoValue().([]interface{}) {
			uintID, ok := v.(uint64)
			if !ok {
				panic("unable to parse uint")
			}
			res = append(res, common.FlowID{Int64: int64(uintID), Valid: true})
		}
	}

	return res, err
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
