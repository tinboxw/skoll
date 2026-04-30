package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
)

func upsertConfigHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req upsertConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Key = strings.TrimSpace(req.Key)
		req.Value = strings.TrimSpace(req.Value)
		req.Description = strings.TrimSpace(req.Description)
		if req.Key == "" || req.Value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "key and value are required"})
			return
		}

		respondJSON(w, http.StatusCreated, svc.Set(req.Key, req.Value, req.Description))
	}
}

func upsertConfigsBulkHandler(svc ConfigService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req upsertConfigsBulkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if len(req.Items) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "items are required"})
			return
		}
		for i := range req.Items {
			req.Items[i].Key = strings.TrimSpace(req.Items[i].Key)
			req.Items[i].Value = strings.TrimSpace(req.Items[i].Value)
			req.Items[i].Description = strings.TrimSpace(req.Items[i].Description)
			if req.Items[i].Key == "" || req.Items[i].Value == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid item at index %d", i)})
				return
			}
		}

		out := make([]config.Entry, 0, len(req.Items))
		for _, item := range req.Items {
			out = append(out, svc.Set(item.Key, item.Value, item.Description))
		}
		auditSvc.Append("admin", "configs_bulk_upsert", fmt.Sprintf("count:%d", len(out)))
		respondJSON(w, http.StatusCreated, upsertConfigsBulkResponse{Atomic: true, Count: len(out), Items: out})
	}
}

func listConfigsHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func listConfigsQueryHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size, err := parsePageSizeQuery(r, audit.DefaultPage, audit.DefaultSize, auditQueryMaxSize)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		prefix := strings.TrimSpace(r.URL.Query().Get("key_prefix"))
		keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		sortMode := strings.TrimSpace(r.URL.Query().Get("sort"))

		items := svc.List()
		filtered := make([]config.Entry, 0, len(items))
		for _, item := range items {
			if prefix != "" && !strings.HasPrefix(item.Key, prefix) {
				continue
			}
			if keyword != "" {
				blob := strings.ToLower(item.Key + " " + item.Value + " " + item.Description)
				if !strings.Contains(blob, keyword) {
					continue
				}
			}
			filtered = append(filtered, item)
		}

		if sortMode == "key_desc" {
			sort.Slice(filtered, func(i, j int) bool { return filtered[i].Key > filtered[j].Key })
		}

		pageItems, total := paginateSlice(filtered, page, size)
		respondJSON(w, http.StatusOK, configQueryResponse{
			Items:   pageItems,
			Page:    page,
			Size:    size,
			Total:   total,
			HasNext: page*size < total,
		})
	}
}

func getConfigHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, err := parsePathString(r, "key")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		item, err := svc.Get(key)
		if err != nil {
			if err == config.ErrConfigNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, item)
	}
}

func createDictionaryHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createDictionaryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Type = strings.TrimSpace(req.Type)
		req.Label = strings.TrimSpace(req.Label)
		req.Value = strings.TrimSpace(req.Value)
		if req.Type == "" || req.Label == "" || req.Value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "type, label and value are required"})
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		respondJSON(w, http.StatusCreated, svc.Create(req.Type, req.Label, req.Value, req.Sort, enabled))
	}
}

func createDictionariesBulkHandler(svc DictionaryService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createDictionariesBulkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if len(req.Items) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "items are required"})
			return
		}
		for i := range req.Items {
			req.Items[i].Type = strings.TrimSpace(req.Items[i].Type)
			req.Items[i].Label = strings.TrimSpace(req.Items[i].Label)
			req.Items[i].Value = strings.TrimSpace(req.Items[i].Value)
			if req.Items[i].Type == "" || req.Items[i].Label == "" || req.Items[i].Value == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid item at index %d", i)})
				return
			}
		}

		out := make([]dictionary.Item, 0, len(req.Items))
		for _, item := range req.Items {
			enabled := true
			if item.Enabled != nil {
				enabled = *item.Enabled
			}
			out = append(out, svc.Create(item.Type, item.Label, item.Value, item.Sort, enabled))
		}
		auditSvc.Append("admin", "dictionaries_bulk_create", fmt.Sprintf("count:%d", len(out)))
		respondJSON(w, http.StatusCreated, createDictionariesBulkResponse{Atomic: true, Count: len(out), Items: out})
	}
}

func listDictionariesHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemType := strings.TrimSpace(r.URL.Query().Get("type"))
		if itemType == "" {
			respondJSON(w, http.StatusOK, svc.List())
			return
		}
		respondJSON(w, http.StatusOK, svc.ListByType(itemType))
	}
}

func listDictionariesQueryHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size, err := parsePageSizeQuery(r, audit.DefaultPage, audit.DefaultSize, auditQueryMaxSize)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		itemType := strings.TrimSpace(r.URL.Query().Get("type"))
		keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		enabledRaw := strings.TrimSpace(r.URL.Query().Get("enabled"))
		enabledFilter := false
		enabledValue := false
		if enabledRaw != "" {
			enabledFilter = true
			parsed, parseErr := strconv.ParseBool(enabledRaw)
			if parseErr != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "enabled must be true or false"})
				return
			}
			enabledValue = parsed
		}

		items := svc.List()
		filtered := make([]dictionary.Item, 0, len(items))
		for _, item := range items {
			if itemType != "" && item.Type != itemType {
				continue
			}
			if enabledFilter && item.Enabled != enabledValue {
				continue
			}
			if keyword != "" {
				blob := strings.ToLower(item.Type + " " + item.Label + " " + item.Value)
				if !strings.Contains(blob, keyword) {
					continue
				}
			}
			filtered = append(filtered, item)
		}

		pageItems, total := paginateSlice(filtered, page, size)
		respondJSON(w, http.StatusOK, dictionaryQueryResponse{
			Items:   pageItems,
			Page:    page,
			Size:    size,
			Total:   total,
			HasNext: page*size < total,
		})
	}
}

func getDictionaryHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		item, err := svc.Get(id)
		if err != nil {
			if err == dictionary.ErrItemNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, item)
	}
}
