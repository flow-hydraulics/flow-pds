package app

import (
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/common"
)

// Seal should
// - validate the pack
// - decide on a random salt value
// - calculate the commitment hash for the pack
// - set the pack as sealed
func (p *Pack) Seal() error {
	if p.State != common.PackStateInit {
		return fmt.Errorf("pack in unexpected state: %d", p.State)
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("pack validation error: %w", err)
	}

	p.Salt = "TODO"
	p.CommitmentHash = "TODO"
	p.State = common.PackStateSealed

	return nil
}
