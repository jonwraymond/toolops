package secret

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Resolver resolves secret references using registered providers.
//
// Values with the prefix "secretref:" are resolved via providers.
// Other values are returned after strict environment expansion.
type Resolver struct {
	providers map[string]Provider
	strict    bool
}

// NewResolver creates a resolver.
func NewResolver(strict bool, providers ...Provider) *Resolver {
	r := &Resolver{
		providers: make(map[string]Provider),
		strict:    strict,
	}
	for _, p := range providers {
		if p == nil {
			continue
		}
		r.providers[p.Name()] = p
	}
	return r
}

// Register registers a provider with the resolver.
func (r *Resolver) Register(provider Provider) {
	if r == nil || provider == nil {
		return
	}
	if r.providers == nil {
		r.providers = make(map[string]Provider)
	}
	r.providers[provider.Name()] = provider
}

// ResolveValue resolves environment variables and secret refs in value.
func (r *Resolver) ResolveValue(ctx context.Context, value string) (string, error) {
	if r == nil {
		expanded, err := ExpandEnvStrict(value)
		if err != nil {
			return "", err
		}
		return expanded, nil
	}

	expanded, err := ExpandEnvStrict(value)
	if err != nil {
		return "", err
	}

	if providerName, ref, ok := ParseSecretRef(expanded); ok {
		return r.resolveSingle(ctx, providerName, ref)
	}
	return r.resolveInline(ctx, expanded)
}

// ResolveSlice resolves each value in values.
func (r *Resolver) ResolveSlice(ctx context.Context, values []string) ([]string, error) {
	resolved := make([]string, len(values))
	for i, v := range values {
		out, err := r.ResolveValue(ctx, v)
		if err != nil {
			return nil, err
		}
		resolved[i] = out
	}
	return resolved, nil
}

// ResolveMap resolves each string value in input.
func (r *Resolver) ResolveMap(ctx context.Context, input map[string]string) (map[string]string, error) {
	if input == nil {
		return nil, nil
	}
	out := make(map[string]string, len(input))
	for k, v := range input {
		resolved, err := r.ResolveValue(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", k, err)
		}
		out[k] = resolved
	}
	return out, nil
}

// ParseSecretRef parses a full secret reference of the form:
//
//	secretref:<provider>:<ref>
func ParseSecretRef(value string) (provider string, ref string, ok bool) {
	const prefix = "secretref:"
	if !strings.HasPrefix(value, prefix) {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(value, prefix), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (r *Resolver) resolveSingle(ctx context.Context, providerName string, ref string) (string, error) {
	if strings.TrimSpace(providerName) == "" {
		return "", errors.New("secret provider name is required")
	}
	if strings.TrimSpace(ref) == "" {
		return "", errors.New("secret ref is required")
	}
	provider, ok := r.providers[providerName]
	if !ok || provider == nil {
		return "", fmt.Errorf("secret provider %q is not registered", providerName)
	}
	resolved, err := provider.Resolve(ctx, ref)
	if err != nil {
		return "", err
	}
	if r.strict && resolved == "" {
		return "", fmt.Errorf("secret provider %q returned empty value", providerName)
	}
	return resolved, nil
}

var inlineSecretRefPattern = regexp.MustCompile(`secretref:([^:\s]+):([^\s]+)`) // provider:ref

func (r *Resolver) resolveInline(ctx context.Context, value string) (string, error) {
	matches := inlineSecretRefPattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	out := value
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]

		// Match indexes are stable because we replace from end to start.
		providerName := out[match[2]:match[3]]
		ref := out[match[4]:match[5]]

		resolved, err := r.resolveSingle(ctx, providerName, ref)
		if err != nil {
			return "", err
		}

		out = out[:match[0]] + resolved + out[match[1]:]
	}
	return out, nil
}
