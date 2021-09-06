package flowpds

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"
)

type CollectibleID uint64
type PackSalt string
type PackCommitmentHash string

type DistributionState int
type PackState int
type PackSlotState int

const (
	DistributionStateInit DistributionState = iota
	DistributionStateResolved
	DistributionStateSettling
	DistributionStateSettled
	DistributionStateComplete
)

const (
	PackStateInit PackState = iota
	PackStateSealed
	PackStateRevealed
	PackStateEmpty
)

const (
	PackSlotStateInit PackSlotState = iota
	PackSlotStateInTransit
	PackSlotStateInStorage
	PackSlotStateEmtpy
)

type Distribution struct {
	Issuer       flow.Address
	State        DistributionState
	PackTemplate PackTemplate
	Packs        []Pack
}

type PackTemplate struct {
	PackCount            uint64                 // How many packs to create
	Buckets              []Bucket               // How to distribute collectibles in a pack
	PackReference        common.AddressLocation // Reference to the pack NFT contract
	CollectibleReference common.AddressLocation // Reference to the collectible NFT contract
}

type Bucket struct {
	CollectibleCount      uint64        // How many collectibles to pick from this bucket
	CollectibleCollection []Collectible // Collection of collectibles to pick from
}

type Pack struct {
	State          PackState          // public
	Salt           PackSalt           // public
	CommitmentHash PackCommitmentHash // public
	Slots          []PackSlot         // private
}

type PackSlot struct {
	State       PackSlotState
	Collectible Collectible
}

type Collectible struct {
	ID cadence.UInt64
}

// Resolve should
// - validate the distribution
// - distribute given collectibles into packs based on given template
// - seal each pack??
// - set the distributions state to resolved
func (dist *Distribution) Resolve() error {
	if dist.State != DistributionStateInit {
		return fmt.Errorf("distribution can not be resolved anymore")
	}

	if err := dist.Validate(); err != nil {
		return fmt.Errorf("distribution validation error: %w", err)
	}

	packCount := int(dist.PackTemplate.PackCount)
	packSlotCount := int(dist.PackTemplate.PackSlotCount())

	// Init packs and their slots
	packs := make([]Pack, packCount)
	for i := range packs {
		packs[i].Slots = make([]PackSlot, packSlotCount)
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

		for i, randomIndex := range r.Perm(countTotal) {
			collectible := bucket.CollectibleCollection[randomIndex]
			slot := PackSlot{Collectible: collectible}
			packIndex := i % packCount
			slotIndex := (i / packCount) + slotBaseIndex
			packs[packIndex].Slots[slotIndex] = slot
		}

		slotBaseIndex += countPerPack
	}

	// Sealing each pack
	for i := range packs {
		if err := packs[i].Seal(); err != nil {
			return fmt.Errorf("error while sealing pack %d: %w", i+1, err)
		}
	}

	dist.Packs = packs
	dist.State = DistributionStateResolved

	return nil
}

func (dist *Distribution) StartSettlement() error {
	if dist.State != DistributionStateResolved {
		return fmt.Errorf("settlement can not be started for distribution")
	}

	dist.State = DistributionStateSettling

	// TODO (latenssi)

	return nil
}

func (dist Distribution) packSlots() []PackSlot {
	res := make([]PackSlot, 0, dist.PackTemplate.PackSlotCount())
	for _, pack := range dist.Packs {
		res = append(res, pack.Slots...)
	}
	return res
}

// ResolvedCollection should publicly present what collectibles got in the distribution
// without revealing in which pack each one resides
func (dist Distribution) ResolvedCollection() []Collectible {
	slots := dist.packSlots()
	res := make([]Collectible, len(slots))
	for i := range slots {
		res[i] = slots[i].Collectible
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(res), func(i, j int) { res[i], res[j] = res[j], res[i] })
	return res
}

// Seal should
// - validate the pack
// - decide on a random salt value
// - calculate the commitment hash for the pack
// - set the pack as sealed
func (p *Pack) Seal() error {
	if p.State != PackStateInit {
		return fmt.Errorf("pack in unexpected state: %d", p.State)
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("pack validation error: %w", err)
	}

	p.Salt = "TODO"
	p.CommitmentHash = "TODO"
	p.State = PackStateSealed

	return nil
}

// PackSlotCount returns the number of slots in each pack
func (pt PackTemplate) PackSlotCount() int {
	res := 0
	for _, bucket := range pt.Buckets {
		res += int(bucket.CollectibleCount)
	}
	return res
}

func (dist Distribution) Validate() error {
	if dist.Issuer == flow.EmptyAddress {
		return fmt.Errorf("issuer must be defined")
	}

	if err := dist.PackTemplate.Validate(); err != nil {
		return fmt.Errorf("error while validating pack template: %w", err)
	}

	return nil
}

func (pt PackTemplate) Validate() error {
	if pt.PackCount == 0 {
		return fmt.Errorf("pack count can not be zero")
	}

	if len(pt.Buckets) == 0 {
		return fmt.Errorf("no slot templates provided")
	}

	if err := ValidateContractReference(pt.PackReference); err != nil {
		return fmt.Errorf("error while validating PackReference: %w", err)
	}

	if err := ValidateContractReference(pt.CollectibleReference); err != nil {
		return fmt.Errorf("error while validating CollectibleReference: %w", err)
	}

	for i, bucket := range pt.Buckets {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("error in slot template %d: %w", i+1, err)
		}

		requiredCount := int(pt.PackCount * bucket.CollectibleCount)
		allocatedCount := len(bucket.CollectibleCollection)
		if requiredCount > allocatedCount {
			return fmt.Errorf(
				"collection too small for slot template %d, required %d got %d",
				i+1, requiredCount, allocatedCount,
			)
		}
	}

	return nil
}

func (bucket Bucket) Validate() error {
	if bucket.CollectibleCount == 0 {
		return fmt.Errorf("collectible count can not be zero")
	}

	if len(bucket.CollectibleCollection) == 0 {
		return fmt.Errorf("empty collection")
	}

	if int(bucket.CollectibleCount) > len(bucket.CollectibleCollection) {
		return fmt.Errorf(
			"collection too small, required %d got %d",
			int(bucket.CollectibleCount), len(bucket.CollectibleCollection),
		)
	}

	return nil
}

func (p Pack) Validate() error {
	if len(p.Slots) == 0 {
		return fmt.Errorf("no slots")
	}

	for i, slot := range p.Slots {
		if slot.Collectible.ID == cadence.NewUInt64(0) {
			return fmt.Errorf("uninitilized collectible in slot %d", i+1)
		}
	}

	return nil
}

func ValidateContractReference(ref common.AddressLocation) error {
	empty, err := common.HexToAddress("0")
	if err != nil {
		return err
	}
	if ref.Address == empty {
		return fmt.Errorf("empty address")
	}
	if ref.Name == "" {
		return fmt.Errorf("empty name")
	}
	return nil
}
