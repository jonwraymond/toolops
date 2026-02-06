package secret

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	name    string
	values  map[string]string
	resolve func(ref string) (string, error)
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Resolve(_ context.Context, ref string) (string, error) {
	if s.resolve != nil {
		return s.resolve(ref)
	}
	if s.values == nil {
		return "", nil
	}
	return s.values[ref], nil
}

func (s *stubProvider) Close() error { return nil }

func TestParseSecretRef(t *testing.T) {
	provider, ref, ok := ParseSecretRef("secretref:stub:alpha")
	if !ok {
		t.Fatalf("expected secretref to parse")
	}
	if provider != "stub" || ref != "alpha" {
		t.Fatalf("unexpected values: %q %q", provider, ref)
	}

	_, _, ok = ParseSecretRef("not-a-secretref")
	if ok {
		t.Fatalf("expected non-secretref to fail")
	}
}

func TestResolver_ResolvesFullSecretRef(t *testing.T) {
	r := NewResolver(true, &stubProvider{name: "stub", values: map[string]string{"alpha": "one"}})

	got, err := r.ResolveValue(context.Background(), "secretref:stub:alpha")
	if err != nil {
		t.Fatalf("ResolveValue() error = %v", err)
	}
	if got != "one" {
		t.Fatalf("ResolveValue() = %q, want %q", got, "one")
	}
}

func TestResolver_ResolvesInlineSecretRef(t *testing.T) {
	r := NewResolver(true, &stubProvider{name: "stub", values: map[string]string{"beta": "two"}})

	got, err := r.ResolveValue(context.Background(), "Bearer secretref:stub:beta")
	if err != nil {
		t.Fatalf("ResolveValue() error = %v", err)
	}
	if got != "Bearer two" {
		t.Fatalf("ResolveValue() = %q, want %q", got, "Bearer two")
	}
}

func TestResolver_StrictEmptyProviderValueErrors(t *testing.T) {
	r := NewResolver(true, &stubProvider{name: "stub", values: map[string]string{"empty": ""}})

	_, err := r.ResolveValue(context.Background(), "secretref:stub:empty")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolver_ResolveMapAndSlice(t *testing.T) {
	r := NewResolver(true, &stubProvider{name: "stub", values: map[string]string{"alpha": "one"}})

	slice, err := r.ResolveSlice(context.Background(), []string{"a", "secretref:stub:alpha"})
	if err != nil {
		t.Fatalf("ResolveSlice() error = %v", err)
	}
	if slice[0] != "a" || slice[1] != "one" {
		t.Fatalf("unexpected slice: %#v", slice)
	}

	m, err := r.ResolveMap(context.Background(), map[string]string{"k": "Bearer secretref:stub:alpha"})
	if err != nil {
		t.Fatalf("ResolveMap() error = %v", err)
	}
	if m["k"] != "Bearer one" {
		t.Fatalf("ResolveMap()[\"k\"] = %q, want %q", m["k"], "Bearer one")
	}
}

func TestResolver_ProviderResolveErrorPropagates(t *testing.T) {
	r := NewResolver(true, &stubProvider{name: "stub", resolve: func(ref string) (string, error) {
		if ref == "boom" {
			return "", errors.New("explode")
		}
		return "ok", nil
	}})

	_, err := r.ResolveValue(context.Background(), "secretref:stub:boom")
	if err == nil {
		t.Fatalf("expected error")
	}
}
