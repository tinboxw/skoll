package pharmaoa

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

const (
	MasterDataResourceEmployees = "employees"
	MasterDataResourceProducts  = "products"
	MasterDataResourceSuppliers = "suppliers"
	MasterDataResourceCustomers = "customers"
)

type MasterDataExchangeService interface {
	Template(resource string) (MasterDataTemplate, error)
	Import(ctx context.Context, in MasterDataImportInput) (MasterDataImportResult, error)
	Export(ctx context.Context, in MasterDataExportInput) (MasterDataExportJob, error)
}

type MasterDataImportInput struct {
	Resource string
	Rows     []map[string]string
	ActorID  string
}

type MasterDataExportInput struct {
	Resource string
	ActorID  string
}

type MasterDataTemplate struct {
	Resource    string     `json:"resource"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"contentType"`
	Headers     []string   `json:"headers"`
	Rows        [][]string `json:"rows"`
	FileBase64  string     `json:"fileBase64"`
}

type MasterDataImportResult struct {
	Resource string                 `json:"resource"`
	Created  int                    `json:"created"`
	Failed   []MasterDataImportFail `json:"failed"`
}

type MasterDataImportFail struct {
	Row     int    `json:"row"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MasterDataExportJob struct {
	JobID       string     `json:"jobId"`
	Resource    string     `json:"resource"`
	Status      string     `json:"status"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"contentType"`
	Headers     []string   `json:"headers"`
	Rows        [][]string `json:"rows"`
	FileBase64  string     `json:"fileBase64"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type masterDataExchangeService struct {
	employees EmployeeService
	products  ProductService
	suppliers SupplierService
	customers CustomerService
	nowFn     func() time.Time
}

func NewMasterDataExchangeService(employees EmployeeService, products ProductService, suppliers SupplierService, customers CustomerService) MasterDataExchangeService {
	return &masterDataExchangeService{
		employees: employees,
		products:  products,
		suppliers: suppliers,
		customers: customers,
		nowFn:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *masterDataExchangeService) Template(resource string) (MasterDataTemplate, error) {
	resource, err := normalizeMasterDataResource(resource)
	if err != nil {
		return MasterDataTemplate{}, err
	}
	headers := masterDataHeaders(resource)
	rows := [][]string{headers}
	return MasterDataTemplate{
		Resource:    resource,
		Filename:    resource + "_template.xlsx",
		ContentType: masterDataExcelContentType,
		Headers:     headers,
		Rows:        rows,
		FileBase64:  xlsxBase64(rows),
	}, nil
}

func (s *masterDataExchangeService) Import(ctx context.Context, in MasterDataImportInput) (MasterDataImportResult, error) {
	resource, err := normalizeMasterDataResource(in.Resource)
	if err != nil {
		return MasterDataImportResult{}, err
	}
	result := MasterDataImportResult{Resource: resource, Failed: []MasterDataImportFail{}}
	for idx, row := range in.Rows {
		rowNumber := idx + 1
		code := strings.TrimSpace(row["code"])
		if code == "" {
			result.Failed = append(result.Failed, MasterDataImportFail{Row: rowNumber, Code: code, Message: "code is required"})
			continue
		}
		if err := s.importRow(ctx, resource, row, in.ActorID); err != nil {
			result.Failed = append(result.Failed, MasterDataImportFail{Row: rowNumber, Code: code, Message: err.Error()})
			continue
		}
		result.Created++
	}
	return result, nil
}

func (s *masterDataExchangeService) Export(ctx context.Context, in MasterDataExportInput) (MasterDataExportJob, error) {
	resource, err := normalizeMasterDataResource(in.Resource)
	if err != nil {
		return MasterDataExportJob{}, err
	}
	headers := masterDataHeaders(resource)
	rows, err := s.exportRows(ctx, resource)
	if err != nil {
		return MasterDataExportJob{}, err
	}
	outputRows := append([][]string{headers}, rows...)
	return MasterDataExportJob{
		JobID:       "pharma-master-export-" + resource + "-" + strconv.FormatInt(s.nowFn().Unix(), 10),
		Resource:    resource,
		Status:      "completed",
		Filename:    resource + "_export.xlsx",
		ContentType: masterDataExcelContentType,
		Headers:     headers,
		Rows:        outputRows,
		FileBase64:  xlsxBase64(outputRows),
		CreatedAt:   s.nowFn(),
	}, nil
}

func (s *masterDataExchangeService) importRow(ctx context.Context, resource string, row map[string]string, actorID string) error {
	switch resource {
	case MasterDataResourceEmployees:
		if s.employees == nil {
			return fmt.Errorf("employee service is unavailable")
		}
		_, err := s.employees.Create(ctx, EmployeeWriteInput{
			Code:         row["code"],
			Name:         row["name"],
			DepartmentID: row["departmentId"],
			PositionID:   row["positionId"],
			Phone:        row["phone"],
			Email:        row["email"],
			Certificates: employeeCertificatesFromRow(row),
			ActorID:      actorID,
		})
		return err
	case MasterDataResourceProducts:
		if s.products == nil {
			return fmt.Errorf("product service is unavailable")
		}
		_, err := s.products.Create(ctx, ProductWriteInput{
			Code:           row["code"],
			Name:           row["name"],
			Spec:           row["spec"],
			DosageForm:     row["dosageForm"],
			Manufacturer:   row["manufacturer"],
			ApprovalNumber: row["approvalNumber"],
			Temperature:    productTemperatureFromRow(row),
			ActorID:        actorID,
		})
		return err
	case MasterDataResourceSuppliers:
		if s.suppliers == nil {
			return fmt.Errorf("supplier service is unavailable")
		}
		_, err := s.suppliers.Create(ctx, SupplierWriteInput{
			Code:           row["code"],
			Name:           row["name"],
			Rating:         intFromRow(row, "rating"),
			Contacts:       supplierContactsFromRow(row),
			Qualifications: supplierQualificationsFromRow(row),
			ActorID:        actorID,
		})
		return err
	case MasterDataResourceCustomers:
		if s.customers == nil {
			return fmt.Errorf("customer service is unavailable")
		}
		_, err := s.customers.Create(ctx, CustomerWriteInput{
			Code:           row["code"],
			Name:           row["name"],
			Region:         row["region"],
			OrganizationID: row["organizationId"],
			OwnerID:        row["ownerId"],
			Rating:         intFromRow(row, "rating"),
			Contacts:       customerContactsFromRow(row),
			Qualifications: customerQualificationsFromRow(row),
			ActorID:        actorID,
			Scope:          CustomerAccessScope{IncludeAll: true},
		})
		return err
	default:
		return fmt.Errorf("unsupported master data resource: %s", resource)
	}
}

func (s *masterDataExchangeService) exportRows(ctx context.Context, resource string) ([][]string, error) {
	switch resource {
	case MasterDataResourceEmployees:
		items, err := s.employees.List(ctx, EmployeeListInput{Limit: 1000})
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			certName, certNumber, certExpires := "", "", ""
			if len(item.Certificates) > 0 {
				certName = item.Certificates[0].Name
				certNumber = item.Certificates[0].Number
				certExpires = formatMasterDate(item.Certificates[0].ExpiresAt)
			}
			rows = append(rows, []string{item.Code, item.Name, item.DepartmentID, item.PositionID, item.Phone, item.Email, certName, certNumber, certExpires})
		}
		return rows, nil
	case MasterDataResourceProducts:
		items, err := s.products.List(ctx, ProductListInput{Limit: 1000})
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{item.Code, item.Name, item.Spec, item.DosageForm, item.Manufacturer, item.ApprovalNumber, strconv.FormatBool(item.Temperature.Required), floatString(item.Temperature.MinCelsius), floatString(item.Temperature.MaxCelsius)})
		}
		return rows, nil
	case MasterDataResourceSuppliers:
		items, err := s.suppliers.List(ctx, SupplierListInput{Limit: 1000})
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			contactName, qualificationName, qualificationNumber, qualificationExpires := "", "", "", ""
			if len(item.Contacts) > 0 {
				contactName = item.Contacts[0].Name
			}
			if len(item.Qualifications) > 0 {
				qualificationName = item.Qualifications[0].Name
				qualificationNumber = item.Qualifications[0].Number
				qualificationExpires = formatMasterDate(item.Qualifications[0].ExpiresAt)
			}
			rows = append(rows, []string{item.Code, item.Name, strconv.Itoa(item.Rating), contactName, qualificationName, qualificationNumber, qualificationExpires})
		}
		return rows, nil
	case MasterDataResourceCustomers:
		items, err := s.customers.List(ctx, CustomerListInput{Limit: 1000, Scope: CustomerAccessScope{IncludeAll: true}})
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			contactName, qualificationName, qualificationNumber, qualificationExpires := "", "", "", ""
			if len(item.Contacts) > 0 {
				contactName = item.Contacts[0].Name
			}
			if len(item.Qualifications) > 0 {
				qualificationName = item.Qualifications[0].Name
				qualificationNumber = item.Qualifications[0].Number
				qualificationExpires = formatMasterDate(item.Qualifications[0].ExpiresAt)
			}
			rows = append(rows, []string{item.Code, item.Name, item.Region, item.OrganizationID, item.OwnerID, strconv.Itoa(item.Rating), contactName, qualificationName, qualificationNumber, qualificationExpires})
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unsupported master data resource: %s", resource)
	}
}

