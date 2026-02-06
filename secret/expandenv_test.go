package secret

import (
	"strings"
	"testing"
)

func TestExpandEnvStrict_MissingVarErrors(t *testing.T) {
	t.Setenv("PRESENT", "ok")

	_, err := ExpandEnvStrict("a=${PRESENT} b=${MISSING}")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("expected missing var name in error, got: %v", err)
	}
}

func TestExpandEnvStrict_DollarEscape(t *testing.T) {
	t.Setenv("X", "y")

	out, err := ExpandEnvStrict("$$${X}")
	if err != nil {
		t.Fatalf("ExpandEnvStrict() error = %v", err)
	}
	if out != "$y" {
		t.Fatalf("ExpandEnvStrict() = %q, want %q", out, "$y")
	}
}
