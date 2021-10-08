package pds

import (
	"errors"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func CreatePackIssuer(
	g *gwtf.GoWithTheFlow,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txFilename := "../cadence-transactions/pds/create_new_pack_issuer.cdc"
	txScript := util.ParseCadenceTemplate(txFilename)

	e, err := g.TransactionFromFile(txFilename, txScript).
		SignProposeAndPayAs(account).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func SetPackIssuerCap(
	g *gwtf.GoWithTheFlow,
	issuer string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	setDistCap := "../cadence-transactions/pds/set_pack_issuer_cap.cdc"
	setDistCapCode := util.ParseCadenceTemplate(setDistCap)
	_, err = g.
		TransactionFromFile(setDistCap, setDistCapCode).
		SignProposeAndPayAs("pds").
		AccountArgument("issuer").
		RunE()
	return
}

func CreateDistribution(
	g *gwtf.GoWithTheFlow,
	account string,
	title string,
	metadata cadence.Value,
) (events []*gwtf.FormatedEvent, err error) {
	createDist := "../cadence-transactions/pds/create_distribution.cdc"
	createDistCode := util.ParseCadenceTemplate(createDist)

	// Private path must match the PackNFT contract
	e, err := g.
		TransactionFromFile(createDist, createDistCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "exampleNFTCollectionProvider"}).
		StringArgument(title).
		Argument(metadata).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func GetDistID(
	g *gwtf.GoWithTheFlow,
) (distId uint64, err error) {
	pdsDistId := "../cadence-scripts/pds/get_current_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
	d, err := g.ScriptFromFile(pdsDistId, pdsDistIdCode).RunReturns()
	distId = d.ToGoValue().(uint64)
	return
}

func GetDistTitle(
	g *gwtf.GoWithTheFlow,
	distId uint64,
) (title string, err error) {
	script := "../cadence-scripts/pds/get_dist_title.cdc"
	code := util.ParseCadenceTemplate(script)
	r, err := g.ScriptFromFile(script, code).UInt64Argument(distId).RunReturns()
	title = r.ToGoValue().(string)
	return
}

func GetDistState(
	g *gwtf.GoWithTheFlow,
	distId uint64,
) (state string, err error) {
	script := "../cadence-scripts/pds/get_dist_state.cdc"
	code := util.ParseCadenceTemplate(script)
	r, err := g.ScriptFromFile(script, code).UInt64Argument(distId).RunReturns()
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
	g *gwtf.GoWithTheFlow,
	distId uint64,
) (metadata string, err error) {
	script := "../cadence-scripts/pds/get_dist_metadata.cdc"
	code := util.ParseCadenceTemplate(script)
	r, err := g.ScriptFromFile(script, code).UInt64Argument(distId).RunReturns()
	metadata = r.String()
	return
}

func PDSWithdrawNFT(
	g *gwtf.GoWithTheFlow,
	distId uint64,
	nftIds cadence.Value,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	withdraw := "../cadence-transactions/pds/settle_exampleNFT.cdc"
	withdrawCode := util.ParseCadenceTemplate(withdraw)
	e, err := g.
		TransactionFromFile(withdraw, withdrawCode).
		SignProposeAndPayAs("pds").
		UInt64Argument(distId).
		Argument(nftIds).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSMintPackNFT(
	g *gwtf.GoWithTheFlow,
	distId uint64,
	commitHash string,
	issuer string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/mint_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	var arr []cadence.Value
	arr = append(arr, cadence.String(commitHash))
	hashes := cadence.NewArray(arr)
	e, err := g.
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
		UInt64Argument(distId).
		Argument(hashes).
		AccountArgument(issuer).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSUpdateDistState(
	g *gwtf.GoWithTheFlow,
	distId uint64,
	state string,
) (events []*gwtf.FormatedEvent, err error) {
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
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
		UInt64Argument(distId).
		UInt8Argument(stateInt).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSRevealPackNFT(
	g *gwtf.GoWithTheFlow,
	distId uint64,
	packId uint64,
	nftContractAddrs cadence.Value,
	nftContractNames cadence.Value,
	nftIds cadence.Value,
	salt string,
	receiver string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/reveal_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	var recv cadence.Value
	if len(receiver) != 0 {
		addr := g.Account(receiver).Address()
		recv = cadence.NewAddress(addr)
	} else {
		recv = nil
	}
	e, err := g.
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
		UInt64Argument(distId).
		UInt64Argument(packId).
		Argument(nftContractAddrs).
		Argument(nftContractNames).
		Argument(nftIds).
		StringArgument(salt).
		Argument(cadence.NewOptional(recv)).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func PDSOpenPackNFT(
	g *gwtf.GoWithTheFlow,
	distId uint64,
	packId uint64,
	nftContractAddrs cadence.Value,
	nftContractNames cadence.Value,
	nftIds cadence.Value,
	owner string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txScript := "../cadence-transactions/pds/open_packNFT.cdc"
	code := util.ParseCadenceTemplate(txScript)
	e, err := g.
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
		UInt64Argument(distId).
		UInt64Argument(packId).
		Argument(nftContractAddrs).
		Argument(nftContractNames).
		Argument(nftIds).
		AccountArgument(owner).
		RunE()
	events = util.ParseTestEvents(e)
	return
}
