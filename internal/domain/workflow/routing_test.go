package workflow

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestConditionalParallelQuorumWorkflowResolvesAllBranchesOnce(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	definition, err := NewDefinition(
		"governed-definition",
		"medical.purchase.approval",
		"Medical Purchase Approval",
		1,
		[]Node{
			{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
			{ID: "manager", Key: "manager", Name: "Manager", Type: NodeApproval, Assignees: []shared.ID{"manager-1"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
			{ID: "quality", Key: "quality", Name: "Quality", Type: NodeApproval, Assignees: []shared.ID{"quality-1", "quality-2", "quality-3"}, Decision: DecisionRule{Strategy: DecisionQuorum, Quorum: 2}},
			{ID: "finance", Key: "finance", Name: "Finance", Type: NodeApproval, Assignees: []shared.ID{"finance-1", "finance-2"}, Decision: DecisionRule{Strategy: DecisionAll, Quorum: 2}},
			{ID: "end", Key: "end", Name: "End", Type: NodeEnd},
		},
		[]Transition{
			{From: "start", To: "manager"},
			{From: "start", To: "quality", Condition: condition(ConditionAll, Predicate{Field: "risk.level", Operator: PredicateEqual, Value: value(ValueString, "high")})},
			{From: "start", To: "finance", Condition: condition(ConditionAll, Predicate{Field: "amount", Operator: PredicateGreaterEqual, Value: value(ValueNumber, "1000.00")})},
			{From: "manager", To: "end"},
			{From: "quality", To: "end"},
			{From: "finance", To: "end"},
		},
		now,
	)
	if err != nil {
		t.Fatalf("NewDefinition error: %v", err)
	}
	if err := definition.Publish(now); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	instance, err := Start(StartInput{
		ID: "governed-instance", Definition: *definition, BusinessType: "purchase", BusinessID: "purchase-1",
		Title: "Controlled medicine purchase", Starter: Actor{ID: "buyer-1"},
		Variables: map[string]Value{
			"risk.level": {Type: ValueString, Value: "high"},
			"amount":     {Type: ValueNumber, Value: "1000"},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if len(instance.ActiveNodes) != 3 || len(instance.Tasks) != 6 {
		t.Fatalf("expected three parallel branches and six tasks, got nodes=%v tasks=%d", instance.ActiveNodes, len(instance.Tasks))
	}

	approveTask(t, instance, *definition, "quality", "quality-1", now.Add(time.Minute))
	if !containsID(instance.ActiveNodes, "quality") {
		t.Fatal("quality branch resolved before quorum")
	}
	approveTask(t, instance, *definition, "quality", "quality-2", now.Add(2*time.Minute))
	if containsID(instance.ActiveNodes, "quality") || taskStatus(instance, "quality", "quality-3") != TaskCanceled {
		t.Fatalf("quality quorum did not resolve exactly once: %+v", instance)
	}

	approveTask(t, instance, *definition, "finance", "finance-1", now.Add(3*time.Minute))
	if !containsID(instance.ActiveNodes, "finance") {
		t.Fatal("all-strategy finance branch resolved too early")
	}
	approveTask(t, instance, *definition, "finance", "finance-2", now.Add(4*time.Minute))
	approveTask(t, instance, *definition, "manager", "manager-1", now.Add(5*time.Minute))

	if instance.Status != InstanceApproved || len(instance.ActiveNodes) != 0 || instance.CurrentNode != "end" {
		t.Fatalf("parallel workflow did not complete after every branch: %+v", instance)
	}
	if len(instance.Timeline) != 6 {
		t.Fatalf("expected start plus five immutable decisions, got %d", len(instance.Timeline))
	}
}

func TestWorkflowConditionAndDecisionContractsFailClosed(t *testing.T) {
	now := time.Date(2026, 7, 27, 11, 0, 0, 0, time.UTC)
	_, err := NewDefinition("invalid-decision", "medical.invalid", "Invalid", 1, []Node{
		{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
		{ID: "approval", Key: "approval", Name: "Approval", Type: NodeApproval, Assignees: []shared.ID{"one", "two"}, Decision: DecisionRule{Strategy: DecisionQuorum, Quorum: 3}},
		{ID: "end", Key: "end", Name: "End", Type: NodeEnd},
	}, []Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}}, now)
	if err == nil || !strings.Contains(err.Error(), "outside assignee count") {
		t.Fatalf("expected invalid quorum rejection, got %v", err)
	}

	predicates := make([]Predicate, maxConditionPredicates+1)
	for index := range predicates {
		predicates[index] = Predicate{Field: "amount", Operator: PredicateExists}
	}
	if err := (Condition{Match: ConditionAll, Predicates: predicates}).Validate(); err == nil {
		t.Fatal("expected bounded condition predicate rejection")
	}

	_, err = NewDefinition("invalid-join", "medical.invalid.join", "Invalid Join", 1, []Node{
		{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
		{ID: "left", Key: "left", Name: "Left", Type: NodeApproval, Assignees: []shared.ID{"left"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
		{ID: "right", Key: "right", Name: "Right", Type: NodeApproval, Assignees: []shared.ID{"right"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
		{ID: "join", Key: "join", Name: "Join", Type: NodeApproval, Assignees: []shared.ID{"join"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
		{ID: "end", Key: "end", Name: "End", Type: NodeEnd},
	}, []Transition{
		{From: "start", To: "left"}, {From: "start", To: "right"},
		{From: "left", To: "join"}, {From: "right", To: "join"}, {From: "join", To: "end"},
	}, now)
	if err == nil || !strings.Contains(err.Error(), "converge only at the end") {
		t.Fatalf("expected unsafe non-end join rejection, got %v", err)
	}

	numberCondition := Condition{Match: ConditionAll, Predicates: []Predicate{
		{Field: "amount", Operator: PredicateEqual, Value: value(ValueNumber, "1000.00")},
	}}
	matches, err := numberCondition.Matches(map[string]Value{"amount": {Type: ValueNumber, Value: "1000"}})
	if err != nil || !matches {
		t.Fatalf("exact decimal condition should match without float conversion: matches=%v err=%v", matches, err)
	}
	for number := -100; number <= 100; number++ {
		raw := strconv.Itoa(number)
		condition := Condition{Match: ConditionAll, Predicates: []Predicate{
			{Field: "amount", Operator: PredicateGreaterEqual, Value: value(ValueNumber, "0")},
		}}
		matches, err := condition.Matches(map[string]Value{"amount": {Type: ValueNumber, Value: raw}})
		if err != nil || matches != (number >= 0) {
			t.Fatalf("ordered numeric property failed for %d: matches=%v err=%v", number, matches, err)
		}
	}
}

func TestWorkflowDecisionLeavesNoPartialStateWhenRouteDoesNotMatch(t *testing.T) {
	now := time.Date(2026, 7, 27, 11, 30, 0, 0, time.UTC)
	definition, err := NewDefinition("route-failure", "medical.route.failure", "Route Failure", 1, []Node{
		{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
		{ID: "approval", Key: "approval", Name: "Approval", Type: NodeApproval, Assignees: []shared.ID{"approver"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
		{ID: "end", Key: "end", Name: "End", Type: NodeEnd},
	}, []Transition{
		{From: "start", To: "approval"},
		{
			From: "approval", To: "end",
			Condition: condition(ConditionAll, Predicate{Field: "amount", Operator: PredicateGreaterThan, Value: value(ValueNumber, "1000")}),
		},
	}, now)
	if err != nil {
		t.Fatalf("NewDefinition error: %v", err)
	}
	if err := definition.Publish(now); err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	instance, err := Start(StartInput{
		ID: "route-failure-instance", Definition: *definition, BusinessType: "purchase", BusinessID: "purchase-2",
		Title: "Low amount purchase", Starter: Actor{ID: "buyer"}, Variables: map[string]Value{"amount": {Type: ValueNumber, Value: "500"}}, Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	err = instance.Approve(*definition, instance.Tasks[0].ID, Actor{ID: "approver"}, "approve", now.Add(time.Minute))
	if err == nil || !strings.Contains(err.Error(), "no matching transition") {
		t.Fatalf("expected unmatched route rejection, got %v", err)
	}
	if instance.Status != InstanceRunning || instance.Tasks[0].Status != TaskPending || len(instance.Timeline) != 1 || len(instance.ActiveNodes) != 1 {
		t.Fatalf("failed route left partial workflow state: %+v", instance)
	}
}

func condition(match ConditionMatch, predicates ...Predicate) *Condition {
	return &Condition{Match: match, Predicates: predicates}
}

func value(valueType ValueType, raw string) *Value {
	return &Value{Type: valueType, Value: raw}
}

func approveTask(t *testing.T, instance *Instance, definition Definition, nodeID, actorID string, now time.Time) {
	t.Helper()
	for _, task := range instance.Tasks {
		if task.NodeID == shared.ID(nodeID) && task.Assignee.ID == shared.ID(actorID) {
			if err := instance.Approve(definition, task.ID, Actor{ID: shared.ID(actorID)}, "approved", now); err != nil {
				t.Fatalf("approve %s/%s: %v", nodeID, actorID, err)
			}
			return
		}
	}
	t.Fatalf("task not found for %s/%s", nodeID, actorID)
}

func taskStatus(instance *Instance, nodeID, actorID string) TaskStatus {
	for _, task := range instance.Tasks {
		if task.NodeID == shared.ID(nodeID) && task.Assignee.ID == shared.ID(actorID) {
			return task.Status
		}
	}
	return ""
}
