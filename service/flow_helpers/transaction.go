package flow_helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
)

func SignProposeAndPayAs(ctx context.Context, flowClient *client.Client, account *Account, tx *flow.Transaction) error {
	key, err := account.GetProposalKey(ctx, flowClient)
	if err != nil {
		return err
	}

	signer, err := account.GetSigner()
	if err != nil {
		return err
	}

	tx.
		SetProposalKey(account.Address, key.Index, key.SequenceNumber).
		SetPayer(account.Address).
		AddAuthorizer(account.Address).
		SignEnvelope(account.Address, key.Index, signer)

	return nil
}

func PrepareTransaction(arguments []cadence.Value, txScriptPath string) (*flow.Transaction, error) {
	txScript := util.ParseCadenceTemplate(txScriptPath)

	tx := flow.NewTransaction().
		SetScript(txScript).
		SetGasLimit(9999)

	for _, arg := range arguments {
		if err := tx.AddArgument(arg); err != nil {
			return nil, err
		}
	}

	return tx, nil
}

func SendAndWait(ctx context.Context, flowClient *client.Client, timeout time.Duration, tx flow.Transaction) (*flow.TransactionResult, error) {
	if err := flowClient.SendTransaction(ctx, tx); err != nil {
		return nil, err
	}
	return WaitForSeal(ctx, flowClient, timeout, tx.ID())
}

func WaitForSeal(ctx context.Context, flowClient *client.Client, timeout time.Duration, id flow.Identifier) (*flow.TransactionResult, error) {
	var (
		result *flow.TransactionResult
		err    error
	)

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	for {
		result, err = flowClient.GetTransactionResult(ctx, id)
		if err != nil {
			return nil, err
		}

		if result.Error != nil {
			return result, result.Error
		}

		switch result.Status {
		default:
			// Not an interesting state, exit switch and continue loop
		case flow.TransactionStatusExpired:
			// Expired, handle as an error
			return result, fmt.Errorf("transaction expired")
		case flow.TransactionStatusSealed:
			// Sealed, all good
			return result, nil
		}

		time.Sleep(time.Second)
	}
}
