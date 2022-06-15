package examplenft

import (
	"github.com/bjartek/overflow/overflow"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func GetBalance(g *overflow.Overflow, account string) (balance cadence.Value, err error) {
	balanceExampleNFT := "../cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
	balanceExampleNFTCode := util.ParseCadenceTemplate(balanceExampleNFT)
	balance, err = g.Script(string(balanceExampleNFTCode)).Args(g.Arguments().Account(account)).RunReturns()
	return
}
