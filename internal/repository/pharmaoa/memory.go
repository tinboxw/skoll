package pharmaoa

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type memoryAggregate[T any] struct {
	mu             sync.RWMutex
	items          map[string]T
	code           func(T) string
	uniqueKeys     func(T) []string
	status         func(T) string
	search         func(T) string
	region         func(T) string
	organizationID func(T) string
	ownerID        func(T) string
}

func newMemoryAggregate[T any](code, status, search func(T) string) *memoryAggregate[T] {
	return &memoryAggregate[T]{
		items: make(map[string]T), code: code, uniqueKeys: func(item T) []string { return []string{code(item)} },
		status: status, search: search,
	}
}

func (r *memoryAggregate[T]) upsert(_ context.Context, id shared.ID, item T) error {
	copyItem, err := cloneJSON(item)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for existingID, existing := range r.items {
		if existingID == id.String() {
			continue
		}
		candidateKeys := r.uniqueKeys(item)
		currentKeys := r.uniqueKeys(existing)
		for index, candidate := range candidateKeys {
			if index < len(currentKeys) && strings.TrimSpace(candidate) != "" && strings.EqualFold(candidate, currentKeys[index]) {
				return fmt.Errorf("master data unique value already exists")
			}
		}
	}
	r.items[id.String()] = copyItem
	return nil
}

func (r *memoryAggregate[T]) create(_ context.Context, id shared.ID, item T) error {
	copyItem, err := cloneJSON(item)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[id.String()]; exists {
		return fmt.Errorf("master data id already exists")
	}
	for _, existing := range r.items {
		candidateKeys := r.uniqueKeys(item)
		currentKeys := r.uniqueKeys(existing)
		for index, candidate := range candidateKeys {
			if index < len(currentKeys) && strings.TrimSpace(candidate) != "" && strings.EqualFold(candidate, currentKeys[index]) {
				return fmt.Errorf("master data unique value already exists")
			}
		}
	}
	r.items[id.String()] = copyItem
	return nil
}

func (r *memoryAggregate[T]) get(_ context.Context, id shared.ID) (*T, error) {
	r.mu.RLock()
	item, ok := r.items[id.String()]
	r.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	copyItem, err := cloneJSON(item)
	return &copyItem, err
}

func (r *memoryAggregate[T]) getByCode(_ context.Context, code string) (*T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if strings.EqualFold(r.code(item), strings.TrimSpace(code)) {
			copyItem, err := cloneJSON(item)
			return &copyItem, err
		}
	}
	return nil, nil
}

func (r *memoryAggregate[T]) list(_ context.Context, filter ListFilter) ([]T, error) {
	page, err := r.listPage(filter)
	return page.Items, err
}

func (r *memoryAggregate[T]) listPage(filter ListFilter) (ListPage[T], error) {
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	region := strings.ToLower(strings.TrimSpace(filter.Region))
	organizationIDs := filter.NormalizedOrganizationIDs()
	organizationSet := make(map[string]struct{}, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		organizationSet[organizationID] = struct{}{}
	}
	ownerID := filter.OwnerID.String()

	r.mu.RLock()
	items := make([]T, 0, len(r.items))
	for _, item := range r.items {
		if status != "" && strings.ToLower(r.status(item)) != status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(r.search(item)), keyword) {
			continue
		}
		if region != "" && (r.region == nil || strings.ToLower(r.region(item)) != region) {
			continue
		}
		_, organizationMatch := organizationSet[organizationValue(r, item)]
		ownerMatch := ownerID != "" && r.ownerID != nil && r.ownerID(item) == ownerID
		if filter.ScopeAny {
			if (len(organizationIDs) > 0 || ownerID != "") && !organizationMatch && !ownerMatch {
				continue
			}
		} else if (len(organizationIDs) > 0 && !organizationMatch) || (ownerID != "" && !ownerMatch) {
			continue
		}
		copyItem, err := cloneJSON(item)
		if err != nil {
			r.mu.RUnlock()
			return ListPage[T]{}, err
		}
		items = append(items, copyItem)
	}
	r.mu.RUnlock()

	sort.SliceStable(items, func(i, j int) bool { return r.code(items[i]) < r.code(items[j]) })
	total := int64(len(items))
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return ListPage[T]{Items: []T{}, Total: total}, nil
	}
	end := len(items)
	if filter.Limit > 0 && offset+filter.Limit < end {
		end = offset + filter.Limit
	}
	return ListPage[T]{Items: items[offset:end], Total: total}, nil
}

