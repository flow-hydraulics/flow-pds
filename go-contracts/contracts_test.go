package main

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/bjartek/overflow/overflow"
	"github.com/flow-hydraulics/flow-pds/go-contracts/examplenft"
	"github.com/flow-hydraulics/flow-pds/go-contracts/packnft"
	"github.com/flow-hydraulics/flow-pds/go-contracts/pds"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/utils"
	"github.com/onflow/cadence"

	"github.com/stretchr/testify/assert"
)

// Create all required resources for different accounts
func TestMintExampleNFTs(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)

	mintExampleNFT := "../cadence-transactions/exampleNFT/mint_exampleNFT.cdc"
	mintExampleNFTCode := util.ParseCadenceTemplate(mintExampleNFT)
	for i := 0; i < 4; i++ {
		_, err := g.
			Transaction(string(mintExampleNFTCode)).
			SignProposeAndPayAs("issuer").
			Args(g.Arguments().Account("issuer")).
			RunE()
		assert.NoError(t, err)
	}
}

func TestCanCreateExampleCollection(t *testing.T) {
	// for both pds and owner
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	setupExampleNFT := "../cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	setupExampleNFTCode := util.ParseCadenceTemplate(setupExampleNFT)
	_, err = g.Transaction(string(setupExampleNFTCode)).
		SignProposeAndPayAs("owner").
		RunE()
	assert.NoError(t, err)

	_, err = g.Transaction(string(setupExampleNFTCode)).
		SignProposeAndPayAs("pds").
		RunE()
	assert.NoError(t, err)
}

func TestCanCreatePackNFTCollection(t *testing.T) {
	// for both issuer and owner
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	createPackNFTCollection := "../cadence-transactions/packNFT/create_new_packNFT_collection.cdc"
	createPackNFTCollectionCode := util.ParseCadenceTemplate(createPackNFTCollection)
	_, err = g.
		Transaction(string(createPackNFTCollectionCode)).
		SignProposeAndPayAs("issuer").
		RunE()
	assert.NoError(t, err)

	_, err = g.
		Transaction(string(createPackNFTCollectionCode)).
		SignProposeAndPayAs("owner").
		RunE()
	assert.NoError(t, err)
}

func TestCanCreatePackIssuer(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	_, err = pds.CreatePackIssuer(g, "issuer")
	assert.NoError(t, err)
}

// Issuer and PDS  must first link their NFT Provider Cap
// Issuer: for PDS to withdraw to escrow
// PDS: for the PDS contract to release escrow
func TestIssuerCanLinkProviderCap(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	script := "../cadence-transactions/exampleNFT/link_providerCap_exampleNFT.cdc"
	code := util.ParseCadenceTemplate(script)
	_, err = g.Transaction(string(code)).
		SignProposeAndPayAs("issuer").
		Args(g.Arguments().Argument(cadence.Path{Domain: "private", Identifier: "NFTCollectionProvider"})).
		RunE()
	assert.NoError(t, err)

	_, err = g.Transaction(string(code)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().Argument(cadence.Path{Domain: "private", Identifier: "NFTCollectionProvider"})).
		RunE()
	assert.NoError(t, err)
}

// Setup - sharing capabilities

func TestCannotCreateDistWithoutCap(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	keyPair := cadence.KeyValuePair{Key: utils.NewCadenceString("metadataKey"), Value: utils.NewCadenceString("metadataValue")}
	var keypairArr []cadence.KeyValuePair
	keypairArr = append(keypairArr, keyPair)
	metadata := cadence.NewDictionary(keypairArr)
	_, err = pds.CreateDistribution(g, "NFTCollectionProvider", "issuer", "title", metadata)
	assert.Error(t, err)
}

func TestSetDistCap(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	_, err = pds.SetPackIssuerCap(g, "issuer", "pds")
	assert.NoError(t, err)
}

// Create Distribution and Minting

func TestCreateDistWithCap(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	nextDistId, err := pds.GetNextDistID(g)
	assert.NoError(t, err)

	keyPair := cadence.KeyValuePair{Key: utils.NewCadenceString("metadataKey"), Value: utils.NewCadenceString("metadataValue")}
	stringifiedKeyPair := "{\"metadataKey\": \"metadataValue\"}"
	var keypairArr []cadence.KeyValuePair
	keypairArr = append(keypairArr, keyPair)
	expMetadata := cadence.NewDictionary(keypairArr)
	expTitle := "ExampleDistTitle"

	events, err := pds.CreateDistribution(g, "NFTCollectionProvider", "issuer", expTitle, expMetadata)
	assert.NoError(t, err)

	util.NewExpectedPDSEvent("DistributionCreated").
		AddField("DistId", strconv.Itoa(int(nextDistId))).
		AddField("state", "0").
		AddField("title", expTitle).
		AddField("metadata", stringifiedKeyPair).
		AssertEqual(t, events[0])

	newDistId, err := pds.GetNextDistID(g)
	assert.NoError(t, err)
	assert.Equal(t, nextDistId+1, newDistId)

	title, err := pds.GetDistTitle(g, nextDistId)
	assert.NoError(t, err)
	assert.Equal(t, title, title)

	state, err := pds.GetDistState(g, nextDistId)
	assert.NoError(t, err)
	assert.Equal(t, "initialized", state)

	metadata, err := pds.GetDistMetadata(g, nextDistId)
	assert.NoError(t, err)
	assert.Equal(t, stringifiedKeyPair, metadata)
}

