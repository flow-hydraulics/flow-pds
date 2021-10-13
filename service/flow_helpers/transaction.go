package flow_helpers

import (
	"context"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
)

func SignProposeAndPayAs(ctx context.Context, flowClient *client.Client, account *Account, tx *flow.Transaction) error {
	key, err := account.GetProposalKey(ctx, flowClient, tx.ReferenceBlockID)
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
