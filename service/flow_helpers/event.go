package flow_helpers

import (
	"github.com/onflow/cadence"
	flow "github.com/onflow/flow-go-sdk"
)

// Convert the values in an flow.Event to a map for accessing by identifier
// NOTE: May be deprecated once such a helper function exists in the cadence lib
func EventValuesToMap(e flow.Event) map[string]cadence.Value {
	valueMap := make(map[string]cadence.Value)

	for i, field := range e.Value.EventType.Fields {
		value := e.Value.Fields[i]
		valueMap[field.Identifier] = value
	}

	return valueMap
}
