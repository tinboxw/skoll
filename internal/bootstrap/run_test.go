package bootstrap

import (
	"context"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestResolveAdminNonceStore(t *testing.T) {
	t.Run("skip when hmac flow disabled", func(t *testing.T) {
		store, closeFn, err := resolveAdminNonceStore("static-token", "secret", "", "redis", "127.0.0.1:6379", "", 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if store != nil || closeFn != nil {
			t.Fatalf("expected nil store and closeFn when hmac flow disabled")
		}
	})

	t.Run("memory mode returns nil store", func(t *testing.T) {
		store, closeFn, err := resolveAdminNonceStore("hmac-sha256", "", "secret", "memory", "", "", 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if store != nil || closeFn != nil {
			t.Fatalf("expected nil store and closeFn for memory mode")
		}
	})

	t.Run("redis mode requires addr", func(t *testing.T) {
		_, _, err := resolveAdminNonceStore("hmac-sha256", "", "secret", "redis", "", "", 0, "")
		if err == nil {
			t.Fatalf("expected error for missing redis addr")
		}
		if !strings.Contains(err.Error(), "requires non-empty addr") {
			t.Fatalf("expected missing addr error, got %v", err)
		}
	})

	t.Run("invalid mode rejected", func(t *testing.T) {
		_, _, err := resolveAdminNonceStore("hmac-sha256", "", "secret", "etcd", "", "", 0, "")
		if err == nil {
			t.Fatalf("expected invalid mode error")
		}
		if !strings.Contains(err.Error(), "unsupported admin auth nonce store mode") {
			t.Fatalf("expected invalid mode error, got %v", err)
		}
	})
}

func TestIsHMACFlowEnabled(t *testing.T) {
	tests := []struct {
		name string
		mode string
		tok  string
		hmac string
		want bool
	}{
		{name: "auto uses hmac secret", mode: "auto", hmac: "secret", want: true},
		{name: "auto prefers token", mode: "auto", tok: "token", hmac: "secret", want: false},
		{name: "explicit hmac missing secret", mode: "hmac-sha256", hmac: "", want: false},
		{name: "explicit hmac with secret", mode: "hmac-sha256", hmac: "secret", want: true},
		{name: "static token mode", mode: "static-token", tok: "token", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isHMACFlowEnabled(tc.mode, tc.tok, tc.hmac); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	tests := []struct {
		name            string
		shutdownTimeout time.Duration
		drainTime       time.Duration
		wantErrContains string
	}{
		{
			name:            "valid defaults",
			shutdownTimeout: 10 * time.Second,
			drainTime:       2 * time.Second,
		},
		{
			name:            "valid no drain",
			shutdownTimeout: 5 * time.Second,
			drainTime:       0,
		},
		{
			name:            "invalid non positive shutdown timeout",
			shutdownTimeout: 0,
			drainTime:       2 * time.Second,
			wantErrContains: "shutdown-timeout",
		},
		{
			name:            "invalid negative drain time",
			shutdownTimeout: 10 * time.Second,
			drainTime:       -1 * time.Second,
			wantErrContains: "drain-time",
		},
		{
			name:            "invalid excessive drain time",
			shutdownTimeout: 10 * time.Second,
			drainTime:       31 * time.Second,
			wantErrContains: "drain-time",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRuntimeConfig(tc.shutdownTimeout, tc.drainTime)
			if tc.wantErrContains == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErrContains, err)
			}
		})
	}
}

func TestDurationFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		fallback     time.Duration
		want         time.Duration
		wantErrParts []string
	}{
		{
			name:     "fallback when empty",
			envValue: "",
			fallback: 2 * time.Second,
			want:     2 * time.Second,
		},
		{
			name:     "parse valid duration",
			envValue: "7s",
			fallback: 2 * time.Second,
			want:     7 * time.Second,
		},
		{
			name:         "invalid duration",
			envValue:     "seven",
			fallback:     2 * time.Second,
			wantErrParts: []string{"SKOLL_TEST_DURATION", "parse failed"},
		},
	}

	const envKey = "SKOLL_TEST_DURATION"
	original, hadOriginal := os.LookupEnv(envKey)
	defer func() {
		if hadOriginal {
			_ = os.Setenv(envKey, original)
			return
		}
		_ = os.Unsetenv(envKey)
	}()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, tc.envValue)
			}

			got, err := durationFromEnv(envKey, tc.fallback)
			if len(tc.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if got != tc.want {
					t.Fatalf("expected %s, got %s", tc.want, got)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			for _, part := range tc.wantErrParts {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("expected error to contain %q, got %v", part, err)
				}
			}
		})
	}
}