func organizationValue[T any](repo *memoryAggregate[T], item T) string {
	if repo.organizationID == nil {
		return ""
	}
	return repo.organizationID(item)
}

type MemoryEmployeeRepository struct {
	*memoryAggregate[domainpharma.Employee]
}

func NewMemoryEmployeeRepository() *MemoryEmployeeRepository {
	return &MemoryEmployeeRepository{newMemoryAggregate(
		func(v domainpharma.Employee) string { return v.Code },
		func(v domainpharma.Employee) string { return string(v.Status) },
		func(v domainpharma.Employee) string {
			return strings.Join([]string{v.ID.String(), v.Code, v.Name, v.DepartmentID, v.PositionID}, " ")
		},
	)}
}

func (r *MemoryEmployeeRepository) Upsert(ctx context.Context, item *domainpharma.Employee) error {
	if item == nil {
		return nil
	}
	return r.upsert(ctx, item.ID, *item)
}
func (r *MemoryEmployeeRepository) Create(ctx context.Context, item *domainpharma.Employee) error {
	if item == nil {
		return nil
	}
	return r.create(ctx, item.ID, *item)
}
func (r *MemoryEmployeeRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.Employee, error) {
	return r.get(ctx, id)
}
func (r *MemoryEmployeeRepository) GetByCode(ctx context.Context, code string) (*domainpharma.Employee, error) {
	return r.getByCode(ctx, code)
}
func (r *MemoryEmployeeRepository) List(ctx context.Context, filter ListFilter) ([]domainpharma.Employee, error) {
	return r.list(ctx, filter)
}
func (r *MemoryEmployeeRepository) ListPage(_ context.Context, filter ListFilter) (ListPage[domainpharma.Employee], error) {
	return r.listPage(filter)
}

type MemoryProductRepository struct {
	*memoryAggregate[domainpharma.Product]
}

func NewMemoryProductRepository() *MemoryProductRepository {
	base := newMemoryAggregate(
		func(v domainpharma.Product) string { return v.Code },
		func(v domainpharma.Product) string { return string(v.Status) },
		func(v domainpharma.Product) string {
			return strings.Join([]string{v.ID.String(), v.Code, v.Name, v.Spec, v.DosageForm, v.Manufacturer, v.ApprovalNumber}, " ")
		},
	)
	base.uniqueKeys = func(v domainpharma.Product) []string { return []string{v.Code, v.ApprovalNumber} }
	return &MemoryProductRepository{base}
}

func (r *MemoryProductRepository) Upsert(ctx context.Context, item *domainpharma.Product) error {
	if item == nil {
		return nil
	}
	return r.upsert(ctx, item.ID, *item)
}
func (r *MemoryProductRepository) Create(ctx context.Context, item *domainpharma.Product) error {
	if item == nil {
		return nil
	}
	return r.create(ctx, item.ID, *item)
}
func (r *MemoryProductRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.Product, error) {
	return r.get(ctx, id)
}
func (r *MemoryProductRepository) GetByCode(ctx context.Context, code string) (*domainpharma.Product, error) {
	return r.getByCode(ctx, code)
}
func (r *MemoryProductRepository) GetByApprovalNumber(_ context.Context, approvalNumber string) (*domainpharma.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if strings.EqualFold(item.ApprovalNumber, strings.TrimSpace(approvalNumber)) {
			copyItem, err := cloneJSON(item)
			return &copyItem, err
		}
	}
	return nil, nil
}
func (r *MemoryProductRepository) List(ctx context.Context, filter ListFilter) ([]domainpharma.Product, error) {
	return r.list(ctx, filter)
}
func (r *MemoryProductRepository) ListPage(_ context.Context, filter ListFilter) (ListPage[domainpharma.Product], error) {
	return r.listPage(filter)
}

type MemorySupplierRepository struct {
	*memoryAggregate[domainpharma.Supplier]
}

func NewMemorySupplierRepository() *MemorySupplierRepository {
	return &MemorySupplierRepository{newMemoryAggregate(
		func(v domainpharma.Supplier) string { return v.Code },
		func(v domainpharma.Supplier) string { return string(v.Status) },
		func(v domainpharma.Supplier) string {
			return strings.Join([]string{v.ID.String(), v.Code, v.Name}, " ")
		},
	)}
}

