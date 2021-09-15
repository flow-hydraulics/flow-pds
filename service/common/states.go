package common

type DistributionState uint
type PackState uint

const (
	DistributionStateInit DistributionState = iota
	DistributionStateCancelled
	DistributionStateResolved
	DistributionStateSettling
	DistributionStateSettled
	DistributionStateMinting
	DistributionStateComplete
)

const (
	PackStateInit PackState = iota
	PackStateSealed
	PackStateRevealed
	PackStateEmpty
)
