package flow_helpers

import (
	"github.com/onflow/cadence"
)

// Convert the values in an cadence.Event to a map for accessing by identifier
// NOTE: May be deprecated once such a helper function exists in the cadence lib
func EventValuesToMap(fields []cadence.Field, values []cadence.Value) map[string]cadence.Value {
	valueMap := make(map[string]cadence.Value)

	for i, field := range fields {
		value := values[i]
		valueMap[field.Identifier] = value
	}

	return valueMap
}
