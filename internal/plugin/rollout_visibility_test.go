package plugin

import "testing"

func TestRolloutVisibilityPercentAffectsRouteMenuFeature(t *testing.T) {
	service := NewRolloutVisibilityService([]RolloutVisibilityRule{
		{
			PluginID:     "reports",
			Enabled:      true,
			StrategyType: RolloutStrategyPercent,
			Percent:      0,
			Routes:       []string{"/skoll/plugins/reports"},
			Menus:        []string{"plugin.reports"},
			Features:     []string{"reports.export"},
		},
	})
	subject := RolloutVisibilitySubject{UserID: "alice"}

	for _, tc := range []struct {
		name string
		typ  RolloutResourceType
		key  string
	}{
		{name: "route", typ: RolloutResourceRoute, key: "/skoll/plugins/reports"},
		{name: "menu", typ: RolloutResourceMenu, key: "plugin.reports"},
		{name: "feature", typ: RolloutResourceFeature, key: "reports.export"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decision := service.Decide(RolloutVisibilityRequest{
				PluginID:     "reports",
				ResourceType: tc.typ,
				ResourceKey:  tc.key,
				Subject:      subject,
			})
			if decision.Visible {
				t.Fatalf("expected %s to be hidden by 0 percent rollout, got %+v", tc.name, decision)
			}
		})
	}
}

func TestRolloutVisibilityTagAllowsMatchedSubjects(t *testing.T) {
	service := NewRolloutVisibilityService([]RolloutVisibilityRule{
		{
			PluginID:     "reports",
			Enabled:      true,
			StrategyType: RolloutStrategyTag,
			Tags:         []string{"beta"},
			Routes:       []string{"/skoll/plugins/reports"},
		},
	})

	denied := service.Decide(RolloutVisibilityRequest{
		PluginID:     "reports",
		ResourceType: RolloutResourceRoute,
		ResourceKey:  "/skoll/plugins/reports",
		Subject:      RolloutVisibilitySubject{Tags: []string{"stable"}},
	})
	if denied.Visible {
		t.Fatalf("expected unmatched tag to be hidden, got %+v", denied)
	}
	allowed := service.Decide(RolloutVisibilityRequest{
		PluginID:     "reports",
		ResourceType: RolloutResourceRoute,
		ResourceKey:  "/skoll/plugins/reports",
		Subject:      RolloutVisibilitySubject{Tags: []string{"beta"}},
	})
	if !allowed.Visible {
		t.Fatalf("expected matched tag to be visible, got %+v", allowed)
	}
}

func TestRolloutVisibilityCanaryRequiresVersionMatch(t *testing.T) {
	service := NewRolloutVisibilityService([]RolloutVisibilityRule{
		{
			PluginID:      "reports",
			Enabled:       true,
			StrategyType:  RolloutStrategyCanary,
			CanaryVersion: "1.1.0-canary.1",
			Features:      []string{"reports.export"},
		},
	})

	denied := service.Decide(RolloutVisibilityRequest{
		PluginID:     "reports",
		ResourceType: RolloutResourceFeature,
		ResourceKey:  "reports.export",
		Version:      "1.0.0",
	})
	if denied.Visible {
		t.Fatalf("expected non-canary version to be hidden, got %+v", denied)
	}
	allowed := service.Decide(RolloutVisibilityRequest{
		PluginID:     "reports",
		ResourceType: RolloutResourceFeature,
		ResourceKey:  "reports.export",
		Version:      "1.1.0-canary.1",
	})
	if !allowed.Visible {
		t.Fatalf("expected canary version to be visible, got %+v", allowed)
	}
}

func TestRolloutVisibilityUngovernedResourceStaysVisible(t *testing.T) {
	service := NewRolloutVisibilityService([]RolloutVisibilityRule{
		{
			PluginID:     "reports",
			Enabled:      true,
			StrategyType: RolloutStrategyPercent,
			Percent:      0,
			Routes:       []string{"/skoll/plugins/reports"},
		},
	})

	decision := service.Decide(RolloutVisibilityRequest{
		PluginID:     "reports",
		ResourceType: RolloutResourceMenu,
		ResourceKey:  "plugin.reports",
		Subject:      RolloutVisibilitySubject{UserID: "alice"},
	})
	if !decision.Visible || decision.Reason != "resource not governed" {
		t.Fatalf("expected ungoverned resource to remain visible, got %+v", decision)
	}
}