type fakeShutdownServer struct {
	calls []string
}

func (f *fakeShutdownServer) SetReady(v bool) {
	f.calls = append(f.calls, "SetReady(false)")
}

func (f *fakeShutdownServer) Shutdown(context.Context) error {
	f.calls = append(f.calls, "Shutdown")
	return nil
}

func TestPerformGracefulStopSequence(t *testing.T) {
	fake := &fakeShutdownServer{}
	sigCh := make(chan os.Signal, 2)

	err := performGracefulStop(fake, 0, 50*time.Millisecond, sigCh)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{"SetReady(false)", "Shutdown"}
	if !reflect.DeepEqual(fake.calls, want) {
		t.Fatalf("unexpected call sequence: got %v, want %v", fake.calls, want)
	}
}

func TestWaitForDrainInterruptedBySecondSignal(t *testing.T) {
	sigCh := make(chan os.Signal, 2)
	sigCh <- syscall.SIGTERM

	start := time.Now()
	interrupted := waitForDrain(50*time.Millisecond, sigCh)
	elapsed := time.Since(start)

	if !interrupted {
		t.Fatalf("expected drain interruption")
	}
	if elapsed >= 50*time.Millisecond {
		t.Fatalf("expected interruption before drain timeout, elapsed=%s", elapsed)
	}
}

func TestFormatRuntimeConfigLog(t *testing.T) {
	line := formatRuntimeConfigLog(":8080", 10*time.Second, 2*time.Second, true, "dev", "auto", "static-token", true, false)

	parts := []string{
		"runtime config",
		"addr=:8080",
		"shutdown-timeout=10s",
		"drain-time=2s",
		"go-admin-enabled=true",
		"go-admin-mode=dev",
		"admin-auth-mode=auto",
		"admin-auth-effective-mode=static-token",
		"admin-auth-enabled=true",
		"admin-auth-allow-static-token-in-prod=false",
	}
	for _, p := range parts {
		if !strings.Contains(line, p) {
			t.Fatalf("expected log line to contain %q, got %q", p, line)
		}
	}
}

func TestValidateAdminAuthPolicy(t *testing.T) {
	tests := []struct {
		name                   string
		goAdminEnabled         bool
		goAdminMode            string
		effectiveAdminAuthMode string
		allowStaticInProd      bool
		wantErr                bool
	}{
		{
			name:                   "allow when go-admin disabled",
			goAdminEnabled:         false,
			goAdminMode:            "prod",
			effectiveAdminAuthMode: "static-token",
			allowStaticInProd:      false,
			wantErr:                false,
		},
		{
			name:                   "allow non-prod mode",
			goAdminEnabled:         true,
			goAdminMode:            "dev",
			effectiveAdminAuthMode: "static-token",
			allowStaticInProd:      false,
			wantErr:                false,
		},
		{
			name:                   "allow stronger auth in prod",
			goAdminEnabled:         true,
			goAdminMode:            "prod",
			effectiveAdminAuthMode: "hmac-sha256",
			allowStaticInProd:      false,
			wantErr:                false,
		},
		{
			name:                   "reject static token in prod by default",
			goAdminEnabled:         true,
			goAdminMode:            "prod",
			effectiveAdminAuthMode: "static-token",
			allowStaticInProd:      false,
			wantErr:                true,
		},
		{
			name:                   "allow static token in prod when explicitly enabled",
			goAdminEnabled:         true,
			goAdminMode:            "prod",
			effectiveAdminAuthMode: "static-token",
			allowStaticInProd:      true,
			wantErr:                false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAdminAuthPolicy(tc.goAdminEnabled, tc.goAdminMode, tc.effectiveAdminAuthMode, tc.allowStaticInProd)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestBoolFromEnv(t *testing.T) {
	const envKey = "SKOLL_TEST_BOOL"
	original, hadOriginal := os.LookupEnv(envKey)
	defer func() {
		if hadOriginal {
			_ = os.Setenv(envKey, original)
			return
		}
		_ = os.Unsetenv(envKey)
	}()

	tests := []struct {
		name     string
		envValue string
		fallback bool
		want     bool
	}{
		{name: "empty uses fallback true", envValue: "", fallback: true, want: true},
		{name: "true value", envValue: "true", fallback: false, want: true},
		{name: "numeric false", envValue: "0", fallback: true, want: false},
		{name: "invalid uses fallback", envValue: "wat", fallback: false, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, tc.envValue)
			}
			got := boolFromEnv(envKey, tc.fallback)
			if got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}
