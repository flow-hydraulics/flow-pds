package app

import (
	"reflect"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/onflow/flow-go-sdk"
)

func makeCollection(size int) []common.FlowID {
	collection := make([]common.FlowID, size)
	for i := range collection {
		collection[i] = common.FlowID{Int64: int64(i + 1), Valid: true}
	}
	return collection
}

func TestDistributionValidation(t *testing.T) {
	collection := makeCollection(100)

	bucket1 := collection[:20]
	bucket2 := collection[20:25]
	collectibleRef := AddressLocation{
		Name:    "TestCollectibleNFT",
		Address: common.FlowAddress(flow.HexToAddress("0x2")),
	}

	d := Distribution{
		State:  common.DistributionStateInit,
		FlowID: common.FlowID{Int64: int64(1), Valid: true},
		Issuer: common.FlowAddress(flow.HexToAddress("0x1")),
		PackTemplate: PackTemplate{
			PackReference: AddressLocation{
				Name:    "TestPackNFT",
				Address: common.FlowAddress(flow.HexToAddress("0x2")),
			},
			PackCount: 3,
			Buckets: []Bucket{
				{
					CollectibleReference:  collectibleRef,
					CollectibleCount:      10,
					CollectibleCollection: bucket1,
				},
				{
					CollectibleReference:  collectibleRef,
					CollectibleCount:      3,
					CollectibleCollection: bucket2,
				},
			},
		},
	}

	if err := d.Validate(); err == nil {
		t.Error("expected a validation error")
	}
}

func TestDistributionResolution(t *testing.T) {
	collection := makeCollection(100)

	packCount := 4

	bucket1 := collection[:80]
	bucket2 := collection[80:100]
	collectibleRef := AddressLocation{
		Name:    "TestCollectibleNFT",
		Address: common.FlowAddress(flow.HexToAddress("0x2")),
	}

	d := Distribution{
		State:  common.DistributionStateInit,
		FlowID: common.FlowID{Int64: int64(1), Valid: true},
		Issuer: common.FlowAddress(flow.HexToAddress("0x1")),
		PackTemplate: PackTemplate{
			PackReference: AddressLocation{
				Name:    "TestPackNFT",
				Address: common.FlowAddress(flow.HexToAddress("0x2")),
			},
			PackCount: uint(packCount),
			Buckets: []Bucket{
				{
					CollectibleReference:  collectibleRef,
					CollectibleCount:      2,
					CollectibleCollection: bucket1,
				},
				{
					CollectibleReference:  collectibleRef,
					CollectibleCount:      3,
					CollectibleCollection: bucket2,
				},
			},
		},
	}

	if err := d.Resolve(); err != nil {
		t.Fatalf("didn't expect an error, got %s", err)
	}

	collectibleCount, err := d.TemplateCollectibleCount()
	if err != nil {
		t.Fatal(err)
	}

	r1 := make(Collectibles, 0, collectibleCount)
	for _, pack := range d.Packs {
		r1 = append(r1, pack.Collectibles...)
	}

	r2 := make(Collectibles, 0, collectibleCount)
	for _, pack := range d.Packs {
		r2 = append(r2, pack.Collectibles...)
	}

	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("resolved collections should match")
	}

	if len(d.Packs) != packCount {
		t.Fatalf("expected there to be %d packs", packCount)
	}

	for _, p := range d.Packs {
		expected, err := d.PackTemplate.PackSlotCount()
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Collectibles) != expected {
			t.Fatalf("expected there to be %d slots", expected)
		}
	}
}
