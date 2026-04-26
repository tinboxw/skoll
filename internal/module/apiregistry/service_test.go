package apiregistry

import (
	"reflect"
	"testing"
)

func TestServiceRegisterListExists(t *testing.T) {
	svc := NewService()
	svc.RegisterMany([]string{"GET /admin/v1/users", "POST:/admin/v1/users", "post:/admin/v1/users"})

	if !svc.Exists("GET:/admin/v1/users") {
		t.Fatalf("expected GET api to exist")
	}
	if !svc.Exists("POST /admin/v1/users") {
		t.Fatalf("expected POST api to exist")
	}
	if svc.Exists("DELETE:/admin/v1/users") {
		t.Fatalf("did not expect DELETE api to exist")
	}

	list := svc.List()
	want := []string{"GET:/admin/v1/users", "POST:/admin/v1/users"}
	if !reflect.DeepEqual(list, want) {
		t.Fatalf("unexpected api list: got %v want %v", list, want)
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"GET /admin/v1/users":  "GET:/admin/v1/users",
		"post:/admin/v1/users": "POST:/admin/v1/users",
		"bad":                  "",
	}

	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("normalize %q got %q want %q", input, got, want)
		}
	}
}
