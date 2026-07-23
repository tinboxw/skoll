package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const partyTable = "parties"

var partyFields = []string{"id", "party_type", "code", "name", "unified_social_credit_code", "region", "rating", "status", "disable_reason", "contacts", "addresses", "settlement_terms", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}

type partyContact struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Title   string `json:"title"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Primary bool   `json:"primary"`
}
type partyAddress struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	Detail   string `json:"detail"`
	Default  bool   `json:"default"`
}
type settlementTerms struct {
	Currency    string `json:"currency"`
	PaymentDays int64  `json:"paymentDays"`
	CreditLimit int64  `json:"creditLimit"`
}
type party struct {
	ID                      string          `json:"id"`
	Type                    string          `json:"type"`
	Code                    string          `json:"code"`
	Name                    string          `json:"name"`
	UnifiedSocialCreditCode string          `json:"unifiedSocialCreditCode"`
	Region                  string          `json:"region"`
	Rating                  int64           `json:"rating"`
	Status                  string          `json:"status"`
	DisableReason           string          `json:"disableReason,omitempty"`
	Contacts                []partyContact  `json:"contacts"`
	Addresses               []partyAddress  `json:"addresses"`
	Settlement              settlementTerms `json:"settlementTerms"`
	Version                 int64           `json:"version"`
	CreatedAt               string          `json:"createdAt"`
	UpdatedAt               string          `json:"updatedAt"`
	scope                   employeeScope
}
type partyWriteRequest struct {
	TenantID                string          `json:"tenantId"`
	OrganizationID          string          `json:"organizationId"`
	Code                    string          `json:"code"`
	Name                    string          `json:"name"`
	UnifiedSocialCreditCode string          `json:"unifiedSocialCreditCode"`
	Region                  string          `json:"region"`
	Rating                  int64           `json:"rating"`
	Contacts                []partyContact  `json:"contacts"`
	Addresses               []partyAddress  `json:"addresses"`
	Settlement              settlementTerms `json:"settlementTerms"`
	Version                 int64           `json:"version,omitempty"`
}
type partyStatusRequest struct {
	Reason  string `json:"reason"`
	Version int64  `json:"version"`
}

func (s *server) listParties(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		offset, limit, err := employeePagination(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		items, err := s.queryParties(ctx, kind, partyPermission(kind, "read"), r.URL.Query().Get("keyword"), r.URL.Query().Get("status"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		total := len(items)
		if offset > total {
			offset = total
		}
		end := min(total, offset+limit)
		writeOK(w, map[string]any{"items": items[offset:end], "total": total, "offset": offset, "limit": limit})
	}
}

func (s *server) createParty(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		key, err := mutationKey(r, kind+"-create")
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input partyWriteRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if err = validatePartyWrite(&input, false); err != nil {
			writeServiceError(w, err)
			return
		}
		scope, err := s.exactWriteScope(ctx, partyPermission(kind, "create"), input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if err = s.ensurePartyUnique(ctx, kind, partyPermission(kind, "create"), "", input.Code, input.UnifiedSocialCreditCode); err != nil {
			writeServiceError(w, err)
			return
		}
		item := party{ID: "party-" + strings.TrimPrefix(s.newID(), "employee-"), Type: kind, Code: input.Code, Name: input.Name, UnifiedSocialCreditCode: input.UnifiedSocialCreditCode, Region: input.Region, Rating: input.Rating, Status: "active", Contacts: input.Contacts, Addresses: input.Addresses, Settlement: input.Settlement, scope: scope}
		var created party
		err = s.transaction(ctx, func(tx context.Context) error {
			result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: partyTable, Operation: pluginsdk.DataMutationInsert, Scope: partyIntent(kind, "create", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: partyValues(item), Returning: partyFields, IdempotencyKey: key})
			if mutationErr != nil {
				return mutationErr
			}
			created, mutationErr = partyFromMutation(result)
			if mutationErr != nil {
				return mutationErr
			}
			return s.audit(tx, "pharma_oa."+kind+".create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": created.Code, "creditCode": created.UnifiedSocialCreditCode})
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeCreated(w, map[string]any{"item": created})
	}
}

func (s *server) updateParty(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		key, err := mutationKey(r, kind+"-update")
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input partyWriteRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if err = validatePartyWrite(&input, true); err != nil {
			writeServiceError(w, err)
			return
		}
		current, err := s.getParty(ctx, kind, partyPermission(kind, "update"), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if err = s.ensurePartyUnique(ctx, kind, partyPermission(kind, "update"), current.ID, input.Code, input.UnifiedSocialCreditCode); err != nil {
			writeServiceError(w, err)
			return
		}
		current.Code, current.Name, current.UnifiedSocialCreditCode, current.Region, current.Rating = input.Code, input.Name, input.UnifiedSocialCreditCode, input.Region, input.Rating
		current.Contacts, current.Addresses, current.Settlement = input.Contacts, input.Addresses, input.Settlement
		updated, err := s.mutateParty(ctx, kind, "update", key, current, input.Version, pluginsdk.AuditRiskMedium)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) changePartyStatus(kind string, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		action := "disable"
		if enabled {
			action = "enable"
		}
		key, err := mutationKey(r, kind+"-"+action)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input partyStatusRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		input.Reason = strings.TrimSpace(input.Reason)
		if input.Version < 1 || (!enabled && input.Reason == "") || len(input.Reason) > 500 {
			writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_request", "current version and a bounded disable reason are required"))
			return
		}
		current, err := s.getParty(ctx, kind, partyPermission(kind, action), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if enabled {
			current.Status, current.DisableReason = "active", ""
		} else {
			current.Status, current.DisableReason = "disabled", input.Reason
		}
		updated, err := s.mutateParty(ctx, kind, action, key, current, input.Version, pluginsdk.AuditRiskHigh)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) mutateParty(ctx context.Context, kind, action, key string, item party, version int64, risk pluginsdk.AuditRisk) (party, error) {
	if version != item.Version {
		return party{}, newHTTPError(http.StatusConflict, "stale_party", "party version is stale")
	}
	var updated party
	err := s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: partyTable, Operation: pluginsdk.DataMutationUpdate, Scope: partyIntent(kind, action, item.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: partyValues(item), Returning: partyFields, IdempotencyKey: key, ExpectedVersion: &version})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = partyFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa."+kind+"."+action, item.ID, risk, map[string]any{"status": item.Status, "code": item.Code})
	})
	return updated, err
}

func (s *server) ensurePartyUnique(ctx context.Context, kind string, permission pluginsdk.Permission, excludeID, code, credit string) error {
	kindValue, codeValue, creditValue := stringValue(kind), stringValue(code), stringValue(credit)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{
		{Field: "party_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue},
		{Any: []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorEqual, Value: &codeValue}, {Field: "unified_social_credit_code", Operator: pluginsdk.DataOperatorEqual, Value: &creditValue}}},
	}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: partyTable, Fields: partyFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 2}})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		item, parseErr := partyFromRecord(record)
		if parseErr != nil {
			return parseErr
		}
		if item.ID != excludeID && (strings.EqualFold(item.Code, code) || strings.EqualFold(item.UnifiedSocialCreditCode, credit)) {
			return newHTTPError(http.StatusConflict, "duplicate_party", "code or unified social credit code already exists")
		}
	}
	return nil
}

func (s *server) getParty(ctx context.Context, kind string, permission pluginsdk.Permission, id string) (party, error) {
	idValue, kindValue := stringValue(strings.TrimSpace(id)), stringValue(kind)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}, {Field: "party_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: partyTable, Fields: partyFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return party{}, err
	}
	if len(page.Records) != 1 {
		return party{}, newHTTPError(http.StatusNotFound, "party_not_found", "party was not found")
	}
	return partyFromRecord(page.Records[0])
}

func (s *server) queryParties(ctx context.Context, kind string, permission pluginsdk.Permission, keyword, status string) ([]party, error) {
	kindValue := stringValue(kind)
	filters := []pluginsdk.DataFilter{{Field: "party_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue}}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "name", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "unified_social_credit_code", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "region", Operator: pluginsdk.DataOperatorContains, Value: &value}}})
	}
	status = strings.TrimSpace(status)
	if status != "" {
		if status != "active" && status != "disabled" {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "party status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	filter := &pluginsdk.DataFilter{All: filters}
	out, cursor := make([]party, 0), ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: partyTable, Fields: partyFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: filter, Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200}})
		if err != nil {
			return nil, err
		}
		for _, record := range page.Records {
			item, parseErr := partyFromRecord(record)
			if parseErr != nil {
				return nil, parseErr
			}
			out = append(out, item)
		}
		if !page.HasMore {
			return out, nil
		}
		cursor = page.NextCursor
	}
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "parties", "party query exceeds 5000 records", false)
}

func partyPermission(kind, action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa." + kind, Action: action}
}
func partyIntent(kind, action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: partyPermission(kind, action), Filter: pluginsdk.ScopeFilter{TenantIDs: []string{scope.TenantID}, OrganizationIDs: []string{scope.OrganizationID}, OwnerIDs: []string{scope.OwnerID}}}
}
func partyValues(item party) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{"party_type": stringValue(item.Type), "code": stringValue(item.Code), "name": stringValue(item.Name), "unified_social_credit_code": stringValue(item.UnifiedSocialCreditCode), "region": stringValue(item.Region), "rating": {Type: pluginsdk.DataValueInteger, Value: strconv.FormatInt(item.Rating, 10)}, "status": stringValue(item.Status), "disable_reason": nullableStringValue(item.DisableReason), "contacts": jsonValue(item.Contacts), "addresses": jsonValue(item.Addresses), "settlement_terms": jsonValue(item.Settlement)}
}
func partyFromMutation(result pluginsdk.DataMutationResult) (party, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return party{}, fmt.Errorf("party mutation returned no record")
	}
	return partyFromRecord(*result.Record)
}
func partyFromRecord(record pluginsdk.DataRecord) (party, error) {
	rating, _ := strconv.ParseInt(dataString(record, "rating"), 10, 64)
	item := party{ID: dataString(record, "id"), Type: dataString(record, "party_type"), Code: dataString(record, "code"), Name: dataString(record, "name"), UnifiedSocialCreditCode: dataString(record, "unified_social_credit_code"), Region: dataString(record, "region"), Rating: rating, Status: dataString(record, "status"), DisableReason: dataString(record, "disable_reason"), Contacts: []partyContact{}, Addresses: []partyAddress{}, Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: employeeScope{TenantID: dataString(record, "tenant_id"), OrganizationID: dataString(record, "organization_id"), OwnerID: dataString(record, "owner_id")}}
	for field, target := range map[string]any{"contacts": &item.Contacts, "addresses": &item.Addresses, "settlement_terms": &item.Settlement} {
		if raw := dataString(record, field); raw != "" {
			if err := json.Unmarshal([]byte(raw), target); err != nil {
				return party{}, fmt.Errorf("decode party %s: %w", field, err)
			}
		}
	}
	return item, nil
}

func validatePartyWrite(input *partyWriteRequest, update bool) error {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.UnifiedSocialCreditCode = strings.ToUpper(strings.TrimSpace(input.UnifiedSocialCreditCode))
	input.Region = strings.TrimSpace(input.Region)
	input.Settlement.Currency = strings.ToUpper(strings.TrimSpace(input.Settlement.Currency))
	if input.Code == "" || len(input.Code) > 64 || input.Name == "" || len(input.Name) > 255 || len(input.UnifiedSocialCreditCode) < 8 || len(input.UnifiedSocialCreditCode) > 32 || input.Region == "" || input.Rating < 1 || input.Rating > 5 || (update && input.Version < 1) {
		return newHTTPError(http.StatusBadRequest, "invalid_party", "code, name, credit code, region, rating, and current version are required")
	}
	if len(input.Contacts) == 0 || len(input.Contacts) > 10 || len(input.Addresses) == 0 || len(input.Addresses) > 10 {
		return newHTTPError(http.StatusBadRequest, "invalid_party", "bounded contacts and addresses are required")
	}
	primary, defaults := 0, 0
	for index := range input.Contacts {
		item := &input.Contacts[index]
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.Email = strings.TrimSpace(item.Email)
		if item.ID == "" {
			item.ID = fmt.Sprintf("contact-%d", index+1)
		}
		if item.Name == "" || (item.Email != "" && !validPartyEmail(item.Email)) {
			return newHTTPError(http.StatusBadRequest, "invalid_contact", "contact name and valid email are required")
		}
		if item.Primary {
			primary++
		}
	}
	for index := range input.Addresses {
		item := &input.Addresses[index]
		item.ID = strings.TrimSpace(item.ID)
		item.Label = strings.TrimSpace(item.Label)
		item.Detail = strings.TrimSpace(item.Detail)
		if item.ID == "" {
			item.ID = fmt.Sprintf("address-%d", index+1)
		}
		if item.Label == "" || item.Detail == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_address", "address label and detail are required")
		}
		if item.Default {
			defaults++
		}
	}
	if primary != 1 || defaults != 1 {
		return newHTTPError(http.StatusBadRequest, "invalid_party", "exactly one primary contact and default address are required")
	}
	if len(input.Settlement.Currency) != 3 || input.Settlement.PaymentDays < 0 || input.Settlement.PaymentDays > 365 || input.Settlement.CreditLimit < 0 {
		return newHTTPError(http.StatusBadRequest, "invalid_settlement", "currency, payment days, and credit limit are invalid")
	}
	return nil
}
func validPartyEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
