package main

import (
	"fmt"
	"strconv"
	"testing"
    "os"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/examplenft"
	"github.com/flow-hydraulics/flow-pds/go-contracts/packnft"
	"github.com/flow-hydraulics/flow-pds/go-contracts/pds"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"

	// 	"github.com/flow-hydraulics/flow-pds/go-contracts/packnft"
	//	"github.com/onflow/cadence"
	"github.com/stretchr/testify/assert"
)

// Create all required resources for different accounts
func TestMintExampleNFTs(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
	mintExampleNFT := "../cadence-transactions/exampleNFT/mint_exampleNFT.cdc"
	mintExampleNFTCode := util.ParseCadenceTemplate(mintExampleNFT)
	for i := 0; i < 3; i++ {
		_, err := g.
			TransactionFromFile(mintExampleNFT, mintExampleNFTCode).
			SignProposeAndPayAs("issuer").
			AccountArgument("issuer").
			RunE()
        assert.NoError(t, err)
	}
}

func TestCanCreateExampleCollection(t *testing.T) {
// for both pds and owner
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
	setupExampleNFT := "../cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	setupExampleNFTCode := util.ParseCadenceTemplate(setupExampleNFT)
	_, err := g.TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("owner").
		RunE()
    assert.NoError(t, err)

	_, err = g.TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("pds").
		RunE()
    assert.NoError(t, err)
}

func TestCanCreatePackNFTCollection(t *testing.T) {
// for both issuer and owner
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
	createPackNFTCollection := "../cadence-transactions/packNFT/create_new_packNFT_collection.cdc"
	createPackNFTCollectionCode := util.ParseCadenceTemplate(createPackNFTCollection)
    _, err := g.
		TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("issuer").
		RunE()
    assert.NoError(t, err)

    _, err = g.
		TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("owner").
		RunE()
    assert.NoError(t, err)
}



func TestCanCreatePackIssuer(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
	_, err := pds.CreatePackIssuer(g, "issuer")
	assert.NoError(t, err)
}

// Setup - sharing capabilities

func TestCannotCreateDistWithoutCap(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
	_, err := pds.CreateDistribution(g, "issuer")
	assert.Error(t, err)
}

func TestSetDistCap(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
	_, err := pds.SetPackIssuerCap(g, "issuer", "pds")
	assert.NoError(t, err)
}

// Create Distribution and Minting

func TestCreateDistWithCap(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    currentDistId, err := pds.GetDistID(g) 
	assert.NoError(t, err)
	events, err := pds.CreateDistribution(g, "issuer")
	assert.NoError(t, err)

	util.NewExpectedPDSEvent("DistributionCreated").AddField("DistId", strconv.Itoa(int(currentDistId)) ).AssertEqual(t, events[0])

    nextDistId, err := pds.GetDistID(g) 
	assert.NoError(t, err)
    assert.Equal(t, currentDistId + 1, nextDistId)

}

func TestPDSEscrowNFTs(t *testing.T){
    // This just tests to transfer all issuer example NFTs into escrow
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nfts, err := examplenft.GetBalance(g, "issuer") 
    nextDistId, err := pds.GetDistID(g)
    gonfts := nfts.ToGoValue().([]interface{})

    fmt.Printf("nfts to settle %s", nfts.String())
	assert.NoError(t, err)
    events, err := pds.PDSWithdrawNFT(g, nextDistId - 1, nfts, "pds")
	assert.NoError(t, err)
    if os.Getenv("NETWORK") == "emulator" {
        fmt.Print("emulator")
        // For emulator there are deposit and withdraw events
        assert.Len(t, events, 2*len(gonfts))
    } else {
        fmt.Print("testnet")
        // For testnet there are deposit and withdraw and fees (withdraw, deposit, fee)
        assert.Len(t, events, 2*len(gonfts) + 3)
    }
}

