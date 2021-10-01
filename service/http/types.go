package http

import (
	"time"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
)

// TODO (latenssi): represent states as strings

type ReqCreateDistribution struct {
	FlowID       common.FlowID        `json:"distFlowID"`
	Issuer       common.FlowAddress   `json:"issuer"`
	MetaData     DistributionMetaData `json:"meta"`
	PackTemplate ReqPackTemplate      `json:"packTemplate"`
}

type ReqPackTemplate struct {
	PackReference AddressLocation `json:"packReference"`
	PackCount     uint            `json:"packCount"`
	Buckets       []ReqBucket     `json:"buckets"`
}

type ReqBucket struct {
	CollectibleReference  AddressLocation   `json:"collectibleReference"`
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
	MetaData     DistributionMetaData     `json:"meta"`
	PackTemplate ResPackTemplate          `json:"packTemplate"`
}

type ResListDistribution struct {
	ID        uuid.UUID                `json:"distID"`
	FlowID    common.FlowID            `json:"distFlowID"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
	Issuer    common.FlowAddress       `json:"issuer"`
	State     common.DistributionState `json:"state"`
	MetaData  DistributionMetaData     `json:"meta"`
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

type DistributionMetaData struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
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
		MetaData:     DistributionMetaData(d.MetaData),
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
			MetaData:  DistributionMetaData(d.MetaData),
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
		FlowID:       d.FlowID,
		Issuer:       d.Issuer,
		MetaData:     app.DistributionMetaData(d.MetaData),
		PackTemplate: d.PackTemplate.ToApp(),
	}
}

func (pt ReqPackTemplate) ToApp() app.PackTemplate {
	buckets := make([]app.Bucket, len(pt.Buckets))
	for i, b := range pt.Buckets {
		buckets[i] = app.Bucket{
			CollectibleReference:  app.AddressLocation(b.CollectibleReference),
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
