package common

type DistributionState uint
type PackState uint
type SettlementState uint
type MintingState uint

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
	PackStateOpened
	PackStateEmpty
)

const (
	SettlementStateStarted SettlementState = iota
	SettlementStateStopped
	SettlementStateDone
)

const (
	MintingStateStarted MintingState = iota
	MintingStateStopped
	MintingStateDone
)
