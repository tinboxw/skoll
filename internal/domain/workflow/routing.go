package workflow

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

const (
	maxWorkflowNodes       = 128
	maxWorkflowTransitions = 256
	maxConditionPredicates = 16
	maxWorkflowVariables   = 64
	maxWorkflowValueLength = 1024
)

var workflowVariablePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)

type DecisionStrategy string

const (
	DecisionAny    DecisionStrategy = "any"
	DecisionAll    DecisionStrategy = "all"
	DecisionQuorum DecisionStrategy = "quorum"
)

type DecisionRule struct {
	Strategy DecisionStrategy
	Quorum   int
}

func (r DecisionRule) Validate(assigneeCount int) error {
	if assigneeCount <= 0 {
		return fmt.Errorf("workflow decision requires assignees")
	}
	switch r.Strategy {
	case DecisionAny:
		if r.Quorum != 1 {
			return fmt.Errorf("workflow any decision requires quorum 1")
		}
	case DecisionAll:
		if r.Quorum != assigneeCount {
			return fmt.Errorf("workflow all decision quorum must equal assignee count")
		}
	case DecisionQuorum:
		if r.Quorum < 1 || r.Quorum > assigneeCount {
			return fmt.Errorf("workflow quorum decision is outside assignee count")
		}
	default:
		return fmt.Errorf("workflow decision strategy is invalid")
	}
	return nil
}

type ValueType string

const (
	ValueString  ValueType = "string"
	ValueNumber  ValueType = "number"
	ValueBoolean ValueType = "boolean"
)

type Value struct {
	Type  ValueType `json:"type"`
	Value string    `json:"value"`
}

func (v Value) Validate() error {
	if len(v.Value) > maxWorkflowValueLength {
		return fmt.Errorf("workflow value is too long")
	}
	switch v.Type {
	case ValueString:
		return nil
	case ValueNumber:
		if strings.TrimSpace(v.Value) == "" {
			return fmt.Errorf("workflow number value is required")
		}
		if _, ok := new(big.Rat).SetString(strings.TrimSpace(v.Value)); !ok {
			return fmt.Errorf("workflow number value is invalid")
		}
		return nil
	case ValueBoolean:
		if _, err := strconv.ParseBool(strings.TrimSpace(v.Value)); err != nil {
			return fmt.Errorf("workflow boolean value is invalid")
		}
		return nil
	default:
		return fmt.Errorf("workflow value type is invalid")
	}
}

type PredicateOperator string

const (
	PredicateEqual        PredicateOperator = "eq"
	PredicateNotEqual     PredicateOperator = "ne"
	PredicateGreaterThan  PredicateOperator = "gt"
	PredicateGreaterEqual PredicateOperator = "gte"
	PredicateLessThan     PredicateOperator = "lt"
	PredicateLessEqual    PredicateOperator = "lte"
	PredicateExists       PredicateOperator = "exists"
	PredicateNotExists    PredicateOperator = "not_exists"
)

type Predicate struct {
	Field    string            `json:"field"`
	Operator PredicateOperator `json:"operator"`
	Value    *Value            `json:"value,omitempty"`
}

func (p Predicate) Validate() error {
	if !workflowVariablePattern.MatchString(strings.TrimSpace(p.Field)) {
		return fmt.Errorf("workflow predicate field is invalid")
	}
	switch p.Operator {
	case PredicateExists, PredicateNotExists:
		if p.Value != nil {
			return fmt.Errorf("workflow existence predicate cannot declare a value")
		}
	case PredicateEqual, PredicateNotEqual, PredicateGreaterThan, PredicateGreaterEqual, PredicateLessThan, PredicateLessEqual:
		if p.Value == nil {
			return fmt.Errorf("workflow comparison predicate requires a value")
		}
		if err := p.Value.Validate(); err != nil {
			return err
		}
		if (p.Operator == PredicateGreaterThan || p.Operator == PredicateGreaterEqual || p.Operator == PredicateLessThan || p.Operator == PredicateLessEqual) &&
			p.Value.Type != ValueNumber {
			return fmt.Errorf("workflow ordered comparison requires a number value")
		}
	default:
		return fmt.Errorf("workflow predicate operator is invalid")
	}
	return nil
}

type ConditionMatch string

const (
	ConditionAll ConditionMatch = "all"
	ConditionAny ConditionMatch = "any"
)

