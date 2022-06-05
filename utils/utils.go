package utils

import "github.com/onflow/cadence"

func NewCadenceString(str string) cadence.String {
	cadenceStr, _ := cadence.NewString(str)
	return cadenceStr
}
