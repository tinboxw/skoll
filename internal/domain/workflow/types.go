package workflow

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var workflowKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

type DefinitionStatus string

const (
	DefinitionDraft     DefinitionStatus = "draft"
	DefinitionPublished DefinitionStatus = "published"
	DefinitionDisabled  DefinitionStatus = "disabled"
)

type NodeType string

const (
	NodeStart    NodeType = "start"
	NodeApproval NodeType = "approval"
	NodeCC       NodeType = "cc"
	NodeEnd      NodeType = "end"
)

type InstanceStatus string

const (
	InstanceRunning   InstanceStatus = "running"
	InstanceApproved  InstanceStatus = "approved"
	InstanceRejected  InstanceStatus = "rejected"
	InstanceWithdrawn InstanceStatus = "withdrawn"
	InstanceCanceled  InstanceStatus = "canceled"
)

type TaskStatus string

const (
	TaskPending     TaskStatus = "pending"
	TaskApproved    TaskStatus = "approved"
	TaskRejected    TaskStatus = "rejected"
	TaskTransferred TaskStatus = "transferred"
	TaskCopied      TaskStatus = "copied"
	TaskCanceled    TaskStatus = "canceled"
)

type ActionType string

const (
	ActionStart    ActionType = "start"
	ActionApprove  ActionType = "approve"
	ActionReject   ActionType = "reject"
	ActionWithdraw ActionType = "withdraw"
	ActionTransfer ActionType = "transfer"
	ActionCopy     ActionType = "copy"
	ActionCancel   ActionType = "cancel"
)

type Actor struct {
	ID   shared.ID
	Name string
}

type Definition struct {
	ID          shared.ID
	Key         string
	Name        string
	Version     int
	Status      DefinitionStatus
	Nodes       []Node
	Transitions []Transition
	Meta        shared.AuditMeta
}

type Node struct {
	ID        shared.ID
	Key       string
	Name      string
	Type      NodeType
	Assignees []shared.ID
	Decision  DecisionRule
}

type Transition struct {
	From      shared.ID
	To        shared.ID
	Condition *Condition
}

type Instance struct {
	ID            shared.ID
	DefinitionID  shared.ID
	DefinitionKey string
	BusinessType  string
	BusinessID    string
	Title         string
	Status        InstanceStatus
	Starter       Actor
	CurrentNode   shared.ID
	ActiveNodes   []shared.ID
	Variables     map[string]Value
	Tasks         []Task
	Timeline      []Action
	Meta          shared.AuditMeta
}

