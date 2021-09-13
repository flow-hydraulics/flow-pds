package common

type DistributionState uint
type PackState uint
type PackSlotState uint

const (
	DistributionStateInit DistributionState = iota
	DistributionStateCancelled
	DistributionStateResolved
	DistributionStateSettling
	DistributionStateSettled
	DistributionStateConfirmed
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
