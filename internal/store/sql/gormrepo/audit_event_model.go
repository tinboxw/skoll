package gormrepo

import (
	"encoding/json"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type AuditEventModel struct {
	ID            string    `gorm:"primaryKey;size:64"`
	EventType     string    `gorm:"column:event_type;size:32;index:idx_audit_events_type_time,priority:1"`
	Action        string    `gorm:"column:action;size:192;index:idx_audit_events_action_time,priority:1"`
	ActorType     string    `gorm:"column:actor_type;size:64"`
	ActorID       string    `gorm:"column:actor_id;size:64;index:idx_audit_events_actor_time,priority:1"`
	ActorName     string    `gorm:"column:actor_name;size:128"`
	ResourceType  string    `gorm:"column:resource_type;size:64;index:idx_audit_events_resource_time,priority:1"`
	ResourceID    string    `gorm:"column:resource_id;size:128;index:idx_audit_events_resource_time,priority:2"`
	ResourceName  string    `gorm:"column:resource_name;size:128"`
	Result        string    `gorm:"column:result;size:32;index:idx_audit_events_result_time,priority:1"`
	Risk          string    `gorm:"column:risk;size:32;index:idx_audit_events_risk_time,priority:1"`
	TraceID       string    `gorm:"column:trace_id;size:128;index:idx_audit_events_trace"`
	RequestID     string    `gorm:"column:request_id;size:128;index:idx_audit_events_request"`
	RequestMethod string    `gorm:"column:request_method;size:16"`
	RequestPath   string    `gorm:"column:request_path;size:512"`
	RequestIP     string    `gorm:"column:request_ip;size:64"`
	UserAgent     string    `gorm:"column:user_agent;size:512"`
	MetadataJSON  string    `gorm:"column:metadata_json;type:longtext"`
	SourceJSON    string    `gorm:"column:source_json;type:longtext"`
	OccurredAt    time.Time `gorm:"column:occurred_at;index:idx_audit_events_type_time,priority:2;index:idx_audit_events_actor_time,priority:2;index:idx_audit_events_action_time,priority:2;index:idx_audit_events_resource_time,priority:3;index:idx_audit_events_result_time,priority:2;index:idx_audit_events_risk_time,priority:2;index:idx_audit_events_occurred"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (AuditEventModel) TableName() string { return "sk_audit_events" }

func AuditEventModelFromDomain(event *domainaudit.Event) AuditEventModel {
	if event == nil {
		return AuditEventModel{MetadataJSON: "{}", SourceJSON: "{}"}
	}
	return AuditEventModel{
		ID:            strings.TrimSpace(event.ID.String()),
		EventType:     string(event.Type),
		Action:        string(event.Action),
		ActorType:     strings.TrimSpace(event.Actor.Type),
		ActorID:       strings.TrimSpace(event.Actor.ID.String()),
		ActorName:     strings.TrimSpace(event.Actor.Name),
		ResourceType:  strings.TrimSpace(event.Resource.Type),
		ResourceID:    strings.TrimSpace(event.Resource.ID),
		ResourceName:  strings.TrimSpace(event.Resource.Name),
		Result:        string(event.Result),
		Risk:          string(event.Risk),
		TraceID:       strings.TrimSpace(event.Trace.TraceID),
		RequestID:     strings.TrimSpace(event.Trace.RequestID),
		RequestMethod: strings.TrimSpace(event.Trace.Method),
		RequestPath:   strings.TrimSpace(event.Trace.Path),
		RequestIP:     strings.TrimSpace(event.Trace.IP),
		UserAgent:     strings.TrimSpace(event.Trace.UserAgent),
		MetadataJSON:  marshalAuditMap(event.Metadata),
		SourceJSON:    marshalAuditMap(event.SourceData),
		OccurredAt:    event.OccurredAt,
	}
}

func (m AuditEventModel) ToDomain() (*domainaudit.Event, error) {
	metadata, err := unmarshalAuditMap(m.MetadataJSON)
	if err != nil {
		return nil, err
	}
	sourceData, err := unmarshalAuditMap(m.SourceJSON)
	if err != nil {
		return nil, err
	}
	return domainaudit.NewEvent(domainaudit.EventInput{
		ID:     shared.ID(strings.TrimSpace(m.ID)),
		Type:   domainaudit.EventType(strings.TrimSpace(m.EventType)),
		Action: domainaudit.AuditAction(strings.TrimSpace(m.Action)),
		Actor: domainaudit.ActorRef{
			Type: strings.TrimSpace(m.ActorType),
			ID:   shared.ID(strings.TrimSpace(m.ActorID)),
			Name: strings.TrimSpace(m.ActorName),
		},
		Resource: domainaudit.ResourceRef{
			Type: strings.TrimSpace(m.ResourceType),
			ID:   strings.TrimSpace(m.ResourceID),
			Name: strings.TrimSpace(m.ResourceName),
		},
		Result: domainaudit.EventResult(strings.TrimSpace(m.Result)),
		Trace: domainaudit.TraceContext{
			TraceID:   strings.TrimSpace(m.TraceID),
			RequestID: strings.TrimSpace(m.RequestID),
			Method:    strings.TrimSpace(m.RequestMethod),
			Path:      strings.TrimSpace(m.RequestPath),
			IP:        strings.TrimSpace(m.RequestIP),
			UserAgent: strings.TrimSpace(m.UserAgent),
		},
		Risk:       domainaudit.EventRisk(strings.TrimSpace(m.Risk)),
		Metadata:   metadata,
		SourceData: sourceData,
		OccurredAt: m.OccurredAt,
	})
}

func marshalAuditMap(values map[string]any) string {
	if len(values) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func unmarshalAuditMap(raw string) (map[string]any, error) {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
