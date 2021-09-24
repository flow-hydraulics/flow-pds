package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const SALT_LENGTH = 8 // TODO (latenssi): is this ok?
const HASH_DELIM = ","

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

	ContractReference AddressLocation    `gorm:"embedded;embeddedPrefix:contract_ref_"` // Reference to the pack NFT contract
	FlowID            common.FlowID      `gorm:"column:flow_id;index"`                  // ID of the pack NFT
	State             common.PackState   `gorm:"column:state"`                          // public
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
		return fmt.Errorf("distribution can not be resolved anymore")
	}

	if err := dist.Validate(); err != nil {
		return fmt.Errorf("distribution validation error: %w", err)
	}

	packCount := int(dist.PackTemplate.PackCount)
	packSlotCount := int(dist.PackSlotCount())

	// Init packs and their slots
	packs := make([]Pack, packCount)
	for i := range packs {
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

// SetSettling sets the status to settling if preceding state was valid
func (dist *Distribution) SetSettling() error {
	if dist.State != common.DistributionStateResolved {
		return fmt.Errorf("distribution can not start settling at this state: %d", dist.State)
	}

	dist.State = common.DistributionStateSettling

	return nil
}

// SetSettled sets the status to settled if preceding state was valid
func (dist *Distribution) SetSettled() error {
	if dist.State != common.DistributionStateSettling {
		return fmt.Errorf("distribution can not be set as settled at this state: %d", dist.State)
	}

	dist.State = common.DistributionStateSettled

	return nil
}

// SetMinting sets the status to minting if preceding state was valid
func (dist *Distribution) SetMinting() error {
	if dist.State != common.DistributionStateSettled {
		return fmt.Errorf("distribution can not start minting at this state: %d", dist.State)
	}

	dist.State = common.DistributionStateMinting

	return nil
}

// SetComplete sets the status to complete if preceding state was valid
func (dist *Distribution) SetComplete() error {
	if dist.State != common.DistributionStateMinting {
		return fmt.Errorf("distribution can not be set as complete at this state: %d", dist.State)
	}

	dist.State = common.DistributionStateComplete

	return nil
}

// SetCancelled sets the status to cancelled if preceding state was valid
func (dist *Distribution) SetCancelled() error {
	if dist.State == common.DistributionStateComplete {
		return fmt.Errorf("distribution can not be cancelled at this state: %d", dist.State)
	}

	dist.State = common.DistributionStateCancelled

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

// SetCommitmentHash should
// - validate the pack
// - decide on a random salt value
// - calculate the commitment hash for the pack
func (p *Pack) SetCommitmentHash() error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("pack validation error: %w", err)
	}

	if !p.Salt.IsEmpty() {
		return fmt.Errorf("salt is already set")
	}

	if !p.CommitmentHash.IsEmpty() {
		return fmt.Errorf("commitmentHash is already set")
	}

	salt, err := common.GenerateRandomBytes(SALT_LENGTH)
	if err != nil {
		return err
	}

	p.Salt = salt
	p.CommitmentHash = p.Hash()

	return nil
}

// Hash outputs the 'commitmentHash' of a pack.
// It is converting inputs to string and joining them with a delim to make the input more readable.
// This will allow anyone to easily copy paste strings and verify the hash.
// We also use the full reference (address and name) of a collectible to make
// it more difficult to fiddle with the types of collectibles inside a pack.
func (p *Pack) Hash() []byte {
	inputs := make([]string, 1+len(p.Collectibles))
	inputs[0] = hex.EncodeToString(p.Salt)
	for i, c := range p.Collectibles {
		inputs[i+1] = c.HashString()
	}
	input := strings.Join(inputs, HASH_DELIM)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

// Seal should set the FlowID of the pack and set it as sealed
func (p *Pack) Seal(id common.FlowID) error {
	if p.State != common.PackStateInit {
		return fmt.Errorf("pack in unexpected state: %d", p.State)
	}

	if p.FlowID.Valid {
		return fmt.Errorf("pack FlowID already set: %v", id)
	}

	p.FlowID = id
	p.State = common.PackStateSealed

	return nil
}

// Reveal should set the pack as revealed
func (p *Pack) Reveal() error {
	if p.State != common.PackStateSealed {
		return fmt.Errorf("pack in unexpected state: %d", p.State)
	}

	p.State = common.PackStateRevealed

	return nil
}

// Open should set the pack as opened
func (p *Pack) Open() error {
	if p.State != common.PackStateRevealed {
		return fmt.Errorf("pack in unexpected state: %d", p.State)
	}

	p.State = common.PackStateOpened

	return nil
}
