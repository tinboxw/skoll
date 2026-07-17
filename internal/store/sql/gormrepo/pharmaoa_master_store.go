package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	"gorm.io/gorm"
)

type PharmaEmployeeStore struct{ db *gorm.DB }
type PharmaProductStore struct{ db *gorm.DB }
type PharmaSupplierStore struct{ db *gorm.DB }
type PharmaCustomerStore struct{ db *gorm.DB }
type PharmaWarehouseStore struct{ db *gorm.DB }

func NewPharmaEmployeeStore(db *gorm.DB) *PharmaEmployeeStore { return &PharmaEmployeeStore{db: db} }
func NewPharmaProductStore(db *gorm.DB) *PharmaProductStore   { return &PharmaProductStore{db: db} }
func NewPharmaSupplierStore(db *gorm.DB) *PharmaSupplierStore { return &PharmaSupplierStore{db: db} }
func NewPharmaCustomerStore(db *gorm.DB) *PharmaCustomerStore { return &PharmaCustomerStore{db: db} }
func NewPharmaWarehouseStore(db *gorm.DB) *PharmaWarehouseStore {
	return &PharmaWarehouseStore{db: db}
}

func (s *PharmaEmployeeStore) Create(ctx context.Context, item *domainpharma.Employee) error {
	if item == nil {
		return nil
	}
	row, err := pharmaEmployeeModelFromDomain(*item)
	return createPharmaModel(ctx, s.db, &row, err)
}

func (s *PharmaEmployeeStore) Upsert(ctx context.Context, item *domainpharma.Employee) error {
	if item == nil {
		return nil
	}
	row, err := pharmaEmployeeModelFromDomain(*item)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaEmployeeStore) Get(ctx context.Context, id shared.ID) (*domainpharma.Employee, error) {
	row, err := getPharmaModel[PharmaEmployeeModel](ctx, s.db, "id = ?", id.String())
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaEmployeeStore) GetByCode(ctx context.Context, code string) (*domainpharma.Employee, error) {
	row, err := getPharmaModel[PharmaEmployeeModel](ctx, s.db, "LOWER(code) = ?", strings.ToLower(strings.TrimSpace(code)))
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaEmployeeStore) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.Employee, error) {
	rows, err := listPharmaModels[PharmaEmployeeModel](ctx, s.db, filter, pharmaListOptions{
		keywordColumns: []string{"id", "code", "name", "department_id", "position_id"},
	})
	if err != nil {
		return nil, err
	}
	return convertPharmaRows(rows, func(row PharmaEmployeeModel) (*domainpharma.Employee, error) { return row.toDomain() })
}

func (s *PharmaProductStore) Create(ctx context.Context, item *domainpharma.Product) error {
	if item == nil {
		return nil
	}
	row, err := pharmaProductModelFromDomain(*item)
	return createPharmaModel(ctx, s.db, &row, err)
}

func (s *PharmaProductStore) Upsert(ctx context.Context, item *domainpharma.Product) error {
	if item == nil {
		return nil
	}
	row, err := pharmaProductModelFromDomain(*item)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaProductStore) Get(ctx context.Context, id shared.ID) (*domainpharma.Product, error) {
	row, err := getPharmaModel[PharmaProductModel](ctx, s.db, "id = ?", id.String())
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaProductStore) GetByCode(ctx context.Context, code string) (*domainpharma.Product, error) {
	row, err := getPharmaModel[PharmaProductModel](ctx, s.db, "LOWER(code) = ?", strings.ToLower(strings.TrimSpace(code)))
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaProductStore) GetByApprovalNumber(ctx context.Context, approvalNumber string) (*domainpharma.Product, error) {
	row, err := getPharmaModel[PharmaProductModel](ctx, s.db, "LOWER(approval_number) = ?", strings.ToLower(strings.TrimSpace(approvalNumber)))
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaProductStore) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.Product, error) {
	rows, err := listPharmaModels[PharmaProductModel](ctx, s.db, filter, pharmaListOptions{
		keywordColumns: []string{"id", "code", "name", "specification", "dosage_form", "manufacturer", "approval_number"},
	})
	if err != nil {
		return nil, err
	}
	return convertPharmaRows(rows, func(row PharmaProductModel) (*domainpharma.Product, error) { return row.toDomain() })
}