func TestPDSMintPackNFTs(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    addr := g.Account("issuer").Address().String()
    toHash := "f24dfdf9911df152,A." + addr + ".ExampleNFT.0,A."+ addr +".ExampleNFT.3"

    fmt.Printf("toHash: %s", toHash)

    hash, err := util.GetHash(g, toHash) 
    assert.NoError(t, err)

    nextDistId, err := pds.GetDistID(g)
    assert.NoError(t, err)

    expectedId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)

    events, err := pds.PDSMintPackNFT(g, nextDistId - 1, hash, "issuer", "pds")
    assert.NoError(t, err)

    fmt.Print(events)

	util.NewExpectedPackNFTEvent("Mint").
        AddField("id", strconv.Itoa(int(expectedId))).
        AddField("commitHash", hash).
        AddField("distId", strconv.Itoa(int(nextDistId - 1))).
        AssertEqual(t, events[1])

    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)
    assert.Equal(t, expectedId + 1, nextPackNFTId)

    actualHash, err := packnft.GetPackCommitHash(g, expectedId)
    assert.NoError(t, err)
    assert.Equal(t, hash, actualHash)

    status, err := packnft.GetPackStatus(g, expectedId)
    assert.NoError(t, err)
    assert.Equal(t, "Sealed", status)
}

// Sold Pack Transfer to Owner

func TestTransfeToOwner(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)

	transferPackNFT := "../cadence-transactions/packNFT/transfer_packNFT.cdc"
	transferPackNFTCode := util.ParseCadenceTemplate(transferPackNFT)
	_, err = g.TransactionFromFile(transferPackNFT, transferPackNFTCode).
		SignProposeAndPayAs("issuer").
		AccountArgument("owner").
		UInt64Argument(nextPackNFTId - 1).
		RunE()
    assert.NoError(t, err)
}

// Reveal

func TestOwnerRevealReq(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    currentPack := nextPackNFTId -1
    assert.NoError(t, err)

    events, err := packnft.OwnerRevealReq(g, currentPack)
    assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("RevealRequest").
        AddField("id", strconv.Itoa(int(currentPack))).
        AssertEqual(t, events[0])

    // Request should not change the state
    status, err := packnft.GetPackStatus(g, currentPack)
    assert.NoError(t, err)
    assert.Equal(t, "Sealed", status)
}

func TestOwnerCannotOpenWithoutRevealed(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    currentPack := nextPackNFTId -1
    assert.NoError(t, err)

    events, err := packnft.OwnerOpenReq(g, currentPack)
    assert.Error(t, err)
    assert.Len(t, events, 0)
}

func TestPDSCannotRevealwithWrongSalt(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)
    currentPack := nextPackNFTId -1

    nextDistId, err := pds.GetDistID(g)
    currentDistId := nextDistId - 1
    assert.NoError(t, err)

    // toHash := "f24dfdf9911df152,A.01cf0e2f2f715450.ExampleNFT.0,A.01cf0e2f2f715450.ExampleNFT.3"
    incorrectSalt := "123"
    var addrs []cadence.Value
    var name []cadence.Value
    var ids []cadence.Value
    addrBytes  := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
    nameString := cadence.NewString("ExampleNFT")
	for i := 0; i < 2; i++ {
        addrs = append(addrs, addrBytes)
        name = append(name, nameString)
	}
    ids = append(ids, cadence.UInt64(0))
    ids = append(ids, cadence.UInt64(3))

    _, err = pds.PDSRevealPackNFT(
        g,
        currentDistId,
        currentPack,
        cadence.NewArray(addrs),
        cadence.NewArray(name),
        cadence.NewArray(ids),
        incorrectSalt,
        "pds",
    )

    assert.Error(t, err)
    status, err := packnft.GetPackStatus(g, currentPack)
    assert.NoError(t, err)
    assert.Equal(t, "Sealed", status)
}

func TestPDSCannotRevealwithWrongNFTs(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)
    currentPack := nextPackNFTId -1

    nextDistId, err := pds.GetDistID(g)
    currentDistId := nextDistId - 1
    assert.NoError(t, err)

    // toHash := "f24dfdf9911df152,A.01cf0e2f2f715450.ExampleNFT.0,A.01cf0e2f2f715450.ExampleNFT.3"
    salt := "f24dfdf9911df152"
    nameString := cadence.NewString("ExampleNFT")
    var addrs []cadence.Value
    var name []cadence.Value
    var ids []cadence.Value
    addrBytes  := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
        addrs = append(addrs, addrBytes)
        name = append(name, nameString)
	}
    // not the correct ids
    ids = append(ids, cadence.UInt64(1))
    ids = append(ids, cadence.UInt64(5))

    _, err = pds.PDSRevealPackNFT(
        g,
        currentDistId,
        currentPack,
        cadence.NewArray(addrs),
        cadence.NewArray(name),
        cadence.NewArray(ids),
        salt,
        "pds",
    )
    assert.Error(t, err)

    status, err := packnft.GetPackStatus(g, currentPack)
    assert.NoError(t, err)
    assert.Equal(t, "Sealed", status)
}

