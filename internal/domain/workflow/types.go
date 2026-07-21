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
}

type Transition struct {
	From shared.ID
	To   shared.ID
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
	return nil
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
	first, err := def.firstApprovalNode()
	if err != nil {
		return nil, err
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
		CurrentNode:   first.ID,
		Tasks:         makeTasks(in.ID, first, now),
		Timeline: []Action{{
			ID:         shared.ID(fmt.Sprintf("%s-start", in.ID)),
			Type:       ActionStart,
			InstanceID: in.ID,
			NodeID:     first.ID,
			Actor:      normalizeActor(in.Starter),
			CreatedAt:  now,
		}},
	}
	instance.Meta.Touch(now)
	return instance, nil
}

func (i *Instance) Approve(taskID shared.ID, actor Actor, comment string, now time.Time) error {
	return i.completeTask(taskID, actor, comment, now, TaskApproved, ActionApprove, InstanceApproved)
}

func (i *Instance) Reject(taskID shared.ID, actor Actor, comment string, now time.Time) error {
	return i.completeTask(taskID, actor, comment, now, TaskRejected, ActionReject, InstanceRejected)
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
	i.appendAction(Action{ID: actionID(i.ID, ActionWithdraw, actor.ID.String()), Type: ActionWithdraw, InstanceID: i.ID, NodeID: i.CurrentNode, Actor: normalizeActor(actor), Comment: strings.TrimSpace(comment), CreatedAt: now})
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

func (d Definition) firstApprovalNode() (Node, error) {
	start := Node{}
	for _, node := range d.Nodes {
		if node.Type == NodeStart {
			start = node
			break
		}
	}
	for _, edge := range d.Transitions {
		if edge.From != start.ID {
			continue
		}
		node, ok := d.nodeByID(edge.To)
		if !ok {
			continue
		}
		if node.Type == NodeApproval {
			return node, nil
		}
	}
	return Node{}, fmt.Errorf("workflow definition has no first approval node")
}

func (d Definition) nodeByID(id shared.ID) (Node, bool) {
	for _, node := range d.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return Node{}, false
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

func makeTasks(instanceID shared.ID, node Node, now time.Time) []Task {
	tasks := make([]Task, 0, len(node.Assignees))
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
	return tasks
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
