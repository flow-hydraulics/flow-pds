package examplenft

import (
	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
    "github.com/onflow/cadence"
)

func GetBalance( g *gwtf.GoWithTheFlow, account string) (balance cadence.Value, err error) {
    balanceExampleNFT := "../cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
    balanceExampleNFTCode := util.ParseCadenceTemplate(balanceExampleNFT)
    balance, err = g.ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).AccountArgument(account).RunReturns()
    return
}
