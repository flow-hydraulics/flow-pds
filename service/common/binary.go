package common

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
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
