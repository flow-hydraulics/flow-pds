package transactions

import (
	"context"
	"encoding/json"
	"strings"

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

// TODO (latenssi): move this to main and use an application wide logger
func init() {
	log.SetLevel(log.InfoLevel)
}

type TransactionState int

const (
	TransactionStateInit = iota
	TransactionStateRetry
	TransactionStateSent
	TransactionStateError
	TransactionStateOk
)

type StorableTransaction struct {
	gorm.Model
	ID uuid.UUID `gorm:"column:id;primary_key;type:uuid;"`

	State         TransactionState `gorm:"column:state"`
	Error         string           `gorm:"column:error"`
	RetryCount    uint             `gorm:"column:retry_count"`
	TransactionID string           `gorm:"column:transaction_id"`

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
		Script:    string(script),
		Arguments: argsJSON,
	}

	return &transaction, nil
}

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

func (t *StorableTransaction) Send(ctx context.Context, flowClient *client.Client, account *flow_helpers.Account) error {
	if t.State == TransactionStateRetry {
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

	t.TransactionID = tx.ID().Hex()
	t.State = TransactionStateSent

	return nil
}

func (t *StorableTransaction) HandleResult(ctx context.Context, flowClient *client.Client) error {
	result, err := flowClient.GetTransactionResult(ctx, flow.HexToID(t.TransactionID))
	if err != nil {
		return err
	}

	t.Error = ""

	if result.Error != nil {
		t.Error = result.Error.Error()
		if strings.Contains(result.Error.Error(), "invalid proposal key") {
			t.State = TransactionStateRetry

			log.WithFields(log.Fields{
				"transactionID": t.TransactionID,
			}).Warn("Invalid proposal key in transaction, retrying later")
		} else {
			t.State = TransactionStateError

			log.WithFields(log.Fields{
				"transactionID": t.TransactionID,
				"error":         result.Error.Error(),
			}).Warn("Error in transaction")
		}
		return nil
	}

	switch result.Status {
	case flow.TransactionStatusExpired:
		t.State = TransactionStateRetry
	case flow.TransactionStatusSealed:
		t.State = TransactionStateOk
	}

	return nil
}