type Condition struct {
	Match      ConditionMatch `json:"match"`
	Predicates []Predicate    `json:"predicates"`
}

func (c Condition) Validate() error {
	if c.Match != ConditionAll && c.Match != ConditionAny {
		return fmt.Errorf("workflow condition match mode is invalid")
	}
	if len(c.Predicates) == 0 || len(c.Predicates) > maxConditionPredicates {
		return fmt.Errorf("workflow condition predicate count is invalid")
	}
	for _, predicate := range c.Predicates {
		if err := predicate.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateVariables(variables map[string]Value) error {
	if len(variables) > maxWorkflowVariables {
		return fmt.Errorf("workflow variable count exceeds limit")
	}
	for field, value := range variables {
		if !workflowVariablePattern.MatchString(strings.TrimSpace(field)) {
			return fmt.Errorf("workflow variable field is invalid")
		}
		if err := value.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateVariables(variables map[string]Value) error {
	return validateVariables(variables)
}

func (c Condition) Matches(variables map[string]Value) (bool, error) {
	if err := c.Validate(); err != nil {
		return false, err
	}
	matched := c.Match == ConditionAll
	for _, predicate := range c.Predicates {
		result, err := predicate.matches(variables)
		if err != nil {
			return false, err
		}
		if c.Match == ConditionAll && !result {
			return false, nil
		}
		if c.Match == ConditionAny && result {
			return true, nil
		}
	}
	return matched, nil
}

func (p Predicate) matches(variables map[string]Value) (bool, error) {
	actual, exists := variables[p.Field]
	switch p.Operator {
	case PredicateExists:
		return exists, nil
	case PredicateNotExists:
		return !exists, nil
	}
	if !exists || p.Value == nil {
		return false, nil
	}
	if actual.Type != p.Value.Type {
		return false, nil
	}
	comparison, err := compareValues(actual, *p.Value)
	if err != nil {
		return false, err
	}
	switch p.Operator {
	case PredicateEqual:
		return comparison == 0, nil
	case PredicateNotEqual:
		return comparison != 0, nil
	case PredicateGreaterThan:
		return comparison > 0, nil
	case PredicateGreaterEqual:
		return comparison >= 0, nil
	case PredicateLessThan:
		return comparison < 0, nil
	case PredicateLessEqual:
		return comparison <= 0, nil
	default:
		return false, fmt.Errorf("workflow predicate operator is invalid")
	}
}

func compareValues(left, right Value) (int, error) {
	if err := left.Validate(); err != nil {
		return 0, err
	}
	if err := right.Validate(); err != nil {
		return 0, err
	}
	switch left.Type {
	case ValueNumber:
		leftNumber, _ := new(big.Rat).SetString(strings.TrimSpace(left.Value))
		rightNumber, _ := new(big.Rat).SetString(strings.TrimSpace(right.Value))
		return leftNumber.Cmp(rightNumber), nil
	case ValueBoolean:
		leftBool, _ := strconv.ParseBool(strings.TrimSpace(left.Value))
		rightBool, _ := strconv.ParseBool(strings.TrimSpace(right.Value))
		if leftBool == rightBool {
			return 0, nil
		}
		if !leftBool {
			return -1, nil
		}
		return 1, nil
	default:
		return strings.Compare(left.Value, right.Value), nil
	}
}

func (d Definition) validateGraph() error {
	if len(d.Nodes) < 2 || len(d.Nodes) > maxWorkflowNodes {
		return fmt.Errorf("workflow node count is invalid")
	}
	if len(d.Transitions) == 0 || len(d.Transitions) > maxWorkflowTransitions {
		return fmt.Errorf("workflow transition count is invalid")
	}
	start := shared.ID("")
	end := shared.ID("")
	nodeIDs := make(map[shared.ID]Node, len(d.Nodes))
	for _, node := range d.Nodes {
		nodeIDs[node.ID] = node
		if node.Type == NodeStart {
			start = node.ID
		}
		if node.Type == NodeEnd {
			end = node.ID
		}
	}
	outgoing := make(map[shared.ID][]shared.ID, len(d.Nodes))
	incoming := make(map[shared.ID][]shared.ID, len(d.Nodes))
	edges := make(map[string]struct{}, len(d.Transitions))
	for _, transition := range d.Transitions {
		if transition.From == transition.To {
			return fmt.Errorf("workflow transition cannot reference itself")
		}
		identity := transition.From.String() + "\x00" + transition.To.String()
		if _, exists := edges[identity]; exists {
			return fmt.Errorf("workflow transition is duplicated")
		}
		edges[identity] = struct{}{}
		if transition.Condition != nil {
			if err := transition.Condition.Validate(); err != nil {
				return err
			}
		}
		outgoing[transition.From] = append(outgoing[transition.From], transition.To)
		incoming[transition.To] = append(incoming[transition.To], transition.From)
	}
	if len(incoming[start]) != 0 || len(outgoing[start]) == 0 {
		return fmt.Errorf("workflow start node connections are invalid")
	}
	if len(outgoing[end]) != 0 || len(incoming[end]) == 0 {
		return fmt.Errorf("workflow end node connections are invalid")
	}
	for id, node := range nodeIDs {
		if node.Type != NodeEnd && len(outgoing[id]) == 0 {
			return fmt.Errorf("workflow node has no outgoing transition: %s", id)
		}
		if node.Type != NodeEnd && len(incoming[id]) > 1 {
			return fmt.Errorf("workflow parallel branches may converge only at the end node")
		}
	}
	if err := validateAcyclic(start, outgoing, map[shared.ID]uint8{}); err != nil {
		return err
	}
	reachable := walkGraph(start, outgoing)
	canReachEnd := walkGraph(end, incoming)
	for id := range nodeIDs {
		if !reachable[id] {
			return fmt.Errorf("workflow node is unreachable: %s", id)
		}
		if !canReachEnd[id] {
			return fmt.Errorf("workflow node cannot reach end: %s", id)
		}
	}
	return nil
}

func validateAcyclic(node shared.ID, outgoing map[shared.ID][]shared.ID, state map[shared.ID]uint8) error {
	if state[node] == 1 {
		return fmt.Errorf("workflow graph cannot contain cycles")
	}
	if state[node] == 2 {
		return nil
	}
	state[node] = 1
	for _, target := range outgoing[node] {
		if err := validateAcyclic(target, outgoing, state); err != nil {
			return err
		}
	}
	state[node] = 2
	return nil
}

func walkGraph(origin shared.ID, edges map[shared.ID][]shared.ID) map[shared.ID]bool {
	visited := map[shared.ID]bool{}
	var walk func(shared.ID)
	walk = func(node shared.ID) {
		if visited[node] {
			return
		}
		visited[node] = true
		for _, target := range edges[node] {
			walk(target)
		}
	}
	walk(origin)
	return visited
}

func (d Definition) activateFrom(source shared.ID, variables map[string]Value, active []shared.ID) ([]shared.ID, error) {
	activeSet := make(map[shared.ID]struct{}, len(active))
	for _, id := range active {
		activeSet[id] = struct{}{}
	}
	matched := false
	var visit func(shared.ID) error
	visit = func(nodeID shared.ID) error {
		node, ok := d.nodeByID(nodeID)
		if !ok {
			return fmt.Errorf("workflow transition target is unknown")
		}
		switch node.Type {
		case NodeApproval:
			if _, exists := activeSet[node.ID]; !exists {
				activeSet[node.ID] = struct{}{}
				active = append(active, node.ID)
			}
		case NodeCC:
			next, err := d.matchingTargets(node.ID, variables)
			if err != nil {
				return err
			}
			if len(next) == 0 {
				return fmt.Errorf("workflow route has no matching transition")
			}
			for _, target := range next {
				if err := visit(target); err != nil {
					return err
				}
			}
		case NodeEnd:
			return nil
		default:
			return fmt.Errorf("workflow runtime route targets an invalid node type")
		}
		return nil
	}
	targets, err := d.matchingTargets(source, variables)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		matched = true
		if err := visit(target); err != nil {
			return nil, err
		}
	}
	if !matched {
		return nil, fmt.Errorf("workflow route has no matching transition")
	}
	return active, nil
}

func (d Definition) matchingTargets(source shared.ID, variables map[string]Value) ([]shared.ID, error) {
	targets := make([]shared.ID, 0)
	for _, transition := range d.Transitions {
		if transition.From != source {
			continue
		}
		if transition.Condition == nil {
			targets = append(targets, transition.To)
			continue
		}
		matches, err := transition.Condition.Matches(variables)
		if err != nil {
			return nil, err
		}
		if matches {
			targets = append(targets, transition.To)
		}
	}
	return targets, nil
}