type Task struct {
	ID          shared.ID
	InstanceID  shared.ID
	NodeID      shared.ID
	Assignee    Actor
	Status      TaskStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type Action struct {
	ID         shared.ID
	Type       ActionType
	InstanceID shared.ID
	TaskID     shared.ID
	NodeID     shared.ID
	Actor      Actor
	Target     Actor
	Comment    string
	CreatedAt  time.Time
}

type StartInput struct {
	ID           shared.ID
	Definition   Definition
	BusinessType string
	BusinessID   string
	Title        string
	Starter      Actor
	Variables    map[string]Value
	Now          time.Time
}

func NewDefinition(id shared.ID, key, name string, version int, nodes []Node, transitions []Transition, now time.Time) (*Definition, error) {
	def := &Definition{
		ID:          id,
		Key:         strings.TrimSpace(strings.ToLower(key)),
		Name:        strings.TrimSpace(name),
		Version:     version,
		Status:      DefinitionDraft,
		Nodes:       normalizeNodes(nodes),
		Transitions: append([]Transition(nil), transitions...),
	}
	if err := def.Validate(); err != nil {
		return nil, err
	}
	def.Meta.Touch(now)
	return def, nil
}

func (d *Definition) Publish(now time.Time) error {
	if d == nil {
		return fmt.Errorf("workflow definition is required")
	}
	if err := d.Validate(); err != nil {
		return err
	}
	d.Status = DefinitionPublished
	d.Meta.Touch(now)
	return nil
}

func (d Definition) Validate() error {
	if d.ID.IsZero() {
		return fmt.Errorf("workflow definition id is required")
	}
	if !workflowKeyPattern.MatchString(strings.TrimSpace(d.Key)) {
		return fmt.Errorf("workflow definition key is invalid")
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("workflow definition name is required")
	}
	if d.Version <= 0 {
		return fmt.Errorf("workflow definition version must be positive")
	}
	starts, ends := 0, 0
	nodeIDs := map[shared.ID]struct{}{}
	for _, node := range d.Nodes {
		if err := node.Validate(); err != nil {
			return err
		}
		if _, ok := nodeIDs[node.ID]; ok {
			return fmt.Errorf("workflow node id conflict: %s", node.ID)
		}
		nodeIDs[node.ID] = struct{}{}
		if node.Type == NodeStart {
			starts++
		}
		if node.Type == NodeEnd {
			ends++
		}
	}
	if starts != 1 || ends != 1 {
		return fmt.Errorf("workflow definition requires exactly one start and one end node")
	}
	for _, edge := range d.Transitions {
		if _, ok := nodeIDs[edge.From]; !ok {
			return fmt.Errorf("workflow transition references unknown from node: %s", edge.From)
		}
		if _, ok := nodeIDs[edge.To]; !ok {
			return fmt.Errorf("workflow transition references unknown to node: %s", edge.To)
		}
	}
	return d.validateGraph()
}

func (n Node) Validate() error {
	if n.ID.IsZero() {
		return fmt.Errorf("workflow node id is required")
	}
	if !workflowKeyPattern.MatchString(strings.TrimSpace(n.Key)) {
		return fmt.Errorf("workflow node key is invalid")
	}
	if strings.TrimSpace(n.Name) == "" {
		return fmt.Errorf("workflow node name is required")
	}
	switch n.Type {
	case NodeStart, NodeApproval, NodeCC, NodeEnd:
	default:
		return fmt.Errorf("workflow node type is invalid")
	}
	if n.Type == NodeApproval && len(n.Assignees) == 0 {
		return fmt.Errorf("workflow approval node requires assignees")
	}
	if n.Type == NodeApproval {
		return n.Decision.Validate(len(n.Assignees))
	}
	if n.Decision.Strategy != "" || n.Decision.Quorum != 0 {
		return fmt.Errorf("workflow non-approval node cannot declare a decision rule")
	}
	return nil
}

func Start(in StartInput) (*Instance, error) {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	def := in.Definition
	if def.Status != DefinitionPublished {
		return nil, fmt.Errorf("workflow definition must be published")
	}
	if err := def.Validate(); err != nil {
		return nil, err
	}
	if in.ID.IsZero() || strings.TrimSpace(in.BusinessType) == "" || strings.TrimSpace(in.BusinessID) == "" || strings.TrimSpace(in.Title) == "" || in.Starter.ID.IsZero() {
		return nil, fmt.Errorf("workflow start input is incomplete")
	}
	if err := validateVariables(in.Variables); err != nil {
		return nil, err
	}
	start, _ := def.nodeByType(NodeStart)
	active, err := def.activateFrom(start.ID, in.Variables, nil)
	if err != nil {
		return nil, err
	}
	current := def.endNodeID()
	if len(active) > 0 {
		current = active[0]
	}
	instance := &Instance{
		ID:            in.ID,
		DefinitionID:  def.ID,
		DefinitionKey: def.Key,
		BusinessType:  strings.TrimSpace(in.BusinessType),
		BusinessID:    strings.TrimSpace(in.BusinessID),
		Title:         strings.TrimSpace(in.Title),
		Status:        InstanceRunning,
		Starter:       normalizeActor(in.Starter),
		CurrentNode:   current,
		ActiveNodes:   append([]shared.ID(nil), active...),
		Variables:     cloneVariables(in.Variables),
		Tasks:         makeTasksForNodes(in.ID, def, active, now),
		Timeline: []Action{{
			ID:         shared.ID(fmt.Sprintf("%s-start", in.ID)),
			Type:       ActionStart,
			InstanceID: in.ID,
			NodeID:     current,
			Actor:      normalizeActor(in.Starter),
			CreatedAt:  now,
		}},
	}
	if len(active) == 0 {
		instance.Status = InstanceApproved
	}
	instance.Meta.Touch(now)
	return instance, nil
}

func (i *Instance) Approve(definition Definition, taskID shared.ID, actor Actor, comment string, now time.Time) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	if definition.ID != i.DefinitionID || definition.Status != DefinitionPublished {
		return fmt.Errorf("workflow instance definition is invalid")
	}
	if err := definition.Validate(); err != nil {
		return err
	}
	task, err := i.pendingTask(taskID, actor)
	if err != nil {
		return err
	}
	if !containsID(i.ActiveNodes, task.NodeID) {
		return fmt.Errorf("workflow task node is not active")
	}
	node, ok := definition.nodeByID(task.NodeID)
	if !ok || node.Type != NodeApproval {
		return fmt.Errorf("workflow approval node is invalid")
	}
	willResolve := i.approvedCount(task.NodeID)+1 >= node.Decision.Quorum
	nextActive := i.ActiveNodes
	newNodes := []shared.ID(nil)
	if willResolve {
		remaining := removeID(i.ActiveNodes, task.NodeID)
		nextActive, err = definition.activateFrom(task.NodeID, i.Variables, remaining)
		if err != nil {
			return err
		}
		newNodes = differenceIDs(nextActive, remaining)
	}
	now = normalizeNow(now)
	task.Status = TaskApproved
	task.CompletedAt = &now
	i.appendAction(Action{ID: actionID(i.ID, ActionApprove, taskID.String(), actor.ID.String()), Type: ActionApprove, InstanceID: i.ID, TaskID: taskID, NodeID: task.NodeID, Actor: normalizeActor(actor), Comment: strings.TrimSpace(comment), CreatedAt: now})
	if willResolve {
		i.cancelPendingNodeTasks(task.NodeID, now)
		i.ActiveNodes = nextActive
		i.Tasks = append(i.Tasks, makeTasksForNodes(i.ID, definition, newNodes, now)...)
		if len(i.ActiveNodes) == 0 {
			i.Status = InstanceApproved
			i.CurrentNode = definition.endNodeID()
		} else {
			i.CurrentNode = i.ActiveNodes[0]
		}
	}
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) Reject(taskID shared.ID, actor Actor, comment string, now time.Time) error {
	if err := i.completeTask(taskID, actor, comment, now, TaskRejected, ActionReject, InstanceRejected); err != nil {
		return err
	}
	i.cancelAllPendingTasks(normalizeNow(now))
	i.ActiveNodes = nil
	return nil
}

