package transactions

import (
	"context"
	"encoding/json"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/flow_helpers"
	"github.com/google/uuid"
	"github.com/onflow/cadence"
	c_json "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
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
	RetryCount    uint                    `gorm:"column:retry_count"`
	TransactionID string                  `gorm:"column:transaction_id"`

	Name      string         `gorm:"column:name"` // Just a way to identify a transaction
	Script    string         `gorm:"column:script"`
	Arguments datatypes.JSON `gorm:"column:arguments"`
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

func (t *StorableTransaction) ArgumentsAsCadence() ([]cadence.Value, error) {
	bytes := [][]byte{}
	if err := json.Unmarshal(t.Arguments, &bytes); err != nil {
		return nil, err
	}

	argsCadence := make([]cadence.Value, len(bytes))
	for i, a := range bytes {
		b, err := c_json.Decode(a)
		if err != nil {
			return nil, err
		}
		argsCadence[i] = b
	}

	return argsCadence, nil
}

// Prepare parses the transaction into a sendable state.
func (t *StorableTransaction) Prepare(ctx context.Context, flowClient *client.Client, account *flow_helpers.Account) (*flow.Transaction, error) {
	args, err := t.ArgumentsAsCadence()
	if err != nil {
		return nil, err
	}

	tx := flow.NewTransaction().
		SetScript([]byte(t.Script)).
		SetGasLimit(9999)

	for _, a := range args {
		if err := tx.AddArgument(a); err != nil {
			return nil, err
		}
	}

	latestBlock, err := flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return nil, err
	}

	tx.SetReferenceBlockID(latestBlock.ID)

	if err := flow_helpers.SignProposeAndPayAs(ctx, flowClient, account, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// HandleResult checks the results of a transaction onchain and updates the
// StorableTransaction accordingly.
func (t *StorableTransaction) HandleResult(ctx context.Context, flowClient *client.Client) error {
	result, err := flowClient.GetTransactionResult(ctx, flow.HexToID(t.TransactionID))
	if err != nil {
		return err
	}

	t.Error = ""

	if result.Error != nil {
		args, err := t.ArgumentsAsCadence()
		if err != nil {
			args = nil
		}

		logWithContext := log.WithFields(log.Fields{
			"name":          t.Name,
			"transactionID": t.TransactionID,
			"error":         result.Error.Error(),
			"arguments":     args,
		})

		t.Error = result.Error.Error()

		if flow_helpers.IsInvalidProposalSeqNumberError(result.Error) {
			t.State = common.TransactionStateRetry
			logWithContext.Info("Invalid proposal key in transaction, retrying later")
		} else {
			t.State = common.TransactionStateFailed
			logWithContext.Warn("Error in transaction")
		}
		return nil
	}

	switch result.Status {
	case flow.TransactionStatusExpired:
		t.State = common.TransactionStateRetry
	case flow.TransactionStatusSealed:
		t.State = common.TransactionStateComplete
	}

	return nil
}
