package utils

import "github.com/onflow/cadence"

func NewCadenceString(str string) cadence.String {
	cadenceString, _ := cadence.NewString(str)
	return cadenceString
}
