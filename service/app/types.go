package app

import (
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TODO (latenssi): foreign key constraints

type Distribution struct {
	gorm.Model
	ID uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	DistID       common.FlowID            `gorm:"column:dist_id"` // A reference on the PDS Contract to this distribution
	Issuer       common.FlowAddress       `gorm:"column:issuer"`
	State        common.DistributionState `gorm:"column:state"`
	MetaData     DistributionMetaData     `gorm:"embedded;embeddedPrefix:meta_"`
	PackTemplate PackTemplate             `gorm:"embedded;embeddedPrefix:template_"`
	Packs        []Pack
}

type DistributionMetaData struct {
	Title       string    `gorm:"column:title"`
	Description string    `gorm:"column:description"`
	Image       string    `gorm:"column:image"`
	StartDate   time.Time `gorm:"column:start_date"`
	EndDate     time.Time `gorm:"column:end_date"`
}

type PackTemplate struct {
	PackReference        AddressLocation `gorm:"embedded;embeddedPrefix:packref_"`        // Reference to the pack NFT contract
	CollectibleReference AddressLocation `gorm:"embedded;embeddedPrefix:collectibleref_"` // Reference to the collectible NFT contract
	PackCount            uint            `gorm:"column:pack_count"`                       // How many packs to create
	Buckets              []Bucket        // How to distribute collectibles in a pack
}

type Bucket struct {
	gorm.Model
	DistributionID uuid.UUID
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	CollectibleCount      uint              `gorm:"column:collectible_count"`      // How many collectibles to pick from this bucket
	CollectibleCollection common.FlowIDList `gorm:"column:collectible_collection"` // Collection of collectibles to pick from
}

type Pack struct {
	gorm.Model
	DistributionID uuid.UUID
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	FlowID         common.FlowID             `gorm:"column:flow_id;index"`         // ID of the Pack NFT
	State          common.PackState          `gorm:"column:state"`                 // public
	Salt           common.PackSalt           `gorm:"column:salt"`                  // private
	CommitmentHash common.PackCommitmentHash `gorm:"column:commitment_hash;index"` // public
	Slots          []PackSlot                // private
}

type PackSlot struct {
	gorm.Model
	PackID uuid.UUID
	ID     uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	State             common.PackSlotState `gorm:"column:state"`
	CollectibleFlowID common.FlowID        `gorm:"column:collectible_flow_id"`
}

// AddressLocation is a reference to a contract on flow chain
type AddressLocation struct {
	Name    string             `gorm:"column:name"`
	Address common.FlowAddress `gorm:"column:address"`
}

func (Distribution) TableName() string {
	return "distributions"
}

func (d *Distribution) BeforeCreate(tx *gorm.DB) (err error) {
	d.ID = uuid.New()
	return nil
}

func (Bucket) TableName() string {
	return "distribution_buckets"
}

func (b *Bucket) BeforeCreate(tx *gorm.DB) (err error) {
	b.ID = uuid.New()
	return nil
}

func (Pack) TableName() string {
	return "distribution_packs"
}

func (p *Pack) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = uuid.New()
	return nil
}

func (PackSlot) TableName() string {
	return "distribution_packslots"
}

func (ps *PackSlot) BeforeCreate(tx *gorm.DB) (err error) {
	ps.ID = uuid.New()
	return nil
}
