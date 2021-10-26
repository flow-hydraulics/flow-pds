package http

import (
	"time"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
)

type ReqCreateDistribution struct {
	FlowID       common.FlowID      `json:"distFlowID"`
	Issuer       common.FlowAddress `json:"issuer"`
	PackTemplate ReqPackTemplate    `json:"packTemplate"`
}

type ReqPackTemplate struct {
	PackReference AddressLocation `json:"packReference"`
	PackCount     uint            `json:"packCount"`
	Buckets       []ReqBucket     `json:"buckets"`

	// This is here to provide compatibility between backend and onchain contracts.
	// Backend handles CollectibleReferences per bucket but onchain contracts
	// currently handle CollectibleReferences per distribution.
	CollectibleReference AddressLocation `json:"collectibleReference"`
}

type ReqBucket struct {
	// NOTE: read about compatibility above
	// CollectibleReference  AddressLocation   `json:"collectibleReference"`
	CollectibleCount      uint              `json:"collectibleCount"`
	CollectibleCollection common.FlowIDList `json:"collectibleCollection"`
}

type ResCreateDistribution struct {
	ID     uuid.UUID     `json:"distID"`
	FlowID common.FlowID `json:"distFlowID"`
}

type ResGetDistribution struct {
	ID           uuid.UUID                `json:"distID"`
	FlowID       common.FlowID            `json:"distFlowID"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
	Issuer       common.FlowAddress       `json:"issuer"`
	State        common.DistributionState `json:"state"`
	PackTemplate ResPackTemplate          `json:"packTemplate"`
}

type ResListDistribution struct {
	ID        uuid.UUID                `json:"distID"`
	FlowID    common.FlowID            `json:"distFlowID"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
	Issuer    common.FlowAddress       `json:"issuer"`
	State     common.DistributionState `json:"state"`
}

type ResPackTemplate struct {
	PackReference AddressLocation `json:"packReference"`
	PackCount     uint            `json:"packCount"`
	Buckets       []ResBucket     `json:"buckets"`
}

type ResBucket struct {
	CollectibleReference AddressLocation `json:"collectibleReference"`
	CollectibleCount     uint            `json:"collectibleCount"`
}

type AddressLocation struct {
	Name    string             `json:"name"`
	Address common.FlowAddress `json:"address"`
}

func ResGetDistributionFromApp(d *app.Distribution) ResGetDistribution {
	return ResGetDistribution{
		ID:           d.ID,
		FlowID:       d.FlowID,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
		Issuer:       d.Issuer,
		State:        d.State,
		PackTemplate: ResPackTemplateFromApp(d.PackTemplate),
	}
}

func ResDistributionListFromApp(dd []app.Distribution) []ResListDistribution {
	res := make([]ResListDistribution, len(dd))
	for i, d := range dd {
		res[i] = ResListDistribution{
			ID:        d.ID,
			FlowID:    d.FlowID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			Issuer:    d.Issuer,
			State:     d.State,
		}
	}
	return res
}

func ResPackTemplateFromApp(pt app.PackTemplate) ResPackTemplate {
	return ResPackTemplate{
		PackReference: AddressLocation(pt.PackReference),
		PackCount:     pt.PackCount,
		Buckets:       ResBucketsFromApp(pt),
	}
}

func ResBucketsFromApp(pt app.PackTemplate) []ResBucket {
	buckets := make([]ResBucket, len(pt.Buckets))
	for i, b := range pt.Buckets {
		buckets[i] = ResBucket{
			CollectibleReference: AddressLocation(b.CollectibleReference),
			CollectibleCount:     b.CollectibleCount,
		}
	}
	return buckets
}

func (d ReqCreateDistribution) ToApp() app.Distribution {
	return app.Distribution{
		State:        common.DistributionStateInit,
		FlowID:       d.FlowID,
		Issuer:       d.Issuer,
		PackTemplate: d.PackTemplate.ToApp(),
	}
}

func (pt ReqPackTemplate) ToApp() app.PackTemplate {
	buckets := make([]app.Bucket, len(pt.Buckets))
	for i, b := range pt.Buckets {
		// ref := b.CollectibleReference
		ref := pt.CollectibleReference

		buckets[i] = app.Bucket{
			CollectibleReference:  app.AddressLocation(ref),
			CollectibleCount:      b.CollectibleCount,
			CollectibleCollection: b.CollectibleCollection,
		}
	}
	return app.PackTemplate{
		PackReference: app.AddressLocation(pt.PackReference),
		PackCount:     pt.PackCount,
		Buckets:       buckets,
	}
}
