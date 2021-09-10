package common

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/cadence"
)

const delim = ","

type FlowID cadence.UInt64
type FlowIDList []FlowID

func FlowIDFromInt64(i int64) FlowID {
	return FlowID(cadence.NewUInt64(uint64(i)))
}

func FlowIDFromStr(s string) (FlowID, error) {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return FlowID(cadence.NewUInt64(u)), nil
}

func (FlowIDList) GormDataType() string {
	return "text"
}

func (l *FlowIDList) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal FlowIDList value: %v", value)
	}
	strSplit := strings.Split(string(str), delim)
	list := make([]FlowID, len(strSplit))
	for i, s := range strSplit {
		id, err := FlowIDFromStr(s)
		if err != nil {
			return err
		}
		list[i] = id
	}
	*l = list
	return nil
}

func (l FlowIDList) Value() (driver.Value, error) {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(l)), delim), "[]"), nil
}
