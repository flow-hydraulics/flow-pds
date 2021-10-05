package common

type DistributionState string
type PackState string
type TransactionState string

const (
	DistributionStateInit     DistributionState = "init"
	DistributionStateInvalid  DistributionState = "invalid"
	DistributionStateResolved DistributionState = "resolved"
	DistributionStateSettling DistributionState = "settling"
	DistributionStateSettled  DistributionState = "settled"
	DistributionStateMinting  DistributionState = "minting"
	DistributionStateComplete DistributionState = "complete"
)

const (
	PackStateInit     PackState = "init"
	PackStateSealed   PackState = "sealed"
	PackStateRevealed PackState = "revealed"
	PackStateOpened   PackState = "opened"
	PackStateEmpty    PackState = "empty"
)

const (
	TransactionStateInit     TransactionState = "init"
	TransactionStateRetry    TransactionState = "retry"
	TransactionStateSent     TransactionState = "sent"
	TransactionStateFailed   TransactionState = "failed"
	TransactionStateComplete TransactionState = "complete"
)
