package packnft 

import (
	"testing"
    "fmt"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"

	"github.com/stretchr/testify/assert"
)

func TestCreateMinterProxy(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, "emulator", false, 1)

	_, err := CreateMinterProxy(g, "pds")
	assert.NoError(t, err)
}

func TestMintWithProxyWithoutCap(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, "emulator", false, 1)

	_, err := MinterProxyMint(g, "pds", "issuer", "commitHash123")
	assert.Error(t, err)

}

func TestSetProxyCapability(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, "emulator", false, 1)
	err := SetMinterProxyCapability(g, "pds", "issuer")
	assert.NoError(t, err)
}

func TestMintWithProxyWithCap(t *testing.T) {
	g := gwtf.NewGoWithTheFlow(util.FlowJSON, "emulator", false, 1)

	events, err := MinterProxyMint(g, "pds", "issuer", "commitHash123")
	assert.NoError(t, err)

    // TODO get id from contract 

	// Test event
    
    fmt.Print(events)
    // First event is deposit
	util.NewExpectedPackNFTEvent("Mint").AddField("id", "0").AddField("commitHash", "commitHash123").AssertEqual(t, events[1])

    // TODO get id from contract check incremented
}

 
