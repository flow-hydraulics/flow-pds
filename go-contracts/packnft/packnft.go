package packnft

import (
	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
)

func GetPackCommitHash(
	g *gwtf.GoWithTheFlow,
    id uint64,
) (commitHash string, err error) {
	txScript:= "../cadence-scripts/packNFT/packNFT_commitHash.cdc"
	code:= util.ParseCadenceTemplate(txScript)
	d, err := g.ScriptFromFile(txScript, code).UInt64Argument(id).RunReturns()
	commitHash = d.ToGoValue().(string)
	return
}

func GetPackStatus(
	g *gwtf.GoWithTheFlow,
    id uint64,
) (status string, err error) {
	txScript:= "../cadence-scripts/packNFT/packNFT_status.cdc"
	code:= util.ParseCadenceTemplate(txScript)
	d, err := g.ScriptFromFile(txScript, code).UInt64Argument(id).RunReturns()
	status = d.ToGoValue().(string)
	return
}

func GetTotalPacks(
	g *gwtf.GoWithTheFlow,
) (total uint64, err error) {
	txScript:= "../cadence-scripts/packNFT/packNFT_total_supply.cdc"
	code:= util.ParseCadenceTemplate(txScript)
	d, err := g.ScriptFromFile(txScript, code).RunReturns()
	total = d.ToGoValue().(uint64)
	return
}

func Verify(
	g *gwtf.GoWithTheFlow,
    id uint64,
    nftString string,
) (verified bool, err error) {
	txScript:= "../cadence-scripts/packNFT/verify.cdc"
	code:= util.ParseCadenceTemplate(txScript)
	d, err := g.ScriptFromFile(txScript, code).UInt64Argument(id).StringArgument(nftString).RunReturns()
    if err != nil {
        return
    } 
	verified = d.ToGoValue().(bool)
	return
}

func OwnerRevealReq(g *gwtf.GoWithTheFlow, id uint64) (events []*gwtf.FormatedEvent, err error) {
	revealRequest := "../cadence-transactions/packNFT/reveal_request.cdc"
	revealRequestCode := util.ParseCadenceTemplate(revealRequest)
    e, err := g.
		TransactionFromFile(revealRequest, revealRequestCode).
		SignProposeAndPayAs("owner").
		UInt64Argument(id).
		RunE()
	events = util.ParseTestEvents(e)
    return
}

func OwnerOpenReq(
    g *gwtf.GoWithTheFlow, 
    id uint64,
) (events []*gwtf.FormatedEvent, err error) {
	openRequest := "../cadence-transactions/packNFT/open_request.cdc"
	openRequestCode := util.ParseCadenceTemplate(openRequest)
    e, err := g.TransactionFromFile(openRequest, openRequestCode).
		SignProposeAndPayAs("owner").
		UInt64Argument(id).
		RunE()
	events = util.ParseTestEvents(e)
    return
}

