package gormrepo

import (
	"context"
	"errors"
	"fmt"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	"gorm.io/gorm"
	"strings"
)

type PharmaPurchaseStore struct{ db *gorm.DB }

func NewPharmaPurchaseStore(db *gorm.DB) *PharmaPurchaseStore { return &PharmaPurchaseStore{db: db} }
func (s *PharmaPurchaseStore) CreateRequest(ctx context.Context, v *domainpharma.PurchaseRequest) error {
	if v == nil {
		return fmt.Errorf("purchase request is required")
	}
	row, err := purchaseRequestModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaPurchaseStore) UpsertRequest(ctx context.Context, v *domainpharma.PurchaseRequest) error {
	if v == nil {
		return fmt.Errorf("purchase request is required")
	}
	row, err := purchaseRequestModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaPurchaseStore) GetRequest(ctx context.Context, id shared.ID) (*domainpharma.PurchaseRequest, error) {
	row, err := getOrderRow[PharmaPurchaseRequestModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaPurchaseStore) ListRequests(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.PurchaseRequest, error) {
	rows, err := listOrderRows[PharmaPurchaseRequestModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.PurchaseRequest, 0, len(rows))
	for _, row := range rows {
		v, e := row.toDomain()
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func (s *PharmaPurchaseStore) CreateOrder(ctx context.Context, v *domainpharma.PurchaseOrder) error {
	if v == nil {
		return fmt.Errorf("purchase order is required")
	}
	row, err := purchaseOrderModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaPurchaseStore) ApproveRequest(ctx context.Context, request *domainpharma.PurchaseRequest, order *domainpharma.PurchaseOrder) error {
	if request == nil || order == nil {
		return fmt.Errorf("purchase approval aggregates are required")
	}
	requestRow, err := purchaseRequestModelFromDomain(*request)
	if err != nil {
		return err
	}
	orderRow, err := purchaseOrderModelFromDomain(*order)
	if err != nil {
		return err
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&orderRow).Error; err != nil {
				return err
			}
			return tx.Save(&requestRow).Error
		})
	})
}
func (s *PharmaPurchaseStore) UpsertOrder(ctx context.Context, v *domainpharma.PurchaseOrder) error {
	if v == nil {
		return fmt.Errorf("purchase order is required")
	}
	row, err := purchaseOrderModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaPurchaseStore) GetOrder(ctx context.Context, id shared.ID) (*domainpharma.PurchaseOrder, error) {
	row, err := getOrderRow[PharmaPurchaseOrderModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaPurchaseStore) ListOrders(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.PurchaseOrder, error) {
	rows, err := listOrderRows[PharmaPurchaseOrderModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.PurchaseOrder, 0, len(rows))
	for _, row := range rows {
		v, e := row.toDomain()
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}

type PharmaPurchaseInboundStore struct{ db *gorm.DB }

func NewPharmaPurchaseInboundStore(db *gorm.DB) *PharmaPurchaseInboundStore {
	return &PharmaPurchaseInboundStore{db: db}
}
func (s *PharmaPurchaseInboundStore) Create(ctx context.Context, v *domainpharma.PurchaseInbound) error {
	if v == nil {
		return fmt.Errorf("purchase inbound is required")
	}
	row, err := purchaseInboundModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaPurchaseInboundStore) Upsert(ctx context.Context, v *domainpharma.PurchaseInbound) error {
	if v == nil {
		return fmt.Errorf("purchase inbound is required")
	}
	row, err := purchaseInboundModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaPurchaseInboundStore) Get(ctx context.Context, id shared.ID) (*domainpharma.PurchaseInbound, error) {
	row, err := getOrderRow[PharmaPurchaseInboundModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaPurchaseInboundStore) List(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.PurchaseInbound, error) {
	rows, err := listOrderRows[PharmaPurchaseInboundModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.PurchaseInbound, 0, len(rows))
	for _, row := range rows {
		v, e := row.toDomain()
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}

type PharmaSalesStore struct{ db *gorm.DB }

func NewPharmaSalesStore(db *gorm.DB) *PharmaSalesStore { return &PharmaSalesStore{db: db} }
func (s *PharmaSalesStore) CreateOrder(ctx context.Context, v *domainpharma.SalesOrder) error {
	if v == nil {
		return fmt.Errorf("sales order is required")
	}
	row, err := salesOrderModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaSalesStore) UpsertOrder(ctx context.Context, v *domainpharma.SalesOrder) error {
	if v == nil {
		return fmt.Errorf("sales order is required")
	}
	row, err := salesOrderModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaSalesStore) GetOrder(ctx context.Context, id shared.ID) (*domainpharma.SalesOrder, error) {
	row, err := getOrderRow[PharmaSalesOrderModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaSalesStore) ListOrders(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.SalesOrder, error) {
	rows, err := listOrderRows[PharmaSalesOrderModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.SalesOrder, 0, len(rows))
	for _, row := range rows {
		v, e := row.toDomain()
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func (s *PharmaSalesStore) CreateOutbound(ctx context.Context, v *domainpharma.SalesOutbound) error {
	if v == nil {
		return fmt.Errorf("sales outbound is required")
	}
	row, err := salesOutboundModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaSalesStore) UpsertOutbound(ctx context.Context, v *domainpharma.SalesOutbound) error {
	if v == nil {
		return fmt.Errorf("sales outbound is required")
	}
	row, err := salesOutboundModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, err)
}
func (s *PharmaSalesStore) GetOutbound(ctx context.Context, id shared.ID) (*domainpharma.SalesOutbound, error) {
	row, err := getOrderRow[PharmaSalesOutboundModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain()
}
func (s *PharmaSalesStore) ListOutbounds(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.SalesOutbound, error) {
	rows, err := listOrderRows[PharmaSalesOutboundModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.SalesOutbound, 0, len(rows))
	for _, row := range rows {
		v, e := row.toDomain()
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}

type PharmaStocktakeStore struct{ db *gorm.DB }

func NewPharmaStocktakeStore(db *gorm.DB) *PharmaStocktakeStore { return &PharmaStocktakeStore{db: db} }
func (s *PharmaStocktakeStore) Create(ctx context.Context, v *domainpharma.StocktakeOrder) error {
	if v == nil {
		return fmt.Errorf("stocktake is required")
	}
	row := stocktakeModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, nil)
}
func (s *PharmaStocktakeStore) Upsert(ctx context.Context, v *domainpharma.StocktakeOrder) error {
	if v == nil {
		return fmt.Errorf("stocktake is required")
	}
	row := stocktakeModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, nil)
}
func (s *PharmaStocktakeStore) Get(ctx context.Context, id shared.ID) (*domainpharma.StocktakeOrder, error) {
	row, err := getOrderRow[PharmaStocktakeModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain(), nil
}
func (s *PharmaStocktakeStore) List(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.StocktakeOrder, error) {
	rows, err := listOrderRows[PharmaStocktakeModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.StocktakeOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, *row.toDomain())
	}
	return out, nil
}

type PharmaTransferStore struct{ db *gorm.DB }

func NewPharmaTransferStore(db *gorm.DB) *PharmaTransferStore { return &PharmaTransferStore{db: db} }
func (s *PharmaTransferStore) Create(ctx context.Context, v *domainpharma.TransferOrder) error {
	if v == nil {
		return fmt.Errorf("transfer is required")
	}
	row := transferModelFromDomain(*v)
	return createPharmaModel(ctx, s.db, &row, nil)
}
func (s *PharmaTransferStore) Upsert(ctx context.Context, v *domainpharma.TransferOrder) error {
	if v == nil {
		return fmt.Errorf("transfer is required")
	}
	row := transferModelFromDomain(*v)
	return savePharmaModel(ctx, s.db, &row, nil)
}
func (s *PharmaTransferStore) Get(ctx context.Context, id shared.ID) (*domainpharma.TransferOrder, error) {
	row, err := getOrderRow[PharmaTransferModel](ctx, s.db, id.String())
	if err != nil || row == nil {
		return nil, err
	}
	return row.toDomain(), nil
}
func (s *PharmaTransferStore) List(ctx context.Context, f pharmaoarepo.ListFilter) ([]domainpharma.TransferOrder, error) {
	rows, err := listOrderRows[PharmaTransferModel](ctx, s.db, f)
	if err != nil {
		return nil, err
	}
	out := make([]domainpharma.TransferOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, *row.toDomain())
	}
	return out, nil
}

func getOrderRow[M any](ctx context.Context, db *gorm.DB, id string) (*M, error) {
	if db == nil {
		return nil, fmt.Errorf("pharma OA database is required")
	}
	var row M
	err := db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
func listOrderRows[M any](ctx context.Context, db *gorm.DB, f pharmaoarepo.ListFilter) ([]M, error) {
	if db == nil {
		return nil, fmt.Errorf("pharma OA database is required")
	}
	q := db.WithContext(ctx).Model(new(M)).Order("number asc")
	if k := strings.ToLower(strings.TrimSpace(f.Keyword)); k != "" {
		q = q.Where("LOWER(number) LIKE ?", "%"+k+"%")
	}
	if st := strings.TrimSpace(f.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	var rows []M
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
