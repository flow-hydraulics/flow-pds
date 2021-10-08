package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/flow-hydraulics/flow-pds/service/common"
)

const SALT_LENGTH_IN_BYTES = 32 // 256-bit
const HASH_DELIM = ","

// SetCommitmentHash should
// - validate the pack
// - decide on a random salt value
// - calculate the commitment hash for the pack
func (p *Pack) SetCommitmentHash() error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("pack validation error: %w", err)
	}

	if !p.Salt.IsEmpty() {
		return fmt.Errorf("salt is already set")
	}

	if !p.CommitmentHash.IsEmpty() {
		return fmt.Errorf("commitmentHash is already set")
	}

	salt, err := common.GenerateRandomBytes(SALT_LENGTH_IN_BYTES)
	if err != nil {
		return err
	}

	p.Salt = salt
	p.CommitmentHash = p.Hash()

	return nil
}

// Hash outputs the 'commitmentHash' of a pack.
// It is converting inputs to string and joining them with a delim to make the input more readable.
// This will allow anyone to easily copy paste strings and verify the hash.
// We also use the full reference (address and name) of a collectible to make
// it more difficult to fiddle with the types of collectibles inside a pack.
func (p *Pack) Hash() []byte {
	inputs := make([]string, 1+len(p.Collectibles))
	inputs[0] = hex.EncodeToString(p.Salt)
	for i, c := range p.Collectibles {
		inputs[i+1] = c.HashString()
	}
	input := strings.Join(inputs, HASH_DELIM)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

// Seal should set the FlowID of the pack and set it as sealed
func (p *Pack) Seal(id common.FlowID) error {
	if p.State != common.PackStateInit {
		return fmt.Errorf("pack in unexpected state: %s", p.State)
	}

	if p.FlowID.Valid {
		return fmt.Errorf("pack FlowID already set: %v", id)
	}

	p.FlowID = id
	p.State = common.PackStateSealed

	return nil
}

// RevealRequestHandled should set the pack as "reveal-request-handled"
// given the previous state was correct
func (p *Pack) RevealRequestHandled() error {
	if p.State != common.PackStateSealed {
		return fmt.Errorf("pack in unexpected state: %s", p.State)
	}

	p.State = common.PackStateRevealRequestHandled

	return nil
}

// Reveal should set the pack as "revealed"
// given the previous state was correct
func (p *Pack) Reveal() error {
	if p.State != common.PackStateRevealRequestHandled {
		return fmt.Errorf("pack in unexpected state: %s", p.State)
	}

	p.State = common.PackStateRevealed

	return nil
}

// OpenRequestHandled should set the pack as "open-request-handled"
// given the previous state was correct
func (p *Pack) OpenRequestHandled() error {
	if p.State != common.PackStateRevealed {
		return fmt.Errorf("pack in unexpected state: %s", p.State)
	}

	p.State = common.PackStateOpenRequestHandled

	return nil
}

// Open should set the pack as "opened"
// given the previous state was correct
func (p *Pack) Open() error {
	if p.State != common.PackStateOpenRequestHandled && p.State != common.PackStateRevealed {
		return fmt.Errorf("pack in unexpected state: %s", p.State)
	}

	p.State = common.PackStateOpened

	return nil
}
