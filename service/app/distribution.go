package app

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
)

// Resolve should
// - validate the distribution
// - distribute given collectibles into packs based on given template
// - seal each pack??
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

		permutation := r.Perm(len(bucket.CollectibleCollection))

		for i := 0; i < countTotal; i++ {
			randomIndex := permutation[i]
			collectible := bucket.CollectibleCollection[randomIndex]
			slot := PackSlot{CollectibleFlowID: collectible}
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
	dist.State = common.DistributionStateResolved

	return nil
}

// Settle starts the transferring of collectible NFTs to escrow
func (dist *Distribution) Settle() error {
	if dist.State != common.DistributionStateResolved {
		return fmt.Errorf("distribution can not be settled at this state")
	}

	dist.State = common.DistributionStateSettling

	// TODO (latenssi)

	return nil
}

// Mint starts the minting of Pack NFTs
func (dist *Distribution) Mint() error {
	if dist.State != common.DistributionStateSettled {
		return fmt.Errorf("distribution can not start minting at this state")
	}

	dist.State = common.DistributionStateMinting

	// TODO (latenssi)

	return nil
}

func (dist *Distribution) Cancel() error {
	if dist.State == common.DistributionStateComplete {
		return fmt.Errorf("distribution can not be cancelled at this state")
	}

	dist.State = common.DistributionStateCancelled

	// TODO (latenssi)

	return nil
}

func (dist Distribution) packSlots() []PackSlot {
	var slots []PackSlot
	for _, pack := range dist.Packs {
		slots = append(slots, pack.Slots...)
	}
	return slots
}

// ResolvedCollection should publicly present what collectibles got in the distribution
// without revealing in which pack each one resides
func (dist Distribution) ResolvedCollection() common.FlowIDList {
	slots := dist.packSlots()
	res := make([]common.FlowID, len(slots))
	for i := range slots {
		res[i] = slots[i].CollectibleFlowID
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(res), func(i, j int) { res[i], res[j] = res[j], res[i] })
	return res
}

// PackSlotCount returns the number of slots in each pack
func (dist Distribution) PackSlotCount() int {
	res := 0
	for _, bucket := range dist.PackTemplate.Buckets {
		res += int(bucket.CollectibleCount)
	}
	return res
}