func (i *Instance) Withdraw(actor Actor, comment string, now time.Time) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	if actor.ID.IsZero() || actor.ID != i.Starter.ID {
		return fmt.Errorf("workflow can only be withdrawn by starter")
	}
	now = normalizeNow(now)
	for idx := range i.Tasks {
		if i.Tasks[idx].Status == TaskPending {
			i.Tasks[idx].Status = TaskCanceled
			i.Tasks[idx].CompletedAt = &now
		}
	}
	i.Status = InstanceWithdrawn
	i.ActiveNodes = nil
	i.appendAction(Action{ID: actionID(i.ID, ActionWithdraw, actor.ID.String()), Type: ActionWithdraw, InstanceID: i.ID, NodeID: i.CurrentNode, Actor: normalizeActor(actor), Comment: strings.TrimSpace(comment), CreatedAt: now})
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) Cancel(actor Actor, comment string, now time.Time) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	if actor.ID.IsZero() {
		return fmt.Errorf("workflow cancellation actor is required")
	}
	now = normalizeNow(now)
	for idx := range i.Tasks {
		if i.Tasks[idx].Status == TaskPending {
			i.Tasks[idx].Status = TaskCanceled
			i.Tasks[idx].CompletedAt = &now
		}
	}
	i.Status = InstanceCanceled
	i.ActiveNodes = nil
	i.appendAction(Action{ID: actionID(i.ID, ActionCancel, actor.ID.String()), Type: ActionCancel, InstanceID: i.ID, NodeID: i.CurrentNode, Actor: normalizeActor(actor), Comment: strings.TrimSpace(comment), CreatedAt: now})
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) Transfer(taskID shared.ID, actor Actor, target Actor, comment string, now time.Time) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	if target.ID.IsZero() {
		return fmt.Errorf("workflow transfer target is required")
	}
	task, err := i.pendingTask(taskID, actor)
	if err != nil {
		return err
	}
	now = normalizeNow(now)
	task.Status = TaskTransferred
	task.CompletedAt = &now
	i.Tasks = append(i.Tasks, Task{
		ID:         shared.ID(fmt.Sprintf("%s-transfer-%s", task.ID, target.ID)),
		InstanceID: i.ID,
		NodeID:     task.NodeID,
		Assignee:   normalizeActor(target),
		Status:     TaskPending,
		CreatedAt:  now,
	})
	i.appendAction(Action{ID: actionID(i.ID, ActionTransfer, taskID.String(), actor.ID.String(), target.ID.String()), Type: ActionTransfer, InstanceID: i.ID, TaskID: taskID, NodeID: task.NodeID, Actor: normalizeActor(actor), Target: normalizeActor(target), Comment: strings.TrimSpace(comment), CreatedAt: now})
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) Copy(taskID shared.ID, actor Actor, target Actor, comment string, now time.Time) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	if target.ID.IsZero() {
		return fmt.Errorf("workflow copy target is required")
	}
	task, err := i.pendingTask(taskID, actor)
	if err != nil {
		return err
	}
	now = normalizeNow(now)
	i.Tasks = append(i.Tasks, Task{
		ID:          shared.ID(fmt.Sprintf("%s-copy-%s", task.ID, target.ID)),
		InstanceID:  i.ID,
		NodeID:      task.NodeID,
		Assignee:    normalizeActor(target),
		Status:      TaskCopied,
		CreatedAt:   now,
		CompletedAt: &now,
	})
	i.appendAction(Action{ID: actionID(i.ID, ActionCopy, taskID.String(), actor.ID.String(), target.ID.String()), Type: ActionCopy, InstanceID: i.ID, TaskID: taskID, NodeID: task.NodeID, Actor: normalizeActor(actor), Target: normalizeActor(target), Comment: strings.TrimSpace(comment), CreatedAt: now})
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) completeTask(taskID shared.ID, actor Actor, comment string, now time.Time, taskStatus TaskStatus, actionType ActionType, instanceStatus InstanceStatus) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	task, err := i.pendingTask(taskID, actor)
	if err != nil {
		return err
	}
	now = normalizeNow(now)
	task.Status = taskStatus
	task.CompletedAt = &now
	i.Status = instanceStatus
	i.appendAction(Action{ID: actionID(i.ID, actionType, taskID.String(), actor.ID.String()), Type: actionType, InstanceID: i.ID, TaskID: taskID, NodeID: task.NodeID, Actor: normalizeActor(actor), Comment: strings.TrimSpace(comment), CreatedAt: now})
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) ensureRunning() error {
	if i == nil {
		return fmt.Errorf("workflow instance is required")
	}
	if i.Status != InstanceRunning {
		return fmt.Errorf("workflow instance is not running")
	}
	return nil
}

