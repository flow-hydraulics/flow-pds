package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
)

func TestE2E(t *testing.T) {
	t.Skip("skipping for now as this requires a flow emulator")

	cfg := getTestCfg()
	a, cleanup := getTestApp(cfg, true)
	defer func() {
		cleanup()
	}()

	jsonPath := "./flow.json"
	var flowJSON []string = []string{jsonPath}
	g := gwtf.NewGoWithTheFlow(flowJSON, "emulator", false, 3)

	issuer := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "issuer")))
	// pds := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "pds")))
	// owner := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "owner")))

	// -- Mint example NFTs as issuer --

	mintExampleNFT := "./cadence-transactions/exampleNFT/mint_exampleNFT.cdc"
	code1 := util.ParseCadenceTemplate(mintExampleNFT)
	for i := 0; i < 5; i++ {
		_, err := g.TransactionFromFile(mintExampleNFT, code1).
			SignProposeAndPayAs("issuer").
			AccountArgument("issuer").
			RunE()
		if err != nil {
			t.Fatal(err)
		}
	}

	balanceExampleNFT := "./cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
	code2 := util.ParseCadenceTemplate(balanceExampleNFT)
	nftIDs, err := g.ScriptFromFile(balanceExampleNFT, code2).
		AccountArgument("issuer").RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	arr, ok := nftIDs.(cadence.Array)
	if !ok {
		t.Fatal("can not convert")
	}
	collection := make(common.FlowIDList, len(arr.Values))
	for i := 0; i < len(arr.Values); i++ {
		v, ok := arr.Values[i].(cadence.UInt64)
		if !ok {
			t.Fatal("can not convert 2")
		}
		collection[i] = common.FlowID(v)
	}

	// -- Use newly minted NFTs to create a distribution as issuer --
	d := app.Distribution{
		DistID: 1, // TODO
		Issuer: issuer,
		PackTemplate: app.PackTemplate{
			PackReference: app.AddressLocation{
				Name:    "PackNFT",
				Address: issuer,
			},
			PackCount: 2,
			Buckets: []app.Bucket{
				{
					CollectibleReference: app.AddressLocation{
						Name:    "ExampleNFT",
						Address: issuer,
					},
					CollectibleCount:      2,
					CollectibleCollection: collection,
				},
			},
		},
	}

	if err := a.CreateDistribution(context.Background(), &d); err != nil {
		t.Fatal(err)
	}

	// -- Settle --

	// setup examplenft collection for pds (placeholder escrow)
	setupExampleNFT := "./cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	code3 := util.ParseCadenceTemplate(setupExampleNFT)
	_, err = g.TransactionFromFile(setupExampleNFT, code3).
		SignProposeAndPayAs("pds").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for settlement

	wg := &sync.WaitGroup{}
	wg.Add(1)
	var waitError error
	go func() {
		// TODO (latenssi): timeout
		for {
			dist, _, err := a.GetDistribution(context.Background(), d.ID)
			if err != nil {
				waitError = err
				break
			}
			if dist.State == common.DistributionStateMinting {
				break
			}
			time.Sleep(time.Second * 1)
		}
		wg.Done()
	}()

	if err := a.SettleDistribution(context.Background(), d.ID); err != nil {
		t.Fatal(err)
	}

	// transfer
	// TODO: use PDS contract interface instead of manually transfering
	transferExampleNFT := "./cadence-transactions/exampleNFT/transfer_exampleNFT.cdc"
	code4 := util.ParseCadenceTemplate(transferExampleNFT)
	for _, c := range d.ResolvedCollection() {
		_, err := g.TransactionFromFile(transferExampleNFT, code4).
			SignProposeAndPayAs("issuer").
			AccountArgument("pds").
			Argument(cadence.UInt64(c.FlowID)).
			RunE()
		if err != nil {
			t.Fatal(err)
		}
	}

	wg.Wait()

	if waitError != nil {
		t.Fatal(waitError)
	}

	// -- Mint --

	// Start minting pack NFTs as pds using mint proxy from issuer (should store nfts righ into issuers collection)

	// Wait for minting to finish

	// -- Reveal --

	// -- Retrieve --
}
