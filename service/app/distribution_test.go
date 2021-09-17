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
		collection[i] = common.FlowID(i + 1)
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
		DistID: common.FlowID(1),
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
		DistID: common.FlowID(1),
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

	r1 := d.ResolvedCollection()
	r2 := d.ResolvedCollection()

	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("resolved collections should match")
	}

	for i := range r1 {
		if i > 0 && r1[i-1].FlowID > r1[i].FlowID {
			t.Fatal("resolved collection should be sorted ascending by flow id")
		}
	}

	if len(d.Packs) != packCount {
		t.Fatalf("expected there to be %d packs", packCount)
	}

	for _, p := range d.Packs {
		expected := d.PackSlotCount()
		if len(p.Collectibles) != expected {
			t.Fatalf("expected there to be %d slots", expected)
		}

		for _, c := range p.Collectibles {
			if c.FlowID == common.FlowID(0) {
				t.Fatalf("did not expect 0 value in a slot")
			}
		}
	}
}