func TestPDSEscrowNFTs(t *testing.T) {
	// This just tests to transfer all issuer example NFTs into escrow
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	nfts, err := examplenft.GetBalance(g, "issuer")
	assert.NoError(t, err)
	nextDistId, err := pds.GetNextDistID(g)
	gonfts := nfts.ToGoValue().([]interface{})

	assert.NoError(t, err)
	events, err := pds.PDSWithdrawNFT(g, nextDistId-1, nfts, "pds")
	assert.NoError(t, err)
	if os.Getenv("NETWORK") == "emulator" {
		fmt.Print("emulator")
		// For emulator there are deposit and withdraw events
		assert.Len(t, events, 2*len(gonfts))
	} else {
		fmt.Print("testnet")
		// For testnet there are deposit and withdraw and fees (withdraw, deposit, fee)
		assert.Len(t, events, 2*len(gonfts)+3)
	}
}

func TestPDSMintPackNFTs(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	addr := g.Account("issuer").Address().String()

	// pds account should have all the escrowed nfts
	nfts, err := examplenft.GetBalance(g, "pds")
	assert.NoError(t, err)
	gonfts := nfts.ToGoValue().([]interface{})

	// Mint a pack nft to test reveal ONLY
	toHash := "f24dfdf9911df152,A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[0].(uint64))) + ",A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[1].(uint64)))

	hash, err := util.GetHash(g, toHash)
	assert.NoError(t, err)

	nextDistId, err := pds.GetNextDistID(g)
	assert.NoError(t, err)

	numOfPacks, err := packnft.GetTotalPacks(g)
	expectedId := numOfPacks + 1
	assert.NoError(t, err)

	events, err := pds.PDSMintPackNFT(g, nextDistId-1, hash, "issuer", "pds")
	assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("Mint").
		AddField("id", strconv.Itoa(int(expectedId))).
		AddField("commitHash", hash).
		AddField("distId", strconv.Itoa(int(nextDistId-1))).
		AssertEqual(t, events[0])

	nextPackNFTId, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	assert.Equal(t, expectedId, nextPackNFTId)

	actualHash, err := packnft.GetPackCommitHash(g, expectedId)
	assert.NoError(t, err)
	assert.Equal(t, hash, actualHash)

	status, err := packnft.GetPackStatus(g, expectedId)
	assert.NoError(t, err)
	assert.Equal(t, "Sealed", status)

	// Mint second pack that will be revealed and opened at the same block
	toHash1 := "f24dfdf9911df152,A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[2].(uint64))) + ",A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[3].(uint64)))
	hash1, err := util.GetHash(g, toHash1)
	assert.NoError(t, err)
	_, err = pds.PDSMintPackNFT(g, nextDistId-1, hash1, "issuer", "pds")
	assert.NoError(t, err)

	// Mint a third pack to be revealed via the public function
	toHash2 := "g24dfdf9911df152,A." + addr + ".ExampleNFT.2,A." + addr + ".ExampleNFT.4"
	hash2, err := util.GetHash(g, toHash2)
	assert.NoError(t, err)
	_, err = pds.PDSMintPackNFT(g, nextDistId-1, hash2, "issuer", "pds")
	assert.NoError(t, err)
}

func TestUpdateDistState(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	stateToUpdate := "complete"
	events, err := pds.PDSUpdateDistState(g, currentDistId, stateToUpdate)
	assert.NoError(t, err)

	util.NewExpectedPDSEvent("DistributionStateUpdated").
		AddField("DistId", strconv.Itoa(int(currentDistId))).
		AddField("state", "2").
		AssertEqual(t, events[0])
}

// Sold Pack Transfer to Owner
func TestTransfeToOwner(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)

	transferPackNFT := "../cadence-transactions/packNFT/transfer_packNFT.cdc"
	transferPackNFTCode := util.ParseCadenceTemplate(transferPackNFT)
	_, err = g.Transaction(string(transferPackNFTCode)).
		SignProposeAndPayAs("issuer").
		Args(g.Arguments().
			Account("owner").
			UInt64(numOfPacks)).
		RunE()
	assert.NoError(t, err)
}

// Reveal

