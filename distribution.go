package flowpds

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/onflow/flow-go-sdk"
)

// Someway of determining what NFTs (collectible) we should be interacting with
// If we run a PDS per collectible type then this should not be needed
type CollectibleContractIdentifier string

type CollectibleID uint64
type PackSalt string
type PackCommitmentHash string

type DistributionState int
type PackState int

const (
	DistributionStateInit DistributionState = iota
	DistributionStateEditable
	DistributionStateComplete
)

const (
	PackStateInit PackState = iota
	PackStateSealed
	PackStateRevealed
	PackStateEmpty
)

type Distribution struct {
	Issuer             flow.Address
	State              DistributionState
	PackTemplate       PackTemplate
	Packs              []Pack
	ContractIdentifier CollectibleContractIdentifier
}

type PackTemplate struct {
	PackCount         uint64             // How many packs to create
	PackSlotTemplates []PackSlotTemplate // How to distribute collectibles in a pack
}

type PackSlotTemplate struct {
	CollectibleCount      uint64          // How many collectibles to pick for this slot
	CollectibleCollection []CollectibleID // Collection of collectibles IDs to pick from
}

type Pack struct {
	State          PackState          // public
	Salt           PackSalt           // public
	CommitmentHash PackCommitmentHash // public
	Slots          []PackSlot         // private
}

type PackSlot struct {
	ColletibleID CollectibleID
}

// Resolve should
// - validate the distribution
// - distribute given collectible IDs into packs based on given template
// - set the distribution state to complete
func (dist *Distribution) Resolve() error {
	if dist.State == DistributionStateComplete {
		return fmt.Errorf("distribution has already been resolved")
	}

	if err := dist.Validate(); err != nil {
		return fmt.Errorf("distribution validation error: %w", err)
	}

	packCount := int(dist.PackTemplate.PackCount)
	slotCount := int(dist.PackTemplate.SlotCount())

	packs := make([]Pack, packCount)
	for i := range packs {
		packs[i].Slots = make([]PackSlot, slotCount)
	}

	slotBaseIndex := 0
	for _, pst := range dist.PackTemplate.PackSlotTemplates {
		collectibleCount := int(pst.CollectibleCount)
		totalCollectibleCount := packCount * collectibleCount

		// TODO (latenssi): Is this safe enough?
		r := rand.New(rand.NewSource(time.Now().Unix()))

		for i, randomIndex := range r.Perm(totalCollectibleCount) {
			slot := PackSlot{ColletibleID: pst.CollectibleCollection[randomIndex]}
			packIndex := i % packCount
			slotIndex := (i / packCount) + slotBaseIndex
			packs[packIndex].Slots[slotIndex] = slot
		}

		slotBaseIndex += collectibleCount
	}

	for i := range packs {
		if err := packs[i].Seal(); err != nil {
			return fmt.Errorf("error while sealing pack %d: %w", i+1, err)
		}
	}

	dist.Packs = packs
	dist.State = DistributionStateComplete

	return nil
}

// ResolvedCollection should publicly present what collectibles got in the distribution
// without revealing in which pack each one resides
func (dist Distribution) ResolvedCollection() []CollectibleID {
	// TODO (latenssi)
	return []CollectibleID{}
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

// SlotCount calculates the sum of CollectibleCounts of PackSlotTemplates
func (pt PackTemplate) SlotCount() int {
	res := 0
	for _, pst := range pt.PackSlotTemplates {
		res += int(pst.CollectibleCount)
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

	if len(pt.PackSlotTemplates) == 0 {
		return fmt.Errorf("no slot templates provided")
	}

	for i, pst := range pt.PackSlotTemplates {
		if err := pst.Validate(); err != nil {
			return fmt.Errorf("error in slot template %d: %w", i+1, err)
		}

		requiredCount := int(pt.PackCount * pst.CollectibleCount)
		allocatedCount := len(pst.CollectibleCollection)
		if requiredCount > allocatedCount {
			return fmt.Errorf(
				"collection too small for slot template %d, required %d got %d",
				i+1, requiredCount, allocatedCount,
			)
		}
	}

	return nil
}

func (pst PackSlotTemplate) Validate() error {
	if pst.CollectibleCount == 0 {
		return fmt.Errorf("collectible count can not be zero")
	}

	if len(pst.CollectibleCollection) == 0 {
		return fmt.Errorf("empty collection")
	}

	if int(pst.CollectibleCount) > len(pst.CollectibleCollection) {
		return fmt.Errorf(
			"collection too small, required %d got %d",
			int(pst.CollectibleCount), len(pst.CollectibleCollection),
		)
	}

	return nil
}

func (p Pack) Validate() error {
	if len(p.Slots) == 0 {
		return fmt.Errorf("no slots")
	}

	for i := range p.Slots {
		if p.Slots[i].ColletibleID == CollectibleID(0) {
			return fmt.Errorf("uninitilized collectible in slot %d", i+1)
		}
	}

	return nil
}
