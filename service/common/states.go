package common

type DistributionState uint
type PackState uint
type SettlementState uint
type MintingState uint
type TransactionState int

// TODO (latenssi): represent states as strings (instead of integers) to allow
// flexibility in database structure?

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

const (
	TransactionStateInit = iota
	TransactionStateRetry
	TransactionStateSent
	TransactionStateFailed
	TransactionStateOk
)
