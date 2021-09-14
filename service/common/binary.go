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
	return []byte(fmt.Sprintf("\"%s\"", hex.EncodeToString(b))), nil
}
