package main

import (
	"fmt"
	"encoding/hex"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/onflow/cadence"
)

func main() {
	// This relative path to flow.json is  different in tests as it is the main package

    jsonPath := "../flow.json"
	var flowJSON []string = []string{jsonPath}

    // g := gwtf.NewGoWithTheFlow(flowJSON, os.Getenv("NETWORK"), false, 3)
	g := gwtf.NewGoWithTheFlow(flowJSON, "emulator", false, 3)

	contractCode := util.ParseCadenceTemplate("../cadence-contracts/PackNFT.cdc")
	txFilename := "../cadence-transactions/deploy/deploy-with-auth.cdc"
	code := util.ParseCadenceTemplate(txFilename)
	encodedStr := hex.EncodeToString(contractCode)

	if g.Network == "emulator" {
		g.CreateAccounts("emulator-account")
	}

	e, err := g.TransactionFromFile(txFilename, code).
		SignProposeAndPayAs("issuer").
		StringArgument("PackNFT").
		StringArgument(encodedStr).
		Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTCollection"}).
		Argument(cadence.Path{Domain: "public", Identifier: "ExamplePackNFTCollectionPub"}).
		Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTMinter"}).
		Argument(cadence.Path{Domain: "private", Identifier: "ExamplePackNFTMinterPriv"}).
		Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTMinterProxy"}).
		Argument(cadence.Path{Domain: "public", Identifier: "ExamplePackNFTMinterProxyPub"}).
		StringArgument("0.1.0").
		RunE()

    if err!=nil {
        fmt.Errorf("deploy error: %s", err)
        return
    } else {
        fmt.Print("deployed")
        fmt.Print(e)
    }

	gwtf.PrintEvents(e, map[string][]string{})
	return
}

