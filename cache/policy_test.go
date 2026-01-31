package cache

import (
	"testing"
	"time"
)

func TestPolicy_DefaultTTL(t *testing.T) {
	p := Policy{
		DefaultTTL: 5 * time.Minute,
		MaxTTL:     10 * time.Minute,
	}

	got := p.EffectiveTTL(0)
	if got != 5*time.Minute {
		t.Errorf("EffectiveTTL(0) = %v, want %v", got, 5*time.Minute)
	}
}

func TestPolicy_OverrideTTL(t *testing.T) {
	p := Policy{
		DefaultTTL: 5 * time.Minute,
		MaxTTL:     10 * time.Minute,
	}

	got := p.EffectiveTTL(3 * time.Minute)
	if got != 3*time.Minute {
		t.Errorf("EffectiveTTL(3m) = %v, want %v", got, 3*time.Minute)
	}
}

func TestPolicy_MaxTTLClamping(t *testing.T) {
	p := Policy{
		DefaultTTL: 5 * time.Minute,
		MaxTTL:     10 * time.Minute,
	}

	got := p.EffectiveTTL(15 * time.Minute)
	if got != 10*time.Minute {
		t.Errorf("EffectiveTTL(15m) = %v, want %v (clamped to MaxTTL)", got, 10*time.Minute)
	}
}

func TestPolicy_DisabledCaching(t *testing.T) {
	p := Policy{
		DefaultTTL: 0,
		MaxTTL:     10 * time.Minute,
	}

	got := p.EffectiveTTL(0)
	if got != 0 {
		t.Errorf("EffectiveTTL(0) with DefaultTTL=0 = %v, want 0", got)
	}

	if p.ShouldCache() {
		t.Error("ShouldCache() = true, want false when DefaultTTL=0")
	}
}

func TestPolicy_OverrideEnablesCaching(t *testing.T) {
	p := Policy{
		DefaultTTL: 0,
		MaxTTL:     10 * time.Minute,
	}

	got := p.EffectiveTTL(5 * time.Minute)
	if got != 5*time.Minute {
		t.Errorf("EffectiveTTL(5m) with DefaultTTL=0 = %v, want %v", got, 5*time.Minute)
	}
}

func TestPolicy_DefaultPolicy(t *testing.T) {
	p := DefaultPolicy()

	if p.DefaultTTL != 5*time.Minute {
		t.Errorf("DefaultPolicy().DefaultTTL = %v, want %v", p.DefaultTTL, 5*time.Minute)
	}
	if p.MaxTTL != 1*time.Hour {
		t.Errorf("DefaultPolicy().MaxTTL = %v, want %v", p.MaxTTL, 1*time.Hour)
	}
	if p.AllowUnsafe {
		t.Error("DefaultPolicy().AllowUnsafe = true, want false")
	}
}

func TestPolicy_NoCachePolicy(t *testing.T) {
	p := NoCachePolicy()

	if p.DefaultTTL != 0 {
		t.Errorf("NoCachePolicy().DefaultTTL = %v, want 0", p.DefaultTTL)
	}
	if p.MaxTTL != 0 {
		t.Errorf("NoCachePolicy().MaxTTL = %v, want 0", p.MaxTTL)
	}
	if p.ShouldCache() {
		t.Error("NoCachePolicy().ShouldCache() = true, want false")
	}
}

func TestPolicy_TTLMatrix(t *testing.T) {
	tests := []struct {
		name       string
		defaultTTL time.Duration
		maxTTL     time.Duration
		override   time.Duration
		want       time.Duration
	}{
		{
			name:       "no override uses default",
			defaultTTL: 5 * time.Minute,
			maxTTL:     10 * time.Minute,
			override:   0,
			want:       5 * time.Minute,
		},
		{
			name:       "override within max",
			defaultTTL: 5 * time.Minute,
			maxTTL:     10 * time.Minute,
			override:   7 * time.Minute,
			want:       7 * time.Minute,
		},
		{
			name:       "override exceeds max, clamped",
			defaultTTL: 5 * time.Minute,
			maxTTL:     10 * time.Minute,
			override:   20 * time.Minute,
			want:       10 * time.Minute,
		},
		{
			name:       "default exceeds max, clamped",
			defaultTTL: 15 * time.Minute,
			maxTTL:     10 * time.Minute,
			override:   0,
			want:       10 * time.Minute,
		},
		{
			name:       "no max TTL, override used as-is",
			defaultTTL: 5 * time.Minute,
			maxTTL:     0,
			override:   1 * time.Hour,
			want:       1 * time.Hour,
		},
		{
			name:       "no max TTL, default used as-is",
			defaultTTL: 30 * time.Minute,
			maxTTL:     0,
			override:   0,
			want:       30 * time.Minute,
		},
		{
			name:       "all zeros means no caching",
			defaultTTL: 0,
			maxTTL:     0,
			override:   0,
			want:       0,
		},
		{
			name:       "override enables caching when default is zero",
			defaultTTL: 0,
			maxTTL:     10 * time.Minute,
			override:   3 * time.Minute,
			want:       3 * time.Minute,
		},
		{
			name:       "override enables caching, clamped by max",
			defaultTTL: 0,
			maxTTL:     5 * time.Minute,
			override:   10 * time.Minute,
			want:       5 * time.Minute,
		},
		{
			name:       "negative override treated as zero (use default)",
			defaultTTL: 5 * time.Minute,
			maxTTL:     10 * time.Minute,
			override:   -1 * time.Minute,
			want:       5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Policy{
				DefaultTTL: tt.defaultTTL,
				MaxTTL:     tt.maxTTL,
			}
			got := p.EffectiveTTL(tt.override)
			if got != tt.want {
				t.Errorf("EffectiveTTL(%v) = %v, want %v", tt.override, got, tt.want)
			}
		})
	}
}

func TestPolicy_ShouldCache(t *testing.T) {
	tests := []struct {
		name       string
		defaultTTL time.Duration
		want       bool
	}{
		{"positive default enables caching", 5 * time.Minute, true},
		{"zero default disables caching", 0, false},
		{"negative default disables caching", -1 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Policy{DefaultTTL: tt.defaultTTL}
			if got := p.ShouldCache(); got != tt.want {
				t.Errorf("ShouldCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
