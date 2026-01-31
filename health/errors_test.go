package health

import (
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrCheckFailed", ErrCheckFailed},
		{"ErrCheckTimeout", ErrCheckTimeout},
		{"ErrCheckerNotFound", ErrCheckerNotFound},
		{"ErrNoCheckers", ErrNoCheckers},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}

			if tt.err.Error() == "" {
				t.Errorf("%s has empty message", tt.name)
			}
		})
	}
}
