package pharmaoa

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

type InventoryOperationService interface {
	CreateStocktake(ctx context.Context, in StocktakeOrderCreateInput) (*domainpharma.StocktakeOrder, error)
	ApproveStocktake(ctx context.Context, id string, in InventoryApprovalInput) (*domainpharma.StocktakeOrder, error)
	RejectStocktake(ctx context.Context, id string, in InventoryApprovalInput) (*domainpharma.StocktakeOrder, error)
	ListStocktakes(ctx context.Context) ([]*domainpharma.StocktakeOrder, error)
	GetStocktake(ctx context.Context, id string) (*domainpharma.StocktakeOrder, error)
	CreateTransfer(ctx context.Context, in TransferOrderCreateInput) (*domainpharma.TransferOrder, error)
	ListTransfers(ctx context.Context) ([]*domainpharma.TransferOrder, error)
	GetTransfer(ctx context.Context, id string) (*domainpharma.TransferOrder, error)
}

type StocktakeOrderCreateInput struct {
	Number, ProductID, WarehouseID, AreaID, LocationID, BatchID string
	ActualQuantity                                              int
	Reason, CreatorID, ApproverID                               string
}
type InventoryApprovalInput struct{ ActorID, Comment string }
type TransferOrderCreateInput struct {
	Number, ProductID, BatchID                                                                  string
	Quantity                                                                                    int
	FromWarehouseID, FromAreaID, FromLocationID, ToWarehouseID, ToAreaID, ToLocationID, ActorID string
}
type InventoryOperationRepositories struct {
	Stocktakes pharmaoarepo.StocktakeRepository
	Transfers  pharmaoarepo.TransferRepository
}

type inventoryOperationService struct {
	stocktakeCreateMu                 sync.Mutex
	transferCreateMu                  sync.Mutex
	approvalMu                        sync.Mutex
	stocktakes                        pharmaoarepo.StocktakeRepository
	transfers                         pharmaoarepo.TransferRepository
	inventory                         InventoryService
	warehouses                        WarehouseService
	workflow                          workflowsvc.Service
	audit                             auditsvc.Service
	nowFn                             func() time.Time
	stocktakeCounter, transferCounter atomic.Int64
}

