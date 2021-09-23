package common

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"

	"github.com/onflow/cadence"
)

type BinaryValue []byte

func (b BinaryValue) IsEmpty() bool {
	return len(b) == 0
}

func (b BinaryValue) Value() (driver.Value, error) {
	return []byte(b), nil
}

func (b BinaryValue) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", b.String())), nil
}

func (b BinaryValue) String() string {
	return hex.EncodeToString(b)
}

func BinaryValueFromHexString(s string) (BinaryValue, error) {
	return hex.DecodeString(s)
}

func BinaryValueFromCadence(v cadence.Value) (BinaryValue, error) {
	hexString, ok := v.ToGoValue().(string)
	if !ok {
		return nil, fmt.Errorf("unable to parse BinaryValue from cadence value: %v", v)
	}
	return BinaryValueFromHexString(hexString)
}