func normalizeMasterDataResource(resource string) (string, error) {
	resource = strings.ToLower(strings.TrimSpace(resource))
	switch resource {
	case MasterDataResourceEmployees, MasterDataResourceProducts, MasterDataResourceSuppliers, MasterDataResourceCustomers:
		return resource, nil
	default:
		return "", fmt.Errorf("unsupported master data resource: %s", resource)
	}
}

func masterDataHeaders(resource string) []string {
	switch resource {
	case MasterDataResourceEmployees:
		return []string{"code", "name", "departmentId", "positionId", "phone", "email", "certificateName", "certificateNumber", "certificateExpiresAt"}
	case MasterDataResourceProducts:
		return []string{"code", "name", "spec", "dosageForm", "manufacturer", "approvalNumber", "temperatureRequired", "temperatureMinCelsius", "temperatureMaxCelsius"}
	case MasterDataResourceSuppliers:
		return []string{"code", "name", "rating", "contactName", "qualificationName", "qualificationNumber", "qualificationExpiresAt"}
	case MasterDataResourceCustomers:
		return []string{"code", "name", "region", "organizationId", "ownerId", "rating", "contactName", "qualificationName", "qualificationNumber", "qualificationExpiresAt"}
	default:
		return []string{}
	}
}

func employeeCertificatesFromRow(row map[string]string) []domainpharma.EmployeeCertificate {
	name := strings.TrimSpace(row["certificateName"])
	if name == "" {
		return nil
	}
	return []domainpharma.EmployeeCertificate{{Name: name, Number: row["certificateNumber"], ExpiresAt: dateFromRow(row, "certificateExpiresAt")}}
}

