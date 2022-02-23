package common

type DistributionState string
type PackState string
type TransactionState string

const (
	DistributionStateInit     DistributionState = "init"
	DistributionStateInvalid  DistributionState = "invalid"
	DistributionStateResolved DistributionState = "resolved"
	DistributionStateSetup    DistributionState = "setup"
	DistributionStateSettling DistributionState = "settling"
	DistributionStateSettled  DistributionState = "settled"
	DistributionStateMinting  DistributionState = "minting"
	DistributionStateComplete DistributionState = "complete"
)

const (
	PackStateInit                 PackState = "init"
	PackStateSealed               PackState = "sealed"
	PackStateRevealRequestHandled PackState = "reveal-request-handled"
	PackStateRevealed             PackState = "revealed"
	PackStateOpenRequestHandled   PackState = "open-request-handled"
	PackStateOpened               PackState = "opened"
	PackStateEmpty                PackState = "empty"
)

const (
	TransactionStateInit     TransactionState = "init"
	TransactionStateRetry    TransactionState = "retry"
	TransactionStateSent     TransactionState = "sent"
	TransactionStateFailed   TransactionState = "failed"
	TransactionStateComplete TransactionState = "complete"
)

const TransactionRPCErrorString string = "rpc error"
