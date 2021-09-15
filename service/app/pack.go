package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/flow-hydraulics/flow-pds/service/common"
)

const SALT_LENGTH = 8
const HASH_DELIM = ","

// SetCommitmentHash should
// - validate the pack
// - decide on a random salt value
// - calculate the commitment hash for the pack
func (p *Pack) SetCommitmentHash() error {
	if !p.Salt.IsEmpty() {
		return fmt.Errorf("salt is already set")
	}

	if !p.CommitmentHash.IsEmpty() {
		return fmt.Errorf("commitmentHash is already set")
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("pack validation error: %w", err)
	}

	salt, err := common.GenerateRandomBytes(SALT_LENGTH)
	if err != nil {
		return err
	}

	p.Salt = salt
	p.CommitmentHash = p.Hash()

	return nil
}

func (p *Pack) Hash() []byte {
	inputs := make([]string, 1+len(p.Collectibles))
	inputs[0] = hex.EncodeToString(p.Salt)
	for i, c := range p.Collectibles {
		inputs[i+1] = c.String()
	}
	h := sha256.Sum256([]byte(strings.Join(inputs, HASH_DELIM)))
	return h[:]
}

// Seal should
// - set the pack as sealed
func (p *Pack) Seal() error {
	if p.State != common.PackStateInit {
		return fmt.Errorf("pack in unexpected state: %d", p.State)
	}

	p.State = common.PackStateSealed

	return nil
}
