package main

import (
	"context"
	"strings"
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
	cfg := getTestCfg()
	a, cleanup := getTestApp(cfg, true)
	defer func() {
		cleanup()
	}()

	g := gwtf.NewGoWithTheFlow([]string{"./flow.json"}, "emulator", false, 0)

	t.Log("Setting up collectible NFT (ExampleNFT) collection for owner")

	setupExampleNFT := "./cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	setupExampleNFTCode := util.ParseCadenceTemplate(setupExampleNFT)
	_, err := g.
		TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Setting up collectible NFT (ExampleNFT) collection for PDS")

	_, err = g.
		TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("pds").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Issuer create PackIssuer resource to store DistCap")

	createPackIssuer := "./cadence-transactions/pds/create_new_pack_issuer.cdc"
	createPackIssuerCode := util.ParseCadenceTemplate(createPackIssuer)
	_, err = g.
		TransactionFromFile(createPackIssuer, createPackIssuerCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Issuer create PackNFT collection resource to store minted PackNFT")

	createPackNFTCollection := "./cadence-transactions/packNFT/create_new_packNFT_collection.cdc"
	createPackNFTCollectionCode := util.ParseCadenceTemplate(createPackNFTCollection)
	_, err = g.
		TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Owner create PackNFT collection resource to store PackNFT after purchase")

	_, err = g.
		TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// -- Mint some collectible NFTs as issuer --
	t.Log("Mint some collectible NFTs as issuer")

	// First check if we need more
	balanceExampleNFT := "./cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
	balanceExampleNFTCode := util.ParseCadenceTemplate(balanceExampleNFT)

	available, err := g.
		ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).
		AccountArgument("issuer").
		RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	// Mint so we have at least 5
	mintCount := 5 - len(available.(cadence.Array).Values)

	mintExampleNFT := "./cadence-transactions/exampleNFT/mint_exampleNFT.cdc"
	mintExampleNFTCode := util.ParseCadenceTemplate(mintExampleNFT)
	for i := 0; i < mintCount; i++ {
		_, err := g.
			TransactionFromFile(mintExampleNFT, mintExampleNFTCode).
			SignProposeAndPayAs("issuer").
			AccountArgument("issuer").
			RunE()
		if err != nil {
			t.Fatal(err)
		}
	}

	issuerCollectibleNFTs, err := g.ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).
		AccountArgument("issuer").RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Issuer available collectible NFTs: %s (%d)\n", issuerCollectibleNFTs.String(), len(issuerCollectibleNFTs.(cadence.Array).Values))

	t.Log("PDS share DistCap to PackIssuer (owned by Issuer)")

	setDistCap := "./cadence-transactions/pds/set_pack_issuer_cap.cdc"
	setDistCapCode := util.ParseCadenceTemplate(setDistCap)
	_, err = g.
		TransactionFromFile(setDistCap, setDistCapCode).
		SignProposeAndPayAs("pds").
		AccountArgument("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Issuer creates distribution on chain")

	pdsDistId := "./cadence-scripts/pds/get_current_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
	currentDistId, err := g.ScriptFromFile(pdsDistId, pdsDistIdCode).RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	createDist := "./cadence-transactions/pds/create_distribution.cdc"
	createDistCode := util.ParseCadenceTemplate(createDist)
	// Private path must match the PackNFT contract
	e, err := g.
		TransactionFromFile(createDist, createDistCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "exampleNFTCollectionProvider"}).
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events := util.ParseTestEvents(e)

	util.NewExpectedPDSEvent("DistributionCreated").AddField("DistId", currentDistId.String()).AssertEqual(t, events[0])

	// -- Create distribution --

	t.Log("Use available NFTs to create a distribution in backend")

	issuer := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "issuer")))

	distId, err := common.FlowIDFromCadence(currentDistId)
	if err != nil {
		t.Fatal(err)
	}

	collection, err := common.FlowIDListFromCadence(issuerCollectibleNFTs)
	if err != nil {
		t.Fatal(err)
	}

	distribution := app.Distribution{
		FlowID: distId,
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

	if err := a.CreateDistribution(context.Background(), &distribution); err != nil {
		t.Fatal(err)
	}

	resolved := distribution.ResolvedCollection()
	resolvedStr := make([]string, len(resolved))
	for i, c := range resolved {
		resolvedStr[i] = c.String()
	}
	t.Logf("Distribution created with collectibles:\n%s\n", strings.Join(resolvedStr, "\n"))

	// -- Resolve, settle and mint --

	t.Log("Wait for the distribution to complete")

	for {
		d, _, err := a.GetDistribution(context.Background(), distribution.ID)
		if err != nil {
			t.Fatal(err)
		}
		if d.State == common.DistributionStateComplete {
			distribution = *d
			break
		}
		time.Sleep(time.Second)
	}

	ownerCollectibleNFTsBefore, err := g.
		ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).
		AccountArgument("owner").
		RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Picking one pack")

	// Pick one pack
	randomPack := distribution.Packs[0]
	randomPackID := cadence.UInt64(randomPack.FlowID.Int64)

	if len(randomPack.Collectibles) != 2 {
		t.Fatal("expected pack to contain 2 collectibles")
	}

	collectibles := randomPack.Collectibles
	collectiblesStr := make([]string, len(collectibles))
	for i, c := range collectibles {
		collectiblesStr[i] = c.String()
	}
	t.Logf("Collectible NFTs in the pack:\n%s\n", strings.Join(collectiblesStr, "\n"))

	// -- Transfer --

	t.Log("Transferring a pack to owner")

	// Issuer transfer PackNFT to owner
	transferPackNFT := "./cadence-transactions/packNFT/transfer_packNFT.cdc"
	transferPackNFTCode := util.ParseCadenceTemplate(transferPackNFT)
	_, err = g.
		TransactionFromFile(transferPackNFT, transferPackNFTCode).
		SignProposeAndPayAs("issuer").
		AccountArgument("owner").
		Argument(randomPackID).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// -- Reveal --

	t.Log("Owner requests to reveal the pack")

	revealRequest := "./cadence-transactions/packNFT/reveal_request.cdc"
	revealRequestCode := util.ParseCadenceTemplate(revealRequest)
	e, err = g.
		TransactionFromFile(revealRequest, revealRequestCode).
		SignProposeAndPayAs("owner").
		Argument(randomPackID).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	events = util.ParseTestEvents(e)
	// Onwer withdraw PackNFT from the collection, calls reveal on it and deposits back into their collection
	util.NewExpectedPackNFTEvent("RevealRequest").AddField("id", randomPackID.String()).AssertEqual(t, events[0])

	t.Log("PDS backend submits reveal transaction")

	t.Log("Wait for the reveal to happen")
	for {
		p, err := a.GetPack(context.Background(), randomPack.ID)
		if err != nil {
			t.Fatal(err)
		}
		if p.State == common.PackStateRevealed {
			break
		}
		time.Sleep(time.Second)
	}

	// -- Open --

	t.Log("Owner requests to open the pack")

	openRequest := "./cadence-transactions/packNFT/open_request.cdc"
	openRequestCode := util.ParseCadenceTemplate(openRequest)
	e, err = g.
		TransactionFromFile(openRequest, openRequestCode).
		SignProposeAndPayAs("owner").
		Argument(randomPackID).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	events = util.ParseTestEvents(e)
	// Onwer withdraw PackNFT from the collection, calls open on it and deposits back into their collection
	util.NewExpectedPackNFTEvent("OpenRequest").AddField("id", randomPackID.String()).AssertEqual(t, events[0])

	t.Log("PDS backend submits open transaction")

	t.Log("Wait for the open to happen")
	for {
		p, err := a.GetPack(context.Background(), randomPack.ID)
		if err != nil {
			t.Fatal(err)
		}
		if p.State == common.PackStateOpened {
			break
		}
		time.Sleep(time.Second)
	}

	// Wait a bit more as the blocktime might be 1s if run from the test script
	time.Sleep(time.Second * 2)

	ownerCollectibleNFTsAfter, err := g.
		ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).
		AccountArgument("owner").
		RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	ownerCollectibleIDs, err := common.FlowIDListFromCadence(ownerCollectibleNFTsAfter)
	if err != nil {
		t.Fatal(err)
	}

	randomPackCollectibleIDs := make(common.FlowIDList, len(randomPack.Collectibles))
	for i, c := range randomPack.Collectibles {
		randomPackCollectibleIDs[i] = c.FlowID
	}

	for _, id := range randomPackCollectibleIDs {
		if _, ok := ownerCollectibleIDs.Contains(id); !ok {
			t.Errorf("expected owner to have collectible NFT: %s", id)
		}
	}

	t.Logf("Owner collectible NFTs before: %s\n", ownerCollectibleNFTsBefore.String())
	t.Logf("Owner collectible NFTs after:  %s\n", ownerCollectibleNFTsAfter.String())
}
