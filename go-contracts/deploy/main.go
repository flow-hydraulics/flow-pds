package main

import (
	"encoding/hex"
	"fmt"

	"github.com/bjartek/overflow/overflow"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func main() {
	g, err := overflow.NewOverflowEmulator().Config("../flow.json").ExistingEmulator().StartE()
	if err != nil {
		ferr := fmt.Errorf("error creating new overflow builder: %s", err)
		fmt.Println(ferr)
	}

	packNFT := util.ParseCadenceTemplate("../cadence-contracts/PackNFT.cdc")
	txFilename := "../cadence-transactions/deploy/deploy-packNFT-with-auth.cdc"
	code := util.ParseCadenceTemplate(txFilename)
	packNFTencodedStr := hex.EncodeToString(packNFT)

	if g.Network == "emulator" {
		g, err = g.CreateAccountsE()
		if err != nil {
			ferr := fmt.Errorf("error creating accounts: %s", err)
			fmt.Println(ferr)
		}
	}

	e, err := g.Transaction(string(code)).
		SignProposeAndPayAs("issuer").
		Args(g.Arguments().
			String("PackNFT").
			String(packNFTencodedStr).
			Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTCollection"}).
			Argument(cadence.Path{Domain: "public", Identifier: "ExamplePackNFTCollectionPub"}).
			Argument(cadence.Path{Domain: "public", Identifier: "ExamplePackNFTIPackNFTCollectionPub"}).
			Argument(cadence.Path{Domain: "storage", Identifier: "ExamplePackNFTOperator"}).
			Argument(cadence.Path{Domain: "private", Identifier: "ExamplePackNFTOperatorPriv"}).
			String("0.1.0")).
		RunE()

	if err != nil {
		ferr := fmt.Errorf("deploy Pack error: %s", err)
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
	e, err = g.Transaction(string(code)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().
			String("PDS").
			String(pdsEncodedStr).
			Argument(cadence.Path{Domain: "storage", Identifier: "PDSPackIssuer"}).
			Argument(cadence.Path{Domain: "public", Identifier: "PDSPackIssuerCapRecv"}).
			Argument(cadence.Path{Domain: "storage", Identifier: "PDSDistCreator"}).
			Argument(cadence.Path{Domain: "private", Identifier: "PDSDistCap"}).
			Argument(cadence.Path{Domain: "storage", Identifier: "PDSDistManager"}).
			String("0.1.0")).
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
