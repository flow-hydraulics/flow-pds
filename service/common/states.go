package common

type DistributionState uint
type PackState uint
type SettlementState uint

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

const (
	SettlementStateStarted SettlementState = iota
	SettlementStateStopped
	SettlementStateDone
)
