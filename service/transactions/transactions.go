package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/google/uuid"
	"github.com/onflow/cadence"
	c_json "github.com/onflow/cadence/encoding/json"
	flow "github.com/onflow/flow-go-sdk"
	flowGrpc "github.com/onflow/flow-go-sdk/access/grpc"
	log "github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// StorableTransaction represents a Flow transaction.
// It stores the script and arguments of a transaction.
type StorableTransaction struct {
	gorm.Model
	ID uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	State         common.TransactionState `gorm:"column:state;not null;default:null;index"`
	Error         string                  `gorm:"column:error"`
	RetryCount    uint                    `gorm:"column:retry_count"` // TODO increment this
	TransactionID string                  `gorm:"column:transaction_id"`

	Name      string         `gorm:"column:name"` // Just a way to identify a transaction
	Script    string         `gorm:"column:script"`
	Arguments datatypes.JSON `gorm:"column:arguments"`

	DistributionID uuid.UUID `gorm:"column:distribution_id;index"` // NOTE: Not a proper foreign key
}

func NewTransaction(name string, script []byte, arguments []cadence.Value) (*StorableTransaction, error) {
	argsBytes := make([][]byte, len(arguments))
	for i, a := range arguments {
		b, err := c_json.Encode(a)
		if err != nil {
			return nil, err
		}
		argsBytes[i] = b
	}

	argsJSON, err := json.Marshal(argsBytes)
	if err != nil {
		return nil, err
	}

	transaction := StorableTransaction{
		State:     common.TransactionStateInit,
		Name:      name,
		Script:    string(script),
		Arguments: argsJSON,
	}

	return &transaction, nil
}

func NewTransactionWithDistributionID(name string, script []byte, arguments []cadence.Value, distributionID uuid.UUID) (*StorableTransaction, error) {
	transaction, err := NewTransaction(name, script, arguments)
	if err != nil {
		return nil, err
	}

	transaction.DistributionID = distributionID

	return transaction, nil
}

func (t *StorableTransaction) ArgumentsAsCadence() ([]cadence.Value, error) {
	bytes := [][]byte{}
	if err := json.Unmarshal(t.Arguments, &bytes); err != nil {
		return nil, err
	}

	argsCadence := make([]cadence.Value, len(bytes))
	for i, a := range bytes {
		b, err := c_json.Decode(nil, a)
		if err != nil {
			return nil, err
		}
		argsCadence[i] = b
	}

	return argsCadence, nil
}

// Prepare parses the transaction into a sendable state.
func (t *StorableTransaction) Prepare(ctx context.Context, flowClient *flowGrpc.BaseClient, account *flow_helpers.Account, gasLimit uint64) (*flow.Transaction, flow_helpers.UnlockKeyFunc, error) {
	args, err := t.ArgumentsAsCadence()
	if err != nil {
		return nil, nil, err
	}

	tx := flow.NewTransaction().
		SetScript([]byte(t.Script)).
		SetGasLimit(gasLimit)

	for _, a := range args {
		if err := tx.AddArgument(a); err != nil {
			return nil, nil, err
		}
	}

	latestBlockHeader, err := flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return nil, nil, err
	}

	tx.SetReferenceBlockID(latestBlockHeader.ID)

	unlock, err := flow_helpers.SignProposeAndPayAs(ctx, flowClient, account, tx)
	if err != nil {
		return nil, unlock, err
	}

	return tx, unlock, nil
}

// HandleResult checks the results of a transaction onchain and updates the
// StorableTransaction accordingly.
func (t *StorableTransaction) HandleResult(ctx context.Context, flowClient *flowGrpc.BaseClient) error {
	logger := log.WithFields(log.Fields{
		"name":           t.Name,
		"transactionID":  t.TransactionID,
		"distributionID": t.DistributionID,
	})

	result, err := flowClient.GetTransactionResult(ctx, flow.HexToID(t.TransactionID))
	if err != nil {
		return err
	}

	t.Error = ""

	if result.Error != nil {
		loggerWithError := logger.WithFields(log.Fields{"error": result.Error.Error()})

		t.Error = result.Error.Error()

		if flow_helpers.IsInvalidProposalSeqNumberError(result.Error) {
			t.State = common.TransactionStateRetry
			// These can be quite numerous so using trace log level here
			loggerWithError.Trace("Invalid sequence number, retrying later")
		} else {
			t.State = common.TransactionStateFailed
			loggerWithError.Warn("Error in transaction")
		}
		return nil
	}

	switch result.Status {
	case flow.TransactionStatusExpired:
		// TODO (latenssi): will we ever get here? if status == expired is result.Error set?
		t.State = common.TransactionStateRetry
	case flow.TransactionStatusSealed:
		logger.Debug("Transaction sealed")
		t.State = common.TransactionStateComplete
	}

	return nil
}

func (t *StorableTransaction) WaitForFinalize(ctx context.Context, flowClient *flowGrpc.BaseClient, pollInterval time.Duration) (*flow.TransactionResult, error) {
	for ctx.Err() == nil {
		result, err := flowClient.GetTransactionResult(ctx, flow.HexToID(t.TransactionID))
		if err != nil {
			return nil, fmt.Errorf("error getting transaction result: %w", err)
		}
		if result.Status == flow.TransactionStatusFinalized || result.Status == flow.TransactionStatusSealed {
			return result, result.Error
		}

		if deadline, hasDeadline := ctx.Deadline(); hasDeadline && deadline.Before(time.Now()) {
			return nil, fmt.Errorf("error getting transaction result within timeout")
		}
		time.Sleep(pollInterval)
	}
	return nil, ctx.Err()
}
