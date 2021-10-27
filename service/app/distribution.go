package app

import (
	"fmt"
	"math/rand"
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
	packSlotCount, err := dist.PackTemplate.PackSlotCount()
	if err != nil {
		return err
	}

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

func (dist *Distribution) SetState(target common.DistributionState, prereq common.DistributionState) error {
	if dist.State != prereq {
		return fmt.Errorf("distribution can not be set to '%s' from '%s'", target, dist.State)
	}

	dist.State = target

	return nil
}

// SetSetup sets the status to "setup" if preceding state was valid
func (dist *Distribution) SetSetup() error {
	return dist.SetState(common.DistributionStateSetup, common.DistributionStateResolved)
}

// SetSettling sets the status to "settling" if preceding state was valid
func (dist *Distribution) SetSettling() error {
	return dist.SetState(common.DistributionStateSettling, common.DistributionStateSetup)
}

// SetSettled sets the status to "settled" if preceding state was valid
func (dist *Distribution) SetSettled() error {
	return dist.SetState(common.DistributionStateSettled, common.DistributionStateSettling)
}

// SetMinting sets the status to "minting" if preceding state was valid
func (dist *Distribution) SetMinting() error {
	return dist.SetState(common.DistributionStateMinting, common.DistributionStateSettled)
}

// SetComplete sets the status to "complete" if preceding state was valid
func (dist *Distribution) SetComplete() error {
	return dist.SetState(common.DistributionStateComplete, common.DistributionStateMinting)
}

// SetInvalid sets the status to "invalid" if preceding state was valid
func (dist *Distribution) SetInvalid() error {
	if dist.State == common.DistributionStateComplete {
		return fmt.Errorf("distribution can not be set to '%s' from '%s'", common.DistributionStateInvalid, dist.State)
	}

	dist.State = common.DistributionStateInvalid

	return nil
}

func (d Distribution) TemplateCollectibleCount() (int, error) {
	packSlotCount, err := d.PackTemplate.PackSlotCount()
	if err != nil {
		return 0, err
	}
	return int(d.PackTemplate.PackCount) * packSlotCount, nil
}

// PackSlotCount returns the number of slots in each Pack described by PackTemplate
// (sum of all buckets ColletibleCounts)
func (pt PackTemplate) PackSlotCount() (int, error) {
	if pt.Buckets == nil {
		return 0, fmt.Errorf("distribution not fully hydrated from database")
	}

	res := 0
	for _, bucket := range pt.Buckets {
		res += int(bucket.CollectibleCount)
	}

	return res, nil
}
