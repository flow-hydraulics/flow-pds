package pds

import (
	"errors"

	"github.com/bjartek/overflow/overflow"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func CreatePackIssuer(
	g *overflow.Overflow,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	txFilename := "../cadence-transactions/pds/create_new_pack_issuer.cdc"
	txScript := util.ParseCadenceTemplate(txFilename)

	e, err := g.Transaction(string(txScript)).
		SignProposeAndPayAs(account).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func SetPackIssuerCap(
	g *overflow.Overflow,
	issuer string,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	setDistCap := "../cadence-transactions/pds/set_pack_issuer_cap.cdc"
	setDistCapCode := util.ParseCadenceTemplate(setDistCap)
	_, err = g.
		Transaction(string(setDistCapCode)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().Account("issuer")).
		RunE()
	return
}

func CreateDistribution(
	g *overflow.Overflow,
	privPath string,
	account string,
	title string,
	metadata cadence.Value,
) (events []*overflow.FormatedEvent, err error) {
	createDist := "../cadence-transactions/pds/create_distribution.cdc"
	createDistCode := util.ParseCadenceTemplate(createDist)

	// Private path must match the PackNFT contract
	e, err := g.
		Transaction(string(createDistCode)).
		SignProposeAndPayAs("issuer").
		Args(g.Arguments().
			Argument(cadence.Path{Domain: "private", Identifier: privPath}).
			String(title).
			Argument(metadata)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func GetNextDistID(
	g *overflow.Overflow,
) (distId uint64, err error) {
	pdsDistId := "../cadence-scripts/pds/get_next_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
	d, err := g.Script(string(pdsDistIdCode)).RunReturns()
	distId = d.ToGoValue().(uint64)
	return
}

func GetDistTitle(
	g *overflow.Overflow,
	distId uint64,
) (title string, err error) {
	script := "../cadence-scripts/pds/get_dist_title.cdc"
	code := util.ParseCadenceTemplate(script)
	r, err := g.Script(string(code)).
		Args(g.Arguments().UInt64(distId)).
		RunReturns()
	title = r.ToGoValue().(string)
	return
}

func GetDistState(
	g *overflow.Overflow,
	distId uint64,
) (state string, err error) {
	script := "../cadence-scripts/pds/get_dist_state.cdc"
	code := util.ParseCadenceTemplate(script)
	r, err := g.Script(string(code)).
		Args(g.Arguments().UInt64(distId)).
		RunReturns()
	rInt := r.ToGoValue().(uint8)
	switch rInt {
	case 0:
		state = "initialized"
	case 1:
		state = "invalid"
	case 2:
		state = "complete"
	}
	return
}

func GetDistMetadata(
	g *overflow.Overflow,
	distId uint64,
) (metadata string, err error) {
	script := "../cadence-scripts/pds/get_dist_metadata.cdc"
	code := util.ParseCadenceTemplate(script)
	r, err := g.Script(string(code)).
		Args(g.Arguments().UInt64(distId)).
		RunReturns()
	metadata = r.String()
	return
}

func PDSWithdrawNFT(
	g *overflow.Overflow,
	distId uint64,
	nftIds cadence.Value,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	withdraw := "../cadence-transactions/pds/settle.cdc"
	withdrawCode := util.ParseCadenceTemplate(withdraw)
	e, err := g.
		Transaction(string(withdrawCode)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().
			UInt64(distId).
			Argument(nftIds)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSMintPackNFT(
	g *overflow.Overflow,
	distId uint64,
	commitHash string,
	issuer string,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/mint_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	var arr []cadence.Value
	arr = append(arr, cadence.String(commitHash))
	hashes := cadence.NewArray(arr)
	e, err := g.
		Transaction(string(code)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().
			UInt64(distId).
			Argument(hashes).
			Account(issuer)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSUpdateDistState(
	g *overflow.Overflow,
	distId uint64,
	state string,
) (events []*overflow.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/update_dist_state.cdc"
	code := util.ParseCadenceTemplate(txScript)
	var stateInt uint8
	switch state {
	case "invalid":
		stateInt = 1
	case "complete":
		stateInt = 2
	default:
		err = errors.New("not supported case")
		return
	}
	e, err := g.
		Transaction(string(code)).
		SignProposeAndPayAs("pds").
		Args(g.Arguments().
			UInt64(distId).
			UInt8(stateInt)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSRevealPackNFT(
	g *overflow.Overflow,
	distId uint64,
	packId uint64,
	nftContractAddrs cadence.Value,
	nftContractNames cadence.Value,
	nftIds cadence.Value,
	salt string,
	owner string,
	openReq bool,
	privPath string,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/reveal_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	e, err := g.
		Transaction(string(code)).
		SignProposeAndPayAs(account).
		Args(g.Arguments().
			UInt64(distId).
			UInt64(packId).
			Argument(nftContractAddrs).
			Argument(nftContractNames).
			Argument(nftIds).
			String(salt).
			Account(owner).
			Boolean(openReq).
			Argument(cadence.Path{Domain: "private", Identifier: privPath})).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSOpenPackNFT(
	g *overflow.Overflow,
	distId uint64,
	packId uint64,
	nftContractAddrs cadence.Value,
	nftContractNames cadence.Value,
	nftIds cadence.Value,
	owner string,
	privPath string,
	account string,
) (events []*overflow.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/open_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	e, err := g.
		Transaction(string(code)).
		SignProposeAndPayAs(account).
		Args(g.Arguments().
			UInt64(distId).
			UInt64(packId).
			Argument(nftContractAddrs).
			Argument(nftContractNames).
			Argument(nftIds).
			Account(owner).
			Argument(cadence.Path{Domain: "private", Identifier: privPath})).
		RunE()
	events = util.ParseTestEvents(e)
	return
}
