package util

import (
	"bytes"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	// "os"

	"text/template"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/stretchr/testify/assert"
)

const flowPath = "../../flow.json"

var FlowJSON []string = []string{flowPath}

type Addresses struct {
	NonFungibleToken string
	ExampleNFT       string
	PackNFT          string
	IPackNFT         string
	PDSInterface     string
	PDS string
}

type TestEvent struct {
	Name   string
	Fields map[string]string
}

var addresses Addresses

func ParseCadenceTemplate(templatePath string) []byte {
	fb, err := ioutil.ReadFile(templatePath)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("Template").Parse(string(fb))
	if err != nil {
		panic(err)
	}

	// Addresss for emulator are
	addresses = Addresses{"f8d6e0586b0a20c7", "01cf0e2f2f715450", "01cf0e2f2f715450", "f3fcd2c1a78f5eee", "f3fcd2c1a78f5eee", "f3fcd2c1a78f5eee"}
	// PDS account deploys IPackNFTInterface, PDSInterface, PDS contracts
	// addresses = Addresses{os.Getenv("NON_FUNGIBLE_TOKEN_ADDRESS"), os.Getenv("EXAMPLE_NFT_ADDRESS"), os.Getenv("PackNFT"), os.Getenv("PDS_ADDRESS"), os.Getenv("PDS_ADDRESS")}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, addresses)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func ParseTestEvents(events []flow.Event) (formatedEvents []*gwtf.FormatedEvent) {
	for _, e := range events {
		formatedEvents = append(formatedEvents, gwtf.ParseEvent(e, uint64(0), time.Now(), nil))
	}
	return
}

func NewExpectedPackNFTEvent(name string) TestEvent {
	return TestEvent{
		Name:   "A." + addresses.PackNFT + ".PackNFT." + name,
		Fields: map[string]string{},
	}
}

func NewExpectedPDSEvent(name string) TestEvent {
	return TestEvent{
		Name:   "A." + addresses.PDSInterface + ".PDS." + name,
		Fields: map[string]string{},
	}
}

func (te TestEvent) AddField(fieldName string, fieldValue string) TestEvent {
	te.Fields[fieldName] = fieldValue
	return te
}

func (te TestEvent) AssertHasKey(t *testing.T, event *gwtf.FormatedEvent, key string) {
	assert.Equal(t, te.Name, event.Name)
	_, exist := event.Fields[key]
	assert.Equal(t, true, exist)
}

func (te TestEvent) AssertEqual(t *testing.T, event *gwtf.FormatedEvent) {
	assert.Equal(t, event.Name, te.Name)
	assert.Equal(t, len(te.Fields), len(event.Fields))
	for k := range te.Fields {
		assert.Equal(t, te.Fields[k], event.Fields[k])
	}
}

// Gets the address in the format of a hex string from an account name
func GetAccountAddr(g *gwtf.GoWithTheFlow, name string) string {
	address := g.Account(name).Address().String()
	zeroPrefix := "0"
	if string(address[0]) == zeroPrefix {
		address = address[1:]
	}
	return "0x" + address
}

func ReadCadenceCode(ContractPath string) []byte {
	b, err := ioutil.ReadFile(ContractPath)
	if err != nil {
		panic(err)
	}
	return b
}

func GetTotalSupply(g *gwtf.GoWithTheFlow) (result cadence.UFix64, err error) {
	filename := "../../../scripts/contract/get_total_supply.cdc"
	script := ParseCadenceTemplate(filename)
	r, err := g.ScriptFromFile(filename, script).RunReturns()
	result = r.(cadence.UFix64)
	return
}

func GetName(g *gwtf.GoWithTheFlow) (result string, err error) {
	filename := "../../../scripts/contract/get_name.cdc"
	script := ParseCadenceTemplate(filename)
	r, err := g.ScriptFromFile(filename, script).RunReturns()
	result = r.ToGoValue().(string)
	return
}

func GetVersion(g *gwtf.GoWithTheFlow) (result string, err error) {
	filename := "../../../scripts/contract/get_version.cdc"
	script := ParseCadenceTemplate(filename)
	r, err := g.ScriptFromFile(filename, script).RunReturns()
	result = r.ToGoValue().(string)
	return
}

func GetBalance(g *gwtf.GoWithTheFlow, account string) (result cadence.UFix64, err error) {
	filename := "../../../scripts/vault/get_balance.cdc"
	script := ParseCadenceTemplate(filename)
	value, err := g.ScriptFromFile(filename, script).AccountArgument(account).RunReturns()
	if err != nil {
		return
	}
	result = value.(cadence.UFix64)
	return
}

func GetUUID(g *gwtf.GoWithTheFlow, account string, resourceName string) (r uint64, err error) {
	filename := "../../../scripts/contract/get_resource_uuid.cdc"
	script := ParseCadenceTemplate(filename)
	value, err := g.ScriptFromFile(filename, script).AccountArgument(account).StringArgument(resourceName).RunReturns()
	if err != nil {
		return
	}
	r, ok := value.ToGoValue().(uint64)
	if !ok {
		err = errors.New("returned not uint64")
	}
	return
}

func ConvertCadenceByteArray(a cadence.Value) (b []uint8) {
	// type assertion of interface
	i := a.ToGoValue().([]interface{})

	for _, e := range i {
		// type assertion of uint8
		b = append(b, e.(uint8))
	}
	return

}

func ConvertCadenceStringArray(a cadence.Value) (b []string) {
	// type assertion of interface
	i := a.ToGoValue().([]interface{})

	for _, e := range i {
		b = append(b, e.(string))
	}
	return
}

// Multisig utility functions and type

// Arguement for Multisig functions `Multisig_SignAndSubmit`
// This allows for generic functions to type cast the arguments into
// correct cadence types.
// i.e. for a cadence.UFix64, Arg {V: "12.00", T: "UFix64"}
type Arg struct {
	V interface{}
	T string
}
