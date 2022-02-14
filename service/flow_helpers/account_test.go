package flow_helpers

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"

	"github.com/onflow/flow-go-sdk"
)

func TestAccountKeyRotation(t *testing.T) {
	pdsAccount, err := GetAccount(
		flow.HexToAddress("0x1"),
		"",
		"",
		[]int{0, 1, 2},
	)

	assert.NoError(t, err)

	for i := 0; i < 4; i++ {
		index, _, err := pdsAccount.PKeyIndexes.Next()
		if err != nil && i < 3 {
			t.Fatalf("didn't expect an error, got: %s\n", err)
		}
		if index == i && i > 2 {
			t.Fatal("expected KeyIndex to rotate")
		}
	}
}

func TestAccountCaching(t *testing.T) {
	pdsAccount1, err := GetAccount(
		flow.HexToAddress("0x1"),
		"key1",
		"",
		[]int{0, 1, 2},
	)
	assert.NoError(t, err)

	pdsAccount2, err := GetAccount(
		flow.HexToAddress("0x1"),
		"key2",
		"",
		[]int{0, 1, 2},
	)

	assert.NoError(t, err)

	pdsAccount3, err := GetAccount(
		flow.HexToAddress("0x2"),
		"key3",
		"",
		[]int{0, 1, 2},
	)

	assert.NoError(t, err)

	if pdsAccount1.PrivateKey != pdsAccount2.PrivateKey {
		t.Fatal("expected accounts to equal")
	}

	if pdsAccount1.PrivateKey == pdsAccount3.PrivateKey {
		t.Fatal("expected accounts to not equal")
	}
}

func TestAccountAvailableKeys(t *testing.T) {

	var keyIndexes []int
	initialNumKeys := 5

	for i := 0; i < initialNumKeys; i++ {
		keyIndexes = append(keyIndexes, i)
	}

	acct, err := GetAccount(
		flow.HexToAddress("0x1ccc"),
		"testAccountAvailableKeys",
		"",
		keyIndexes,
	)

	assert.NoError(t, err)

	assertAvailableKeys(t, acct.AvailableKeys(), initialNumKeys)

	_, unlockFunc1, _ := acct.PKeyIndexes.Next()
	_, unlockFunc2, _ := acct.PKeyIndexes.Next()

	assertAvailableKeys(t, acct.AvailableKeys(), initialNumKeys-2)

	unlockFunc1()
	unlockFunc2()

	assertAvailableKeys(t, acct.AvailableKeys(), initialNumKeys)
}

func TestAccountAvailableKeysConcurrent(t *testing.T) {
	var keyIndexes []int
	initialNumKeys := 1000

	for i := 0; i < initialNumKeys; i++ {
		keyIndexes = append(keyIndexes, i)
	}

	acct, err := GetAccount(
		flow.HexToAddress("0x1ccd"),
		"testAccountAvailableKeysConcurrent",
		"",
		keyIndexes,
	)

	assert.NoError(t, err)

	assertAvailableKeys(t, acct.AvailableKeys(), initialNumKeys)

	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, count int) {
			defer wg.Done()
			idx, unlockFunc, _ := acct.PKeyIndexes.Next()
			defer unlockFunc()
			fmt.Printf("%d key_idx[%d] - available_keys=[%d]\n", count, idx, acct.AvailableKeys())
		}(&wg, i)
	}
	wg.Wait()
	assertAvailableKeys(t, acct.AvailableKeys(), initialNumKeys)
}

func assertAvailableKeys(t *testing.T, actual int, expected int) {
	if actual != expected {
		t.Fatalf("unexpected available keys, actual: %d, expected:%d", actual, expected)
	}
}
