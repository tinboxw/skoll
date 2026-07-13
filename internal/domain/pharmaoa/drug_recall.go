package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type DrugRecallStatus string
type DrugRecallTaskStatus string

const (
	DrugRecallActive    DrugRecallStatus = "active"
	DrugRecallCompleted DrugRecallStatus = "completed"

	DrugRecallTaskPending   DrugRecallTaskStatus = "pending"
	DrugRecallTaskCompleted DrugRecallTaskStatus = "completed"
)

type DrugRecallScope struct {
	CustomerID   string   `json:"customerId"`
	CustomerName string   `json:"customerName"`
	OutboundIDs  []string `json:"outboundIds"`
	Quantity     int      `json:"quantity"`
}

type DrugRecallTask struct {
	ID             string               `json:"id"`
	CustomerID     string               `json:"customerId"`
	CustomerName   string               `json:"customerName"`
	OutboundIDs    []string             `json:"outboundIds"`
	Quantity       int                  `json:"quantity"`
	Status         DrugRecallTaskStatus `json:"status"`
	CompletionNote string               `json:"completionNote,omitempty"`
	CompletedBy    string               `json:"completedBy,omitempty"`
	CompletedAt    *time.Time           `json:"completedAt,omitempty"`
}

type DrugRecall struct {
	ID                shared.ID        `json:"id"`
	Number            string           `json:"number"`
	Title             string           `json:"title"`
	Reason            string           `json:"reason"`
	ProductID         string           `json:"productId"`
	ProductName       string           `json:"productName"`
	BatchID           string           `json:"batchId"`
	BatchNo           string           `json:"batchNo"`
	SourceComplaintID string           `json:"sourceComplaintId,omitempty"`
	Tasks             []DrugRecallTask `json:"tasks"`
	Status            DrugRecallStatus `json:"status"`
	InitiatedBy       string           `json:"initiatedBy"`
	InitiatedAt       time.Time        `json:"initiatedAt"`
	CompletedBy       string           `json:"completedBy,omitempty"`
	CompletedAt       *time.Time       `json:"completedAt,omitempty"`
	Meta              shared.AuditMeta `json:"meta"`
}

type DrugRecallInput struct {
	Number            string
	Title             string
	Reason            string
	ProductID         string
	ProductName       string
	BatchID           string
	BatchNo           string
	SourceComplaintID string
	InitiatedBy       string
	Scopes            []DrugRecallScope
}

func NewDrugRecall(id shared.ID, in DrugRecallInput, now time.Time) (*DrugRecall, error) {
	if id.IsZero() || strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Reason) == "" || strings.TrimSpace(in.ProductID) == "" || strings.TrimSpace(in.ProductName) == "" || strings.TrimSpace(in.BatchID) == "" || strings.TrimSpace(in.BatchNo) == "" || strings.TrimSpace(in.InitiatedBy) == "" {
		return nil, fmt.Errorf("drug recall input is incomplete")
	}
	if len(in.Scopes) == 0 {
		return nil, fmt.Errorf("drug recall requires at least one affected customer")
	}
	tasks := make([]DrugRecallTask, 0, len(in.Scopes))
	seenCustomers := map[string]struct{}{}
	for index, scope := range in.Scopes {
		customerID := strings.TrimSpace(scope.CustomerID)
		customerName := strings.TrimSpace(scope.CustomerName)
		if customerID == "" || customerName == "" || scope.Quantity <= 0 || len(scope.OutboundIDs) == 0 {
			return nil, fmt.Errorf("drug recall scope is incomplete")
		}
		if _, exists := seenCustomers[customerID]; exists {
			return nil, fmt.Errorf("drug recall customer scope is duplicated")
		}
		seenCustomers[customerID] = struct{}{}
		outboundIDs := make([]string, 0, len(scope.OutboundIDs))
		seenOutbound := map[string]struct{}{}
		for _, rawID := range scope.OutboundIDs {
			outboundID := strings.TrimSpace(rawID)
			if outboundID == "" {
				return nil, fmt.Errorf("drug recall outbound id is required")
			}
			if _, exists := seenOutbound[outboundID]; exists {
				continue
			}
			seenOutbound[outboundID] = struct{}{}
			outboundIDs = append(outboundIDs, outboundID)
		}
		tasks = append(tasks, DrugRecallTask{ID: fmt.Sprintf("%s-task-%d", id.String(), index+1), CustomerID: customerID, CustomerName: customerName, OutboundIDs: outboundIDs, Quantity: scope.Quantity, Status: DrugRecallTaskPending})
	}
	now = normalizeContractTime(now)
	item := &DrugRecall{ID: id, Number: strings.TrimSpace(in.Number), Title: strings.TrimSpace(in.Title), Reason: strings.TrimSpace(in.Reason), ProductID: strings.TrimSpace(in.ProductID), ProductName: strings.TrimSpace(in.ProductName), BatchID: strings.TrimSpace(in.BatchID), BatchNo: strings.TrimSpace(in.BatchNo), SourceComplaintID: strings.TrimSpace(in.SourceComplaintID), Tasks: tasks, Status: DrugRecallActive, InitiatedBy: strings.TrimSpace(in.InitiatedBy), InitiatedAt: now}
	item.Meta.Touch(now)
	return item, nil
}

func (r *DrugRecall) CompleteTask(taskID, actorID, note string, now time.Time) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("drug recall is required")
	}
	taskID, actorID, note = strings.TrimSpace(taskID), strings.TrimSpace(actorID), strings.TrimSpace(note)
	if taskID == "" || actorID == "" || note == "" {
		return false, fmt.Errorf("drug recall task, actor, and completion note are required")
	}
	for index := range r.Tasks {
		task := &r.Tasks[index]
		if task.ID != taskID {
			continue
		}
		if task.Status == DrugRecallTaskCompleted {
			return false, nil
		}
		now = normalizeContractTime(now)
		task.Status, task.CompletionNote, task.CompletedBy, task.CompletedAt = DrugRecallTaskCompleted, note, actorID, &now
		r.Meta.Touch(now)
		if r.allTasksCompleted() {
			r.Status, r.CompletedBy, r.CompletedAt = DrugRecallCompleted, actorID, &now
		}
		return true, nil
	}
	return false, fmt.Errorf("drug recall task not found")
}

func (r *DrugRecall) allTasksCompleted() bool {
	if r == nil || len(r.Tasks) == 0 {
		return false
	}
	for _, task := range r.Tasks {
		if task.Status != DrugRecallTaskCompleted {
			return false
		}
	}
	return true
}
