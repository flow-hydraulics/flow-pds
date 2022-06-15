package packnft

import (
	"github.com/bjartek/overflow/overflow"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func GetPackCommitHash(
	g *overflow.Overflow,
	id uint64,
) (commitHash string, err error) {
	txScript := "../cadence-scripts/packNFT/packNFT_commitHash.cdc"
	code := util.ParseCadenceTemplate(txScript)
	d, err := g.Script(string(code)).
		Args(g.Arguments().
			UInt64(id)).
		RunReturns()
	commitHash = d.ToGoValue().(string)
	return
}

func GetPackStatus(
	g *overflow.Overflow,
	id uint64,
) (status string, err error) {
	txScript := "../cadence-scripts/packNFT/packNFT_status.cdc"
	code := util.ParseCadenceTemplate(txScript)
	d, err := g.Script(string(code)).
		Args(g.Arguments().
			UInt64(id)).
		RunReturns()
	rInt := d.ToGoValue().(uint8)
	switch rInt {
	case 0:
		status = "Sealed"
	case 1:
		status = "Revealed"
	case 2:
		status = "Opened"
	}
	return
}

func GetTotalPacks(
	g *overflow.Overflow,
) (total uint64, err error) {
	txScript := "../cadence-scripts/packNFT/packNFT_total_supply.cdc"
	code := util.ParseCadenceTemplate(txScript)
	d, err := g.Script(string(code)).RunReturns()
	total = d.ToGoValue().(uint64)
	return
}

func Verify(
	g *overflow.Overflow,
	id uint64,
	nftString string,
) (verified bool, err error) {
	txScript := "../cadence-scripts/packNFT/verify.cdc"
	code := util.ParseCadenceTemplate(txScript)
	d, err := g.Script(string(code)).
		Args(g.Arguments().
			UInt64(id).
			String(nftString)).
		RunReturns()
	if err != nil {
		return
	}
	verified = d.ToGoValue().(bool)
	return
}

func OwnerRevealReq(g *overflow.Overflow, id uint64, openRequest bool) (events []*overflow.FormatedEvent, err error) {
	revealRequest := "../cadence-transactions/packNFT/reveal_request.cdc"
	revealRequestCode := util.ParseCadenceTemplate(revealRequest)

	e, err := g.
		Transaction(string(revealRequestCode)).
		SignProposeAndPayAs("owner").
		Args(g.Arguments().
			UInt64(id).
			Boolean(openRequest)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func OwnerOpenReq(
	g *overflow.Overflow,
	id uint64,
) (events []*overflow.FormatedEvent, err error) {
	openRequest := "../cadence-transactions/packNFT/open_request.cdc"
	openRequestCode := util.ParseCadenceTemplate(openRequest)
	e, err := g.Transaction(string(openRequestCode)).
		SignProposeAndPayAs("owner").
		Args(g.Arguments().
			UInt64(id)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PublicRevealPackNFT(
	g *overflow.Overflow,
	packId uint64,
	nftContractAddrs cadence.Value,
	nftContractNames cadence.Value,
	nftIds cadence.Value,
	salt string,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	txScript := "../cadence-transactions/packNFT/public_reveal_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	e, err := g.
		Transaction(string(code)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().
			UInt64(packId).
			Argument(nftContractAddrs).
			Argument(nftContractNames).
			Argument(nftIds).String(salt)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}
