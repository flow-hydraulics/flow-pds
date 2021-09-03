package flowpds

import (
	"testing"

	"github.com/onflow/flow-go-sdk"
)

func TestDistributionValidation(t *testing.T) {
	collection := make([]CollectibleID, 100)
	for i := range collection {
		collection[i] = CollectibleID(i + 1)
	}

	bucket1 := collection[:20]
	bucket2 := collection[20:25]

	distribution := Distribution{
		Issuer: flow.HexToAddress("0x1"),
		PackTemplate: PackTemplate{
			PackCount: 3,
			PackSlotTemplates: []PackSlotTemplate{
				{
					CollectibleCount:      10,
					CollectibleCollection: bucket1,
				},
				{
					CollectibleCount:      3,
					CollectibleCollection: bucket2,
				},
			},
		},
	}

	if err := distribution.Validate(); err == nil {
		t.Error("expected a validation error")
	}
}

func TestDistributionResolution(t *testing.T) {
	collection := make([]CollectibleID, 100)
	for i := range collection {
		collection[i] = CollectibleID(i + 1)
	}

	bucket1 := collection[:80]
	bucket2 := collection[80:100]

	distribution := Distribution{
		Issuer: flow.HexToAddress("0x1"),
		PackTemplate: PackTemplate{
			PackCount: 4,
			PackSlotTemplates: []PackSlotTemplate{
				{
					CollectibleCount:      2,
					CollectibleCollection: bucket1,
				},
				{
					CollectibleCount:      2,
					CollectibleCollection: bucket2,
				},
			},
		},
	}

	if err := distribution.Resolve(); err != nil {
		t.Fatalf("didn't expect an error, got %s", err)
	}

	t.Log(distribution)
}
