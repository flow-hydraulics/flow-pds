package flow_helpers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/crypto/cloudkms"
	"github.com/trailofbits/go-mutexasserts"
)

var ErrNoAccountKeyAvailable = errors.New("no account key available")

var accounts map[flow.Address]*Account
var accountsLock = &sync.Mutex{} // Making sure our "accounts" var is a singleton

var seqNumLock = &sync.Mutex{}
var lastAccountKeySeqNumber map[flow.Address]map[int]uint64

const GOOGLE_KMS_KEY_TYPE = "google_kms"

type Account struct {
	Address           flow.Address
	PrivateKey        string
	PrivateKeyType    string
	PKeyIndexes       ProposalKeyIndexes
	nextKeyIndexIndex int
}

type ProposalKeyIndex struct {
	mu    sync.Mutex
	index int
}

type ProposalKeyIndexes []*ProposalKeyIndex

type UnlockKeyFunc func()

func (ii ProposalKeyIndexes) Next() (int, UnlockKeyFunc, error) {
	for _, key := range ii {
		if !mutexasserts.MutexLocked(&key.mu) {
			key.mu.Lock()
			return key.index, key.mu.Unlock, nil
		}
	}
	return 0, nil, ErrNoAccountKeyAvailable
}

// GetAccount either returns an Account from the application wide cache or initiliazes a new Account
func GetAccount(address flow.Address, privateKey, privateKeyType string, keyIndexes []int) *Account {
	accountsLock.Lock()
	defer accountsLock.Unlock()

	if accounts == nil {
		accounts = make(map[flow.Address]*Account, 1)
	}

	if existing, ok := accounts[address]; ok {
		return existing
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

	accounts[address] = new

	return new
}

func (a *Account) GetProposalKey(ctx context.Context, flowClient *client.Client) (*flow.AccountKey, UnlockKeyFunc, error) {
	account, err := flowClient.GetAccount(ctx, a.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("error in flow_helpers.Account.GetProposalKey: %w", err)
	}

	idx, unlock, err := a.PKeyIndexes.Next()
	if err != nil {
		return nil, nil, err
	}

	k := account.Keys[idx]

	k.SequenceNumber = getSequenceNumber(a.Address, k)

	return k, unlock, nil
}

func (a Account) GetSigner() (crypto.Signer, error) {
	// Get Google KMS Signer if using KMS key
	if a.PrivateKeyType == GOOGLE_KMS_KEY_TYPE {
		s, err := getGoogleKMSSigner(a.Address, a.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error in flow_helpers.Account.GetSigner: %w", err)
		}
		return s, nil
	}

	// Default to using local key
	p, err := crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, a.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error in flow_helpers.Account.GetSigner: %w", err)
	}

	return crypto.NewNaiveSigner(p, crypto.SHA3_256), nil
}

func getGoogleKMSSigner(address flow.Address, resourceId string) (crypto.Signer, error) {
	ctx := context.Background()
	c, err := cloudkms.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	k, err := cloudkms.KeyFromResourceID(resourceId)
	if err != nil {
		return nil, err
	}

	s, err := c.SignerForKey(ctx, address, k)

	if err != nil {
		return nil, err
	}

	return s, nil
}

// getSequenceNumber, is a hack around the fact that GetAccount on Flow Client returns
// the latest SequenceNumber on-chain but it might be outdated as we may be
// sending multiple transactions in the current block
// NOTE: This breaks if running in a multi-instance setup
func getSequenceNumber(address flow.Address, accountKey *flow.AccountKey) uint64 {
	seqNumLock.Lock()
	defer seqNumLock.Unlock()

	// Init lastAccountKeySeqNumber
	if lastAccountKeySeqNumber == nil {
		lastAccountKeySeqNumber = make(map[flow.Address]map[int]uint64)
	}

	if lastAccountKeySeqNumber[address] == nil {
		lastAccountKeySeqNumber[address] = make(map[int]uint64)
	}

	// Check if we have a previous sequence number stored
	if _, ok := lastAccountKeySeqNumber[address][accountKey.Index]; !ok {
		lastAccountKeySeqNumber[address][accountKey.Index] = accountKey.SequenceNumber
	} else {
		lastAccountKeySeqNumber[address][accountKey.Index]++
	}

	return lastAccountKeySeqNumber[address][accountKey.Index]
}
