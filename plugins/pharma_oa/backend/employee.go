package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const employeeTable = "employees"

var employeeFields = []string{
	"id", "code", "name", "department_id", "position_id", "employment_status", "phone", "email",
	"hire_date", "left_at", "leave_reason", "attachments", "certificates",
	"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
}

type employeeScope struct {
	TenantID       string
	OrganizationID string
	OwnerID        string
}

type employeeAttachment struct {
	FileID     string `json:"fileId"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MIME       string `json:"mime"`
	RequestKey string `json:"requestKey"`
}

type employeeCertificate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Number    string `json:"number"`
	ExpiresAt string `json:"expiresAt"`
}

type employee struct {
	ID               string                `json:"id"`
	Code             string                `json:"code"`
	Name             string                `json:"name"`
	DepartmentID     string                `json:"departmentId"`
	PositionID       string                `json:"positionId"`
	EmploymentStatus string                `json:"employmentStatus"`
	Phone            string                `json:"phone"`
	Email            string                `json:"email"`
	HireDate         string                `json:"hireDate"`
	LeftAt           string                `json:"leftAt,omitempty"`
	LeaveReason      string                `json:"leaveReason,omitempty"`
	Attachments      []employeeAttachment  `json:"attachments"`
	Certificates     []employeeCertificate `json:"certificates"`
	Version          int64                 `json:"version"`
	CreatedAt        string                `json:"createdAt"`
	UpdatedAt        string                `json:"updatedAt"`
	scope            employeeScope
}

type employeeWriteRequest struct {
	TenantID       string                `json:"tenantId"`
	OrganizationID string                `json:"organizationId"`
	Code           string                `json:"code"`
	Name           string                `json:"name"`
	DepartmentID   string                `json:"departmentId"`
	PositionID     string                `json:"positionId"`
	Phone          string                `json:"phone"`
	Email          string                `json:"email"`
	HireDate       string                `json:"hireDate"`
	Certificates   []employeeCertificate `json:"certificates"`
	Version        int64                 `json:"version,omitempty"`
}

type employeeLeaveRequest struct {
	Reason  string `json:"reason"`
	Version int64  `json:"version"`
}

type employeeAttachmentRequest struct {
	Name          string `json:"name"`
	ContentBase64 string `json:"contentBase64"`
	Version       int64  `json:"version"`
}

type employeeReminder struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeCode string `json:"employeeCode"`
	EmployeeName string `json:"employeeName"`
	Certificate  string `json:"certificate"`
	Number       string `json:"number"`
	ExpiresAt    string `json:"expiresAt"`
}

func (s *server) listEmployees(w http.ResponseWriter, r *http.Request) {
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
	items, err := s.queryEmployees(ctx, employeePermission("read"), strings.TrimSpace(r.URL.Query().Get("keyword")), strings.TrimSpace(r.URL.Query().Get("status")))
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

func (s *server) createEmployee(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input employeeWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateEmployeeWrite(&input, false); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, employeePermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	input.TenantID, input.OrganizationID = scope.TenantID, scope.OrganizationID
	now := s.now().UTC()
	item := employee{
		ID: s.newID(), Code: input.Code, Name: input.Name, DepartmentID: input.DepartmentID, PositionID: input.PositionID,
		EmploymentStatus: "active", Phone: input.Phone, Email: input.Email, HireDate: input.HireDate,
		Attachments: []employeeAttachment{}, Certificates: input.Certificates, scope: scope,
	}
	if item.HireDate == "" {
		item.HireDate = now.Format(time.RFC3339Nano)
	}
	var created employee
	err = s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: employeeTable, Operation: pluginsdk.DataMutationInsert, Scope: employeeIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: employeeValues(item),
			Returning: employeeFields, IdempotencyKey: requestKey,
		})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = employeeFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.employee.create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": created.Code, "departmentId": created.DepartmentID, "positionId": created.PositionID})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created})
}

func (s *server) updateEmployee(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "update")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input employeeWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateEmployeeWrite(&input, true); err != nil {
		writeServiceError(w, err)
		return
	}
	current, err := s.getEmployee(ctx, employeePermission("update"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if current.EmploymentStatus == "left" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "employee_left", "a departed employee cannot be edited"))
		return
	}
	current.Code, current.Name = input.Code, input.Name
	current.DepartmentID, current.PositionID = input.DepartmentID, input.PositionID
	current.Phone, current.Email, current.HireDate = input.Phone, input.Email, input.HireDate
	current.Certificates = input.Certificates
	updated, err := s.mutateEmployee(ctx, "update", "update", requestKey, current, input.Version, pluginsdk.AuditRiskMedium, map[string]any{"code": current.Code, "departmentId": current.DepartmentID, "positionId": current.PositionID})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) leaveEmployee(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "leave")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input employeeLeaveRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" || len(input.Reason) > 500 || input.Version < 1 {
		writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_request", "reason and current version are required"))
		return
	}
	current, err := s.getEmployee(ctx, employeePermission("leave"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if current.EmploymentStatus == "left" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "employee_left", "employee has already departed"))
		return
	}
	current.EmploymentStatus = "left"
	current.LeftAt = s.now().UTC().Format(time.RFC3339Nano)
	current.LeaveReason = input.Reason
	updated, err := s.mutateEmployee(ctx, "leave", "leave", requestKey, current, input.Version, pluginsdk.AuditRiskHigh, map[string]any{"reason": input.Reason})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) attachEmployeeFile(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "attach")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input employeeAttachmentRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(filepath.Base(input.Name))
	content, decodeErr := base64.StdEncoding.DecodeString(input.ContentBase64)
	if input.Name == "" || input.Name == "." || decodeErr != nil || len(content) == 0 || len(content) > 1<<20 || input.Version < 1 {
		writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_attachment", "name, bounded base64 content, and current version are required"))
		return
	}
	current, err := s.getEmployee(ctx, employeePermission("update"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	for _, attachment := range current.Attachments {
		if attachment.RequestKey == requestKey {
			writeOK(w, map[string]any{"item": current, "attachment": attachment, "duplicate": true})
			return
		}
	}
	file, err := s.host.Files.Store(ctx, pluginsdk.FileWrite{
		Key: "employees/" + current.ID + "/" + requestKey, Name: input.Name, Content: content,
		Visibility: pluginsdk.FileVisibilityPrivate, Metadata: map[string]string{"employeeId": current.ID, "requestKey": requestKey},
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	attachment := employeeAttachment{FileID: file.ID, Name: file.Name, Size: file.Size, MIME: file.MIME, RequestKey: requestKey}
	current.Attachments = append(current.Attachments, attachment)
	updated, err := s.mutateEmployee(ctx, "update", "attach", requestKey, current, input.Version, pluginsdk.AuditRiskMedium, map[string]any{"fileId": file.ID, "fileName": file.Name})
	if err != nil {
		_ = s.host.Files.Delete(ctx, file.ID)
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated, "attachment": attachment, "duplicate": false})
}

func (s *server) employeeQualificationReminders(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	days, err := boundedInt(r.URL.Query().Get("days"), 30, 1, 365)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryEmployees(ctx, employeePermission("reminder"), "", "active")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	deadline := s.now().UTC().AddDate(0, 0, days)
	reminders := make([]employeeReminder, 0)
	for _, item := range items {
		for _, certificate := range item.Certificates {
			expiresAt, parseErr := time.Parse(time.RFC3339, certificate.ExpiresAt)
			if parseErr == nil && !expiresAt.After(deadline) {
				reminders = append(reminders, employeeReminder{EmployeeID: item.ID, EmployeeCode: item.Code, EmployeeName: item.Name, Certificate: certificate.Name, Number: certificate.Number, ExpiresAt: certificate.ExpiresAt})
			}
		}
	}
	sort.Slice(reminders, func(i, j int) bool { return reminders[i].ExpiresAt < reminders[j].ExpiresAt })
	writeOK(w, map[string]any{"items": reminders, "days": days})
}

func (s *server) mutateEmployee(ctx context.Context, permissionAction, auditAction, requestKey string, item employee, expectedVersion int64, risk pluginsdk.AuditRisk, detail map[string]any) (employee, error) {
	if expectedVersion < 1 || expectedVersion != item.Version {
		return employee{}, newHTTPError(http.StatusConflict, "stale_employee", "employee version is stale")
	}
	var updated employee
	err := s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: employeeTable, Operation: pluginsdk.DataMutationUpdate, Scope: employeeIntent(permissionAction, item.scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: employeeValues(item),
			Returning: employeeFields, IdempotencyKey: requestKey, ExpectedVersion: &expectedVersion,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = employeeFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.employee."+auditAction, item.ID, risk, detail)
	})
	return updated, err
}

func (s *server) getEmployee(ctx context.Context, permission pluginsdk.Permission, id string) (employee, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return employee{}, newHTTPError(http.StatusBadRequest, "invalid_request", "employee id is required")
	}
	value := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: employeeTable, Fields: employeeFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return employee{}, err
	}
	if len(page.Records) != 1 {
		return employee{}, newHTTPError(http.StatusNotFound, "employee_not_found", "employee was not found in the trusted scope")
	}
	return employeeFromRecord(page.Records[0])
}

func (s *server) queryEmployees(ctx context.Context, permission pluginsdk.Permission, keyword, status string) ([]employee, error) {
	filter, err := employeeFilter(keyword, status)
	if err != nil {
		return nil, err
	}
	items := make([]employee, 0)
	cursor := ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, queryErr := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: employeeTable, Fields: employeeFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: filter,
			Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200},
		})
		if queryErr != nil {
			return nil, queryErr
		}
		for _, record := range page.Records {
			item, parseErr := employeeFromRecord(record)
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
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "employees", "employee query exceeds 5000 records", false)
}

func (s *server) exactWriteScope(ctx context.Context, permission pluginsdk.Permission, tenantID, organizationID string) (employeeScope, error) {
	predicate, err := s.host.DataScopes.Resolve(ctx, permission)
	if err != nil {
		return employeeScope{}, err
	}
	if predicate.Denied() {
		return employeeScope{}, newHTTPError(http.StatusForbidden, "scope_denied", "plugin data access is denied")
	}
	tenantID, err = chooseScopeID(tenantID, predicate.TenantIDs(), predicate.AllTenants(), "tenantId")
	if err != nil {
		return employeeScope{}, err
	}
	organizationID, err = chooseScopeID(organizationID, predicate.OrganizationIDs(), predicate.AllOrganizations(), "organizationId")
	if err != nil {
		return employeeScope{}, err
	}
	ownerID := strings.TrimSpace(predicate.SubjectID())
	if ownerID == "" {
		return employeeScope{}, newHTTPError(http.StatusForbidden, "scope_denied", "trusted owner scope is unavailable")
	}
	requested := pluginsdk.ScopeFilter{TenantIDs: []string{tenantID}, OrganizationIDs: []string{organizationID}, OwnerIDs: []string{ownerID}}
	if predicate.Constrain(requested).Denied() {
		return employeeScope{}, newHTTPError(http.StatusForbidden, "scope_denied", "requested plugin data scope is not allowed")
	}
	return employeeScope{TenantID: tenantID, OrganizationID: organizationID, OwnerID: ownerID}, nil
}

func employeeIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{
		Permission: employeePermission(action),
		Filter:     pluginsdk.ScopeFilter{TenantIDs: []string{scope.TenantID}, OrganizationIDs: []string{scope.OrganizationID}, OwnerIDs: []string{scope.OwnerID}},
	}
}

func employeePermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.employee", Action: action}
}

func employeeValues(item employee) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"code": stringValue(item.Code), "name": stringValue(item.Name), "department_id": stringValue(item.DepartmentID),
		"position_id": stringValue(item.PositionID), "employment_status": stringValue(item.EmploymentStatus),
		"phone": stringValue(item.Phone), "email": stringValue(item.Email), "hire_date": timestampValue(item.HireDate),
		"left_at": nullableTimestampValue(item.LeftAt), "leave_reason": nullableStringValue(item.LeaveReason),
		"attachments": jsonValue(item.Attachments), "certificates": jsonValue(item.Certificates),
	}
}

func employeeFromMutation(result pluginsdk.DataMutationResult) (employee, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return employee{}, fmt.Errorf("employee mutation returned no record")
	}
	return employeeFromRecord(*result.Record)
}

func employeeFromRecord(record pluginsdk.DataRecord) (employee, error) {
	item := employee{
		ID: dataString(record, "id"), Code: dataString(record, "code"), Name: dataString(record, "name"),
		DepartmentID: dataString(record, "department_id"), PositionID: dataString(record, "position_id"),
		EmploymentStatus: dataString(record, "employment_status"), Phone: dataString(record, "phone"), Email: dataString(record, "email"),
		HireDate: dataString(record, "hire_date"), LeftAt: dataString(record, "left_at"), LeaveReason: dataString(record, "leave_reason"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		scope:       employeeScope{TenantID: dataString(record, "tenant_id"), OrganizationID: dataString(record, "organization_id"), OwnerID: dataString(record, "owner_id")},
		Attachments: []employeeAttachment{}, Certificates: []employeeCertificate{},
	}
	if raw := dataString(record, "attachments"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Attachments); err != nil {
			return employee{}, fmt.Errorf("decode employee attachments: %w", err)
		}
	}
	if raw := dataString(record, "certificates"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Certificates); err != nil {
			return employee{}, fmt.Errorf("decode employee certificates: %w", err)
		}
	}
	return item, nil
}

func employeeFilter(keyword, status string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 2)
	keyword = strings.TrimSpace(keyword)
	if len(keyword) > 100 {
		return nil, newHTTPError(http.StatusBadRequest, "invalid_request", "keyword is too long")
	}
	if keyword != "" {
		value := stringValue(keyword)
		fields := []string{"code", "name", "department_id", "position_id", "email"}
		any := make([]pluginsdk.DataFilter, 0, len(fields))
		for _, field := range fields {
			copy := value
			any = append(any, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorContains, Value: &copy})
		}
		filters = append(filters, pluginsdk.DataFilter{Any: any})
	}
	status = strings.TrimSpace(status)
	if status != "" {
		if status != "active" && status != "on_leave" && status != "left" {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "employment status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "employment_status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) == 1 {
		return &filters[0], nil
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

func validateEmployeeWrite(input *employeeWriteRequest, update bool) error {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.DepartmentID = strings.TrimSpace(input.DepartmentID)
	input.PositionID = strings.TrimSpace(input.PositionID)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Email = strings.TrimSpace(input.Email)
	input.HireDate = strings.TrimSpace(input.HireDate)
	if input.Code == "" || len(input.Code) > 64 || input.Name == "" || len(input.Name) > 200 || input.DepartmentID == "" || len(input.DepartmentID) > 128 || input.PositionID == "" || len(input.PositionID) > 128 {
		return newHTTPError(http.StatusBadRequest, "invalid_employee", "code, name, department, and position are required")
	}
	if len(input.Phone) > 64 || len(input.Email) > 250 {
		return newHTTPError(http.StatusBadRequest, "invalid_employee", "phone or email is too long")
	}
	if input.Email != "" {
		address, err := mail.ParseAddress(input.Email)
		if err != nil || address.Address != input.Email {
			return newHTTPError(http.StatusBadRequest, "invalid_employee", "email is invalid")
		}
	}
	if input.HireDate != "" {
		parsed, err := parseDate(input.HireDate)
		if err != nil {
			return newHTTPError(http.StatusBadRequest, "invalid_employee", "hireDate must be RFC3339 or YYYY-MM-DD")
		}
		input.HireDate = parsed.Format(time.RFC3339Nano)
	}
	if update && (input.Version < 1 || input.HireDate == "") {
		return newHTTPError(http.StatusBadRequest, "invalid_employee", "current version and hire date are required")
	}
	if len(input.Certificates) > 20 {
		return newHTTPError(http.StatusBadRequest, "invalid_employee", "certificate count exceeds 20")
	}
	for index := range input.Certificates {
		certificate := &input.Certificates[index]
		certificate.ID = strings.TrimSpace(certificate.ID)
		certificate.Name = strings.TrimSpace(certificate.Name)
		certificate.Number = strings.TrimSpace(certificate.Number)
		if certificate.ID == "" || certificate.Name == "" || certificate.Number == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_certificate", "certificate id, name, and number are required")
		}
		expiresAt, err := parseDate(certificate.ExpiresAt)
		if err != nil {
			return newHTTPError(http.StatusBadRequest, "invalid_certificate", "certificate expiry must be RFC3339 or YYYY-MM-DD")
		}
		certificate.ExpiresAt = expiresAt.Format(time.RFC3339Nano)
	}
	return nil
}

func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Parse("2006-01-02", value)
}

func employeePagination(r *http.Request) (int, int, error) {
	offset, err := boundedInt(r.URL.Query().Get("offset"), 0, 0, 5000)
	if err != nil {
		return 0, 0, err
	}
	limit, err := boundedInt(r.URL.Query().Get("limit"), 25, 1, 200)
	return offset, limit, err
}

func boundedInt(raw string, defaultValue, minimum, maximum int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, newHTTPError(http.StatusBadRequest, "invalid_request", fmt.Sprintf("value must be between %d and %d", minimum, maximum))
	}
	return value, nil
}

func mutationKey(r *http.Request, operation string) (string, error) {
	value := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if value == "" || len(value) > 96 {
		return "", newHTTPError(http.StatusBadRequest, "idempotency_key_required", "a bounded Idempotency-Key header is required")
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') && !(char >= '0' && char <= '9') && !strings.ContainsRune("._:-", char) {
			return "", newHTTPError(http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key contains unsupported characters")
		}
	}
	return value + "." + operation, nil
}

func chooseScopeID(requested string, trusted []string, all bool, label string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		if all || containsString(trusted, requested) {
			return requested, nil
		}
		return "", newHTTPError(http.StatusForbidden, "scope_denied", label+" is outside the trusted scope")
	}
	if !all && len(trusted) == 1 {
		return trusted[0], nil
	}
	return "", newHTTPError(http.StatusBadRequest, "scope_required", label+" is required")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringValue(value string) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: value}
}

func timestampValue(value string) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueTimestamp, Value: value}
}

func nullableTimestampValue(value string) pluginsdk.DataValue {
	if value == "" {
		return pluginsdk.DataValue{Type: pluginsdk.DataValueNull}
	}
	return timestampValue(value)
}

func nullableStringValue(value string) pluginsdk.DataValue {
	if value == "" {
		return pluginsdk.DataValue{Type: pluginsdk.DataValueNull}
	}
	return stringValue(value)
}

func jsonValue(value any) pluginsdk.DataValue {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("[]")
	}
	return pluginsdk.DataValue{Type: pluginsdk.DataValueJSON, Value: string(raw)}
}

func dataString(record pluginsdk.DataRecord, field string) string {
	value, ok := record.Values[field]
	if !ok || value.Type == pluginsdk.DataValueNull {
		return ""
	}
	return value.Value
}
