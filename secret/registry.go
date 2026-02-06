package secret

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ProviderFactory creates a Provider from configuration.
type ProviderFactory func(cfg map[string]any) (Provider, error)

// Registry manages provider factories.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ProviderFactory
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]ProviderFactory)}
}

// Register adds a provider factory.
func (r *Registry) Register(name string, factory ProviderFactory) error {
	if strings.TrimSpace(name) == "" || factory == nil {
		return errors.New("invalid provider registration")
	}
	name = strings.TrimSpace(name)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("secret provider %q already registered", name)
	}
	r.providers[name] = factory
	return nil
}

// Create instantiates a provider by name.
func (r *Registry) Create(name string, cfg map[string]any) (Provider, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("provider name is required")
	}

	r.mu.RLock()
	factory, ok := r.providers[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("secret provider %q is not registered", name)
	}

	return factory(cfg)
}

// List returns registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultRegistry is the global registry for secret providers.
var DefaultRegistry = NewRegistry()