func NewInventoryOperationService(inventory InventoryService, warehouses WarehouseService, workflow workflowsvc.Service, audit auditsvc.Service, repositories ...InventoryOperationRepositories) InventoryOperationService {
	stocktakes := pharmaoarepo.StocktakeRepository(pharmaoarepo.NewMemoryStocktakeRepository())
	transfers := pharmaoarepo.TransferRepository(pharmaoarepo.NewMemoryTransferRepository())
	if len(repositories) > 0 {
		if repositories[0].Stocktakes != nil {
			stocktakes = repositories[0].Stocktakes
		}
		if repositories[0].Transfers != nil {
			transfers = repositories[0].Transfers
		}
	}
	return &inventoryOperationService{stocktakes: stocktakes, transfers: transfers, inventory: inventory, warehouses: warehouses, workflow: workflow, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *inventoryOperationService) CreateStocktake(ctx context.Context, in StocktakeOrderCreateInput) (*domainpharma.StocktakeOrder, error) {
	if s == nil || s.inventory == nil || s.warehouses == nil || s.workflow == nil {
		return nil, fmt.Errorf("inventory operation dependencies are required")
	}
	s.stocktakeCreateMu.Lock()
	defer s.stocktakeCreateMu.Unlock()
	if strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.ProductID) == "" || strings.TrimSpace(in.WarehouseID) == "" || strings.TrimSpace(in.AreaID) == "" || strings.TrimSpace(in.LocationID) == "" || strings.TrimSpace(in.BatchID) == "" || strings.TrimSpace(in.CreatorID) == "" || strings.TrimSpace(in.ApproverID) == "" || in.ActualQuantity < 0 {
		return nil, fmt.Errorf("stocktake order input is incomplete")
	}
	if err := s.validateLocation(ctx, in.WarehouseID, in.AreaID, in.LocationID); err != nil {
		return nil, err
	}
	balance, err := s.findBalance(ctx, in.ProductID, in.WarehouseID, in.AreaID, in.LocationID, in.BatchID)
	if err != nil {
		return nil, err
	}
	if in.ActualQuantity == balance.Quantity {
		return nil, fmt.Errorf("stocktake order requires a non-zero difference")
	}
	exists, err := s.stocktakeNumberExists(ctx, in.Number)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("stocktake number already exists")
	}
	sequence, err := s.nextStocktakeSequence(ctx)
	if err != nil {
		return nil, err
	}
	now := s.nowFn()
	orderID := shared.ID("stocktake-order-" + strconv.FormatInt(sequence, 10))
	definitionID := shared.ID("stocktake-definition-" + strconv.FormatInt(sequence, 10))
	workflowID := shared.ID("stocktake-workflow-" + strconv.FormatInt(sequence, 10))
	definition, err := s.workflow.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{ID: definitionID, Key: "pharma.stocktake." + strconv.FormatInt(sequence, 10), Name: "Stocktake Difference Approval", Version: 1, Now: now, Nodes: []domainworkflow.Node{{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart}, {ID: "approval", Key: "approval", Name: "Stocktake Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{shared.ID(strings.TrimSpace(in.ApproverID))}}, {ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd}}, Transitions: []domainworkflow.Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}}})
	if err != nil {
		return nil, err
	}
	if _, err = s.workflow.PublishDefinition(ctx, definition.ID, now); err != nil {
		return nil, err
	}
	instance, err := s.workflow.Start(ctx, workflowsvc.StartInput{ID: workflowID, DefinitionID: definition.ID, BusinessType: "pharma_oa.stocktake_order", BusinessID: orderID.String(), Title: "Stocktake " + strings.TrimSpace(in.Number), Starter: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.CreatorID))}, Now: now})
	if err != nil {
		return nil, err
	}
	position := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID, BatchID: in.BatchID}
	item, err := domainpharma.NewStocktakeOrder(orderID, in.Number, position, balance.Quantity, in.ActualQuantity, in.Reason, in.CreatorID, in.ApproverID, instance.ID.String(), now)
	if err != nil {
		return nil, err
	}
	if err := s.stocktakes.Create(ctx, item); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.CreatorID, "pharma_oa.stocktake.create", "pharma_oa_stocktake", item.ID.String(), map[string]any{"difference": item.Difference, "workflowInstanceId": item.WorkflowInstanceID})
	return cloneStocktakeOrder(item), nil
}

func (s *inventoryOperationService) ApproveStocktake(ctx context.Context, id string, in InventoryApprovalInput) (*domainpharma.StocktakeOrder, error) {
	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	item, err := s.GetStocktake(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status == domainpharma.StocktakeApproved {
		return item, nil
	}
	if item.Status != domainpharma.StocktakePendingApproval {
		return nil, fmt.Errorf("stocktake is not pending approval")
	}
	current, err := s.findBalance(ctx, item.ProductID, item.WarehouseID, item.AreaID, item.LocationID, item.BatchID)
	if err != nil {
		return nil, err
	}
	if current.Quantity != item.SystemQuantity {
		return nil, fmt.Errorf("stock balance changed after stocktake creation")
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(item.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("stocktake approval task not found")
	}
	now := s.nowFn()
	approved, err := s.workflow.Approve(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Comment, Now: now})
	if err != nil {
		return nil, err
	}
	if approved.Status != domainworkflow.InstanceApproved {
		return nil, fmt.Errorf("stocktake workflow is not approved")
	}
	result, err := s.inventory.Stocktake(ctx, StocktakeInput{ReferenceID: item.ID.String(), ProductID: item.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchID: item.BatchID, ActualQuantity: item.ActualQuantity, ActorID: in.ActorID})
	if err != nil {
		return nil, err
	}
	if err = item.Approve(in.ActorID, result.Ledger.ID.String(), now); err != nil {
		return nil, err
	}
	if err = s.stocktakes.Upsert(ctx, item); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.stocktake.approve", "pharma_oa_stocktake", item.ID.String(), map[string]any{"difference": item.Difference, "ledgerId": item.LedgerID})
	return cloneStocktakeOrder(item), nil
}

func (s *inventoryOperationService) RejectStocktake(ctx context.Context, id string, in InventoryApprovalInput) (*domainpharma.StocktakeOrder, error) {
	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	item, err := s.GetStocktake(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status == domainpharma.StocktakeRejected {
		return item, nil
	}
	if item.Status != domainpharma.StocktakePendingApproval {
		return nil, fmt.Errorf("stocktake is not pending approval")
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(item.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("stocktake approval task not found")
	}
	now := s.nowFn()
	rejected, err := s.workflow.Reject(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Comment, Now: now})
	if err != nil {
		return nil, err
	}
	if rejected.Status != domainworkflow.InstanceRejected {
		return nil, fmt.Errorf("stocktake workflow is not rejected")
	}
	if err = item.Reject(in.ActorID, now); err != nil {
		return nil, err
	}
	if err = s.stocktakes.Upsert(ctx, item); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.stocktake.reject", "pharma_oa_stocktake", item.ID.String(), map[string]any{"difference": item.Difference, "comment": strings.TrimSpace(in.Comment)})
	return cloneStocktakeOrder(item), nil
}
func (s *inventoryOperationService) ListStocktakes(ctx context.Context) ([]*domainpharma.StocktakeOrder, error) {
	rows, err := s.stocktakes.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]*domainpharma.StocktakeOrder, 0, len(rows))
	for index := range rows {
		out = append(out, cloneStocktakeOrder(&rows[index]))
	}
	return out, nil
}
func (s *inventoryOperationService) GetStocktake(ctx context.Context, id string) (*domainpharma.StocktakeOrder, error) {
	item, err := s.stocktakes.Get(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("stocktake not found")
	}
	return item, nil
}