func TestOwnerRevealReq(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	currentPack := numOfPacks
	assert.NoError(t, err)

	events, err := packnft.OwnerRevealReq(g, currentPack, false)
	assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("RevealRequest").
		AddField("id", strconv.Itoa(int(currentPack))).
		AddField("openRequest", "false").
		AssertEqual(t, events[0])

	// Request should not change the state
	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Sealed", status)
}

func TestOwnerCannotOpenWithoutRevealed(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	currentPack := numOfPacks
	assert.NoError(t, err)

	events, err := packnft.OwnerOpenReq(g, currentPack)
	assert.Error(t, err)
	assert.Len(t, events, 0)
}

func TestPDSCannotRevealwithWrongSalt(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	// pds account should have all the escrowed nfts
	nfts, err := examplenft.GetBalance(g, "pds")
	assert.NoError(t, err)
	gonfts := nfts.ToGoValue().([]interface{})

	// toHash := "f24dfdf9911df152,A.01cf0e2f2f715450.ExampleNFT.<id>,A.01cf0e2f2f715450.ExampleNFT.<id>"
	incorrectSalt := "123"
	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	nameString := utils.NewCadenceString("ExampleNFT")
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, nameString)
	}
	ids = append(ids, cadence.UInt64(gonfts[0].(uint64)))
	ids = append(ids, cadence.UInt64(gonfts[1].(uint64)))

	_, err = pds.PDSRevealPackNFT(
		g,
		currentDistId,
		currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		incorrectSalt,
		"owner",
		false,
		"NFTCollectionProvider",
		"pds",
	)

	assert.Error(t, err)
	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Sealed", status)
}

func TestPDSCannotRevealwithWrongNFTs(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	// toHash := "f24dfdf9911df152,A.01cf0e2f2f715450.ExampleNFT.0,A.01cf0e2f2f715450.ExampleNFT.3"
	salt := "f24dfdf9911df152"
	nameString := utils.NewCadenceString("ExampleNFT")
	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, nameString)
	}
	// not the correct ids as tests always put consecutive ids in the pack
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
		"owner",
		false,
		"NFTCollectionProvider",
		"pds",
	)
	assert.Error(t, err)

	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Sealed", status)
}

func TestPDSRevealPackNFTs(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	// This is the first minted pack
	currentPack := numOfPacks - 2

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	salt := "f24dfdf9911df152"
	addr := g.Account("issuer").Address().String()

	nfts, err := examplenft.GetBalance(g, "pds")
	assert.NoError(t, err)
	gonfts := nfts.ToGoValue().([]interface{})

	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, utils.NewCadenceString("ExampleNFT"))
	}
	ids = append(ids, cadence.UInt64(gonfts[0].(uint64)))
	ids = append(ids, cadence.UInt64(gonfts[1].(uint64)))

	events, err := pds.PDSRevealPackNFT(
		g,
		currentDistId,
		currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		salt,
		"owner",
		false,
		"NFTCollectionProvider",
		"pds",
	)
	assert.NoError(t, err)

	nftString := "A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[0].(uint64))) + ",A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[1].(uint64)))
	util.NewExpectedPackNFTEvent("Revealed").
		AddField("id", strconv.Itoa(int(currentPack))).
		AddField("salt", salt).
		AddField("nfts", nftString).
		AssertEqual(t, events[0])

	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Revealed", status)

}

