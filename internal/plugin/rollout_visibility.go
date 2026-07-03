package plugin

import (
	"hash/fnv"
	"sort"
	"strings"
)

type RolloutResourceType string

const (
	RolloutResourceRoute   RolloutResourceType = "route"
	RolloutResourceMenu    RolloutResourceType = "menu"
	RolloutResourceFeature RolloutResourceType = "feature"
)

type RolloutStrategyType string

const (
	RolloutStrategyPercent RolloutStrategyType = "percent"
	RolloutStrategyTag     RolloutStrategyType = "tag"
	RolloutStrategyCanary  RolloutStrategyType = "canary"
)

type RolloutVisibilityRule struct {
	PluginID      string
	Enabled       bool
	StrategyType  RolloutStrategyType
	Percent       int
	Tags          []string
	CanaryVersion string
	Routes        []string
	Menus         []string
	Features      []string
}

type RolloutVisibilitySubject struct {
	UserID string
	Roles  []string
	Tags   []string
	Seed   string
}

type RolloutVisibilityRequest struct {
	PluginID     string
	ResourceType RolloutResourceType
	ResourceKey  string
	Subject      RolloutVisibilitySubject
	Version      string
}

type RolloutVisibilityDecision struct {
	Visible      bool   `json:"visible"`
	PluginID     string `json:"pluginId"`
	ResourceType string `json:"resourceType"`
	ResourceKey  string `json:"resourceKey"`
	StrategyType string `json:"strategyType"`
	Reason       string `json:"reason"`
}

type RolloutVisibilityService struct {
	rules map[string]RolloutVisibilityRule
}

func NewRolloutVisibilityService(rules []RolloutVisibilityRule) *RolloutVisibilityService {
	svc := &RolloutVisibilityService{rules: map[string]RolloutVisibilityRule{}}
	for _, rule := range rules {
		rule.PluginID = strings.TrimSpace(rule.PluginID)
		if rule.PluginID == "" {
			continue
		}
		if rule.StrategyType == "" {
			rule.StrategyType = RolloutStrategyPercent
		}
		rule.Tags = normalizeRolloutStrings(rule.Tags)
		rule.Routes = normalizeRolloutStrings(rule.Routes)
		rule.Menus = normalizeRolloutStrings(rule.Menus)
		rule.Features = normalizeRolloutStrings(rule.Features)
		if rule.Percent < 0 {
			rule.Percent = 0
		}
		if rule.Percent > 100 {
			rule.Percent = 100
		}
		svc.rules[rule.PluginID] = rule
	}
	return svc
}

func (s *RolloutVisibilityService) Decide(req RolloutVisibilityRequest) RolloutVisibilityDecision {
	pluginID := strings.TrimSpace(req.PluginID)
	resourceKey := strings.TrimSpace(req.ResourceKey)
	decision := RolloutVisibilityDecision{
		Visible:      true,
		PluginID:     pluginID,
		ResourceType: string(req.ResourceType),
		ResourceKey:  resourceKey,
		StrategyType: string(RolloutStrategyPercent),
		Reason:       "no rollout rule",
	}
	if s == nil {
		return decision
	}
	rule, ok := s.rules[pluginID]
	if !ok {
		return decision
	}
	decision.StrategyType = string(rule.StrategyType)
	if !rule.Enabled {
		decision.Visible = false
		decision.Reason = "rollout disabled"
		return decision
	}
	if !rolloutRuleCoversResource(rule, req.ResourceType, resourceKey) {
		decision.Visible = true
		decision.Reason = "resource not governed"
		return decision
	}
	switch rule.StrategyType {
	case RolloutStrategyTag:
		decision.Visible = hasRolloutTagMatch(req.Subject.Tags, rule.Tags) || hasRolloutTagMatch(req.Subject.Roles, rule.Tags)
		if decision.Visible {
			decision.Reason = "tag matched"
		} else {
			decision.Reason = "tag not matched"
		}
	case RolloutStrategyCanary:
		decision.Visible = strings.TrimSpace(req.Version) != "" && strings.TrimSpace(req.Version) == strings.TrimSpace(rule.CanaryVersion)
		if decision.Visible {
			decision.Reason = "canary version matched"
		} else {
			decision.Reason = "canary version not matched"
		}
	default:
		bucket := rolloutBucket(pluginID, resourceKey, req.Subject)
		decision.Visible = bucket < rule.Percent
		if decision.Visible {
			decision.Reason = "percent bucket allowed"
		} else {
			decision.Reason = "percent bucket denied"
		}
	}
	return decision
}

func rolloutRuleCoversResource(rule RolloutVisibilityRule, typ RolloutResourceType, key string) bool {
	switch typ {
	case RolloutResourceRoute:
		return containsRolloutResource(rule.Routes, key)
	case RolloutResourceMenu:
		return containsRolloutResource(rule.Menus, key)
	case RolloutResourceFeature:
		return containsRolloutResource(rule.Features, key)
	default:
		return false
	}
}

func containsRolloutResource(values []string, key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, value := range values {
		if value == "*" || value == key {
			return true
		}
	}
	return false
}

func hasRolloutTagMatch(left []string, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	set := map[string]struct{}{}
	for _, item := range normalizeRolloutStrings(left) {
		set[item] = struct{}{}
	}
	for _, item := range normalizeRolloutStrings(right) {
		if _, ok := set[item]; ok {
			return true
		}
	}
	return false
}

func rolloutBucket(pluginID string, resourceKey string, subject RolloutVisibilitySubject) int {
	seed := strings.TrimSpace(subject.Seed)
	if seed == "" {
		seed = strings.TrimSpace(subject.UserID)
	}
	if seed == "" {
		seed = strings.Join(normalizeRolloutStrings(subject.Roles), ",")
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.TrimSpace(pluginID) + "|" + strings.TrimSpace(resourceKey) + "|" + seed))
	return int(hash.Sum32() % 100)
}

func normalizeRolloutStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
