package app

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Distribution struct {
	gorm.Model
	ID uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	FlowID       common.FlowID            `gorm:"column:flow_id"` // A reference on the PDS Contract to this distribution
	Issuer       common.FlowAddress       `gorm:"column:issuer"`
	State        common.DistributionState `gorm:"column:state;not null;default:null"`
	PackTemplate PackTemplate             `gorm:"embedded;embeddedPrefix:template_"`
	Packs        []Pack                   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type PackTemplate struct {
	PackReference AddressLocation `gorm:"embedded;embeddedPrefix:pack_ref_"`             // Reference to the pack NFT contract
	PackCount     uint            `gorm:"column:pack_count"`                             // How many packs to create
	Buckets       []Bucket        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // How to distribute collectibles in a pack
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

	ContractReference AddressLocation    `gorm:"embedded;embeddedPrefix:contract_ref_"` // Reference to the pack NFT contract
	FlowID            common.FlowID      `gorm:"column:flow_id;index"`                  // ID of the pack NFT
	State             common.PackState   `gorm:"column:state;not null;default:null"`    // public
	Salt              common.BinaryValue `gorm:"column:salt"`                           // private
	CommitmentHash    common.BinaryValue `gorm:"column:commitment_hash;index"`          // public
	Collectibles      Collectibles       `gorm:"column:collectibles"`                   // private
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

// Resolve should
// - validate the distribution
// - distribute given collectibles into packs based on given template
// - hash each pack
// - set the distributions state to resolved
func (dist *Distribution) Resolve() error {
	if dist.State != common.DistributionStateInit {
		return fmt.Errorf("distribution has to be in 'init' state, got '%s'", dist.State)
	}

	if err := dist.Validate(); err != nil {
		return fmt.Errorf("distribution validation error: %w", err)
	}

	packCount := int(dist.PackTemplate.PackCount)
	packSlotCount := int(dist.PackSlotCount())

	// Init packs and their slots
	packs := make([]Pack, packCount)
	for i := range packs {
		packs[i].State = common.PackStateInit
		packs[i].ContractReference = dist.PackTemplate.PackReference
		packs[i].Collectibles = make([]Collectible, packSlotCount)
	}

	// Distributing collectibles
	slotBaseIndex := 0
	for _, bucket := range dist.PackTemplate.Buckets {
		// How many collectibles to pick from this bucket per pack
		countPerPack := int(bucket.CollectibleCount)
		// How many collectibles to pick from this bucket in total
		countTotal := packCount * countPerPack

		// TODO (latenssi): Is this safe enough?
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		// Generate a slice of random indexes to bucket.CollectibleCollection
		permutation := r.Perm(len(bucket.CollectibleCollection))

		for i := 0; i < countTotal; i++ {
			randomIndex := permutation[i]
			packIndex := i % packCount
			slotIndex := (i / packCount) + slotBaseIndex

			collectible := Collectible{
				ContractReference: bucket.CollectibleReference,
				FlowID:            bucket.CollectibleCollection[randomIndex],
			}

			packs[packIndex].Collectibles[slotIndex] = collectible
		}

		slotBaseIndex += countPerPack
	}

	// Setting commitment hashes of each pack
	for i := range packs {
		if err := packs[i].SetCommitmentHash(); err != nil {
			return fmt.Errorf("error while hashing pack %d: %w", i+1, err)
		}
	}

	dist.Packs = packs
	dist.State = common.DistributionStateResolved

	return nil
}

// SetSettling sets the status to "settling" if preceding state was valid
func (dist *Distribution) SetSettling() error {
	if dist.State != common.DistributionStateResolved {
		return fmt.Errorf("distribution can not start settling at this state: %s", dist.State)
	}

	dist.State = common.DistributionStateSettling

	return nil
}

// SetSettled sets the status to "settled" if preceding state was valid
func (dist *Distribution) SetSettled() error {
	if dist.State != common.DistributionStateSettling {
		return fmt.Errorf("distribution can not be set as settled at this state: %s", dist.State)
	}

	dist.State = common.DistributionStateSettled

	return nil
}

// SetMinting sets the status to "minting" if preceding state was valid
func (dist *Distribution) SetMinting() error {
	if dist.State != common.DistributionStateSettled {
		return fmt.Errorf("distribution can not start minting at this state: %s", dist.State)
	}

	dist.State = common.DistributionStateMinting

	return nil
}

// SetComplete sets the status to "complete" if preceding state was valid
func (dist *Distribution) SetComplete() error {
	if dist.State != common.DistributionStateMinting {
		return fmt.Errorf("distribution can not be set as complete at this state: %s", dist.State)
	}

	dist.State = common.DistributionStateComplete

	return nil
}

// SetInvalid sets the status to "invalid" if preceding state was valid
func (dist *Distribution) SetInvalid() error {
	if dist.State == common.DistributionStateComplete {
		return fmt.Errorf("distribution can not be set as invalid at this state: %s", dist.State)
	}

	dist.State = common.DistributionStateInvalid

	return nil
}

// ResolvedCollection should publicly present what collectibles got in the distribution
// without revealing in which pack each one resides
func (dist Distribution) ResolvedCollection() Collectibles {
	res := make(Collectibles, 0, dist.SlotCount())
	for _, pack := range dist.Packs {
		res = append(res, pack.Collectibles...)
	}
	sort.Sort(res)
	return res
}

func (dist Distribution) PackCount() int {
	return int(dist.PackTemplate.PackCount)
}

// PackSlotCount returns the number of slots per pack
func (dist Distribution) PackSlotCount() int {
	res := 0
	for _, bucket := range dist.PackTemplate.Buckets {
		res += int(bucket.CollectibleCount)
	}
	return res
}

// SlotCount returns the total number of slots in distribution
func (dist Distribution) SlotCount() int {
	return dist.PackCount() * dist.PackSlotCount()
}
