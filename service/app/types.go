package app

import (
	"fmt"
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
	PackReference AddressLocation `gorm:"embedded;embeddedPrefix:pack_ref_"` // Reference to the pack NFT contract
	PackCount     uint            `gorm:"column:pack_count"`                 // How many packs to create
	Buckets       []Bucket        // How to distribute collectibles in a pack
}

type Bucket struct {
	gorm.Model
	DistributionID uuid.UUID
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	CollectibleReference  AddressLocation   `gorm:"embedded;embeddedPrefix:collectible_ref_"` // Reference to the collectible NFT contract
	CollectibleCount      uint              `gorm:"column:collectible_count"`                 // How many collectibles to pick from this bucket
	CollectibleCollection common.FlowIDList `gorm:"column:collectible_collection"`            // Collection of collectibles to pick from
}

type Pack struct {
	gorm.Model
	DistributionID uuid.UUID
	ID             uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	FlowID         common.FlowID      `gorm:"column:flow_id;index"`         // ID of the pack NFT
	State          common.PackState   `gorm:"column:state"`                 // public
	Salt           common.BinaryValue `gorm:"column:salt"`                  // private
	CommitmentHash common.BinaryValue `gorm:"column:commitment_hash;index"` // public
	Collectibles   []Collectible      // private
}

// AddressLocation is a reference to a contract on chain.
type AddressLocation struct {
	Name    string             `gorm:"column:name"`
	Address common.FlowAddress `gorm:"column:address"`
}
type Collectible struct {
	gorm.Model
	PackID uuid.UUID
	ID     uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	FlowID            common.FlowID          `gorm:"column:flow_id"`                        // ID of the collectible NFT
	ContractReference common.AddressLocation `gorm:"embedded;embeddedPrefix:contract_ref_"` // Reference to the collectible NFT contract
}

// Implement sort.Interface by FlowID for Collectible slice
type CollectibleByFlowID []Collectible

func (c CollectibleByFlowID) Len() int           { return len(c) }
func (c CollectibleByFlowID) Less(i, j int) bool { return c[i].FlowID < c[j].FlowID }
func (c CollectibleByFlowID) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

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

func (Collectible) TableName() string {
	return "distribution_collectibles"
}

func (c *Collectible) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = uuid.New()
	return nil
}

func (c Collectible) String() string {
	return fmt.Sprintf("%s.%d", c.ContractReference, c.FlowID)
}
