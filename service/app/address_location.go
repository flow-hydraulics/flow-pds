package app

import "fmt"

func (al AddressLocation) String() string {
	return fmt.Sprintf("A.%s.%s", al.Address, al.Name)
}
