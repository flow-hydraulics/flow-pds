package app

import (
	"fmt"

	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/onflow/flow-go-sdk"
)

func (dist Distribution) Validate() error {
	if dist.DistID == 0 {
		return fmt.Errorf("distId must be defined")
	}

	if flow.Address(dist.Issuer) == flow.EmptyAddress {
		return fmt.Errorf("issuer must be defined")
	}

	if err := dist.PackTemplate.Validate(); err != nil {
		return fmt.Errorf("error while validating pack template: %w", err)
	}

	return nil
}

func (pt PackTemplate) Validate() error {
	if pt.PackCount == 0 {
		return fmt.Errorf("pack count can not be zero")
	}

	if len(pt.Buckets) == 0 {
		return fmt.Errorf("no slot templates provided")
	}

	if err := ValidateContractReference(pt.PackReference); err != nil {
		return fmt.Errorf("error while validating PackReference: %w", err)
	}

	for i, bucket := range pt.Buckets {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("error in slot template %d: %w", i+1, err)
		}

		requiredCount := int(pt.PackCount * bucket.CollectibleCount)
		allocatedCount := len(bucket.CollectibleCollection)
		if requiredCount > allocatedCount {
			return fmt.Errorf(
				"collection too small for slot template %d, required %d got %d",
				i+1, requiredCount, allocatedCount,
			)
		}
	}

	return nil
}

func (bucket Bucket) Validate() error {
	if bucket.CollectibleCount == 0 {
		return fmt.Errorf("collectible count can not be zero")
	}

	if err := ValidateContractReference(bucket.CollectibleReference); err != nil {
		return fmt.Errorf("error while validating CollectibleReference: %w", err)
	}

	if len(bucket.CollectibleCollection) == 0 {
		return fmt.Errorf("empty collection")
	}

	if int(bucket.CollectibleCount) > len(bucket.CollectibleCollection) {
		return fmt.Errorf(
			"collection too small, required %d got %d",
			int(bucket.CollectibleCount), len(bucket.CollectibleCollection),
		)
	}

	return nil
}

func (p Pack) Validate() error {
	if len(p.Collectibles) == 0 {
		return fmt.Errorf("no slots")
	}

	for i, c := range p.Collectibles {
		err := c.Validate()
		if err != nil {
			return fmt.Errorf("error while validating collectible in slot #%d: %w", i+1, err)
		}
	}

	return nil
}

func ValidateContractReference(ref common.AddressLocation) error {
	if ref.Name == "" {
		return fmt.Errorf("empty name")
	}
	if flow.Address(ref.Address) == flow.EmptyAddress {
		return fmt.Errorf("empty address")
	}
	return nil
}

func (c Collectible) Validate() error {
	if err := ValidateContractReference(c.ContractReference); err != nil {
		return fmt.Errorf("error while validating ContractReference: %w", err)
	}

	if c.FlowId == common.FlowID(0) {
		return fmt.Errorf("uninitialized flowID")
	}

	return nil
}
