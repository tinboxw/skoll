package goadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	adminsdk "github.com/go-admin-team/go-admin-core/sdk/pkg"
)

const AdminPingPath = "/admin/ping"

type Bootstrap struct {
	enabled bool
	mode    adminsdk.Mode
}

func New(enabled bool, mode string) (*Bootstrap, error) {
	resolvedMode, err := resolveMode(mode)
	if err != nil {
		return nil, err
	}

	return &Bootstrap{
		enabled: enabled,
		mode:    resolvedMode,
	}, nil
}

func (b *Bootstrap) Init(context.Context) error {
	if !b.enabled {
		return nil
	}

	// Minimal integration phase: validate mode mapping and dependency wiring only.
	_ = b.mode
	return nil
}

func (b *Bootstrap) AdminPingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		requestID := adminsdk.GenerateRandomKey20()
		w.Header().Set(adminsdk.TrafficKey, requestID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":      "ok",
			"integration": "go-admin",
			"mode":        string(b.mode),
			"request_id":  requestID,
		})
	}
}

func (b *Bootstrap) Enabled() bool {
	return b.enabled
}

func (b *Bootstrap) Mode() string {
	return string(b.mode)
}

func resolveMode(mode string) (adminsdk.Mode, error) {
	switch adminsdk.Mode(mode) {
	case adminsdk.ModeDev, adminsdk.ModeTest, adminsdk.ModeProd:
		return adminsdk.Mode(mode), nil
	default:
		return "", fmt.Errorf("unsupported go-admin mode %q, valid: dev|test|prod", mode)
	}
}
