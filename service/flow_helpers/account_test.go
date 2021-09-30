package flow_helpers

import (
	"testing"

	"github.com/onflow/flow-go-sdk"
)

func TestAccountKeyRotation(t *testing.T) {
	pdsAccount := GetAccount(
		flow.HexToAddress("0x1"),
		"",
		"",
		[]int{0, 1, 2},
	)

	for i := 0; i < 4; i++ {
		index := pdsAccount.KeyIndex()
		if index == i && i > 2 {
			t.Fatal("expected KeyIndex to rotate")
		}
	}
}

func TestAccountCaching(t *testing.T) {
	pdsAccount1 := GetAccount(
		flow.HexToAddress("0x1"),
		"key1",
		"",
		[]int{0, 1, 2},
	)

	pdsAccount2 := GetAccount(
		flow.HexToAddress("0x1"),
		"key2",
		"",
		[]int{0, 1, 2},
	)

	pdsAccount3 := GetAccount(
		flow.HexToAddress("0x2"),
		"key3",
		"",
		[]int{0, 1, 2},
	)

	if pdsAccount1.PrivateKey != pdsAccount2.PrivateKey {
		t.Fatal("expected accounts to equal")
	}

	if pdsAccount1.PrivateKey == pdsAccount3.PrivateKey {
		t.Fatal("expected accounts to not equal")
	}
}
