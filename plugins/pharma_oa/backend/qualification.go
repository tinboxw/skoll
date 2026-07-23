package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	qualificationTypeTable = "qualification_types"
	qualificationTable     = "qualifications"
)

var (
	qualificationTypeFields = []string{"id", "code", "name", "subject_type", "business_gate", "description", "validity_days", "alert_days", "evidence_required", "business_required", "status", "disable_reason", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}
	qualificationFields     = []string{"id", "type_id", "subject_type", "subject_id", "certificate_number", "issuer", "valid_from", "valid_to", "status", "evidence_file_id", "evidence_file_name", "evidence_file_hash", "review_comment", "submitted_at", "reviewed_at", "reviewed_by", "revoked_at", "last_alert_key", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}
)

type qualificationType struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	SubjectType      string `json:"subjectType"`
	BusinessGate     string `json:"businessGate"`
	Description      string `json:"description"`
	ValidityDays     int64  `json:"validityDays"`
	AlertDays        int64  `json:"alertDays"`
	EvidenceRequired bool   `json:"evidenceRequired"`
	BusinessRequired bool   `json:"businessRequired"`
	Status           string `json:"status"`
	DisableReason    string `json:"disableReason,omitempty"`
	Version          int64  `json:"version"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
	scope            employeeScope
}

type qualificationTypeWriteRequest struct {
	TenantID         string `json:"tenantId"`
	OrganizationID   string `json:"organizationId"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	SubjectType      string `json:"subjectType"`
	BusinessGate     string `json:"businessGate"`
	Description      string `json:"description"`
	ValidityDays     int64  `json:"validityDays"`
	AlertDays        int64  `json:"alertDays"`
	EvidenceRequired bool   `json:"evidenceRequired"`
	BusinessRequired bool   `json:"businessRequired"`
	Version          int64  `json:"version,omitempty"`
}

