package util

import (
	"bytes"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"text/template"

	"fmt"
	"reflect"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/stretchr/testify/assert"
)

const flowPath = "../flow.json"

var FlowJSON []string = []string{flowPath}

type Addresses struct {
	NonFungibleToken string
	ExampleNFT       string
	PackNFT          string
	IPackNFT         string
	PDS              string
}

type TestEvent struct {
	Name   string
	Fields map[string]interface{}
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
	// addresses = Addresses{"f8d6e0586b0a20c7", "01cf0e2f2f715450", "01cf0e2f2f715450", "f3fcd2c1a78f5eee", "f3fcd2c1a78f5eee"}
	addresses = Addresses{os.Getenv("NON_FUNGIBLE_TOKEN_ADDRESS"), os.Getenv("EXAMPLE_NFT_ADDRESS"), os.Getenv("PACKNFT_ADDRESS"), os.Getenv("PDS_ADDRESS"), os.Getenv("PDS_ADDRESS")}

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
		Fields: make(map[string]interface{}),
	}
}

func NewExpectedPDSEvent(name string) TestEvent {
	return TestEvent{
		Name:   "A." + addresses.PDS + ".PDS." + name,
		Fields: make(map[string]interface{}),
	}
}

func (te TestEvent) AddField(fieldName string, fieldValue interface{}) TestEvent {
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
		v := reflect.ValueOf(event.Fields[k])
		switch v.Kind() {
		case reflect.String:
			assert.Equal(t, te.Fields[k], event.Fields[k])
		case reflect.Slice:
			assert.Equal(t, len(te.Fields[k].([]interface{})), v.Len())
			for i := 0; i < v.Len(); i++ {
				// This is the special case we are addressing
				u := te.Fields[k].([]interface{})[i].(uint64)
				assert.Equal(t, strconv.FormatUint(u, 10), v.Interface().([]interface{})[i])
				i++
			}
		case reflect.Map:
			fmt.Printf("map: %v\n", v.Interface())
		case reflect.Chan:
			fmt.Printf("chan %v\n", v.Interface())
		default:
			fmt.Println("Unsupported types")
		}
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

func GetHash(g *gwtf.GoWithTheFlow, toHash string) (result string, err error) {
	filename := "../cadence-scripts/packNFT/checksum.cdc"
	script := ParseCadenceTemplate(filename)
	r, err := g.ScriptFromFile(filename, script).StringArgument(toHash).RunReturns()
	result = r.ToGoValue().(string)
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
