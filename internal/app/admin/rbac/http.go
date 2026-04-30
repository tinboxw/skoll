package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	var id int64
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}
		id = id*10 + int64(ch-'0')
	}
	if id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
