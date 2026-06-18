package audit

import "testing"

func TestAuditActionValidateAcceptsModuleResourceAction(t *testing.T) {
	cases := []AuditAction{
		"user.account.create",
		"rbac.role.grant",
		"plugin.release.rollout",
		"file.object.upload",
		"menu.node.update",
		"auth.login.fail",
	}

	for _, action := range cases {
		t.Run(action.String(), func(t *testing.T) {
			if err := action.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestAuditActionValidateRejectsInvalidNames(t *testing.T) {
	cases := []AuditAction{
		"",
		"login",
		"login_failed",
		"plugin_catalog_import",
		"user.create",
		"user.account.create.extra",
		"User.Account.Create",
		"user.account.",
		"user..create",
		"user.account.create!",
		"1user.account.create",
	}

	for _, action := range cases {
		t.Run(action.String(), func(t *testing.T) {
			if err := action.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseAuditActionNormalizesInput(t *testing.T) {
	got, err := ParseAuditAction(" User.Account.Create ")
	if err != nil {
		t.Fatalf("ParseAuditAction() error = %v", err)
	}
	if got != AuditAction("user.account.create") {
		t.Fatalf("ParseAuditAction() = %q", got)
	}
}

func TestParseAuditActionRejectsInvalidInput(t *testing.T) {
	if _, err := ParseAuditAction("plugin_catalog_import"); err == nil {
		t.Fatal("expected invalid audit action error")
	}
}

func TestAuditActionParts(t *testing.T) {
	action := AuditAction("plugin.release.rollout")
	if action.Module() != "plugin" || action.Resource() != "release" || action.Operation() != "rollout" {
		t.Fatalf("unexpected parts: module=%q resource=%q operation=%q", action.Module(), action.Resource(), action.Operation())
	}

	invalid := AuditAction("legacy")
	if invalid.Module() != "" || invalid.Resource() != "" || invalid.Operation() != "" {
		t.Fatalf("invalid action should not expose parts")
	}
}
