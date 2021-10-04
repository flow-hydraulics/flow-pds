package main

import (
	"encoding/hex"
	"fmt"
    "os"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func main() {
	// This relative path to flow.json is  different in tests as it is the main package

	jsonPath := "../flow.json"
	var flowJSON []string = []string{jsonPath}

	g := gwtf.NewGoWithTheFlow(flowJSON, os.Getenv("NETWORK"), false, 3)

	packNFT := util.ParseCadenceTemplate("../cadence-contracts/PackNFT.cdc")
	txFilename := "../cadence-transactions/deploy/deploy-packNFT-with-auth.cdc"
	code := util.ParseCadenceTemplate(txFilename)
	packNFTencodedStr := hex.EncodeToString(packNFT)

	if g.Network == "emulator" {
		g.CreateAccounts("emulator-account")
	}

	e, err := g.TransactionFromFile(txFilename, code).
		SignProposeAndPayAs("issuer").
		StringArgument("PackNFT").
		StringArgument(packNFTencodedStr).
		Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTCollection"}).
		Argument(cadence.Path{Domain: "public", Identifier: "ExamplePackNFTCollectionPub"}).
		Argument(cadence.Path{Domain: "public", Identifier: "ExamplePackNFTIPackNFTCollectionPub"}).
		Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTOperator"}).
		Argument(cadence.Path{Domain: "private", Identifier: "ExamplePackNFTOperatorPriv"}).
		StringArgument("0.1.0").
		RunE()

	if err != nil {
		ferr := fmt.Errorf("deploy Pack: %s", err)
		fmt.Println(ferr)
		return
	} else {
		fmt.Print("deployed packNFT ")
		fmt.Println(e)
	}

	pds := util.ParseCadenceTemplate("../cadence-contracts/PDS.cdc")
	pdsEncodedStr := hex.EncodeToString(pds)
	txFilename = "../cadence-transactions/deploy/deploy-pds-with-auth.cdc"
	code = util.ParseCadenceTemplate(txFilename)
	e, err = g.TransactionFromFile(txFilename, code).
		SignProposeAndPayAs("pds").
		StringArgument("PDS").
		StringArgument(pdsEncodedStr).
		Argument(cadence.Path{Domain: "storage", Identifier: "PDSPackIssuer"}).
		Argument(cadence.Path{Domain: "public", Identifier: "PDSPackIssuerCapRecv"}).
		Argument(cadence.Path{Domain: "storage", Identifier: "PDSDistCreator"}).
		Argument(cadence.Path{Domain: "private", Identifier: "PDSDistCap"}).
		Argument(cadence.Path{Domain: "storage", Identifier: "PDSDistManager"}).
		StringArgument("0.1.0").
		RunE()

	if err != nil {
		ferr := fmt.Errorf("deploy PDS: %s", err)
		fmt.Println(ferr)
		return
	} else {
		fmt.Print("deployed PDS")
		fmt.Println(e)
	}
}
