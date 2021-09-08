package app

import (
	"fmt"
	"math/rand"
	"time"
)

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
	res := make([]PackSlot, 0, dist.PackSlotCount())
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

// PackSlotCount returns the number of slots in each pack
func (dist Distribution) PackSlotCount() int {
	res := 0
	for _, bucket := range dist.PackTemplate.Buckets {
		res += int(bucket.CollectibleCount)
	}
	return res
}