func (i *Instance) pendingTask(taskID shared.ID, actor Actor) (*Task, error) {
	if taskID.IsZero() || actor.ID.IsZero() {
		return nil, fmt.Errorf("workflow task id and actor are required")
	}
	for idx := range i.Tasks {
		task := &i.Tasks[idx]
		if task.ID != taskID {
			continue
		}
		if task.Status != TaskPending {
			return nil, fmt.Errorf("workflow task is not pending")
		}
		if task.Assignee.ID != actor.ID {
			return nil, fmt.Errorf("workflow task assignee mismatch")
		}
		return task, nil
	}
	return nil, fmt.Errorf("workflow task not found")
}

func (i *Instance) appendAction(action Action) {
	i.Timeline = append(i.Timeline, action)
}

func (d Definition) nodeByID(id shared.ID) (Node, bool) {
	for _, node := range d.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return Node{}, false
}

func (d Definition) nodeByType(nodeType NodeType) (Node, bool) {
	for _, node := range d.Nodes {
		if node.Type == nodeType {
			return node, true
		}
	}
	return Node{}, false
}

func (d Definition) endNodeID() shared.ID {
	node, _ := d.nodeByType(NodeEnd)
	return node.ID
}

func normalizeNodes(nodes []Node) []Node {
	out := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		node.Key = strings.TrimSpace(strings.ToLower(node.Key))
		node.Name = strings.TrimSpace(node.Name)
		node.Assignees = normalizeIDs(node.Assignees)
		out = append(out, node)
	}
	return out
}

