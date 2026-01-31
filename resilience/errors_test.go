package resilience

import (
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrCircuitOpen", ErrCircuitOpen},
		{"ErrMaxRetriesExceeded", ErrMaxRetriesExceeded},
		{"ErrRateLimitExceeded", ErrRateLimitExceeded},
		{"ErrBulkheadFull", ErrBulkheadFull},
		{"ErrTimeout", ErrTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}

			// Check error message is not empty
			if tt.err.Error() == "" {
				t.Errorf("%s has empty message", tt.name)
			}
		})
	}
}
