package main

import (
	"context"
	"fmt"
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

	g := gwtf.NewGoWithTheFlow([]string{"./flow.json"}, "emulator", false, 3)

	// s := "./cadence-scripts/packNFT/checksum.cdc"
	// sd := util.ParseCadenceTemplate(s)
	// h, err := g.ScriptFromFile(s, sd).RunReturns()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println(h)

	// Setup exampleNFT collection for owner (for opening PackNFT)
	setupExampleNFT := "./cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	setupExampleNFTCode := util.ParseCadenceTemplate(setupExampleNFT)
	_, err := g.TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Setup exampleNFT collection for PDS (for escrow)
	_, err = g.TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("pds").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Issuer create PackIssuer resource to store DistCap

	createPackIssuer := "./cadence-transactions/pds/create_new_pack_issuer.cdc"
	createPackIssuerCode := util.ParseCadenceTemplate(createPackIssuer)
	_, err = g.TransactionFromFile(createPackIssuer, createPackIssuerCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Issuer create PackNFT collection resource to store minted PackNFT

	createPackNFTCollection := "./cadence-transactions/packNFT/create_new_packNFT_collection.cdc"
	createPackNFTCollectionCode := util.ParseCadenceTemplate(createPackNFTCollection)
	_, err = g.TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Owner create PackNFT collection resource to store PackNFT after purchase

	_, err = g.TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// -- Mint example NFTs as issuer --

	mintExampleNFT := "./cadence-transactions/exampleNFT/mint_exampleNFT.cdc"
	mintExampleNFTCode := util.ParseCadenceTemplate(mintExampleNFT)
	for i := 0; i < 5; i++ {
		_, err := g.TransactionFromFile(mintExampleNFT, mintExampleNFTCode).
			SignProposeAndPayAs("issuer").
			AccountArgument("issuer").
			RunE()
		if err != nil {
			t.Fatal(err)
		}
	}

	balanceExampleNFT := "./cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
	balanceExampleNFTCode := util.ParseCadenceTemplate(balanceExampleNFT)
	nftIDs, err := g.ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).
		AccountArgument("issuer").RunReturns()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Issuer Minted NFTS: %s", nftIDs.String())

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
		collection[i] = common.FlowID{Int64: int64(v), Valid: true}
	}

	// PDS share DistCap to PackIssuer (owned by Issuer)

	setDistCap := "./cadence-transactions/pds/set_pack_issuer_cap.cdc"
	setDistCapCode := util.ParseCadenceTemplate(setDistCap)
	_, err = g.TransactionFromFile(setDistCap, setDistCapCode).
		SignProposeAndPayAs("pds").
		AccountArgument("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Issuer now creates distribution

	pdsDistId := "./cadence-scripts/pds/get_current_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
	currentDistId, err := g.ScriptFromFile(pdsDistId, pdsDistIdCode).RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	createDist := "./cadence-transactions/pds/create_distribution.cdc"
	createDistCode := util.ParseCadenceTemplate(createDist)
	// Private path must match the PackNFT contract
	e, err := g.TransactionFromFile(createDist, createDistCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "exampleNFTCollectionProvider"}).
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events := util.ParseTestEvents(e)

	util.NewExpectedPDSEvent("DistributionCreated").AddField("DistId", currentDistId.String()).AssertEqual(t, events[0])

	// -- Use newly minted NFTs to create a distribution as issuer --
	issuer := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "issuer")))
	distId, err := common.FlowIDFromCadence(e[0].Value.Fields[0])
	if err != nil {
		t.Fatal(err)
	}
	d := app.Distribution{
		DistID: distId,
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

	// Wait for distribution to complete
	for {
		dist, _, err := a.GetDistribution(context.Background(), d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if dist.State == common.DistributionStateComplete {
			break
		}
		time.Sleep(time.Second)
	}

	dist, _, err := a.GetDistribution(context.Background(), d.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Pick one pack
	randomPack := dist.Packs[0]
	randomPackID := cadence.UInt64(randomPack.FlowID.Int64)

	// -- Transfer --
	// Issuer transfer PackNFT to owner
	transferPackNFT := "./cadence-transactions/packNFT/transfer_packNFT.cdc"
	transferPackNFTCode := util.ParseCadenceTemplate(transferPackNFT)
	_, err = g.TransactionFromFile(transferPackNFT, transferPackNFTCode).
		SignProposeAndPayAs("issuer").
		AccountArgument("owner").
		Argument(randomPackID).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// -- Reveal --
	// Owner requests to reveal PackNFT

	revealRequest := "./cadence-transactions/packNFT/reveal_request.cdc"
	revealRequestCode := util.ParseCadenceTemplate(revealRequest)
	e, err = g.TransactionFromFile(revealRequest, revealRequestCode).
		SignProposeAndPayAs("owner").
		Argument(randomPackID).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	events = util.ParseTestEvents(e)
	ownerAddr := util.GetAccountAddr(g, "owner")
	// Onwer withdraw PackNFT from the collection, calls reveal on it and deposits back into their collection
	util.NewExpectedPackNFTEvent("Withdraw").AddField("id", randomPackID.String()).AddField("from", ownerAddr).AssertEqual(t, events[0])
	util.NewExpectedPackNFTEvent("RevealRequest").AddField("id", randomPackID.String()).AssertEqual(t, events[1])
	util.NewExpectedPackNFTEvent("Deposit").AddField("id", randomPackID.String()).AddField("to", ownerAddr).AssertEqual(t, events[2])

	// Wait a moment to let reveal to be noticed
	time.Sleep(time.Second * 1)

	// Parse the collectible IDs in the pack
	randomPackNftIDs := make([]cadence.Value, len(randomPack.Collectibles))
	for i, c := range randomPack.Collectibles {
		randomPackNftIDs[i] = cadence.UInt64(c.FlowID.Int64)
	}
	randomPackNftIDsArr := cadence.NewArray(randomPackNftIDs)

	// Parse the collectible "names" in the pack
	randomPackNftNames := make([]cadence.Value, len(randomPack.Collectibles))
	for i, c := range randomPack.Collectibles {
		randomPackNftNames[i] = cadence.String(c.String())
	}
	randomPackNftNamesArr := cadence.NewArray(randomPackNftNames)

	// Parse the collectible Address in the pack
	randomPackNftAddrs := make([]cadence.Value, len(randomPack.Collectibles))
	for i, c := range randomPack.Collectibles {
		randomPackNftAddrs[i] = cadence.Address(c.ContractReference.Address)
	}
	randomPackNftAddrsArr := cadence.NewArray(randomPackNftAddrs)

	// Parse the collectible ContractName in the pack
	randomPackNftCNames := make([]cadence.Value, len(randomPack.Collectibles))
	for i, c := range randomPack.Collectibles {
		randomPackNftCNames[i] = cadence.String(c.ContractReference.Name)
	}
	randomPackNftCNamesArr := cadence.NewArray(randomPackNftCNames)

	fmt.Println("Pack contents:")
	fmt.Println("IDs", randomPackNftIDsArr)
	fmt.Println("Addrs", randomPackNftAddrsArr)
	fmt.Println("ContractNames", randomPackNftCNamesArr)
	fmt.Println("Names", randomPackNftNamesArr)

	// PDS backend submits revealed information
	reveal := "./cadence-transactions/pds/reveal_packNFT.cdc"
	revealCode := util.ParseCadenceTemplate(reveal)
	e, err = g.TransactionFromFile(reveal, revealCode).
		SignProposeAndPayAs("pds").
		Argument(currentDistId).
		Argument(randomPackID).
		Argument(randomPackNftAddrsArr).
		Argument(randomPackNftCNamesArr).
		Argument(randomPackNftIDsArr).
		StringArgument(randomPack.Salt.String()).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	events = util.ParseTestEvents(e)
	util.NewExpectedPackNFTEvent("Revealed").
		AddField("id", randomPackID.String()).
		AddField("salt", randomPack.Salt.String()).
		AssertEqual(t, events[0])

	// -- Retrieve --
	// Owner requests to open PackNFT

	openRequest := "./cadence-transactions/packNFT/open_request.cdc"
	openRequestCode := util.ParseCadenceTemplate(openRequest)
	e, err = g.TransactionFromFile(openRequest, openRequestCode).
		SignProposeAndPayAs("owner").
		Argument(randomPackID).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	events = util.ParseTestEvents(e)
	// Onwer withdraw PackNFT from the collection, calls open on it and deposits back into their collection
	util.NewExpectedPackNFTEvent("Withdraw").AddField("id", randomPackID.String()).AddField("from", ownerAddr).AssertEqual(t, events[0])
	util.NewExpectedPackNFTEvent("OpenRequest").AddField("id", randomPackID.String()).AssertEqual(t, events[1])
	util.NewExpectedPackNFTEvent("Deposit").AddField("id", randomPackID.String()).AddField("to", ownerAddr).AssertEqual(t, events[2])

	// PDS backend submits open tx and  transfer escrow

	open := "./cadence-transactions/pds/open_packNFT.cdc"
	openCode := util.ParseCadenceTemplate(open)
	e, err = g.TransactionFromFile(open, openCode).
		SignProposeAndPayAs("pds").
		Argument(currentDistId).
		Argument(randomPackID).
		Argument(randomPackNftIDsArr).
		AccountArgument("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// There are withdraw and deposit event for each nft being released from escrow
	// So we only check the last event as "Opened"
	l := len(randomPackNftIDsArr.ToGoValue().([]interface{})) * 2
	events = util.ParseTestEvents(e)
	util.NewExpectedPackNFTEvent("Opened").
		AddField("id", randomPackID.String()).
		AssertEqual(t, events[l])
}
