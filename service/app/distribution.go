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

	// TODO (latenssi): Is this safe enough?
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Copy buckets from template and shuffle each bucket
	buckets := make([]Bucket, len(dist.PackTemplate.Buckets))
	copy(buckets, dist.PackTemplate.Buckets)
	for i := range buckets {
		c := buckets[i].CollectibleCollection
		r.Shuffle(len(c), func(i, j int) { c[i], c[j] = c[j], c[i] })
		buckets[i].CollectibleCollection = c
	}

	for slotIndex, templateSlot := range dist.PackTemplate.Slots {
		for packIndex := 0; packIndex < packCount; packIndex++ {
			// Choose a random entry from 'templateSlot.BucketIndexes'
			index := templateSlot.BucketIndexes[r.Intn(len(templateSlot.BucketIndexes))]
			// Pop a collectible from the chosen bucket
			collectible, rest := buckets[index].CollectibleCollection[0], buckets[index].CollectibleCollection[1:]
			// Store the rest of the bucket
			buckets[index].CollectibleCollection = rest
			packs[packIndex].Slots[slotIndex] = PackSlot{Collectible: collectible}
		}
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
	return len(dist.PackTemplate.Slots)
}
