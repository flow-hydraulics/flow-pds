package main

import (
	"encoding/hex"
    "fmt"
	"github.com/bjartek/go-with-the-flow/v2/gwtf"
    util "github.com/flow-hydraulics/flow-pds"
)

func main() {
	// This relative path to flow.json is different in tests as it is the main package
	g := gwtf.NewGoWithTheFlow("../../flow.json")

	contractCode := util.ReadCadenceCode("../../cadence-contracts/PackNFT.cdc")
	txFilename := "../../cadence-transactions/deploy-with-auth.cdc"
	code := util.ReadCadenceCode(txFilename)
	encodedStr := hex.EncodeToString(contractCode)
	g.CreateAccountPrintEvents(
		"vaulted-account",
		"w-1000",
		"w-500-1",
		"w-500-2",
		"w-250-1",
		"w-250-2",
		"non-registered-account",
	)
	// The "owner" defined in flow.json is the owner of the contracts:
	// - `MultSigFlowToken`
	// - `OnChainMultiSig`
	e, err := g.TransactionFromFile(txFilename, code).
		SignProposeAndPayAs("owner").
		StringArgument("MultiSigFlowToken").
		StringArgument(encodedStr).
		Run()

	if err != nil {
        fmt.Errorf("Cannot deploy contract: %s", err)
	}

	gwtf.PrintEvents(e, map[string][]string{})
}

