package main

import "time"

type Scope struct {
	TenantID       string `json:"tenantId"`
	OrganizationID string `json:"organizationId"`
	OwnerID        string `json:"ownerId"`
}

type Asset struct {
	ID string `json:"id"`
	Scope
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Category          string     `json:"category"`
	Model             string     `json:"model"`
	SerialNumber      string     `json:"serialNumber"`
	Location          string     `json:"location"`
	Status            string     `json:"status"`
	NextMaintenanceAt *time.Time `json:"nextMaintenanceAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type WorkOrder struct {
	ID string `json:"id"`
	Scope
	AssetID            string     `json:"assetId"`
	Number             string     `json:"number"`
	Title              string     `json:"title"`
	Priority           string     `json:"priority"`
	Status             string     `json:"status"`
	AssigneeID         string     `json:"assigneeId,omitempty"`
	WorkflowInstanceID string     `json:"workflowInstanceId,omitempty"`
	EstimatedCost      float64    `json:"estimatedCost"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	ClosedAt           *time.Time `json:"closedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type MaintenancePlan struct {
	ID string `json:"id"`
	Scope
	AssetID      string    `json:"assetId"`
	Name         string    `json:"name"`
	IntervalDays int       `json:"intervalDays"`
	NextRunAt    time.Time `json:"nextRunAt"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Inspection struct {
	ID string `json:"id"`
	Scope
	PlanID        string    `json:"planId"`
	AssetID       string    `json:"assetId"`
	WorkOrderID   string    `json:"workOrderId,omitempty"`
	Status        string    `json:"status"`
	Result        string    `json:"result,omitempty"`
	Finding       string    `json:"finding,omitempty"`
	AttachmentIDs []string  `json:"attachmentIds"`
	InspectedAt   time.Time `json:"inspectedAt"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SparePart struct {
	ID string `json:"id"`
	Scope
	SKU             string    `json:"sku"`
	Name            string    `json:"name"`
	Unit            string    `json:"unit"`
	Quantity        float64   `json:"quantity"`
	MinimumQuantity float64   `json:"minimumQuantity"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type SpareMovement struct {
	ID string `json:"id"`
	Scope
	SparePartID    string    `json:"sparePartId"`
	WorkOrderID    string    `json:"workOrderId,omitempty"`
	MovementType   string    `json:"movementType"`
	Quantity       float64   `json:"quantity"`
	IdempotencyKey string    `json:"idempotencyKey"`
	OccurredAt     time.Time `json:"occurredAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DeliveryRecord struct {
	DeliveryID string    `json:"deliveryId"`
	HandledAt  time.Time `json:"handledAt"`
}

type State struct {
	Sequence        uint64                     `json:"sequence"`
	Assets          map[string]Asset           `json:"assets"`
	WorkOrders      map[string]WorkOrder       `json:"workOrders"`
	Plans           map[string]MaintenancePlan `json:"plans"`
	Inspections     map[string]Inspection      `json:"inspections"`
	SpareParts      map[string]SparePart       `json:"spareParts"`
	SpareMovements  map[string]SpareMovement   `json:"spareMovements"`
	MovementKeys    map[string]string          `json:"movementKeys"`
	EventDeliveries map[string]DeliveryRecord  `json:"eventDeliveries"`
}

func newState() State {
	return State{
		Assets: make(map[string]Asset), WorkOrders: make(map[string]WorkOrder),
		Plans: make(map[string]MaintenancePlan), Inspections: make(map[string]Inspection),
		SpareParts: make(map[string]SparePart), SpareMovements: make(map[string]SpareMovement),
		MovementKeys: make(map[string]string), EventDeliveries: make(map[string]DeliveryRecord),
	}
}

func (s *State) normalize() {
	if s.Assets == nil {
		s.Assets = make(map[string]Asset)
	}
	if s.WorkOrders == nil {
		s.WorkOrders = make(map[string]WorkOrder)
	}
	if s.Plans == nil {
		s.Plans = make(map[string]MaintenancePlan)
	}
	if s.Inspections == nil {
		s.Inspections = make(map[string]Inspection)
	}
	if s.SpareParts == nil {
		s.SpareParts = make(map[string]SparePart)
	}
	if s.SpareMovements == nil {
		s.SpareMovements = make(map[string]SpareMovement)
	}
	if s.MovementKeys == nil {
		s.MovementKeys = make(map[string]string)
	}
	if s.EventDeliveries == nil {
		s.EventDeliveries = make(map[string]DeliveryRecord)
	}
}
