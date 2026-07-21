package pharmaoa

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const pharmaListSort = "code:asc"

type pharmaListPagination struct {
	Offset int
	Limit  int
}

func parsePharmaListPagination(r *http.Request) (pharmaListPagination, error) {
	values := r.URL.Query()
	offset, err := parseNonNegativeInt(values.Get("offset"), "offset")
	if err != nil {
		return pharmaListPagination{}, err
	}
	if cursor := strings.TrimSpace(values.Get("cursor")); cursor != "" {
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(cursor)
		if decodeErr != nil {
			return pharmaListPagination{}, fmt.Errorf("cursor must be a valid page cursor")
		}
		offset, err = parseNonNegativeInt(string(decoded), "cursor")
		if err != nil {
			return pharmaListPagination{}, fmt.Errorf("cursor must be a valid page cursor")
		}
	}
	limit, err := parseNonNegativeInt(values.Get("limit"), "limit")
	if err != nil {
		return pharmaListPagination{}, err
	}
	offset, limit = normalizePagination(offset, limit)
	return pharmaListPagination{Offset: offset, Limit: limit}, nil
}

func parseNonNegativeInt(raw, name string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return value, nil
}

func writePharmaListPage[T any](w http.ResponseWriter, page pharmaoasvc.ListPage[T], pagination pharmaListPagination) {
	nextOffset := pagination.Offset + len(page.Items)
	hasMore := int64(nextOffset) < page.Total
	nextCursor := ""
	if hasMore {
		nextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(nextOffset)))
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items": page.Items, "offset": pagination.Offset, "limit": pagination.Limit,
		"total": page.Total, "hasMore": hasMore, "nextCursor": nextCursor, "sort": pharmaListSort,
	})
}
