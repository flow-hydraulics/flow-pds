package app

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/onflow/flow-go-sdk"
)

type Collectible struct {
	FlowID            common.FlowID   // ID of the collectible NFT
	ContractReference AddressLocation // Reference to the collectible NFT contract
}

type Collectibles []Collectible

func CollectibleFromString(s string) (Collectible, error) {
	split := strings.Split(string(s), ".")
	address := common.FlowAddress(flow.HexToAddress(split[1]))
	name := split[2]
	id, err := common.FlowIDFromString(split[3])
	if err != nil {
		return Collectible{}, err
	}
	return Collectible{
		FlowID: id,
		ContractReference: AddressLocation{
			Name:    name,
			Address: address,
		},
	}, nil
}

func (c Collectible) String() string {
	return fmt.Sprintf("A.%s.%s.%d", c.ContractReference.Address, c.ContractReference.Name, c.FlowID.Int64)
}

func (c Collectible) HashString() string {
	return c.String()
}

// Implement sort.Interface for Collectible slice
func (cc Collectibles) Len() int           { return len(cc) }
func (cc Collectibles) Less(i, j int) bool { return cc[i].FlowID.LessThan(cc[j].FlowID) }
func (cc Collectibles) Swap(i, j int)      { cc[i], cc[j] = cc[j], cc[i] }

func (Collectibles) GormDataType() string {
	return "text"
}

func (cc *Collectibles) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal Collectible value: %v", value)
	}
	strSplit := strings.Split(string(str), ",")
	list := make([]Collectible, len(strSplit))
	for i, s := range strSplit {
		c, err := CollectibleFromString(s)
		if err != nil {
			return err
		}
		list[i] = c
	}
	*cc = list
	return nil
}

func (cc Collectibles) Value() (driver.Value, error) {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cc)), ","), "[]"), nil
}
