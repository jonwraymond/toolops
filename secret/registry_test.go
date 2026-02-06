package secret

import (
	"testing"
)

func TestRegistry_RegisterAndCreate(t *testing.T) {
	reg := NewRegistry()

	if err := reg.Register("stub", func(cfg map[string]any) (Provider, error) {
		return &stubProvider{name: "stub"}, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	p, err := reg.Create("stub", map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if p == nil || p.Name() != "stub" {
		t.Fatalf("unexpected provider: %#v", p)
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register("stub", func(cfg map[string]any) (Provider, error) { return &stubProvider{name: "stub"}, nil })

	if err := reg.Register("stub", func(cfg map[string]any) (Provider, error) { return &stubProvider{name: "stub"}, nil }); err == nil {
		t.Fatalf("expected duplicate registration error")
	}
}

func TestRegistry_CreateUnknown(t *testing.T) {
	reg := NewRegistry()
	if _, err := reg.Create("missing", nil); err == nil {
		t.Fatalf("expected error")
	}
}