type qualification struct {
	ID                string `json:"id"`
	TypeID            string `json:"typeId"`
	SubjectType       string `json:"subjectType"`
	SubjectID         string `json:"subjectId"`
	CertificateNumber string `json:"certificateNumber"`
	Issuer            string `json:"issuer"`
	ValidFrom         string `json:"validFrom"`
	ValidTo           string `json:"validTo"`
	Status            string `json:"status"`
	EvidenceFileID    string `json:"evidenceFileId,omitempty"`
	EvidenceFileName  string `json:"evidenceFileName,omitempty"`
	EvidenceFileHash  string `json:"evidenceFileHash,omitempty"`
	ReviewComment     string `json:"reviewComment,omitempty"`
	SubmittedAt       string `json:"submittedAt,omitempty"`
	ReviewedAt        string `json:"reviewedAt,omitempty"`
	ReviewedBy        string `json:"reviewedBy,omitempty"`
	RevokedAt         string `json:"revokedAt,omitempty"`
	LastAlertKey      string `json:"lastAlertKey,omitempty"`
	Version           int64  `json:"version"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	scope             employeeScope
}

type qualificationEvidenceRequest struct {
	Name          string `json:"name"`
	ContentBase64 string `json:"contentBase64"`
}

type qualificationWriteRequest struct {
	TenantID          string                       `json:"tenantId"`
	OrganizationID    string                       `json:"organizationId"`
	TypeID            string                       `json:"typeId"`
	SubjectType       string                       `json:"subjectType"`
	SubjectID         string                       `json:"subjectId"`
	CertificateNumber string                       `json:"certificateNumber"`
	Issuer            string                       `json:"issuer"`
	ValidFrom         string                       `json:"validFrom"`
	ValidTo           string                       `json:"validTo"`
	Evidence          qualificationEvidenceRequest `json:"evidence,omitempty"`
	Version           int64                        `json:"version,omitempty"`
}

type qualificationActionRequest struct {
	Comment string `json:"comment"`
	Version int64  `json:"version"`
}

func (s *server) listQualificationTypes(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	limit, err := catalogPageLimit(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := qualificationTypeFilter(r.URL.Query().Get("keyword"), r.URL.Query().Get("status"), r.URL.Query().Get("subjectType"), r.URL.Query().Get("businessGate"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTypeTable, Fields: qualificationTypeFields, Scope: pluginsdk.DataScopeIntent{Permission: qualificationTypePermission("read")}, Filter: filter, Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}, {Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Cursor: r.URL.Query().Get("cursor"), Limit: limit}})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]qualificationType, 0, len(page.Records))
	for _, record := range page.Records {
		item, parseErr := qualificationTypeFromRecord(record)
		if parseErr != nil {
			writeServiceError(w, parseErr)
			return
		}
		items = append(items, item)
	}
	writeOK(w, map[string]any{"items": items, "pageInfo": pageInfo(page, limit)})
}

func (s *server) createQualificationType(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "qualification-type-create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input qualificationTypeWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateQualificationTypeWrite(&input, false); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, qualificationTypePermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item := qualificationType{ID: resourceID("qualification-type", s.newID()), Code: input.Code, Name: input.Name, SubjectType: input.SubjectType, BusinessGate: input.BusinessGate, Description: input.Description, ValidityDays: input.ValidityDays, AlertDays: input.AlertDays, EvidenceRequired: input.EvidenceRequired, BusinessRequired: input.BusinessRequired, Status: "active", scope: scope}
	var created qualificationType
	err = s.transaction(ctx, func(tx context.Context) error {
		if uniqueErr := s.ensureQualificationTypeUnique(tx, qualificationTypePermission("create"), scope, "", item.Code); uniqueErr != nil {
			return uniqueErr
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: qualificationTypeTable, Operation: pluginsdk.DataMutationInsert, Scope: qualificationTypeIntent("create", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: qualificationTypeValues(item), Returning: qualificationTypeFields, IdempotencyKey: key})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = qualificationTypeFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.qualification_type.create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": created.Code, "subjectType": created.SubjectType})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created})
}

func (s *server) updateQualificationType(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "qualification-type-update")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input qualificationTypeWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateQualificationTypeWrite(&input, true); err != nil {
		writeServiceError(w, err)
		return
	}
	current, err := s.getQualificationTypeUnscoped(ctx, qualificationTypePermission("update"), r.PathValue("id"), false)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	contractChanged := current.SubjectType != input.SubjectType || current.BusinessGate != input.BusinessGate || current.ValidityDays != input.ValidityDays || current.AlertDays != input.AlertDays || current.EvidenceRequired != input.EvidenceRequired || current.BusinessRequired != input.BusinessRequired
	if contractChanged {
		if err = s.ensureQualificationTypeUnused(ctx, qualificationTypePermission("update"), current); err != nil {
			writeServiceError(w, err)
			return
		}
	}
	current.Code, current.Name, current.SubjectType, current.BusinessGate, current.Description = input.Code, input.Name, input.SubjectType, input.BusinessGate, input.Description
	current.ValidityDays, current.AlertDays, current.EvidenceRequired, current.BusinessRequired = input.ValidityDays, input.AlertDays, input.EvidenceRequired, input.BusinessRequired
	updated, err := s.mutateQualificationType(ctx, "update", key, current, input.Version, pluginsdk.AuditRiskMedium, true)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) changeQualificationTypeStatus(enabled bool) http.HandlerFunc {
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
		key, err := mutationKey(r, "qualification-type-"+action)
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
		current, err := s.getQualificationTypeUnscoped(ctx, qualificationTypePermission(action), r.PathValue("id"), false)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if enabled {
			current.Status, current.DisableReason = "active", ""
		} else {
			current.Status, current.DisableReason = "disabled", input.Reason
		}
		updated, err := s.mutateQualificationType(ctx, action, key, current, input.Version, pluginsdk.AuditRiskHigh, false)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) mutateQualificationType(ctx context.Context, action, key string, item qualificationType, version int64, risk pluginsdk.AuditRisk, checkUnique bool) (qualificationType, error) {
	if version != item.Version {
		return qualificationType{}, newHTTPError(http.StatusConflict, "stale_qualification_type", "qualification type version is stale")
	}
	var updated qualificationType
	err := s.transaction(ctx, func(tx context.Context) error {
		if checkUnique {
			if uniqueErr := s.ensureQualificationTypeUnique(tx, qualificationTypePermission(action), item.scope, item.ID, item.Code); uniqueErr != nil {
				return uniqueErr
			}
		}
		if action == "disable" {
			if inUseErr := s.ensureQualificationTypeNotPending(tx, qualificationTypePermission(action), item); inUseErr != nil {
				return inUseErr
			}
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: qualificationTypeTable, Operation: pluginsdk.DataMutationUpdate, Scope: qualificationTypeIntent(action, item.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: qualificationTypeValues(item), Returning: qualificationTypeFields, IdempotencyKey: key, ExpectedVersion: &version})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = qualificationTypeFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.qualification_type."+action, updated.ID, risk, map[string]any{"code": updated.Code, "status": updated.Status})
	})
	return updated, err
}

func (s *server) listQualifications(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	limit, err := catalogPageLimit(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := qualificationFilter(r.URL.Query().Get("keyword"), r.URL.Query().Get("status"), r.URL.Query().Get("subjectType"), r.URL.Query().Get("subjectId"), r.URL.Query().Get("typeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: qualificationFields, Scope: pluginsdk.DataScopeIntent{Permission: qualificationPermission("read")}, Filter: filter, Sort: []pluginsdk.DataSort{{Field: "valid_to", Direction: pluginsdk.DataSortAscending}, {Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Cursor: r.URL.Query().Get("cursor"), Limit: limit}})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]qualification, 0, len(page.Records))
	for _, record := range page.Records {
		item, parseErr := qualificationFromRecord(record)
		if parseErr != nil {
			writeServiceError(w, parseErr)
			return
		}
		items = append(items, item)
	}
	writeOK(w, map[string]any{"items": items, "pageInfo": pageInfo(page, limit)})
}

func (s *server) createQualification(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "qualification-create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input qualificationWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateQualificationWrite(&input, false); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, qualificationPermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	typeItem, err := s.getQualificationType(ctx, qualificationPermission("create"), scope, input.TypeID, true)
	if err != nil {
		writeServiceError(w, newHTTPError(http.StatusUnprocessableEntity, "invalid_qualification_type", "qualification type is missing, disabled, or outside the trusted scope"))
		return
	}
	if typeItem.SubjectType != input.SubjectType {
		writeServiceError(w, newHTTPError(http.StatusUnprocessableEntity, "subject_type_mismatch", "qualification subject type does not match its type"))
		return
	}
	if err = s.validateQualificationSubject(ctx, qualificationPermission("create"), scope, input.SubjectType, input.SubjectID); err != nil {
		writeServiceError(w, err)
		return
	}
	if err = validateQualificationValidity(typeItem, input.ValidFrom, input.ValidTo); err != nil {
		writeServiceError(w, err)
		return
	}
	id := stableID("qualification", key)
	if existing, found, findErr := s.findQualification(ctx, qualificationPermission("create"), scope, id); findErr != nil {
		writeServiceError(w, findErr)
		return
	} else if found {
		writeOK(w, map[string]any{"item": existing, "duplicate": true})
		return
	}
	file, content, err := s.storeQualificationEvidence(ctx, id, key, input.Evidence, typeItem.EvidenceRequired, input.SubjectType, input.SubjectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item := qualification{ID: id, TypeID: input.TypeID, SubjectType: input.SubjectType, SubjectID: input.SubjectID, CertificateNumber: input.CertificateNumber, Issuer: input.Issuer, ValidFrom: input.ValidFrom, ValidTo: input.ValidTo, Status: "draft", scope: scope}
	if file != nil {
		item.EvidenceFileID, item.EvidenceFileName, item.EvidenceFileHash = file.ID, file.Name, file.Hash
		if item.EvidenceFileHash == "" {
			hash := sha256.Sum256(content)
			item.EvidenceFileHash = hex.EncodeToString(hash[:])
		}
	}
	var created qualification
	err = s.transaction(ctx, func(tx context.Context) error {
		if uniqueErr := s.ensureQualificationUnique(tx, qualificationPermission("create"), scope, "", item); uniqueErr != nil {
			return uniqueErr
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: qualificationTable, Operation: pluginsdk.DataMutationInsert, Scope: qualificationIntent("create", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: qualificationValues(item), Returning: qualificationFields, IdempotencyKey: key})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = qualificationFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.qualification.create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"subjectType": created.SubjectType, "subjectId": created.SubjectID, "fileId": created.EvidenceFileID})
	})
	if err != nil {
		if file != nil {
			_ = s.host.Files.Delete(ctx, file.ID)
		}
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created, "duplicate": false})
}

func (s *server) updateQualification(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "qualification-update")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input qualificationWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateQualificationWrite(&input, true); err != nil {
		writeServiceError(w, err)
		return
	}
	current, err := s.getQualificationUnscoped(ctx, qualificationPermission("update"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if current.Status != "draft" && current.Status != "rejected" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "qualification_locked", "only draft or rejected qualifications can be edited"))
		return
	}
	typeItem, err := s.getQualificationType(ctx, qualificationPermission("update"), current.scope, input.TypeID, true)
	if err != nil || typeItem.SubjectType != input.SubjectType {
		writeServiceError(w, newHTTPError(http.StatusUnprocessableEntity, "invalid_qualification_type", "qualification type is invalid"))
		return
	}
	if err = s.validateQualificationSubject(ctx, qualificationPermission("update"), current.scope, input.SubjectType, input.SubjectID); err != nil {
		writeServiceError(w, err)
		return
	}
	if err = validateQualificationValidity(typeItem, input.ValidFrom, input.ValidTo); err != nil {
		writeServiceError(w, err)
		return
	}
	current.TypeID, current.SubjectType, current.SubjectID, current.CertificateNumber, current.Issuer = input.TypeID, input.SubjectType, input.SubjectID, input.CertificateNumber, input.Issuer
	current.ValidFrom, current.ValidTo, current.Status, current.ReviewComment, current.LastAlertKey = input.ValidFrom, input.ValidTo, "draft", "", ""
	updated, err := s.mutateQualification(ctx, "update", key, current, input.Version, pluginsdk.AuditRiskMedium, true)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) submitQualification(w http.ResponseWriter, r *http.Request) {
	s.transitionQualification(w, r, "submit")
}

func (s *server) approveQualification(w http.ResponseWriter, r *http.Request) {
	s.transitionQualification(w, r, "approve")
}

func (s *server) rejectQualification(w http.ResponseWriter, r *http.Request) {
	s.transitionQualification(w, r, "reject")
}

func (s *server) revokeQualification(w http.ResponseWriter, r *http.Request) {
	s.transitionQualification(w, r, "revoke")
}

func (s *server) transitionQualification(w http.ResponseWriter, r *http.Request, action string) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "qualification-"+action)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input qualificationActionRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Comment = strings.TrimSpace(input.Comment)
	if input.Version < 1 || len(input.Comment) > 1000 || ((action == "reject" || action == "revoke") && input.Comment == "") {
		writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_review", "current version and a bounded required review comment are required"))
		return
	}
	current, err := s.getQualificationUnscoped(ctx, qualificationPermission(action), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, qualificationPermission(action), current.scope.TenantID, current.scope.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	switch action {
	case "submit":
		typeItem, typeErr := s.getQualificationType(ctx, qualificationPermission(action), current.scope, current.TypeID, false)
		if typeErr != nil {
			err = typeErr
			break
		}
		if current.Status != "draft" && current.Status != "rejected" {
			err = newHTTPError(http.StatusConflict, "invalid_qualification_transition", "only draft or rejected qualifications can be submitted")
		} else if typeItem.EvidenceRequired && current.EvidenceFileID == "" {
			err = newHTTPError(http.StatusUnprocessableEntity, "evidence_required", "owned evidence is required before submission")
		} else {
			current.Status, current.SubmittedAt, current.ReviewComment = "pending", now, ""
		}
	case "approve":
		validTo, _ := time.Parse(time.RFC3339Nano, current.ValidTo)
		if current.Status != "pending" {
			err = newHTTPError(http.StatusConflict, "invalid_qualification_transition", "only pending qualifications can be approved")
		} else if !validTo.After(s.now().UTC()) {
			err = newHTTPError(http.StatusUnprocessableEntity, "qualification_expired", "expired qualification cannot be approved")
		} else {
			current.Status, current.ReviewComment, current.ReviewedAt, current.ReviewedBy = "approved", input.Comment, now, scope.OwnerID
		}
	case "reject":
		if current.Status != "pending" {
			err = newHTTPError(http.StatusConflict, "invalid_qualification_transition", "only pending qualifications can be rejected")
		} else {
			current.Status, current.ReviewComment, current.ReviewedAt, current.ReviewedBy = "rejected", input.Comment, now, scope.OwnerID
		}
	case "revoke":
		if current.Status != "approved" {
			err = newHTTPError(http.StatusConflict, "invalid_qualification_transition", "only approved qualifications can be revoked")
		} else {
			current.Status, current.ReviewComment, current.RevokedAt, current.ReviewedBy = "revoked", input.Comment, now, scope.OwnerID
		}
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	risk := pluginsdk.AuditRiskMedium
	if action == "approve" || action == "reject" || action == "revoke" {
		risk = pluginsdk.AuditRiskHigh
	}
	updated, err := s.mutateQualification(ctx, action, key, current, input.Version, risk, false)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) scanQualificationExpiry(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "qualification-expiry-scan")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryAllQualifications(ctx, qualificationPermission("expiry.run"), "approved")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	now := s.now().UTC()
	scheduled, skipped := 0, 0
	jobs := make([]pluginsdk.Job, 0)
	for _, item := range items {
		typeItem, typeErr := s.getQualificationType(ctx, qualificationPermission("expiry.run"), item.scope, item.TypeID, false)
		if typeErr != nil {
			writeServiceError(w, typeErr)
			return
		}
		validTo, parseErr := time.Parse(time.RFC3339Nano, item.ValidTo)
		if parseErr != nil || validTo.After(now.AddDate(0, 0, int(typeItem.AlertDays))) {
			continue
		}
		alertKey := fmt.Sprintf("qualification-expiry:%s:%s:%d", item.ID, validTo.Format("20060102"), typeItem.AlertDays)
		if item.LastAlertKey == alertKey {
			skipped++
			continue
		}
		payload, _ := json.Marshal(map[string]any{"qualificationId": item.ID, "typeId": item.TypeID, "subjectType": item.SubjectType, "subjectId": item.SubjectID, "validTo": item.ValidTo, "alertDays": typeItem.AlertDays})
		job, scheduleErr := s.host.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{ID: stableID("job", alertKey), Kind: "pharma_oa.qualification.expiring", IdempotencyKey: alertKey, Payload: payload, RunAt: now, MaxAttempts: 5})
		if scheduleErr != nil {
			writeServiceError(w, scheduleErr)
			return
		}
		item.LastAlertKey = alertKey
		if !validTo.After(now) {
			item.Status = "expired"
		}
		if _, mutationErr := s.mutateQualification(ctx, "expiry.run", stableID("alert", alertKey), item, item.Version, pluginsdk.AuditRiskLow, false); mutationErr != nil {
			writeServiceError(w, mutationErr)
			return
		}
		jobs, scheduled = append(jobs, job), scheduled+1
	}
	if err = s.transaction(ctx, func(tx context.Context) error {
		return s.audit(tx, "pharma_oa.qualification.expiry_scan", key, pluginsdk.AuditRiskHigh, map[string]any{"scheduled": scheduled, "skipped": skipped})
	}); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"scheduled": scheduled, "skipped": skipped, "jobs": jobs})
}

func (s *server) customerSalesEligibility(w http.ResponseWriter, r *http.Request) {
	s.writeQualificationEligibility(w, r, "customer", r.PathValue("id"), "sales", partyPermission("customer", "sales"))
}

func (s *server) supplierPurchaseEligibility(w http.ResponseWriter, r *http.Request) {
	s.writeQualificationEligibility(w, r, "supplier", r.PathValue("id"), "purchase", partyPermission("supplier", "purchase"))
}

func (s *server) productSaleEligibility(w http.ResponseWriter, r *http.Request) {
	s.writeQualificationEligibility(w, r, "product", r.PathValue("id"), "sale", productPermission("sales"))
}

func (s *server) manufacturerSupplyEligibility(w http.ResponseWriter, r *http.Request) {
	s.writeQualificationEligibility(w, r, "manufacturer", r.PathValue("id"), "supply", catalogPermission("manufacturer", "supply"))
}

func (s *server) writeQualificationEligibility(w http.ResponseWriter, r *http.Request, subjectType, subjectID, gate string, permission pluginsdk.Permission) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.subjectScope(ctx, permission, subjectType, subjectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	types, err := s.queryRequiredQualificationTypes(ctx, permission, scope, subjectType, gate)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	qualifications, err := s.querySubjectQualifications(ctx, permission, scope, subjectType, subjectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	now, approved := s.now().UTC(), make(map[string]bool)
	for _, item := range qualifications {
		validTo, _ := time.Parse(time.RFC3339Nano, item.ValidTo)
		if item.Status == "approved" && validTo.After(now) {
			approved[item.TypeID] = true
		}
	}
	missing := make([]qualificationType, 0)
	for _, item := range types {
		if !approved[item.ID] {
			missing = append(missing, item)
		}
	}
	writeOK(w, map[string]any{"eligible": len(types) > 0 && len(missing) == 0, "subjectType": subjectType, "subjectId": subjectID, "businessGate": gate, "required": types, "missing": missing})
}

func (s *server) mutateQualification(ctx context.Context, action, key string, item qualification, version int64, risk pluginsdk.AuditRisk, checkUnique bool) (qualification, error) {
	if version != item.Version {
		return qualification{}, newHTTPError(http.StatusConflict, "stale_qualification", "qualification version is stale")
	}
	var updated qualification
	err := s.transaction(ctx, func(tx context.Context) error {
		if checkUnique {
			if uniqueErr := s.ensureQualificationUnique(tx, qualificationPermission(action), item.scope, item.ID, item); uniqueErr != nil {
				return uniqueErr
			}
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: qualificationTable, Operation: pluginsdk.DataMutationUpdate, Scope: qualificationIntent(action, item.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: qualificationValues(item), Returning: qualificationFields, IdempotencyKey: key, ExpectedVersion: &version})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = qualificationFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.qualification."+action, updated.ID, risk, map[string]any{"status": updated.Status, "subjectType": updated.SubjectType, "subjectId": updated.SubjectID})
	})
	return updated, err
}

func (s *server) storeQualificationEvidence(ctx context.Context, qualificationID, requestKey string, input qualificationEvidenceRequest, required bool, subjectType, subjectID string) (*pluginsdk.FileObject, []byte, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" && input.ContentBase64 == "" && !required {
		return nil, nil, nil
	}
	base := filepath.Base(name)
	content, err := base64.StdEncoding.DecodeString(input.ContentBase64)
	if name == "" || base != name || base == "." || err != nil || len(content) == 0 || len(content) > 5<<20 {
		return nil, nil, newHTTPError(http.StatusBadRequest, "invalid_evidence", "a safe PDF, JPEG, or PNG evidence file up to 5 MB is required")
	}
	extension := strings.ToLower(filepath.Ext(base))
	mime := http.DetectContentType(content)
	allowed := (extension == ".pdf" && mime == "application/pdf") || ((extension == ".jpg" || extension == ".jpeg") && mime == "image/jpeg") || (extension == ".png" && mime == "image/png")
	if !allowed {
		return nil, nil, newHTTPError(http.StatusBadRequest, "invalid_evidence", "evidence extension and detected content type must be PDF, JPEG, or PNG")
	}
	file, err := s.host.Files.Store(ctx, pluginsdk.FileWrite{Key: "qualifications/" + qualificationID + "/" + requestKey, Name: base, Content: content, Visibility: pluginsdk.FileVisibilityPrivate, Metadata: map[string]string{"qualificationId": qualificationID, "subjectType": subjectType, "subjectId": subjectID, "requestKey": requestKey}})
	if err != nil {
		return nil, nil, err
	}
	return &file, content, nil
}

func (s *server) validateQualificationSubject(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, subjectType, subjectID string) error {
	_, err := s.subjectScopeExact(ctx, permission, scope, subjectType, subjectID)
	return err
}

func (s *server) subjectScope(ctx context.Context, permission pluginsdk.Permission, subjectType, subjectID string) (employeeScope, error) {
	switch subjectType {
	case "customer", "supplier":
		item, err := s.getParty(ctx, subjectType, permission, subjectID)
		if err != nil || item.Status != "active" {
			return employeeScope{}, newHTTPError(http.StatusUnprocessableEntity, "invalid_subject", "party subject is missing, disabled, or outside the trusted scope")
		}
		return item.scope, nil
	case "product":
		item, err := s.getProduct(ctx, permission, subjectID)
		if err != nil || item.Status != "active" {
			return employeeScope{}, newHTTPError(http.StatusUnprocessableEntity, "invalid_subject", "product subject is missing, disabled, or outside the trusted scope")
		}
		return item.scope, nil
	case "manufacturer":
		item, err := s.getCatalogUnscoped(ctx, "manufacturer", permission, subjectID)
		if err != nil || item.Status != "active" {
			return employeeScope{}, newHTTPError(http.StatusUnprocessableEntity, "invalid_subject", "manufacturer subject is missing, disabled, or outside the trusted scope")
		}
		return item.scope, nil
	default:
		return employeeScope{}, newHTTPError(http.StatusBadRequest, "invalid_subject_type", "qualification subject type is invalid")
	}
}

func (s *server) subjectScopeExact(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, subjectType, subjectID string) (employeeScope, error) {
	resolved, err := s.subjectScope(ctx, permission, subjectType, subjectID)
	if err != nil {
		return employeeScope{}, err
	}
	if resolved != scope {
		return employeeScope{}, newHTTPError(http.StatusForbidden, "scope_mismatch", "qualification subject is outside the trusted write scope")
	}
	return resolved, nil
}

func (s *server) ensureQualificationTypeUnique(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, excludeID, code string) error {
	value := stringValue(code)
	filter := pluginsdk.DataFilter{Field: "code", Operator: pluginsdk.DataOperatorEqual, Value: &value}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTypeTable, Fields: qualificationTypeFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 2}})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		item, parseErr := qualificationTypeFromRecord(record)
		if parseErr != nil {
			return parseErr
		}
		if item.ID != excludeID {
			return newHTTPError(http.StatusConflict, "duplicate_qualification_type", "qualification type code already exists")
		}
	}
	return nil
}

func (s *server) ensureQualificationTypeUnused(ctx context.Context, permission pluginsdk.Permission, item qualificationType) error {
	value := stringValue(item.ID)
	filter := pluginsdk.DataFilter{Field: "type_id", Operator: pluginsdk.DataOperatorEqual, Value: &value}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: []string{"id"}, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(item.scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return err
	}
	if len(page.Records) > 0 {
		return newHTTPError(http.StatusConflict, "qualification_type_in_use", "referenced qualification type policy cannot be changed; create a new type")
	}
	return nil
}

func (s *server) ensureQualificationUnique(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, excludeID string, candidate qualification) error {
	typeID, subjectID, number := stringValue(candidate.TypeID), stringValue(candidate.SubjectID), stringValue(candidate.CertificateNumber)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "type_id", Operator: pluginsdk.DataOperatorEqual, Value: &typeID}, {Field: "subject_id", Operator: pluginsdk.DataOperatorEqual, Value: &subjectID}, {Field: "certificate_number", Operator: pluginsdk.DataOperatorEqual, Value: &number}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: qualificationFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 2}})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		item, parseErr := qualificationFromRecord(record)
		if parseErr != nil {
			return parseErr
		}
		if item.ID != excludeID {
			return newHTTPError(http.StatusConflict, "duplicate_qualification", "qualification certificate already exists for this subject and type")
		}
	}
	return nil
}

func (s *server) ensureQualificationTypeNotPending(ctx context.Context, permission pluginsdk.Permission, item qualificationType) error {
	typeID := stringValue(item.ID)
	statuses := []pluginsdk.DataValue{stringValue("pending"), stringValue("approved")}
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "type_id", Operator: pluginsdk.DataOperatorEqual, Value: &typeID}, {Field: "status", Operator: pluginsdk.DataOperatorIn, Values: statuses}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: []string{"id"}, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(item.scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return err
	}
	if len(page.Records) > 0 {
		return newHTTPError(http.StatusConflict, "qualification_type_in_use", "pending or approved qualifications still use this type")
	}
	return nil
}

func (s *server) getQualificationTypeUnscoped(ctx context.Context, permission pluginsdk.Permission, id string, activeOnly bool) (qualificationType, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filters := []pluginsdk.DataFilter{{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}}
	if activeOnly {
		active := stringValue("active")
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &active})
	}
	filter := pluginsdk.DataFilter{All: filters}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTypeTable, Fields: qualificationTypeFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return qualificationType{}, err
	}
	if len(page.Records) != 1 {
		return qualificationType{}, newHTTPError(http.StatusNotFound, "qualification_type_not_found", "qualification type was not found")
	}
	return qualificationTypeFromRecord(page.Records[0])
}

func (s *server) getQualificationType(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, id string, activeOnly bool) (qualificationType, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filters := []pluginsdk.DataFilter{{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}}
	if activeOnly {
		active := stringValue("active")
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &active})
	}
	filter := pluginsdk.DataFilter{All: filters}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTypeTable, Fields: qualificationTypeFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return qualificationType{}, err
	}
	if len(page.Records) != 1 {
		return qualificationType{}, newHTTPError(http.StatusNotFound, "qualification_type_not_found", "qualification type was not found")
	}
	return qualificationTypeFromRecord(page.Records[0])
}

func (s *server) getQualificationUnscoped(ctx context.Context, permission pluginsdk.Permission, id string) (qualification, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filter := pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: qualificationFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return qualification{}, err
	}
	if len(page.Records) != 1 {
		return qualification{}, newHTTPError(http.StatusNotFound, "qualification_not_found", "qualification was not found")
	}
	return qualificationFromRecord(page.Records[0])
}

func (s *server) findQualification(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, id string) (qualification, bool, error) {
	idValue := stringValue(id)
	filter := pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: qualificationFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return qualification{}, false, err
	}
	if len(page.Records) == 0 {
		return qualification{}, false, nil
	}
	item, err := qualificationFromRecord(page.Records[0])
	return item, true, err
}

func (s *server) queryAllQualifications(ctx context.Context, permission pluginsdk.Permission, status string) ([]qualification, error) {
	statusValue := stringValue(status)
	filter := pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &statusValue}
	items, cursor := make([]qualification, 0), ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: qualificationFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "valid_to", Direction: pluginsdk.DataSortAscending}, {Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200}})
		if err != nil {
			return nil, err
		}
		for _, record := range page.Records {
			item, parseErr := qualificationFromRecord(record)
			if parseErr != nil {
				return nil, parseErr
			}
			items = append(items, item)
		}
		if !page.HasMore {
			return items, nil
		}
		cursor = page.NextCursor
	}
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "qualifications", "qualification scan exceeds 5000 records", false)
}

func (s *server) queryRequiredQualificationTypes(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, subjectType, gate string) ([]qualificationType, error) {
	subject, businessGate, active, required := stringValue(subjectType), stringValue(gate), stringValue("active"), booleanValue(true)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "subject_type", Operator: pluginsdk.DataOperatorEqual, Value: &subject}, {Field: "business_gate", Operator: pluginsdk.DataOperatorEqual, Value: &businessGate}, {Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &active}, {Field: "business_required", Operator: pluginsdk.DataOperatorEqual, Value: &required}}}
	items, cursor := make([]qualificationType, 0), ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTypeTable, Fields: qualificationTypeFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "code", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200}})
		if err != nil {
			return nil, err
		}
		for _, record := range page.Records {
			item, parseErr := qualificationTypeFromRecord(record)
			if parseErr != nil {
				return nil, parseErr
			}
			items = append(items, item)
		}
		if !page.HasMore {
			return items, nil
		}
		cursor = page.NextCursor
	}
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "qualification_types", "qualification gate exceeds 5000 required types", false)
}

func (s *server) querySubjectQualifications(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, subjectType, subjectID string) ([]qualification, error) {
	typeValue, idValue := stringValue(subjectType), stringValue(subjectID)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "subject_type", Operator: pluginsdk.DataOperatorEqual, Value: &typeValue}, {Field: "subject_id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}}}
	items, cursor := make([]qualification, 0), ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: qualificationTable, Fields: qualificationFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "valid_to", Direction: pluginsdk.DataSortDescending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200}})
		if err != nil {
			return nil, err
		}
		for _, record := range page.Records {
			item, parseErr := qualificationFromRecord(record)
			if parseErr != nil {
				return nil, parseErr
			}
			items = append(items, item)
		}
		if !page.HasMore {
			return items, nil
		}
		cursor = page.NextCursor
	}
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "qualifications", "qualification gate exceeds 5000 records", false)
}

func qualificationTypeFilter(keyword, status, subjectType, gate string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 4)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "name", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "description", Operator: pluginsdk.DataOperatorContains, Value: &value}}})
	}
	if status = strings.TrimSpace(status); status != "" {
		if status != "active" && status != "disabled" {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "qualification type status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	for field, raw := range map[string]string{"subject_type": subjectType, "business_gate": gate} {
		if raw = strings.TrimSpace(raw); raw != "" {
			value := stringValue(raw)
			filters = append(filters, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &value})
		}
	}
	return combineFilters(filters), nil
}

func qualificationFilter(keyword, status, subjectType, subjectID, typeID string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 5)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{{Field: "certificate_number", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "issuer", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "subject_id", Operator: pluginsdk.DataOperatorContains, Value: &value}}})
	}
	if status = strings.TrimSpace(status); status != "" {
		valid := map[string]bool{"draft": true, "pending": true, "approved": true, "rejected": true, "revoked": true, "expired": true}
		if !valid[status] {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "qualification status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	for field, raw := range map[string]string{"subject_type": subjectType, "subject_id": subjectID, "type_id": typeID} {
		if raw = strings.TrimSpace(raw); raw != "" {
			value := stringValue(raw)
			filters = append(filters, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &value})
		}
	}
	return combineFilters(filters), nil
}

func combineFilters(filters []pluginsdk.DataFilter) *pluginsdk.DataFilter {
	if len(filters) == 0 {
		return nil
	}
	if len(filters) == 1 {
		return &filters[0]
	}
	return &pluginsdk.DataFilter{All: filters}
}

func validateQualificationTypeWrite(input *qualificationTypeWriteRequest, update bool) error {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name, input.SubjectType, input.BusinessGate, input.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.SubjectType), strings.TrimSpace(input.BusinessGate), strings.TrimSpace(input.Description)
	gates := map[string]string{"customer": "sales", "supplier": "purchase", "product": "sale", "manufacturer": "supply"}
	if input.Code == "" || len(input.Code) > 64 || input.Name == "" || len(input.Name) > 255 || len(input.Description) > 512 || gates[input.SubjectType] != input.BusinessGate || input.ValidityDays < 1 || input.ValidityDays > 3650 || input.AlertDays < 1 || input.AlertDays > input.ValidityDays || (update && input.Version < 1) {
		return newHTTPError(http.StatusBadRequest, "invalid_qualification_type", "code, name, matching subject/gate, validity, alert days, and current version are required")
	}
	return nil
}

func validateQualificationWrite(input *qualificationWriteRequest, update bool) error {
	input.TypeID, input.SubjectType, input.SubjectID = strings.TrimSpace(input.TypeID), strings.TrimSpace(input.SubjectType), strings.TrimSpace(input.SubjectID)
	input.CertificateNumber, input.Issuer = strings.ToUpper(strings.TrimSpace(input.CertificateNumber)), strings.TrimSpace(input.Issuer)
	from, fromErr := parseDate(input.ValidFrom)
	to, toErr := parseDate(input.ValidTo)
	if input.TypeID == "" || input.SubjectID == "" || input.CertificateNumber == "" || len(input.CertificateNumber) > 128 || input.Issuer == "" || len(input.Issuer) > 255 || fromErr != nil || toErr != nil || !to.After(from) || (update && input.Version < 1) {
		return newHTTPError(http.StatusBadRequest, "invalid_qualification", "type, subject, certificate, issuer, ordered validity dates, and current version are required")
	}
	input.ValidFrom, input.ValidTo = from.Format(time.RFC3339Nano), to.Format(time.RFC3339Nano)
	return nil
}

func validateQualificationValidity(typeItem qualificationType, validFrom, validTo string) error {
	from, err := time.Parse(time.RFC3339Nano, validFrom)
	if err != nil {
		return err
	}
	to, err := time.Parse(time.RFC3339Nano, validTo)
	if err != nil {
		return err
	}
	if to.After(from.AddDate(0, 0, int(typeItem.ValidityDays)+1)) {
		return newHTTPError(http.StatusBadRequest, "validity_exceeded", "qualification validity exceeds its type policy")
	}
	return nil
}

func qualificationTypePermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.qualification_type", Action: action}
}

func qualificationPermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.qualification", Action: action}
}

func qualificationTypeIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: qualificationTypePermission(action), Filter: scopeFilter(scope)}
}

func qualificationIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: qualificationPermission(action), Filter: scopeFilter(scope)}
}

func qualificationTypeValues(item qualificationType) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{"code": stringValue(item.Code), "name": stringValue(item.Name), "subject_type": stringValue(item.SubjectType), "business_gate": stringValue(item.BusinessGate), "description": stringValue(item.Description), "validity_days": integerValue(item.ValidityDays), "alert_days": integerValue(item.AlertDays), "evidence_required": booleanValue(item.EvidenceRequired), "business_required": booleanValue(item.BusinessRequired), "status": stringValue(item.Status), "disable_reason": nullableStringValue(item.DisableReason)}
}

func qualificationValues(item qualification) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{"type_id": stringValue(item.TypeID), "subject_type": stringValue(item.SubjectType), "subject_id": stringValue(item.SubjectID), "certificate_number": stringValue(item.CertificateNumber), "issuer": stringValue(item.Issuer), "valid_from": timestampValue(item.ValidFrom), "valid_to": timestampValue(item.ValidTo), "status": stringValue(item.Status), "evidence_file_id": nullableStringValue(item.EvidenceFileID), "evidence_file_name": nullableStringValue(item.EvidenceFileName), "evidence_file_hash": nullableStringValue(item.EvidenceFileHash), "review_comment": nullableStringValue(item.ReviewComment), "submitted_at": nullableTimestampValue(item.SubmittedAt), "reviewed_at": nullableTimestampValue(item.ReviewedAt), "reviewed_by": nullableStringValue(item.ReviewedBy), "revoked_at": nullableTimestampValue(item.RevokedAt), "last_alert_key": nullableStringValue(item.LastAlertKey)}
}

func booleanValue(value bool) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueBoolean, Value: strconv.FormatBool(value)}
}

func stableID(prefix, value string) string {
	hash := sha256.Sum256([]byte(value))
	return prefix + "-" + hex.EncodeToString(hash[:12])
}

func pageInfo(page pluginsdk.DataPage, limit int) map[string]any {
	return map[string]any{"nextCursor": page.NextCursor, "hasMore": page.HasMore, "limit": limit}
}

func qualificationTypeFromMutation(result pluginsdk.DataMutationResult) (qualificationType, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return qualificationType{}, fmt.Errorf("qualification type mutation returned no record")
	}
	return qualificationTypeFromRecord(*result.Record)
}

func qualificationFromMutation(result pluginsdk.DataMutationResult) (qualification, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return qualification{}, fmt.Errorf("qualification mutation returned no record")
	}
	return qualificationFromRecord(*result.Record)
}

func qualificationTypeFromRecord(record pluginsdk.DataRecord) (qualificationType, error) {
	validity, err := parseRecordInteger(record, "validity_days")
	if err != nil {
		return qualificationType{}, err
	}
	alert, err := parseRecordInteger(record, "alert_days")
	if err != nil {
		return qualificationType{}, err
	}
	return qualificationType{ID: dataString(record, "id"), Code: dataString(record, "code"), Name: dataString(record, "name"), SubjectType: dataString(record, "subject_type"), BusinessGate: dataString(record, "business_gate"), Description: dataString(record, "description"), ValidityDays: validity, AlertDays: alert, EvidenceRequired: dataString(record, "evidence_required") == "true", BusinessRequired: dataString(record, "business_required") == "true", Status: dataString(record, "status"), DisableReason: dataString(record, "disable_reason"), Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record)}, nil
}

func qualificationFromRecord(record pluginsdk.DataRecord) (qualification, error) {
	return qualification{ID: dataString(record, "id"), TypeID: dataString(record, "type_id"), SubjectType: dataString(record, "subject_type"), SubjectID: dataString(record, "subject_id"), CertificateNumber: dataString(record, "certificate_number"), Issuer: dataString(record, "issuer"), ValidFrom: dataString(record, "valid_from"), ValidTo: dataString(record, "valid_to"), Status: dataString(record, "status"), EvidenceFileID: dataString(record, "evidence_file_id"), EvidenceFileName: dataString(record, "evidence_file_name"), EvidenceFileHash: dataString(record, "evidence_file_hash"), ReviewComment: dataString(record, "review_comment"), SubmittedAt: dataString(record, "submitted_at"), ReviewedAt: dataString(record, "reviewed_at"), ReviewedBy: dataString(record, "reviewed_by"), RevokedAt: dataString(record, "revoked_at"), LastAlertKey: dataString(record, "last_alert_key"), Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record)}, nil
}
