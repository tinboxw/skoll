package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PurchaseInboundStatus string

const PurchaseInboundCompleted PurchaseInboundStatus = "completed"

type InboundAttachment struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

type PurchaseInboundLine struct {
	ProductID      string    `json:"productId"`
	Quantity       int       `json:"quantity"`
	BatchNo        string    `json:"batchNo"`
	ProductionDate time.Time `json:"productionDate"`
	ExpiresAt      time.Time `json:"expiresAt"`
	BatchID        string    `json:"batchId,omitempty"`
	LedgerID       string    `json:"ledgerId,omitempty"`
}

type PurchaseInbound struct {
	ID              shared.ID             `json:"id"`
	Number          string                `json:"number"`
	PurchaseOrderID string                `json:"purchaseOrderId"`
	WarehouseID     string                `json:"warehouseId"`
	AreaID          string                `json:"areaId"`
	LocationID      string                `json:"locationId"`
	Lines           []PurchaseInboundLine `json:"lines"`
	Attachments     []InboundAttachment   `json:"attachments"`
	Status          PurchaseInboundStatus `json:"status"`
	ReceivedBy      string                `json:"receivedBy"`
	ReceivedAt      time.Time             `json:"receivedAt"`
	Meta            shared.AuditMeta      `json:"meta"`
}

func NewPurchaseInbound(id shared.ID, number, orderID, warehouseID, areaID, locationID, actorID string, lines []PurchaseInboundLine, attachments []InboundAttachment, now time.Time) (*PurchaseInbound, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || strings.TrimSpace(orderID) == "" || strings.TrimSpace(warehouseID) == "" || strings.TrimSpace(areaID) == "" || strings.TrimSpace(locationID) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("purchase inbound input is incomplete")
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("purchase inbound requires at least one line")
	}
	for _, line := range lines {
		if strings.TrimSpace(line.ProductID) == "" || line.Quantity <= 0 || strings.TrimSpace(line.BatchNo) == "" || line.ProductionDate.IsZero() || line.ExpiresAt.IsZero() || line.ExpiresAt.Before(line.ProductionDate) {
			return nil, fmt.Errorf("purchase inbound line is invalid")
		}
	}
	for _, attachment := range attachments {
		name := strings.TrimSpace(attachment.FileName)
		if strings.TrimSpace(attachment.FileID) == "" || name == "" || attachment.Size < 0 || strings.Contains(name, "..") || strings.ContainsAny(name, `/\\`) {
			return nil, fmt.Errorf("purchase inbound attachment is invalid")
		}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	item := &PurchaseInbound{ID: id, Number: strings.TrimSpace(number), PurchaseOrderID: strings.TrimSpace(orderID), WarehouseID: strings.TrimSpace(warehouseID), AreaID: strings.TrimSpace(areaID), LocationID: strings.TrimSpace(locationID), Lines: append([]PurchaseInboundLine(nil), lines...), Attachments: append([]InboundAttachment(nil), attachments...), Status: PurchaseInboundCompleted, ReceivedBy: strings.TrimSpace(actorID), ReceivedAt: now}
	item.Meta.Touch(now)
	return item, nil
}