func productTemperatureFromRow(row map[string]string) domainpharma.ProductTemperature {
	required := boolFromRow(row, "temperatureRequired")
	return domainpharma.ProductTemperature{
		Required:   required,
		MinCelsius: floatFromRow(row, "temperatureMinCelsius"),
		MaxCelsius: floatFromRow(row, "temperatureMaxCelsius"),
	}
}

func supplierContactsFromRow(row map[string]string) []domainpharma.SupplierContact {
	name := strings.TrimSpace(row["contactName"])
	if name == "" {
		return nil
	}
	return []domainpharma.SupplierContact{{Name: name, Primary: true}}
}

func supplierQualificationsFromRow(row map[string]string) []domainpharma.SupplierQualification {
	name := strings.TrimSpace(row["qualificationName"])
	if name == "" {
		return nil
	}
	return []domainpharma.SupplierQualification{{Name: name, Number: row["qualificationNumber"], ExpiresAt: dateFromRow(row, "qualificationExpiresAt")}}
}

func customerContactsFromRow(row map[string]string) []domainpharma.CustomerContact {
	name := strings.TrimSpace(row["contactName"])
	if name == "" {
		return nil
	}
	return []domainpharma.CustomerContact{{Name: name}}
}

func customerQualificationsFromRow(row map[string]string) []domainpharma.CustomerQualification {
	name := strings.TrimSpace(row["qualificationName"])
	if name == "" {
		return nil
	}
	return []domainpharma.CustomerQualification{{Name: name, Number: row["qualificationNumber"], ExpiresAt: dateFromRow(row, "qualificationExpiresAt")}}
}

func intFromRow(row map[string]string, key string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(row[key]))
	return value
}

func boolFromRow(row map[string]string, key string) bool {
	value, _ := strconv.ParseBool(strings.TrimSpace(row[key]))
	return value
}

func floatFromRow(row map[string]string, key string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(row[key]), 64)
	return value
}

func dateFromRow(row map[string]string, key string) time.Time {
	raw := strings.TrimSpace(row[key])
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func formatMasterDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func floatString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func sortedMasterRows(rows [][]string) [][]string {
	sort.Slice(rows, func(i, j int) bool {
		if len(rows[i]) == 0 || len(rows[j]) == 0 {
			return len(rows[i]) < len(rows[j])
		}
		return rows[i][0] < rows[j][0]
	})
	return rows
}

const masterDataExcelContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func xlsxBase64(rows [][]string) string {
	raw, err := buildMinimalXLSX(rows)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func buildMinimalXLSX(rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="master_data" sheetId="1" r:id="rId1"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/worksheets/sheet1.xml": buildWorksheetXML(rows),
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildWorksheetXML(rows [][]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rIdx, row := range rows {
		b.WriteString(`<row r="`)
		b.WriteString(strconv.Itoa(rIdx + 1))
		b.WriteString(`">`)
		for cIdx, cell := range row {
			ref := excelColumnName(cIdx+1) + strconv.Itoa(rIdx+1)
			b.WriteString(`<c r="`)
			b.WriteString(ref)
			b.WriteString(`" t="inlineStr"><is><t>`)
			b.WriteString(xmlEscape(cell))
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func excelColumnName(index int) string {
	if index <= 0 {
		return "A"
	}
	var out []byte
	for index > 0 {
		index--
		out = append([]byte{byte('A' + index%26)}, out...)
		index /= 26
	}
	return string(out)
}

func xmlEscape(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}
