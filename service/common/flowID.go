package common

import (
	sql "database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/cadence"
)

// Note (latenssi): flow IDs are actually uint64s so we are not supporting as many IDs as flow,
// 									but to my knowledge int64 is the largest MySQL can store.
// 									For reference:
//									https://www.reddit.com/r/golang/comments/7eycli/why_is_there_no_sqlnulluint64/

type FlowID sql.NullInt64 // Nullable int 64
type FlowIDList []FlowID

func (i FlowID) LessThan(j FlowID) bool {
	return j.Valid && i.Int64 < j.Int64
}

func (i FlowID) EqualTo(j FlowID) bool {
	return (!i.Valid && !j.Valid) || i.Valid && j.Valid && i.Int64 == j.Int64
}

func (i FlowID) Value() (driver.Value, error) {
	if !i.Valid {
		return nil, nil
	}
	return i.Int64, nil
}

func (i *FlowID) Scan(value interface{}) error {
	temp := sql.NullInt64(*i)
	err := temp.Scan(value)
	if err != nil {
		return err
	}
	*i = FlowID(temp)
	return nil
}

func (i FlowID) MarshalJSON() ([]byte, error) {
	if i.Valid {
		return json.Marshal(i.Int64)
	} else {
		return json.Marshal(nil)
	}
}

func (i *FlowID) UnmarshalJSON(data []byte) error {
	temp, err := FlowIDFromString(string(data))
	if err != nil {
		return err
	}
	*i = temp
	return nil
}

func (i FlowID) String() string {
	return fmt.Sprint(i.Int64)
}

func FlowIDFromString(s string) (FlowID, error) {
	if s == "" || s == "null" {
		return FlowID{Int64: 0, Valid: false}, nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return FlowID{Int64: 0, Valid: false}, err
	}
	return FlowID{Int64: i, Valid: true}, nil
}

func FlowIDFromCadence(v cadence.Value) (FlowID, error) {
	uintID, ok := v.ToGoValue().(uint64)
	if !ok {
		return FlowID{}, fmt.Errorf("unable to parse FlowID from cadence value: %v", v)
	}
	return FlowID{Int64: int64(uintID), Valid: true}, nil
}

func (FlowIDList) GormDataType() string {
	return "text"
}

func (l *FlowIDList) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal FlowIDList value: %v", value)
	}
	strSplit := strings.Split(string(str), ",")
	list := make([]FlowID, len(strSplit))
	for i, s := range strSplit {
		id, err := FlowIDFromString(s)
		if err != nil {
			return err
		}
		list[i] = id
	}
	*l = list
	return nil
}

func (l FlowIDList) Value() (driver.Value, error) {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(l)), ","), "[]"), nil
}

func (l FlowIDList) Contains(b FlowID) (int, bool) {
	for i, a := range l {
		if a == b {
			return i, true
		}
	}
	return -1, false
}

func FlowIDListFromCadence(cArr cadence.Value) (FlowIDList, error) {
	arr, ok := cArr.(cadence.Array)
	if !ok {
		return nil, fmt.Errorf("unable to parse as cadence.Array: %s", cArr)
	}
	res := make(FlowIDList, len(arr.Values))
	for i := 0; i < len(arr.Values); i++ {
		id, err := FlowIDFromCadence(arr.Values[i])
		if err != nil {
			return nil, err
		}
		res[i] = id
	}
	return res, nil
}
