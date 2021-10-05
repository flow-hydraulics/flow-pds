package transactions

import (
	"context"
	"encoding/json"
	"strings"

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

	State         common.TransactionState `gorm:"column:state;not null;default:null"`
	Error         string                  `gorm:"column:error"`
	RetryCount    uint                    `gorm:"column:retry_count"`
	TransactionID string                  `gorm:"column:transaction_id"`

	Script    string         `gorm:"column:script"`
	Arguments datatypes.JSON `gorm:"column:arguments"`
}

func NewTransaction(script []byte, arguments []cadence.Value) (*StorableTransaction, error) {
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
		Script:    string(script),
		Arguments: argsJSON,
	}

	return &transaction, nil
}

// Prepare parses the transaction into a sendable state.
func (t *StorableTransaction) Prepare() (*flow.Transaction, error) {
	argsBytes := [][]byte{}
	if err := json.Unmarshal(t.Arguments, &argsBytes); err != nil {
		return nil, err
	}

	argsCadence := make([]cadence.Value, len(argsBytes))
	for i, a := range argsBytes {
		b, err := c_json.Decode(a)
		if err != nil {
			return nil, err
		}
		argsCadence[i] = b
	}

	tx := flow.NewTransaction().
		SetScript([]byte(t.Script)).
		SetGasLimit(9999)

	for _, a := range argsCadence {
		if err := tx.AddArgument(a); err != nil {
			return nil, err
		}
	}

	return tx, nil
}

// Send prepares a Flow transaction, signs it and then sends it.
// Updates the TransactionID each time.
func (t *StorableTransaction) Send(ctx context.Context, flowClient *client.Client, account *flow_helpers.Account) error {
	if t.State == common.TransactionStateRetry {
		t.RetryCount++
	}

	tx, err := t.Prepare()
	if err != nil {
		return err
	}

	latestBlock, err := flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		return err
	}

	tx.SetReferenceBlockID(latestBlock.ID)

	if err := flow_helpers.SignProposeAndPayAs(ctx, flowClient, account, tx); err != nil {
		return err
	}

	if err := flowClient.SendTransaction(ctx, *tx); err != nil {
		return err
	}

	// Update TransactionID
	t.TransactionID = tx.ID().Hex()

	// Update state
	t.State = common.TransactionStateSent

	return nil
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
		t.Error = result.Error.Error()
		if strings.Contains(result.Error.Error(), "invalid proposal key") {
			t.State = common.TransactionStateRetry

			log.WithFields(log.Fields{
				"transactionID": t.TransactionID,
			}).Warn("Invalid proposal key in transaction, retrying later")
		} else {
			t.State = common.TransactionStateFailed

			log.WithFields(log.Fields{
				"transactionID": t.TransactionID,
				"error":         result.Error.Error(),
			}).Warn("Error in transaction")
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
