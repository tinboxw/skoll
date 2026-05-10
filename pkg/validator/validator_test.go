package validator

import "testing"

func TestValidatorSuccessAndFailure(t *testing.T) {
	v := New().
		Add("email", Required(), Email()).
		Add("name", Required(), MinLen(2), MaxLen(20))

	okValues := map[string]any{
		"email": "a@example.com",
		"name":  "alex",
	}
	if err := v.Validate(okValues); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	badValues := map[string]any{
		"email": "not-mail",
		"name":  "",
	}
	err := v.Validate(badValues)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	ve, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(ve.Errors) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(ve.Errors))
	}
}

func TestValidateFunction(t *testing.T) {
	rules := map[string][]Rule{
		"email": {Required(), Email()},
	}
	if err := Validate(map[string]any{"email": "ok@test.com"}, rules); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if err := Validate(map[string]any{"email": "bad"}, rules); err == nil {
		t.Fatalf("expected error")
	}
}
