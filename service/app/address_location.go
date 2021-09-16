package app

import (
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/common"
)

// AddressLocation is a reference to a contract on chain.
type AddressLocation struct {
	Name    string             `gorm:"column:name"`
	Address common.FlowAddress `gorm:"column:address"`
}

func (al AddressLocation) String() string {
	return fmt.Sprintf("A.%s.%s", al.Address, al.Name)
}