func normalizeIDs(ids []shared.ID) []shared.ID {
	out := make([]shared.ID, 0, len(ids))
	seen := map[shared.ID]struct{}{}
	for _, id := range ids {
		id = shared.ID(strings.TrimSpace(id.String()))
		if id.IsZero() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func makeTasksForNodes(instanceID shared.ID, definition Definition, nodeIDs []shared.ID, now time.Time) []Task {
	tasks := make([]Task, 0)
	for _, nodeID := range nodeIDs {
		node, ok := definition.nodeByID(nodeID)
		if !ok {
			continue
		}
		for _, assignee := range node.Assignees {
			tasks = append(tasks, Task{
				ID:         shared.ID(fmt.Sprintf("%s-%s-%s", instanceID, node.ID, assignee)),
				InstanceID: instanceID,
				NodeID:     node.ID,
				Assignee:   Actor{ID: assignee},
				Status:     TaskPending,
				CreatedAt:  now,
			})
		}
	}
	return tasks
}

func cloneVariables(variables map[string]Value) map[string]Value {
	cloned := make(map[string]Value, len(variables))
	for key, value := range variables {
		cloned[key] = value
	}
	return cloned
}

func containsID(ids []shared.ID, candidate shared.ID) bool {
	for _, id := range ids {
		if id == candidate {
			return true
		}
	}
	return false
}

func removeID(ids []shared.ID, candidate shared.ID) []shared.ID {
	out := make([]shared.ID, 0, len(ids))
	for _, id := range ids {
		if id != candidate {
			out = append(out, id)
		}
	}
	return out
}

func differenceIDs(left, right []shared.ID) []shared.ID {
	out := make([]shared.ID, 0)
	for _, id := range left {
		if !containsID(right, id) {
			out = append(out, id)
		}
	}
	return out
}

func (i *Instance) approvedCount(nodeID shared.ID) int {
	count := 0
	for _, task := range i.Tasks {
		if task.NodeID == nodeID && task.Status == TaskApproved {
			count++
		}
	}
	return count
}

func (i *Instance) cancelPendingNodeTasks(nodeID shared.ID, now time.Time) {
	for index := range i.Tasks {
		task := &i.Tasks[index]
		if task.NodeID == nodeID && task.Status == TaskPending {
			task.Status = TaskCanceled
			task.CompletedAt = &now
		}
	}
}

func (i *Instance) cancelAllPendingTasks(now time.Time) {
	for index := range i.Tasks {
		task := &i.Tasks[index]
		if task.Status == TaskPending {
			task.Status = TaskCanceled
			task.CompletedAt = &now
		}
	}
}

func normalizeActor(actor Actor) Actor {
	return Actor{ID: shared.ID(strings.TrimSpace(actor.ID.String())), Name: strings.TrimSpace(actor.Name)}
}

func normalizeNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func actionID(instanceID shared.ID, action ActionType, identity ...string) shared.ID {
	digest := sha256.Sum256([]byte(strings.Join(identity, "\x00")))
	return shared.ID(fmt.Sprintf("%s-%s-%x", instanceID, action, digest[:8]))
}
