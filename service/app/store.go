package app

import "github.com/google/uuid"

type Store interface {
	// Insert distribution
	InsertDistribution(*Distribution) error

	// Update distribution
	UpdateDistribution(*Distribution) error

	// Remove distribution
	RemoveDistribution(*Distribution) error

	// List distributions
	ListDistributions(ListOptions) ([]Distribution, error)

	// Get distribution
	GetDistribution(id uuid.UUID) (*Distribution, error)
}

type ListOptions struct {
	Limit  int
	Offset int
}

const DefaultLimit = 1000

func ParseListOptions(limit, offset int) ListOptions {
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < 0 {
		limit = -1
		offset = 0
	}
	if offset < 0 {
		offset = 0
	}
	return ListOptions{Limit: limit, Offset: offset}
}
