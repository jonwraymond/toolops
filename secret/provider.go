package secret

import "context"

// Provider resolves secrets by reference string.
//
// Implementations must be safe for concurrent use and must not log secret values.
type Provider interface {
	Name() string
	Resolve(ctx context.Context, ref string) (string, error)
	Close() error
}
