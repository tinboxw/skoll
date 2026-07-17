package gormrepo

import (
	"encoding/json"
	"fmt"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PharmaEmployeeModel struct {
	ID               string    `gorm:"column:id;type:varchar(64);primaryKey"`
	Code             string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_pharma_employees_code"`
	Name             string    `gorm:"column:name;type:varchar(255);not null;index:idx_pharma_employees_name"`
	DepartmentID     string    `gorm:"column:department_id;type:varchar(64);index:idx_pharma_employees_org_status,priority:1"`
	PositionID       string    `gorm:"column:position_id;type:varchar(64)"`
	Phone            string    `gorm:"column:phone;type:varchar(64)"`
	Email            string    `gorm:"column:email;type:varchar(255)"`
	Status           string    `gorm:"column:status;type:varchar(32);not null;index:idx_pharma_employees_org_status,priority:2"`
	LeaveReason      string    `gorm:"column:leave_reason;type:varchar(512)"`
	CertificatesJSON string    `gorm:"column:certificates_json;type:text"`
	CreatedAt        time.Time `gorm:"column:created_at;not null"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null"`
	CreatedBy        string    `gorm:"column:created_by;type:varchar(64)"`
	UpdatedBy        string    `gorm:"column:updated_by;type:varchar(64)"`
}

func (PharmaEmployeeModel) TableName() string { return "pharma_oa_employees" }

type PharmaProductModel struct {
	ID              string    `gorm:"column:id;type:varchar(64);primaryKey"`
	Code            string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_pharma_products_code"`
	Name            string    `gorm:"column:name;type:varchar(255);not null;index:idx_pharma_products_status_name,priority:2"`
	Specification   string    `gorm:"column:specification;type:varchar(255)"`
	DosageForm      string    `gorm:"column:dosage_form;type:varchar(128)"`
	Manufacturer    string    `gorm:"column:manufacturer;type:varchar(255)"`
	ApprovalNumber  string    `gorm:"column:approval_number;type:varchar(191);not null;uniqueIndex:uk_pharma_products_approval"`
	Status          string    `gorm:"column:status;type:varchar(32);not null;index:idx_pharma_products_status_name,priority:1"`
	DisableReason   string    `gorm:"column:disable_reason;type:varchar(512)"`
	TemperatureJSON string    `gorm:"column:temperature_json;type:text"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
	CreatedBy       string    `gorm:"column:created_by;type:varchar(64)"`
	UpdatedBy       string    `gorm:"column:updated_by;type:varchar(64)"`
}

func (PharmaProductModel) TableName() string { return "pharma_oa_products" }

type PharmaSupplierModel struct {
	ID                 string    `gorm:"column:id;type:varchar(64);primaryKey"`
	Code               string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_pharma_suppliers_code"`
	Name               string    `gorm:"column:name;type:varchar(255);not null;index:idx_pharma_suppliers_status_name,priority:2"`
	Rating             int       `gorm:"column:rating;not null"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;index:idx_pharma_suppliers_status_name,priority:1"`
	DisableReason      string    `gorm:"column:disable_reason;type:varchar(512)"`
	ContactsJSON       string    `gorm:"column:contacts_json;type:text"`
	QualificationsJSON string    `gorm:"column:qualifications_json;type:text"`
	CreatedAt          time.Time `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null"`
	CreatedBy          string    `gorm:"column:created_by;type:varchar(64)"`
	UpdatedBy          string    `gorm:"column:updated_by;type:varchar(64)"`
}

func (PharmaSupplierModel) TableName() string { return "pharma_oa_suppliers" }

type PharmaCustomerModel struct {
	ID                 string    `gorm:"column:id;type:varchar(64);primaryKey"`
	Code               string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_pharma_customers_code"`
	Name               string    `gorm:"column:name;type:varchar(255);not null;index:idx_pharma_customers_name"`
	Region             string    `gorm:"column:region;type:varchar(128)"`
	OrganizationID     string    `gorm:"column:organization_id;type:varchar(64);index:idx_pharma_customers_scope_status,priority:1"`
	OwnerID            string    `gorm:"column:owner_id;type:varchar(64);index:idx_pharma_customers_scope_status,priority:2"`
	Rating             int       `gorm:"column:rating;not null"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;index:idx_pharma_customers_scope_status,priority:3"`
	DisableReason      string    `gorm:"column:disable_reason;type:varchar(512)"`
	ContactsJSON       string    `gorm:"column:contacts_json;type:text"`
	QualificationsJSON string    `gorm:"column:qualifications_json;type:text"`
	CreatedAt          time.Time `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null"`
	CreatedBy          string    `gorm:"column:created_by;type:varchar(64)"`
	UpdatedBy          string    `gorm:"column:updated_by;type:varchar(64)"`
}

func (PharmaCustomerModel) TableName() string { return "pharma_oa_customers" }

type PharmaWarehouseModel struct {
	ID              string    `gorm:"column:id;type:varchar(64);primaryKey"`
	Code            string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_pharma_warehouses_code"`
	Name            string    `gorm:"column:name;type:varchar(255);not null;index:idx_pharma_warehouses_status_name,priority:2"`
	Region          string    `gorm:"column:region;type:varchar(128)"`
	Status          string    `gorm:"column:status;type:varchar(32);not null;index:idx_pharma_warehouses_status_name,priority:1"`
	DisableReason   string    `gorm:"column:disable_reason;type:varchar(512)"`
	TemperatureJSON string    `gorm:"column:temperature_json;type:text"`
	AreasJSON       string    `gorm:"column:areas_json;type:text"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
	CreatedBy       string    `gorm:"column:created_by;type:varchar(64)"`
	UpdatedBy       string    `gorm:"column:updated_by;type:varchar(64)"`
}

func (PharmaWarehouseModel) TableName() string { return "pharma_oa_warehouses" }

func pharmaEmployeeModelFromDomain(item domainpharma.Employee) (PharmaEmployeeModel, error) {
	certificates, err := encodePharmaJSON(item.Certificates)
	return PharmaEmployeeModel{
		ID: item.ID.String(), Code: item.Code, Name: item.Name, DepartmentID: item.DepartmentID, PositionID: item.PositionID,
		Phone: item.Phone, Email: item.Email, Status: string(item.Status), LeaveReason: item.LeaveReason,
		CertificatesJSON: certificates, CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, err
}

func (m PharmaEmployeeModel) toDomain() (*domainpharma.Employee, error) {
	var certificates []domainpharma.EmployeeCertificate
	if err := decodePharmaJSON(m.CertificatesJSON, &certificates); err != nil {
		return nil, fmt.Errorf("decode employee certificates: %w", err)
	}
	return &domainpharma.Employee{
		ID: shared.ID(m.ID), Code: m.Code, Name: m.Name, DepartmentID: m.DepartmentID, PositionID: m.PositionID,
		Phone: m.Phone, Email: m.Email, Status: domainpharma.EmployeeStatus(m.Status), LeaveReason: m.LeaveReason,
		Certificates: certificates, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
	}, nil
}

func pharmaProductModelFromDomain(item domainpharma.Product) (PharmaProductModel, error) {
	temperature, err := encodePharmaJSON(item.Temperature)
	return PharmaProductModel{
		ID: item.ID.String(), Code: item.Code, Name: item.Name, Specification: item.Spec, DosageForm: item.DosageForm,
		Manufacturer: item.Manufacturer, ApprovalNumber: item.ApprovalNumber, Status: string(item.Status),
		DisableReason: item.DisableReason, TemperatureJSON: temperature, CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, err
}

func (m PharmaProductModel) toDomain() (*domainpharma.Product, error) {
	var temperature domainpharma.ProductTemperature
	if err := decodePharmaJSON(m.TemperatureJSON, &temperature); err != nil {
		return nil, fmt.Errorf("decode product temperature: %w", err)
	}
	return &domainpharma.Product{
		ID: shared.ID(m.ID), Code: m.Code, Name: m.Name, Spec: m.Specification, DosageForm: m.DosageForm,
		Manufacturer: m.Manufacturer, ApprovalNumber: m.ApprovalNumber, Temperature: temperature,
		Status: domainpharma.ProductStatus(m.Status), DisableReason: m.DisableReason,
		Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
	}, nil
}

func pharmaSupplierModelFromDomain(item domainpharma.Supplier) (PharmaSupplierModel, error) {
	contacts, err := encodePharmaJSON(item.Contacts)
	if err != nil {
		return PharmaSupplierModel{}, err
	}
	qualifications, err := encodePharmaJSON(item.Qualifications)
	return PharmaSupplierModel{
		ID: item.ID.String(), Code: item.Code, Name: item.Name, Rating: item.Rating, Status: string(item.Status),
		DisableReason: item.DisableReason, ContactsJSON: contacts, QualificationsJSON: qualifications,
		CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, err
}

func (m PharmaSupplierModel) toDomain() (*domainpharma.Supplier, error) {
	var contacts []domainpharma.SupplierContact
	var qualifications []domainpharma.SupplierQualification
	if err := decodePharmaJSON(m.ContactsJSON, &contacts); err != nil {
		return nil, fmt.Errorf("decode supplier contacts: %w", err)
	}
	if err := decodePharmaJSON(m.QualificationsJSON, &qualifications); err != nil {
		return nil, fmt.Errorf("decode supplier qualifications: %w", err)
	}
	return &domainpharma.Supplier{
		ID: shared.ID(m.ID), Code: m.Code, Name: m.Name, Rating: m.Rating, Status: domainpharma.SupplierStatus(m.Status),
		DisableReason: m.DisableReason, Contacts: contacts, Qualifications: qualifications,
		Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
	}, nil
}

func pharmaCustomerModelFromDomain(item domainpharma.Customer) (PharmaCustomerModel, error) {
	contacts, err := encodePharmaJSON(item.Contacts)
	if err != nil {
		return PharmaCustomerModel{}, err
	}
	qualifications, err := encodePharmaJSON(item.Qualifications)
	return PharmaCustomerModel{
		ID: item.ID.String(), Code: item.Code, Name: item.Name, Region: item.Region, OrganizationID: item.OrganizationID,
		OwnerID: item.OwnerID, Rating: item.Rating, Status: string(item.Status), DisableReason: item.DisableReason,
		ContactsJSON: contacts, QualificationsJSON: qualifications, CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, err
}

func (m PharmaCustomerModel) toDomain() (*domainpharma.Customer, error) {
	var contacts []domainpharma.CustomerContact
	var qualifications []domainpharma.CustomerQualification
	if err := decodePharmaJSON(m.ContactsJSON, &contacts); err != nil {
		return nil, fmt.Errorf("decode customer contacts: %w", err)
	}
	if err := decodePharmaJSON(m.QualificationsJSON, &qualifications); err != nil {
		return nil, fmt.Errorf("decode customer qualifications: %w", err)
	}
	return &domainpharma.Customer{
		ID: shared.ID(m.ID), Code: m.Code, Name: m.Name, Region: m.Region, OrganizationID: m.OrganizationID,
		OwnerID: m.OwnerID, Rating: m.Rating, Status: domainpharma.CustomerStatus(m.Status), DisableReason: m.DisableReason,
		Contacts: contacts, Qualifications: qualifications, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
	}, nil
}

func pharmaWarehouseModelFromDomain(item domainpharma.Warehouse) (PharmaWarehouseModel, error) {
	temperature, err := encodePharmaJSON(item.Temperature)
	if err != nil {
		return PharmaWarehouseModel{}, err
	}
	areas, err := encodePharmaJSON(item.Areas)
	return PharmaWarehouseModel{
		ID: item.ID.String(), Code: item.Code, Name: item.Name, Region: item.Region, Status: string(item.Status),
		DisableReason: item.DisableReason, TemperatureJSON: temperature, AreasJSON: areas,
		CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, err
}

func (m PharmaWarehouseModel) toDomain() (*domainpharma.Warehouse, error) {
	var temperature domainpharma.WarehouseTemperature
	var areas []domainpharma.WarehouseArea
	if err := decodePharmaJSON(m.TemperatureJSON, &temperature); err != nil {
		return nil, fmt.Errorf("decode warehouse temperature: %w", err)
	}
	if err := decodePharmaJSON(m.AreasJSON, &areas); err != nil {
		return nil, fmt.Errorf("decode warehouse areas: %w", err)
	}
	return &domainpharma.Warehouse{
		ID: shared.ID(m.ID), Code: m.Code, Name: m.Name, Region: m.Region, Status: domainpharma.WarehouseStatus(m.Status),
		DisableReason: m.DisableReason, Temperature: temperature, Areas: areas,
		Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
	}, nil
}

func encodePharmaJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	return string(raw), err
}

func decodePharmaJSON(raw string, target any) error {
	if raw == "" || raw == "null" {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}
