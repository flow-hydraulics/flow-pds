package app

import (
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
