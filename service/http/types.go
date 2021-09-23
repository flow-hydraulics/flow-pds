package http

import (
	"time"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
)

type ResCreateDistribution struct {
	DistributionId uuid.UUID `json:"distributionId"`
}

type ReqCreateDistribution struct {
	DistID       common.FlowID        `json:"distId"`
	Issuer       common.FlowAddress   `json:"issuer"`
	MetaData     DistributionMetaData `json:"meta"`
	PackTemplate PackTemplate         `json:"packTemplate"`
}

type ResDistribution struct {
	ID                 uuid.UUID                `json:"id"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
	DistID             common.FlowID            `json:"distId"`
	Issuer             common.FlowAddress       `json:"issuer"`
	State              common.DistributionState `json:"state"`
	MetaData           DistributionMetaData     `json:"meta"`
	PackTemplate       PackTemplate             `json:"packTemplate"`
	Packs              []Pack                   `json:"packs"`
	ResolvedCollection []string                 `json:"resolvedCollection"`
	SettlementStatus   SettlementStatus         `json:"settlementStatuts"`
}

type ResDistributionListItem struct {
	ID        uuid.UUID                `json:"id"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
	DistID    common.FlowID            `json:"distId"`
	Issuer    common.FlowAddress       `json:"issuer"`
	State     common.DistributionState `json:"state"`
	MetaData  DistributionMetaData     `json:"meta"`
}

type DistributionMetaData struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
}

type PackTemplate struct {
	PackReference AddressLocation `json:"packReference"`
	PackCount     uint            `json:"packCount"`
	Buckets       []Bucket        `json:"buckets"`
}

type Bucket struct {
	CollectibleReference  AddressLocation   `json:"collectibleReference"`
	CollectibleCount      uint              `json:"collectibleCount"`
	CollectibleCollection common.FlowIDList `json:"collectibleCollection"`
}

type Pack struct {
	FlowID         common.FlowID      `json:"flowID"`
	State          common.PackState   `json:"state"`
	CommitmentHash common.BinaryValue `json:"commitmentHash"`
}

type SettlementStatus struct {
	Settled uint `json:"settled"`
	Total   uint `json:"total"`
}

type AddressLocation struct {
	Name    string             `json:"name"`
	Address common.FlowAddress `json:"address"`
}

func ResDistributionFromApp(d *app.Distribution, s *app.Settlement) ResDistribution {
	resolvedCollection := make([]string, d.SlotCount())
	for i, c := range d.ResolvedCollection() {
		resolvedCollection[i] = c.String()
	}

	settlement := SettlementStatus{}
	if s != nil {
		settlement.Settled = s.CurrentCount
		settlement.Total = s.TotalCount
	}

	return ResDistribution{
		ID:                 d.ID,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
		DistID:             d.DistID,
		Issuer:             d.Issuer,
		State:              d.State,
		MetaData:           DistributionMetaData(d.MetaData),
		PackTemplate:       PackTemplateFromApp(d.PackTemplate),
		Packs:              PacksFromApp(d),
		ResolvedCollection: resolvedCollection,
		SettlementStatus:   settlement,
	}
}

func ResDistributionListFromApp(dd []app.Distribution) []ResDistributionListItem {
	res := make([]ResDistributionListItem, len(dd))
	for i, d := range dd {
		res[i] = ResDistributionListItem{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DistID:    d.DistID,
			Issuer:    d.Issuer,
			State:     d.State,
			MetaData:  DistributionMetaData(d.MetaData),
		}
	}
	return res
}

func PackTemplateFromApp(pt app.PackTemplate) PackTemplate {
	return PackTemplate{
		PackReference: AddressLocation(pt.PackReference),
		PackCount:     pt.PackCount,
		Buckets:       BucketsFromApp(pt),
	}
}

func BucketsFromApp(pt app.PackTemplate) []Bucket {
	buckets := make([]Bucket, len(pt.Buckets))
	for i, b := range pt.Buckets {
		buckets[i] = Bucket{
			CollectibleReference:  AddressLocation(b.CollectibleReference),
			CollectibleCount:      b.CollectibleCount,
			CollectibleCollection: b.CollectibleCollection,
		}
	}
	return buckets
}

func PacksFromApp(d *app.Distribution) []Pack {
	packs := make([]Pack, len(d.Packs))
	for i, p := range d.Packs {
		packs[i] = Pack{
			FlowID:         p.FlowID,
			State:          p.State,
			CommitmentHash: p.CommitmentHash,
		}
	}
	return packs
}

func (d ReqCreateDistribution) ToApp() app.Distribution {
	return app.Distribution{
		DistID:       d.DistID,
		Issuer:       d.Issuer,
		MetaData:     app.DistributionMetaData(d.MetaData),
		PackTemplate: d.PackTemplate.ToApp(),
	}
}

func (pt PackTemplate) ToApp() app.PackTemplate {
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