func (s *inventoryOperationService) CreateTransfer(ctx context.Context, in TransferOrderCreateInput) (*domainpharma.TransferOrder, error) {
	if s == nil || s.inventory == nil || s.warehouses == nil {
		return nil, fmt.Errorf("inventory operation dependencies are required")
	}
	s.transferCreateMu.Lock()
	defer s.transferCreateMu.Unlock()
	if strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.ProductID) == "" || strings.TrimSpace(in.BatchID) == "" || in.Quantity <= 0 || strings.TrimSpace(in.FromWarehouseID) == "" || strings.TrimSpace(in.FromAreaID) == "" || strings.TrimSpace(in.FromLocationID) == "" || strings.TrimSpace(in.ToWarehouseID) == "" || strings.TrimSpace(in.ToAreaID) == "" || strings.TrimSpace(in.ToLocationID) == "" || strings.TrimSpace(in.ActorID) == "" {
		return nil, fmt.Errorf("transfer order input is incomplete")
	}
	if err := s.validateLocation(ctx, in.FromWarehouseID, in.FromAreaID, in.FromLocationID); err != nil {
		return nil, err
	}
	if err := s.validateLocation(ctx, in.ToWarehouseID, in.ToAreaID, in.ToLocationID); err != nil {
		return nil, err
	}
	if salesStockKey(in.ProductID, in.FromWarehouseID, in.FromAreaID, in.FromLocationID, in.BatchID) == salesStockKey(in.ProductID, in.ToWarehouseID, in.ToAreaID, in.ToLocationID, in.BatchID) {
		return nil, fmt.Errorf("transfer source and destination must differ")
	}
	exists, err := s.transferNumberExists(ctx, in.Number)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("transfer number already exists")
	}
	sequence, err := s.nextTransferSequence(ctx)
	if err != nil {
		return nil, err
	}
	id := shared.ID("transfer-order-" + strconv.FormatInt(sequence, 10))
	result, err := s.inventory.Transfer(ctx, StockTransferInput{ReferenceID: id.String(), IdempotencyKey: "transfer:" + strings.ToLower(strings.TrimSpace(in.Number)), ProductID: in.ProductID, FromWarehouseID: in.FromWarehouseID, FromAreaID: in.FromAreaID, FromLocationID: in.FromLocationID, ToWarehouseID: in.ToWarehouseID, ToAreaID: in.ToAreaID, ToLocationID: in.ToLocationID, BatchID: in.BatchID, Quantity: in.Quantity, ActorID: in.ActorID})
	if err != nil {
		return nil, err
	}
	if len(result.Ledgers) != 2 {
		return nil, fmt.Errorf("transfer must produce paired ledger entries")
	}
	snapshot := domainpharma.StockTransferInputSnapshot{ProductID: in.ProductID, BatchID: in.BatchID, Quantity: in.Quantity, FromWarehouseID: in.FromWarehouseID, FromAreaID: in.FromAreaID, FromLocationID: in.FromLocationID, ToWarehouseID: in.ToWarehouseID, ToAreaID: in.ToAreaID, ToLocationID: in.ToLocationID}
	item, err := domainpharma.NewTransferOrder(id, in.Number, snapshot, result.Ledgers[0].ID.String(), result.Ledgers[1].ID.String(), in.ActorID, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err := s.transfers.Create(ctx, item); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.transfer.create", "pharma_oa_transfer", item.ID.String(), map[string]any{"quantity": item.Quantity, "outLedgerId": item.OutLedgerID, "inLedgerId": item.InLedgerID})
	return cloneTransferOrder(item), nil
}
func (s *inventoryOperationService) ListTransfers(ctx context.Context) ([]*domainpharma.TransferOrder, error) {
	rows, err := s.transfers.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]*domainpharma.TransferOrder, 0, len(rows))
	for index := range rows {
		out = append(out, cloneTransferOrder(&rows[index]))
	}
	return out, nil
}
func (s *inventoryOperationService) GetTransfer(ctx context.Context, id string) (*domainpharma.TransferOrder, error) {
	item, err := s.transfers.Get(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("transfer not found")
	}
	return item, nil
}
func (s *inventoryOperationService) validateLocation(ctx context.Context, warehouseID, areaID, locationID string) error {
	eligible, err := s.warehouses.ValidateMovementLocation(ctx, WarehouseMovementLocationInput{WarehouseID: warehouseID, AreaID: areaID, LocationID: locationID})
	if err != nil {
		return err
	}
	if !eligible.Allowed {
		return fmt.Errorf("warehouse location is not eligible: %s", eligible.Reason)
	}
	return nil
}
func (s *inventoryOperationService) findBalance(ctx context.Context, productID, warehouseID, areaID, locationID, batchID string) (domainpharma.StockBalance, error) {
	items, err := s.inventory.ListBalances(ctx)
	if err != nil {
		return domainpharma.StockBalance{}, err
	}
	key := salesStockKey(productID, warehouseID, areaID, locationID, batchID)
	for _, item := range items {
		if salesStockKey(item.ProductID, item.WarehouseID, item.AreaID, item.LocationID, item.BatchID) == key {
			return item, nil
		}
	}
	return domainpharma.StockBalance{}, fmt.Errorf("stock balance not found")
}
func (s *inventoryOperationService) stocktakeNumberExists(ctx context.Context, number string) (bool, error) {
	items, err := s.stocktakes.List(ctx, pharmaoarepo.ListFilter{Keyword: strings.TrimSpace(number)})
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true, nil
		}
	}
	return false, nil
}
func (s *inventoryOperationService) transferNumberExists(ctx context.Context, number string) (bool, error) {
	items, err := s.transfers.List(ctx, pharmaoarepo.ListFilter{Keyword: strings.TrimSpace(number)})
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true, nil
		}
	}
	return false, nil
}
func (s *inventoryOperationService) nextStocktakeSequence(ctx context.Context) (int64, error) {
	for {
		sequence := s.stocktakeCounter.Add(1)
		item, err := s.stocktakes.Get(ctx, shared.ID("stocktake-order-"+strconv.FormatInt(sequence, 10)))
		if err != nil {
			return 0, err
		}
		if item == nil {
			return sequence, nil
		}
	}
}
func (s *inventoryOperationService) nextTransferSequence(ctx context.Context) (int64, error) {
	for {
		sequence := s.transferCounter.Add(1)
		item, err := s.transfers.Get(ctx, shared.ID("transfer-order-"+strconv.FormatInt(sequence, 10)))
		if err != nil {
			return 0, err
		}
		if item == nil {
			return sequence, nil
		}
	}
}
func cloneStocktakeOrder(item *domainpharma.StocktakeOrder) *domainpharma.StocktakeOrder {
	if item == nil {
		return nil
	}
	out := *item
	if item.CompletedAt != nil {
		completed := *item.CompletedAt
		out.CompletedAt = &completed
	}
	return &out
}
func cloneTransferOrder(item *domainpharma.TransferOrder) *domainpharma.TransferOrder {
	if item == nil {
		return nil
	}
	out := *item
	return &out
}
func (s *inventoryOperationService) appendAudit(ctx context.Context, actorID, action, resource, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "system"
	}
	_, _ = s.audit.Append(ctx, actorID, action, resource, resourceID, detail)
}