func (s *PharmaSupplierStore) Create(ctx context.Context, item *domainpharma.Supplier) error {
	if item == nil {
		return nil
	}
	row, err := pharmaSupplierModelFromDomain(*item)
	return createPharmaModel(ctx, s.db, &row, err)
}

func (s *PharmaSupplierStore) Upsert(ctx context.Context, item *domainpharma.Supplier) error {
	if item == nil {
		return nil
	}
	row, err := pharmaSupplierModelFromDomain(*item)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaSupplierStore) Get(ctx context.Context, id shared.ID) (*domainpharma.Supplier, error) {
	row, err := getPharmaModel[PharmaSupplierModel](ctx, s.db, "id = ?", id.String())
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaSupplierStore) GetByCode(ctx context.Context, code string) (*domainpharma.Supplier, error) {
	row, err := getPharmaModel[PharmaSupplierModel](ctx, s.db, "LOWER(code) = ?", strings.ToLower(strings.TrimSpace(code)))
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaSupplierStore) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.Supplier, error) {
	rows, err := listPharmaModels[PharmaSupplierModel](ctx, s.db, filter, pharmaListOptions{keywordColumns: []string{"id", "code", "name"}})
	if err != nil {
		return nil, err
	}
	return convertPharmaRows(rows, func(row PharmaSupplierModel) (*domainpharma.Supplier, error) { return row.toDomain() })
}

func (s *PharmaCustomerStore) Create(ctx context.Context, item *domainpharma.Customer) error {
	if item == nil {
		return nil
	}
	row, err := pharmaCustomerModelFromDomain(*item)
	return createPharmaModel(ctx, s.db, &row, err)
}

func (s *PharmaCustomerStore) Upsert(ctx context.Context, item *domainpharma.Customer) error {
	if item == nil {
		return nil
	}
	row, err := pharmaCustomerModelFromDomain(*item)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaCustomerStore) Get(ctx context.Context, id shared.ID) (*domainpharma.Customer, error) {
	row, err := getPharmaModel[PharmaCustomerModel](ctx, s.db, "id = ?", id.String())
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaCustomerStore) GetByCode(ctx context.Context, code string) (*domainpharma.Customer, error) {
	row, err := getPharmaModel[PharmaCustomerModel](ctx, s.db, "LOWER(code) = ?", strings.ToLower(strings.TrimSpace(code)))
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaCustomerStore) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.Customer, error) {
	rows, err := listPharmaModels[PharmaCustomerModel](ctx, s.db, filter, pharmaListOptions{
		keywordColumns: []string{"id", "code", "name", "region", "owner_id", "organization_id"}, regionColumn: "region",
		organizationColumn: "organization_id", ownerColumn: "owner_id",
	})
	if err != nil {
		return nil, err
	}
	return convertPharmaRows(rows, func(row PharmaCustomerModel) (*domainpharma.Customer, error) { return row.toDomain() })
}

func (s *PharmaWarehouseStore) Create(ctx context.Context, item *domainpharma.Warehouse) error {
	if item == nil {
		return nil
	}
	row, err := pharmaWarehouseModelFromDomain(*item)
	return createPharmaModel(ctx, s.db, &row, err)
}

func (s *PharmaWarehouseStore) Upsert(ctx context.Context, item *domainpharma.Warehouse) error {
	if item == nil {
		return nil
	}
	row, err := pharmaWarehouseModelFromDomain(*item)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaWarehouseStore) Get(ctx context.Context, id shared.ID) (*domainpharma.Warehouse, error) {
	row, err := getPharmaModel[PharmaWarehouseModel](ctx, s.db, "id = ?", id.String())
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaWarehouseStore) GetByCode(ctx context.Context, code string) (*domainpharma.Warehouse, error) {
	row, err := getPharmaModel[PharmaWarehouseModel](ctx, s.db, "LOWER(code) = ?", strings.ToLower(strings.TrimSpace(code)))
	if row == nil || err != nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaWarehouseStore) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.Warehouse, error) {
	rows, err := listPharmaModels[PharmaWarehouseModel](ctx, s.db, filter, pharmaListOptions{
		keywordColumns: []string{"id", "code", "name", "region", "areas_json"}, regionColumn: "region",
	})
	if err != nil {
		return nil, err
	}
	return convertPharmaRows(rows, func(row PharmaWarehouseModel) (*domainpharma.Warehouse, error) { return row.toDomain() })
}