func TestPDSRevealPackNFTs(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)
    currentPack := nextPackNFTId - 1

    nextDistId, err := pds.GetDistID(g)
    currentDistId := nextDistId - 1
    assert.NoError(t, err)

    salt := "f24dfdf9911df152"
    addr := g.Account("issuer").Address().String()
    nftString := "A." + addr + ".ExampleNFT.0,A."+ addr +".ExampleNFT.3"
    var addrs []cadence.Value
    var name []cadence.Value
    var ids []cadence.Value
    addrBytes  := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
        addrs = append(addrs, addrBytes)
        name = append(name, cadence.NewString("ExampleNFT"))
	}
    ids = append(ids, cadence.UInt64(0))
    ids = append(ids, cadence.UInt64(3))

    events, err := pds.PDSRevealPackNFT(
        g,
        currentDistId,
        currentPack,
        cadence.NewArray(addrs),
        cadence.NewArray(name),
        cadence.NewArray(ids),
        salt,
        "pds",
    )
    assert.NoError(t, err)

   util.NewExpectedPackNFTEvent("Revealed").
       AddField("id", strconv.Itoa(int(currentPack))).
       AddField("salt", salt).
       AddField("nfts", nftString).
       AssertEqual(t, events[0])

    status, err := packnft.GetPackStatus(g, currentPack)
    assert.NoError(t, err)
    assert.Equal(t, "Revealed", status)

}

// Open 

func TestOwnerOpenReq(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    currentPack := nextPackNFTId -1
    assert.NoError(t, err)

    events, err := packnft.OwnerOpenReq(g, currentPack)
    assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("OpenRequest").
        AddField("id", strconv.Itoa(int(currentPack))).
        AssertEqual(t, events[0])
}

func TestPDSOpenPackNFTs(t *testing.T){
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)
    currentPack := nextPackNFTId - 1

    nextDistId, err := pds.GetDistID(g)
    currentDistId := nextDistId - 1
    assert.NoError(t, err)

    // we are just sending all nft that the pds owns here
    // it is down to the pds to send the correct ones
    nfts, err := examplenft.GetBalance(g, "pds") 
    events, err := pds.PDSOpenPackNFT(
        g, currentDistId, currentPack, nfts, "owner", "pds",
    )
    assert.NoError(t, err)

   gonfts := nfts.ToGoValue().([]interface{})

   fmt.Print(events)

   util.NewExpectedPackNFTEvent("Opened").
       AddField("id", strconv.Itoa(int(currentPack))).
       AssertEqual(t, events[0])

    if os.Getenv("NETWORK") == "emulator" {
        fmt.Print("emulator")
        // each NFT goes through withdraw and deposit events
        assert.Len(t, events, (2*len(gonfts) + 1))
    } else {
        fmt.Print("testnet")
        // each NFT goes through withdraw and deposit events and 3 events for fees
        assert.Len(t, events, (2*len(gonfts) + 1 + 3))
    }

    status, err := packnft.GetPackStatus(g, currentPack)
    assert.NoError(t, err)
    assert.Equal(t, "Opened", status)
}

func TestPublicVerify(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, os.Getenv("NETWORK"), false, 3)
    nextPackNFTId, err := packnft.GetTotalPacks(g)
    assert.NoError(t, err)
    currentPack := nextPackNFTId - 1

    addr := g.Account("issuer").Address().String()
    nfts:= "A." + addr + ".ExampleNFT.0,A."+ addr +".ExampleNFT.3"
    v, err := packnft.Verify(g, currentPack, nfts)
    assert.NoError(t, err)
    assert.Equal(t, true, v)

    notNfts:= "A." + addr + ".ExampleNFT.2,A."+ addr +".ExampleNFT.4"
    v, err = packnft.Verify(g, currentPack, notNfts)
    assert.NoError(t, err)
    assert.Equal(t, false, v)
}
