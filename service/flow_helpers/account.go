package flow_helpers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	flow "github.com/onflow/flow-go-sdk"
	flowGrpc "github.com/onflow/flow-go-sdk/access/grpc"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/crypto/cloudkms"
	mutexasserts "github.com/trailofbits/go-mutexasserts"
)

var ErrNoAccountKeyAvailable = errors.New("no account key available")

var accounts map[flow.Address]*Account
var accountsLock = &sync.Mutex{} // Making sure our "accounts" var is a singleton

var availableKeysLock = &sync.Mutex{}

const GOOGLE_KMS_KEY_TYPE = "google_kms"

type Account struct {
	Address           flow.Address
	PrivateKey        string
	PrivateKeyType    string
	PKeyIndexes       ProposalKeyIndexes
	nextKeyIndexIndex int
	//kmsSigner         crypto.Signer
	kmsClient *cloudkms.Client
}

type ProposalKeyIndex struct {
	index int
	mu    sync.Mutex
}

type ProposalKeyIndexes []*ProposalKeyIndex

type UnlockKeyFunc func()

var EmptyUnlockKey UnlockKeyFunc = func() {}

func (ii ProposalKeyIndexes) Next() (int, UnlockKeyFunc, error) {
	for _, key := range ii {
		if !mutexasserts.MutexLocked(&key.mu) {
			key.mu.Lock()
			// Use Once here so multiple calls to unlock, won't unlock this key
			// if it is already given to another caller
			var once sync.Once
			unlock := func() {
				once.Do(key.mu.Unlock)
			}
			return key.index, unlock, nil
		}
	}
	return -1, EmptyUnlockKey, ErrNoAccountKeyAvailable
}

// GetAccount either returns an Account from the application wide cache or initiliazes a new Account
func GetAccount(address flow.Address, privateKey, privateKeyType string, keyIndexes []int) (*Account, error) {
	accountsLock.Lock()
	defer accountsLock.Unlock()

	if accounts == nil {
		accounts = make(map[flow.Address]*Account, 1)
	}

	if existing, ok := accounts[address]; ok {
		return existing, nil
	}

	// Pick a random index to start from
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(keyIndexes))

	// TODO(nanuuki): Check KMS key exists, if using KMS key

	pKeyIndexes := make(ProposalKeyIndexes, len(keyIndexes))
	for i, idx := range keyIndexes {
		pKeyIndexes[i] = &ProposalKeyIndex{index: idx}
	}

	new := &Account{
		Address:           address,
		PrivateKey:        privateKey,
		PrivateKeyType:    privateKeyType,
		PKeyIndexes:       pKeyIndexes,
		nextKeyIndexIndex: randomIndex,
	}

	if privateKeyType == GOOGLE_KMS_KEY_TYPE {
		c, err := getGoogleKMSClient(context.Background())
		if err != nil {
			return nil, err
		}
		new.kmsClient = c
	}

	accounts[address] = new

	return new, nil
}

func (a *Account) GetProposalKey(ctx context.Context, flowClient *flowGrpc.BaseClient) (*flow.AccountKey, UnlockKeyFunc, error) {
	account, err := flowClient.GetAccount(ctx, a.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("error in flow_helpers.Account.GetProposalKey: %w", err)
	}

	idx, unlock, err := a.PKeyIndexes.Next()
	if err != nil {
		return nil, unlock, err
	}

	k := account.Keys[idx]

	k.SequenceNumber = getSequenceNumber(a.Address, k)

	return k, unlock, nil
}

func (a Account) GetSigner() (crypto.Signer, error) {
	// Get Google KMS Signer if using KMS key
	if a.PrivateKeyType == GOOGLE_KMS_KEY_TYPE {
		signer, err := getGoogleKMSSignerFromClient(context.Background(), a.kmsClient, a.PrivateKey)
		if err != nil {
			return nil, err
		}
		return signer, nil
	}

	// Default to using local key
	p, err := crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, a.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error in flow_helpers.Account.GetSigner: %w", err)
	}

	signer, err := crypto.NewNaiveSigner(p, crypto.SHA3_256)
	if err != nil {
		return nil, fmt.Errorf("error in flow_helpers.Account.GetSigner.NewNaiveSigner: %w", err)
	}
	return signer, nil
}

func (a *Account) AvailableKeys() int {
	availableKeysLock.Lock()
	defer availableKeysLock.Unlock()
	var numAvailableKeys int
	for _, key := range a.PKeyIndexes {
		if !mutexasserts.MutexLocked(&key.mu) {
			numAvailableKeys++
		}
	}
	return numAvailableKeys
}

func getGoogleKMSClient(ctx context.Context) (*cloudkms.Client, error) {
	c, err := cloudkms.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func getGoogleKMSSignerFromClient(ctx context.Context, client *cloudkms.Client, resourceId string) (crypto.Signer, error) {
	k, err := cloudkms.KeyFromResourceID(resourceId)
	if err != nil {
		return nil, err
	}

	s, err := client.SignerForKey(ctx, k)

	if err != nil {
		return nil, err
	}

	return s, nil
}

//func getGoogleKMSSigner(address flow.Address, resourceId string) (crypto.Signer, error) {
//	ctx := context.Background()
//	c, err := cloudkms.NewClient(ctx)
//	if err != nil {
//		return nil, err
//	}
//
//	k, err := cloudkms.KeyFromResourceID(resourceId)
//	if err != nil {
//		return nil, err
//	}
//
//	s, err := c.SignerForKey(ctx, address, k)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return s, nil
//}

// getSequenceNumber returns the sequence number to use for sending transactions.
func getSequenceNumber(address flow.Address, accountKey *flow.AccountKey) uint64 {
	return accountKey.SequenceNumber
}