type pharmaListOptions struct {
	keywordColumns     []string
	regionColumn       string
	organizationColumn string
	ownerColumn        string
}

func savePharmaModel[M any](ctx context.Context, db *gorm.DB, row *M, conversionErr error) error {
	if conversionErr != nil {
		return conversionErr
	}
	if db == nil {
		return fmt.Errorf("pharma OA database is required")
	}
	return withDBRetry(func() error { return db.WithContext(ctx).Save(row).Error })
}

func createPharmaModel[M any](ctx context.Context, db *gorm.DB, row *M, conversionErr error) error {
	if conversionErr != nil {
		return conversionErr
	}
	if db == nil {
		return fmt.Errorf("pharma OA database is required")
	}
	return withDBRetry(func() error { return db.WithContext(ctx).Create(row).Error })
}

func getPharmaModel[M any](ctx context.Context, db *gorm.DB, where string, value any) (*M, error) {
	if db == nil {
		return nil, fmt.Errorf("pharma OA database is required")
	}
	var row M
	err := withDBRetry(func() error { return db.WithContext(ctx).Where(where, value).First(&row).Error })
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func listPharmaModels[M any](ctx context.Context, db *gorm.DB, filter pharmaoarepo.ListFilter, options pharmaListOptions) ([]M, error) {
	if db == nil {
		return nil, fmt.Errorf("pharma OA database is required")
	}
	query := db.WithContext(ctx).Model(new(M)).Order("code asc")
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	if keyword != "" && len(options.keywordColumns) > 0 {
		conditions := make([]string, 0, len(options.keywordColumns))
		args := make([]any, 0, len(options.keywordColumns))
		for _, column := range options.keywordColumns {
			conditions = append(conditions, "LOWER("+column+") LIKE ?")
			args = append(args, "%"+keyword+"%")
		}
		query = query.Where("("+strings.Join(conditions, " OR ")+")", args...)
	}
	if status := strings.ToLower(strings.TrimSpace(filter.Status)); status != "" {
		query = query.Where("status = ?", status)
	}
	if region := strings.ToLower(strings.TrimSpace(filter.Region)); region != "" && options.regionColumn != "" {
		query = query.Where("LOWER("+options.regionColumn+") = ?", region)
	}
	organizationID := filter.OrganizationID.String()
	ownerID := filter.OwnerID.String()
	if filter.ScopeAny && (organizationID != "" || ownerID != "") {
		parts := make([]string, 0, 2)
		args := make([]any, 0, 2)
		if organizationID != "" && options.organizationColumn != "" {
			parts, args = append(parts, options.organizationColumn+" = ?"), append(args, organizationID)
		}
		if ownerID != "" && options.ownerColumn != "" {
			parts, args = append(parts, options.ownerColumn+" = ?"), append(args, ownerID)
		}
		if len(parts) > 0 {
			query = query.Where("("+strings.Join(parts, " OR ")+")", args...)
		}
	} else {
		if organizationID != "" && options.organizationColumn != "" {
			query = query.Where(options.organizationColumn+" = ?", organizationID)
		}
		if ownerID != "" && options.ownerColumn != "" {
			query = query.Where(options.ownerColumn+" = ?", ownerID)
		}
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var rows []M
	if err := withDBRetry(func() error { return query.Find(&rows).Error }); err != nil {
		return nil, err
	}
	return rows, nil
}

func convertPharmaRows[M any, T any](rows []M, convert func(M) (*T, error)) ([]T, error) {
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		item, err := convert(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}
