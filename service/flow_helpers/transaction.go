package flow_helpers

import (
	"context"

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

func SendTransactionAs(ctx context.Context, flowClient *client.Client, account *Account, referenceBlock *flow.Block, arguments []cadence.Value, txScriptPath string) error {
	txScript := util.ParseCadenceTemplate(txScriptPath)

	tx := flow.NewTransaction().
		SetScript(txScript).
		SetGasLimit(9999).
		SetReferenceBlockID(referenceBlock.ID)

	for _, arg := range arguments {
		if err := tx.AddArgument(arg); err != nil {
			return err
		}
	}

	if err := SignProposeAndPayAs(ctx, flowClient, account, tx); err != nil {
		return err
	}

	if err := flowClient.SendTransaction(ctx, *tx); err != nil {
		return err
	}

	return nil
}