func (r *MemorySupplierRepository) Upsert(ctx context.Context, item *domainpharma.Supplier) error {
	if item == nil {
		return nil
	}
	return r.upsert(ctx, item.ID, *item)
}
func (r *MemorySupplierRepository) Create(ctx context.Context, item *domainpharma.Supplier) error {
	if item == nil {
		return nil
	}
	return r.create(ctx, item.ID, *item)
}
func (r *MemorySupplierRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.Supplier, error) {
	return r.get(ctx, id)
}
func (r *MemorySupplierRepository) GetByCode(ctx context.Context, code string) (*domainpharma.Supplier, error) {
	return r.getByCode(ctx, code)
}
func (r *MemorySupplierRepository) List(ctx context.Context, filter ListFilter) ([]domainpharma.Supplier, error) {
	return r.list(ctx, filter)
}
func (r *MemorySupplierRepository) ListPage(_ context.Context, filter ListFilter) (ListPage[domainpharma.Supplier], error) {
	return r.listPage(filter)
}

type MemoryCustomerRepository struct {
	*memoryAggregate[domainpharma.Customer]
}

func NewMemoryCustomerRepository() *MemoryCustomerRepository {
	base := newMemoryAggregate(
		func(v domainpharma.Customer) string { return v.Code },
		func(v domainpharma.Customer) string { return string(v.Status) },
		func(v domainpharma.Customer) string {
			return strings.Join([]string{v.ID.String(), v.Code, v.Name, v.Region, v.OwnerID, v.OrganizationID}, " ")
		},
	)
	base.region = func(v domainpharma.Customer) string { return v.Region }
	base.organizationID = func(v domainpharma.Customer) string { return v.OrganizationID }
	base.ownerID = func(v domainpharma.Customer) string { return v.OwnerID }
	return &MemoryCustomerRepository{base}
}

func (r *MemoryCustomerRepository) Upsert(ctx context.Context, item *domainpharma.Customer) error {
	if item == nil {
		return nil
	}
	return r.upsert(ctx, item.ID, *item)
}
func (r *MemoryCustomerRepository) Create(ctx context.Context, item *domainpharma.Customer) error {
	if item == nil {
		return nil
	}
	return r.create(ctx, item.ID, *item)
}
func (r *MemoryCustomerRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.Customer, error) {
	return r.get(ctx, id)
}
func (r *MemoryCustomerRepository) GetByCode(ctx context.Context, code string) (*domainpharma.Customer, error) {
	return r.getByCode(ctx, code)
}
func (r *MemoryCustomerRepository) List(ctx context.Context, filter ListFilter) ([]domainpharma.Customer, error) {
	return r.list(ctx, filter)
}
func (r *MemoryCustomerRepository) ListPage(_ context.Context, filter ListFilter) (ListPage[domainpharma.Customer], error) {
	return r.listPage(filter)
}

type MemoryWarehouseRepository struct {
	*memoryAggregate[domainpharma.Warehouse]
}

func NewMemoryWarehouseRepository() *MemoryWarehouseRepository {
	base := newMemoryAggregate(
		func(v domainpharma.Warehouse) string { return v.Code },
		func(v domainpharma.Warehouse) string { return string(v.Status) },
		func(v domainpharma.Warehouse) string {
			return strings.Join([]string{v.ID.String(), v.Code, v.Name, v.Region, fmt.Sprint(v.Areas)}, " ")
		},
	)
	base.region = func(v domainpharma.Warehouse) string { return v.Region }
	return &MemoryWarehouseRepository{base}
}

func (r *MemoryWarehouseRepository) Upsert(ctx context.Context, item *domainpharma.Warehouse) error {
	if item == nil {
		return nil
	}
	return r.upsert(ctx, item.ID, *item)
}
func (r *MemoryWarehouseRepository) Create(ctx context.Context, item *domainpharma.Warehouse) error {
	if item == nil {
		return nil
	}
	return r.create(ctx, item.ID, *item)
}
func (r *MemoryWarehouseRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.Warehouse, error) {
	return r.get(ctx, id)
}
func (r *MemoryWarehouseRepository) GetByCode(ctx context.Context, code string) (*domainpharma.Warehouse, error) {
	return r.getByCode(ctx, code)
}
func (r *MemoryWarehouseRepository) List(ctx context.Context, filter ListFilter) ([]domainpharma.Warehouse, error) {
	return r.list(ctx, filter)
}
func (r *MemoryWarehouseRepository) ListPage(_ context.Context, filter ListFilter) (ListPage[domainpharma.Warehouse], error) {
	return r.listPage(filter)
}

func cloneJSON[T any](item T) (T, error) {
	var out T
	raw, err := json.Marshal(item)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}
