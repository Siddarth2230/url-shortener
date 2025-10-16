package idgen

import "context"

// Generator defines the interface for generating short codes.
type Generator interface {
	Generate(ctx context.Context) (string, error)
}
