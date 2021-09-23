package flow_helpers

import (
	"context"
	"fmt"
	"sync"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
)

var accounts map[flow.Address]*Account
var accountsLock = &sync.Mutex{} // Making sure our "accounts" var is a singleton
var keyIndexLock = &sync.Mutex{}

type Account struct {
	Address           flow.Address
	PrivateKeyInHex   string
	KeyIndexes        []int
	nextKeyIndexIndex int
}

// GetAccount either returns an Account from the application wide cache or initiliazes a new Account
func GetAccount(address flow.Address, privateKeyInHex string, keyIndexes []int) *Account {
	accountsLock.Lock()
	defer accountsLock.Unlock()

	if accounts == nil {
		accounts = make(map[flow.Address]*Account, 1)
	}

	if existing, ok := accounts[address]; ok {
		return existing
	}

	new := &Account{
		Address:         address,
		PrivateKeyInHex: privateKeyInHex,
		KeyIndexes:      keyIndexes,
	}

	accounts[address] = new

	return new
}

func (a *Account) KeyIndex() int {
	// NOTE: This won't help if having multiple instances of the PDS service running
	keyIndexLock.Lock()
	defer keyIndexLock.Unlock()

	i := a.KeyIndexes[a.nextKeyIndexIndex]
	a.nextKeyIndexIndex = (a.nextKeyIndexIndex + 1) % len(a.KeyIndexes)

	return i
}

func (a Account) GetProposalKey(ctx context.Context, flowClient *client.Client) (*flow.AccountKey, error) {
	account, err := flowClient.GetAccount(ctx, a.Address)
	k := account.Keys[a.KeyIndex()]
	if err != nil {
		return nil, fmt.Errorf("error in flow_helpers.Account.GetProposalKey: %w", err)
	}
	return k, nil
}

func (a Account) GetSigner() (crypto.Signer, error) {
	p, err := crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, a.PrivateKeyInHex)
	if err != nil {
		return nil, fmt.Errorf("error in flow_helpers.Account.GetSigner: %w", err)
	}
	return crypto.NewNaiveSigner(p, crypto.SHA3_256), nil
}
