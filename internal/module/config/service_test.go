package config

import (
	"reflect"
	"testing"
)

func TestServiceSetGetList(t *testing.T) {
	svc := NewService()
	svc.Set("system.name", "skoll", "display name")
	svc.Set("system.env", "dev", "environment")
	svc.Set("system.env", "prod", "environment")

	item, err := svc.Get("system.env")
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if item.Value != "prod" {
		t.Fatalf("unexpected config value: %s", item.Value)
	}

	list := svc.List()
	wantKeys := []string{"system.env", "system.name"}
	gotKeys := make([]string, 0, len(list))
	for _, v := range list {
		gotKeys = append(gotKeys, v.Key)
	}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("unexpected config keys: got %v want %v", gotKeys, wantKeys)
	}
}
