package workflowaudit

import (
	"fmt"
	"sort"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
)

type ReplayRecord struct {
	EventID       string
	Action        string
	ActorID       string
	TargetActorID string
	ResourceID    string
	OccurredAt    string
}

type ReplayResult struct {
	InstanceID string
	Terminal   string
	Records    []ReplayRecord
}

func ReplayApprovalChain(events []*domainaudit.Event) (ReplayResult, error) {
	ordered := make([]*domainaudit.Event, 0, len(events))
	for _, event := range events {
		if event != nil && event.Resource.Type == "workflow_instance" {
			ordered = append(ordered, event)
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].OccurredAt.Before(ordered[j].OccurredAt)
	})
	if len(ordered) == 0 {
		return ReplayResult{}, fmt.Errorf("workflow audit replay has no events")
	}

	expected := []string{"start", "copy", "delegate", "approve"}
	if len(ordered) != len(expected) {
		return ReplayResult{}, fmt.Errorf("workflow audit replay expected %d events, got %d", len(expected), len(ordered))
	}
	instanceID := ordered[0].Resource.ID
	result := ReplayResult{InstanceID: instanceID, Records: make([]ReplayRecord, 0, len(ordered))}
	for idx, event := range ordered {
		action := stringValue(event.SourceData["action"])
		if action != expected[idx] {
			return ReplayResult{}, fmt.Errorf("workflow audit replay expected action %s at index %d, got %s", expected[idx], idx, action)
		}
		if event.Resource.ID != instanceID {
			return ReplayResult{}, fmt.Errorf("workflow audit replay crossed instance ids: %s != %s", event.Resource.ID, instanceID)
		}
		if event.Result != domainaudit.EventResultSuccess {
			return ReplayResult{}, fmt.Errorf("workflow audit replay event %s is not successful", event.ID)
		}
		result.Records = append(result.Records, ReplayRecord{
			EventID:       event.ID.String(),
			Action:        action,
			ActorID:       event.Actor.ID.String(),
			TargetActorID: stringValue(event.SourceData["targetId"]),
			ResourceID:    event.Resource.ID,
			OccurredAt:    event.OccurredAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if result.Records[0].ActorID == "" || result.Records[2].TargetActorID != "approver-2" || result.Records[3].ActorID != "approver-2" {
		return ReplayResult{}, fmt.Errorf("workflow audit replay actor chain is invalid")
	}
	result.Terminal = "approved"
	return result, nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}
