package flow_helpers

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/crypto/cloudkms"
)

var accounts map[flow.Address]*Account
var accountsLock = &sync.Mutex{} // Making sure our "accounts" var is a singleton
var keyIndexLock = &sync.Mutex{}

var seqNumLock = &sync.Mutex{}
var lastAccountKeySeqNumber map[flow.Address]map[int]uint64
var lastAccountKeyBlock map[flow.Address]map[int]flow.Identifier

const GOOGLE_KMS_KEY_TYPE = "google_kms"

type Account struct {
	Address           flow.Address
	PrivateKey        string
	PrivateKeyType    string
	KeyIndexes        []int
	nextKeyIndexIndex int
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

	new := &Account{
		Address:           address,
		PrivateKey:        privateKey,
		PrivateKeyType:    privateKeyType,
		KeyIndexes:        keyIndexes,
		nextKeyIndexIndex: randomIndex,
	}

	accounts[address] = new

	return new
}

// KeyIndex rotates the given indexes ('KeyIndexes') and returns the next index
// TODO (latenssi): sync over database as this currently only works in a single instance situation?
func (a *Account) KeyIndex() int {
	// NOTE: This won't help if having multiple instances of the PDS service running
	keyIndexLock.Lock()
	defer keyIndexLock.Unlock()

	i := a.KeyIndexes[a.nextKeyIndexIndex]
	a.nextKeyIndexIndex = (a.nextKeyIndexIndex + 1) % len(a.KeyIndexes)

	return i
}

func (a *Account) GetProposalKey(ctx context.Context, flowClient *client.Client) (*flow.AccountKey, error) {
	account, err := flowClient.GetAccount(ctx, a.Address)
	if err != nil {
		return nil, fmt.Errorf("error in flow_helpers.Account.GetProposalKey: %w", err)
	}

	k := account.Keys[a.KeyIndex()]

	if latestBlockHeader, err := flowClient.GetLatestBlockHeader(ctx, true); err == nil {
		k.SequenceNumber = getSequenceNumber(a.Address, k, latestBlockHeader.ID)
	}

	return k, nil
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
func getSequenceNumber(address flow.Address, accountKey *flow.AccountKey, currentBlockID flow.Identifier) uint64 {
	seqNumLock.Lock()
	defer seqNumLock.Unlock()

	// Init lastAccountKeySeqNumber
	if lastAccountKeySeqNumber == nil {
		lastAccountKeySeqNumber = make(map[flow.Address]map[int]uint64)
	}

	if lastAccountKeySeqNumber[address] == nil {
		lastAccountKeySeqNumber[address] = make(map[int]uint64)
	}

	// Init lastAccountKeyBlock
	if lastAccountKeyBlock == nil {
		lastAccountKeyBlock = make(map[flow.Address]map[int]flow.Identifier)
	}

	if lastAccountKeyBlock[address] == nil {
		lastAccountKeyBlock[address] = make(map[int]flow.Identifier)
	}

	useGiven := true

	// Check if operating in the same block as before
	if prevID, ok := lastAccountKeyBlock[address][accountKey.Index]; ok && prevID == currentBlockID {
		// Check if we have a previous number stored and if it is larger or equal to new number
		if prevNumber, ok := lastAccountKeySeqNumber[address][accountKey.Index]; ok && accountKey.SequenceNumber <= prevNumber {
			lastAccountKeySeqNumber[address][accountKey.Index]++
			useGiven = false
		}
	}

	if useGiven {
		lastAccountKeySeqNumber[address][accountKey.Index] = accountKey.SequenceNumber
	}

	lastAccountKeyBlock[address][accountKey.Index] = currentBlockID

	return lastAccountKeySeqNumber[address][accountKey.Index]
}
