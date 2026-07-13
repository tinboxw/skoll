package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type SalesOrderStatus string
type SalesOutboundStatus string

const (
	SalesOrderOpen         SalesOrderStatus    = "open"
	SalesOutboundCompleted SalesOutboundStatus = "completed"
)

type SalesLine struct {
	ProductID string  `json:"productId"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
	Amount    float64 `json:"amount"`
}

type SalesOrder struct {
	ID          shared.ID        `json:"id"`
	Number      string           `json:"number"`
	CustomerID  string           `json:"customerId"`
	Lines       []SalesLine      `json:"lines"`
	TotalAmount float64          `json:"totalAmount"`
	Status      SalesOrderStatus `json:"status"`
	CreatedBy   string           `json:"createdBy"`
	CreatedAt   time.Time        `json:"createdAt"`
	Meta        shared.AuditMeta `json:"meta"`
}

type SalesOutboundLine struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	BatchID   string `json:"batchId"`
	LedgerID  string `json:"ledgerId,omitempty"`
}

type SalesOutbound struct {
	ID           shared.ID           `json:"id"`
	Number       string              `json:"number"`
	SalesOrderID string              `json:"salesOrderId"`
	CustomerID   string              `json:"customerId"`
	WarehouseID  string              `json:"warehouseId"`
	AreaID       string              `json:"areaId"`
	LocationID   string              `json:"locationId"`
	Lines        []SalesOutboundLine `json:"lines"`
	Status       SalesOutboundStatus `json:"status"`
	ShippedBy    string              `json:"shippedBy"`
	ShippedAt    time.Time           `json:"shippedAt"`
	Meta         shared.AuditMeta    `json:"meta"`
}

func NewSalesOrder(id shared.ID, number, customerID, actorID string, lines []SalesLine, now time.Time) (*SalesOrder, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || strings.TrimSpace(customerID) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("sales order input is incomplete")
	}
	normalized, total, err := normalizeSalesLines(lines)
	if err != nil {
		return nil, err
	}
	now = normalizeSalesTime(now)
	item := &SalesOrder{ID: id, Number: strings.TrimSpace(number), CustomerID: strings.TrimSpace(customerID), Lines: normalized, TotalAmount: total, Status: SalesOrderOpen, CreatedBy: strings.TrimSpace(actorID), CreatedAt: now}
	item.Meta.Touch(now)
	return item, nil
}

func NewSalesOutbound(id shared.ID, number, orderID, customerID, warehouseID, areaID, locationID, actorID string, lines []SalesOutboundLine, now time.Time) (*SalesOutbound, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || strings.TrimSpace(orderID) == "" || strings.TrimSpace(customerID) == "" || strings.TrimSpace(warehouseID) == "" || strings.TrimSpace(areaID) == "" || strings.TrimSpace(locationID) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("sales outbound input is incomplete")
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("sales outbound requires at least one line")
	}
	for _, line := range lines {
		if strings.TrimSpace(line.ProductID) == "" || strings.TrimSpace(line.BatchID) == "" || line.Quantity <= 0 {
			return nil, fmt.Errorf("sales outbound line is invalid")
		}
	}
	now = normalizeSalesTime(now)
	item := &SalesOutbound{ID: id, Number: strings.TrimSpace(number), SalesOrderID: strings.TrimSpace(orderID), CustomerID: strings.TrimSpace(customerID), WarehouseID: strings.TrimSpace(warehouseID), AreaID: strings.TrimSpace(areaID), LocationID: strings.TrimSpace(locationID), Lines: append([]SalesOutboundLine(nil), lines...), Status: SalesOutboundCompleted, ShippedBy: strings.TrimSpace(actorID), ShippedAt: now}
	item.Meta.Touch(now)
	return item, nil
}

func normalizeSalesLines(lines []SalesLine) ([]SalesLine, float64, error) {
	if len(lines) == 0 {
		return nil, 0, fmt.Errorf("sales order requires at least one line")
	}
	out := make([]SalesLine, 0, len(lines))
	total := 0.0
	for _, line := range lines {
		line.ProductID = strings.TrimSpace(line.ProductID)
		if line.ProductID == "" || line.Quantity <= 0 || line.UnitPrice < 0 {
			return nil, 0, fmt.Errorf("sales order line is invalid")
		}
		line.Amount = float64(line.Quantity) * line.UnitPrice
		total += line.Amount
		out = append(out, line)
	}
	return out, total, nil
}

func normalizeSalesTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