func TestPublicFailRevealPackNFTsWithWrongIds(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks

	salt := "g24dfdf9911df152"
	// wrong ids
	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, utils.NewCadenceString("ExampleNFT"))
	}
	ids = append(ids, cadence.UInt64(5))
	ids = append(ids, cadence.UInt64(5))

	_, err = packnft.PublicRevealPackNFT(
		g,
		currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		salt,
		"owner",
	)
	assert.Error(t, err)

	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Sealed", status)

}
func TestPublicRevealPackNFTs(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks

	salt := "g24dfdf9911df152"
	addr := g.Account("issuer").Address().String()
	nftString := "A." + addr + ".ExampleNFT.2,A." + addr + ".ExampleNFT.4"
	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, utils.NewCadenceString("ExampleNFT"))
	}
	ids = append(ids, cadence.UInt64(2))
	ids = append(ids, cadence.UInt64(4))

	events, err := packnft.PublicRevealPackNFT(
		g,
		currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		salt,
		"owner",
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

func TestPDSRevealAndOpenPackNFT(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	// This is the second minted NFT pack
	currentPack := numOfPacks - 1

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	salt := "f24dfdf9911df152"
	addr := g.Account("issuer").Address().String()

	nfts, err := examplenft.GetBalance(g, "pds")
	assert.NoError(t, err)
	gonfts := nfts.ToGoValue().([]interface{})

	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, utils.NewCadenceString("ExampleNFT"))
	}
	ids = append(ids, cadence.UInt64(gonfts[2].(uint64)))
	ids = append(ids, cadence.UInt64(gonfts[3].(uint64)))

	// First time submitting the transaction to reveal
	events, err := pds.PDSRevealPackNFT(
		g,
		currentDistId,
		currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		salt,
		"owner",
		true,
		"NFTCollectionProvider",
		"pds",
	)
	assert.NoError(t, err)

	nftString := "A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[2].(uint64))) + ",A." + addr + ".ExampleNFT." + strconv.Itoa(int(gonfts[3].(uint64)))
	util.NewExpectedPackNFTEvent("Revealed").
		AddField("id", strconv.Itoa(int(currentPack))).
		AddField("salt", salt).
		AddField("nfts", nftString).
		AssertEqual(t, events[0])

		// Second time submitting the transaction to open
	events, err = pds.PDSRevealPackNFT(
		g,
		currentDistId,
		currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		salt,
		"owner",
		true,
		"NFTCollectionProvider",
		"pds",
	)
	assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("Opened").
		AddField("id", strconv.Itoa(int(currentPack))).
		AssertEqual(t, events[0])

	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Opened", status)

	if os.Getenv("NETWORK") == "emulator" {
		fmt.Print("emulator")
		// each NFT goes through withdraw and deposit events
		assert.Len(t, events, (4 + 1))
	} else {
		fmt.Print("testnet")
		// each NFT goes through withdraw and deposit events and 3 events for fees
		assert.Len(t, events, (4 + 1 + 3))
	}
}

// Open

func TestOwnerOpenReq(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	currentPack := numOfPacks
	assert.NoError(t, err)

	events, err := packnft.OwnerOpenReq(g, currentPack)
	assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("OpenRequest").
		AddField("id", strconv.Itoa(int(currentPack))).
		AssertEqual(t, events[0])
}

func TestPDSFailOpenPackNFTsWithWrongIds(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks - 2

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, utils.NewCadenceString("ExampleNFT"))
	}
	ids = append(ids, cadence.UInt64(11))
	ids = append(ids, cadence.UInt64(33))

	_, err = pds.PDSOpenPackNFT(
		g, currentDistId, currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		"owner",
		"NFTCollectionProvider",
		"pds",
	)
	assert.Error(t, err)

	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Revealed", status)
}

func TestPDSOpenPackNFTs(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks - 2

	nextDistId, err := pds.GetNextDistID(g)
	currentDistId := nextDistId - 1
	assert.NoError(t, err)

	escrowedNfts, err := examplenft.GetBalance(g, "pds")
	assert.NoError(t, err)
	gonfts := escrowedNfts.ToGoValue().([]interface{})

	var addrs []cadence.Value
	var name []cadence.Value
	var ids []cadence.Value
	addrBytes := cadence.BytesToAddress(g.Account("issuer").Address().Bytes())
	for i := 0; i < 2; i++ {
		addrs = append(addrs, addrBytes)
		name = append(name, utils.NewCadenceString("ExampleNFT"))
	}
	ids = append(ids, cadence.UInt64(gonfts[0].(uint64)))
	ids = append(ids, cadence.UInt64(gonfts[1].(uint64)))

	events, err := pds.PDSOpenPackNFT(
		g, currentDistId, currentPack,
		cadence.NewArray(addrs),
		cadence.NewArray(name),
		cadence.NewArray(ids),
		"owner", "NFTCollectionProvider", "pds",
	)
	assert.NoError(t, err)

	util.NewExpectedPackNFTEvent("Opened").
		AddField("id", strconv.Itoa(int(currentPack))).
		AssertEqual(t, events[0])

	if os.Getenv("NETWORK") == "emulator" {
		fmt.Print("emulator")
		// each NFT goes through withdraw and deposit events
		assert.Len(t, events, (4 + 1))
	} else {
		fmt.Print("testnet")
		// each NFT goes through withdraw and deposit events and 3 events for fees
		assert.Len(t, events, (4 + 1 + 3))
	}

	status, err := packnft.GetPackStatus(g, currentPack)
	assert.NoError(t, err)
	assert.Equal(t, "Opened", status)
}

func TestPublicVerify(t *testing.T) {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	assert.NoError(t, err)
	numOfPacks, err := packnft.GetTotalPacks(g)
	assert.NoError(t, err)
	currentPack := numOfPacks

	addr := g.Account("issuer").Address().String()
	nfts := "A." + addr + ".ExampleNFT.2,A." + addr + ".ExampleNFT.4"
	v, err := packnft.Verify(g, currentPack, nfts)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	notNfts := "A." + addr + ".ExampleNFT.1,A." + addr + ".ExampleNFT.5"
	_, err = packnft.Verify(g, currentPack, notNfts)
	assert.Error(t, err)
}
