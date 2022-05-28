package main

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/flow-hydraulics/flow-pds/utils"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestE2E(t *testing.T) {
	cfg := getTestCfg(t, nil)
	a, cleanup := getTestApp(cfg, true)
	defer cleanup()

	no_packs := cfg.TestPackCount
	no_collectibles_per_pack := 5
	no_collectibles := no_packs * no_collectibles_per_pack

	g := gwtf.NewGoWithTheFlow([]string{"./flow.json"}, "emulator", false, 0)

	flowClient, err := client.New("localhost:3569", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	issuer := common.FlowAddress(g.Account("issuer").Address())

	t.Log("Setting up collectible NFT (ExampleNFT) collection for owner")

	// The caller wishing to create the collection will choose which Private Path they would like to link the
	// the Collection Provider Capability (when shared, to withdraw from their collection)
	// The Private Path string in this case is "NFTCollectionProvider"
	setupExampleNFT := "./cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	setupExampleNFTCode := util.ParseCadenceTemplate(setupExampleNFT)
	_, err = g.
		TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Issuer link NFT collection capability to share when create dist")

	linkScript := "./cadence-transactions/exampleNFT/link_providerCap_exampleNFT.cdc"
	linkCode := util.ParseCadenceTemplate(linkScript)
	_, err = g.TransactionFromFile(linkScript, linkCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "NFTCollectionProvider"}).
		RunE()
	assert.NoError(t, err)

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

	// -- Check if issuer needs more collectible NFTs --
	t.Log("Check if issuer needs more collectible NFTs")
	balance, err := getExampleNFTBalance(flowClient, flow.Address(issuer))
	if err != nil {
		t.Fatal(err)
	}

	if balance < uint64(no_collectibles) {
		// Mint so we have enough collectible NFTs
		mintBatchSize := int(math.Min(100, float64(no_collectibles)))
		mintBatchCount := int(math.Ceil(float64(no_collectibles-int(balance)) / float64(mintBatchSize)))

		t.Logf("Need to mint %d collectibles \n", mintBatchCount*mintBatchSize)

		mintExampleNFT := "./cadence-transactions/exampleNFT/mint_exampleNFTBatched.cdc"
		mintExampleNFTCode := util.ParseCadenceTemplate(mintExampleNFT)
		for i := 0; i < mintBatchCount; i++ {
			_, err := g.
				TransactionFromFile(mintExampleNFT, mintExampleNFTCode).
				SignProposeAndPayAs("issuer").
				AccountArgument("issuer").
				IntArgument(mintBatchSize).
				RunE()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	balance, err = getExampleNFTBalance(flowClient, flow.Address(issuer))
	if err != nil {
		t.Fatal(err)
	}

	collection, err := getExampleNFTIDs(flowClient, flow.Address(issuer), balance)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("PDS share DistCap to PackIssuer (owned by Issuer)")
	err = a.SetDistCap(context.Background(), issuer)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Issuer creates distribution on chain")

	pdsDistId := "./cadence-scripts/pds/get_next_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
	currentDistId, err := g.ScriptFromFile(pdsDistId, pdsDistIdCode).RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	keyPair := cadence.KeyValuePair{Key: utils.NewCadenceString("metadataKey"), Value: utils.NewCadenceString("metadataValue")}
	stringifiedKeyPair := "{\"metadataKey\": \"metadataValue\"}"
	var keypairArr []cadence.KeyValuePair
	keypairArr = append(keypairArr, keyPair)
	expMetadata := cadence.NewDictionary(keypairArr)
	expTitle := "ExampleDistTitle"

	createDist := "./cadence-transactions/pds/create_distribution.cdc"
	createDistCode, err := flow_helpers.ParseCadenceTemplate(
		createDist,
		&flow_helpers.CadenceTemplateVars{
			PackNFTName:    "PackNFT",
			PackNFTAddress: os.Getenv("PACKNFT_ADDRESS"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Private path must match the PackNFT contract
	e, err := g.
		TransactionFromFile(createDist, createDistCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "NFTCollectionProvider"}).
		StringArgument(expTitle).
		Argument(expMetadata).
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events := util.ParseTestEvents(e)

	util.NewExpectedPDSEvent("DistributionCreated").
		AddField("DistId", currentDistId.String()).
		AddField("state", "0").
		AddField("title", expTitle).
		AddField("metadata", stringifiedKeyPair).
		AssertEqual(t, events[0])
	// -- Create distribution --

	t.Log("Use available NFTs to create a distribution in backend")

	distId, err := common.FlowIDFromCadence(currentDistId)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()

	distribution := app.Distribution{
		State:  common.DistributionStateInit,
		FlowID: distId,
		Issuer: issuer,
		PackTemplate: app.PackTemplate{
			PackReference: app.AddressLocation{
				Name:    "PackNFT",
				Address: issuer,
			},
			PackCount: uint(no_packs),
			Buckets: []app.Bucket{
				{
					CollectibleReference: app.AddressLocation{
						Name:    "ExampleNFT",
						Address: issuer,
					},
					CollectibleCount:      uint(no_collectibles_per_pack),
					CollectibleCollection: collection,
				},
			},
		},
	}

	if err := a.CreateDistribution(context.Background(), &distribution); err != nil {
		t.Fatal(err)
	}

	collectibleCount, err := distribution.TemplateCollectibleCount()
	if err != nil {
		t.Fatal(err)
	}
	resolved := make(app.Collectibles, 0, collectibleCount)
	for _, pack := range distribution.Packs {
		resolved = append(resolved, pack.Collectibles...)
	}
	resolvedStr := make([]string, len(resolved))
	for i, c := range resolved {
		resolvedStr[i] = c.String()
	}
	if len(resolved) > 20 {
		t.Logf("Distribution created with collectibles: %d\n", len(resolved))
	} else {
		t.Logf("Distribution created with collectibles:\n%s\n", strings.Join(resolvedStr, "\n"))
	}

	// -- Resolve, settle and mint --

	t.Log("Wait for the distribution to complete")

	for {
		state, err := a.GetDistributionState(context.Background(), distribution.ID)
		if err != nil {
			if strings.Contains(err.Error(), "database is locked") {
				continue
			}
			t.Fatal(err)
		}
		if state == common.DistributionStateComplete {
			d, err := a.GetDistribution(context.Background(), distribution.ID)
			if err != nil {
				if strings.Contains(err.Error(), "database is locked") {
					continue
				}
				t.Fatal(err)
			}
			distribution = *d
			break
		}
		time.Sleep(time.Second)
	}

	t.Logf("Distribution settled and minted, took %s\n", time.Since(start))

	balanceExampleNFT := "./cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
	balanceExampleNFTCode := util.ParseCadenceTemplate(balanceExampleNFT)

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

	if len(randomPack.Collectibles) != no_collectibles_per_pack {
		t.Fatalf("expected pack to contain %d collectibles", no_collectibles_per_pack)
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

	// -- Reveal & open --

	t.Log("Owner requests to reveal and open the pack")

	revealRequest := "./cadence-transactions/packNFT/reveal_request.cdc"
	revealRequestCode := util.ParseCadenceTemplate(revealRequest)
	e, err = g.
		TransactionFromFile(revealRequest, revealRequestCode).
		SignProposeAndPayAs("owner").
		Argument(randomPackID).
		BooleanArgument(true).
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events = util.ParseTestEvents(e)

	// Owner withdraw PackNFT from the collection, calls reveal & open on it and deposits back into their collection
	util.NewExpectedPackNFTEvent("RevealRequest").AddField("id", randomPackID.String()).AddField("openRequest", "true").AssertEqual(t, events[0])

	t.Log("PDS backend submits reveal transaction w/ openRequest=true")

	t.Log("Wait for the reveal & open to happen")
	for {
		p, err := a.GetPack(context.Background(), randomPack.ID)
		if err != nil {
			if strings.Contains(err.Error(), "database is locked") {
				continue
			}
			t.Fatal(err)
		}
		if p.State == common.PackStateRevealed || p.State == common.PackStateOpened {
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

	distStateScript := "./cadence-scripts/pds/get_dist_state.cdc"
	distStateCode := util.ParseCadenceTemplate(distStateScript)
	distStateR, err := g.ScriptFromFile(distStateScript, distStateCode).UInt64Argument(uint64(distribution.FlowID.Int64)).RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint8(2), distStateR.ToGoValue().(uint8), "Expected distribution to be in state 2 (complete)")
}
