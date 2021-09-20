 package packnft 

import (
	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
)

func CreateMinterProxy(
	g *gwtf.GoWithTheFlow,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txFilename := "../../cadence-transactions/packNFT/create_new_minter_proxy.cdc"
	txScript := util.ParseCadenceTemplate(txFilename)

	e, err := g.TransactionFromFile(txFilename, txScript).
		SignProposeAndPayAs(account).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func MinterProxyMint(g *gwtf.GoWithTheFlow, minterProxyAcct string, packnftAcct string, commitHash string) (events []*gwtf.FormatedEvent, err error) {
	txFilename := "../../cadence-transactions/packNFT/minter_proxy_mint.cdc"
	txScript := util.ParseCadenceTemplate(txFilename)

    e, err := g.TransactionFromFile(txFilename, txScript).
		SignProposeAndPayAs(minterProxyAcct).
        StringArgument(commitHash).
		AccountArgument(packnftAcct).
		RunE()
	events = util.ParseTestEvents(e)
	return

}

func SetMinterProxyCapability(
	g *gwtf.GoWithTheFlow,
	minterProxy string,
	issuer string,
) (err error) {
	txFilename := "../../cadence-transactions/packNFT/set_minter_proxy_cap.cdc"
	txScript := util.ParseCadenceTemplate(txFilename)

	_, err = g.TransactionFromFile(txFilename, txScript).
		SignProposeAndPayAs(issuer).
		AccountArgument(minterProxy).
		RunE()
	return
}

