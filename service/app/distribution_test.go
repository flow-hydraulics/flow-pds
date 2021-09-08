package app

import (
	"reflect"
	"testing"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"
)

func makeCollection(size int) []Collectible {
	collection := make([]Collectible, size)
	for i := range collection {
		collection[i] = Collectible{ID: cadence.NewUInt64(uint64(i + 1))}
	}
	return collection
}

func TestDistributionValidation(t *testing.T) {
	collection := makeCollection(100)

	bucket1 := collection[:20]
	bucket2 := collection[20:25]

	distribution := Distribution{
		Issuer: flow.HexToAddress("0x1"),
		PackTemplate: PackTemplate{
			PackCount: 3,
			Buckets: []Bucket{
				{
					CollectibleCollection: bucket1,
				},
				{
					CollectibleCollection: bucket2,
				},
			},
			Slots: []SlotTemplate{
				{BucketIndexes: []int{0}},
				{BucketIndexes: []int{0, 1}},
				{BucketIndexes: []int{0, 1, 1}},
				{BucketIndexes: []int{1}},
			},
		},
	}

	if err := distribution.Validate(); err == nil {
		t.Error("expected a validation error")
	}

	t.Log(distribution.PackTemplate.CollectibleReference.ID())
	t.Log(distribution.PackTemplate.CollectibleReference.Address.Bytes())
}

func TestDistributionResolution(t *testing.T) {
	collection := makeCollection(100)

	bucket1 := collection[:80]
	bucket2 := collection[80:100]

	addr1, err := common.HexToAddress("0x1")
	if err != nil {
		t.Fatal(err)
	}

	distribution := Distribution{
		Issuer: flow.HexToAddress("0x1"),
		PackTemplate: PackTemplate{
			PackCount: 4,
			Buckets: []Bucket{
				{
					CollectibleCollection: bucket1,
				},
				{
					CollectibleCollection: bucket2,
				},
			},
			Slots: []SlotTemplate{
				{BucketIndexes: []int{0}},
				{BucketIndexes: []int{0, 1}},
				{BucketIndexes: []int{0, 1, 1}},
				{BucketIndexes: []int{1}},
			},
			PackReference: common.AddressLocation{
				Name:    "TestPackNFT",
				Address: addr1,
			},
			CollectibleReference: common.AddressLocation{
				Name:    "TestCollectibleNFT",
				Address: addr1,
			},
		},
	}

	if err := distribution.Resolve(); err != nil {
		t.Fatalf("didn't expect an error, got %s", err)
	}

	r1 := distribution.ResolvedCollection()
	r2 := distribution.ResolvedCollection()

	if reflect.DeepEqual(r1, r2) {
		t.Fatalf("resolved collections should not match")
	}

	t.Log(distribution.Packs)
	t.Log(distribution.PackTemplate.CollectibleReference.ID())
}
