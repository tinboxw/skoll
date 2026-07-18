package security

import (
	"testing"
	"time"
)

func TestSignAndParseJWT(t *testing.T) {
	now := time.Now().UTC()
	token, err := SignJWT("secret", JWTIdentity{
		Subject:          "u-1",
		OrganizationID:   "org-child",
		OrganizationPath: []string{"org-root", "org-child"},
		Role:             "admin",
		Roles:            []string{"admin", "auditor"},
	}, time.Minute, now)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	claims, err := ParseJWT("secret", token)
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	if claims.Subject != "u-1" || claims.OrganizationID != "org-child" || claims.Role != "admin" {
		t.Fatalf("unexpected identity claims: %+v", claims)
	}
	if len(claims.OrganizationPath) != 2 || claims.OrganizationPath[0] != "org-root" || claims.OrganizationPath[1] != "org-child" {
		t.Fatalf("unexpected organization path: %v", claims.OrganizationPath)
	}
	if len(claims.Roles) != 2 || !claims.HasRole("admin") || !claims.HasRole("auditor") {
		t.Fatalf("unexpected role claims: %v", claims.Roles)
	}
}

func TestParseJWTRejectsBadSignature(t *testing.T) {
	token, err := SignJWT("secret", JWTIdentity{Subject: "u-1", Role: "admin", Roles: []string{"admin"}}, time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	if _, err := ParseJWT("wrong", token); err == nil {
		t.Fatalf("expected signature error")
	}
}

func TestSignJWTRejectsUntrustedIdentityShape(t *testing.T) {
	tests := []struct {
		name     string
		identity JWTIdentity
	}{
		{name: "missing subject", identity: JWTIdentity{Roles: []string{"user"}}},
		{name: "missing roles", identity: JWTIdentity{Subject: "u-1"}},
		{name: "primary role outside roles", identity: JWTIdentity{Subject: "u-1", Role: "admin", Roles: []string{"user"}}},
		{name: "path without organization", identity: JWTIdentity{Subject: "u-1", OrganizationPath: []string{"org-1"}, Roles: []string{"user"}}},
		{name: "path does not end at organization", identity: JWTIdentity{Subject: "u-1", OrganizationID: "org-2", OrganizationPath: []string{"org-1"}, Roles: []string{"user"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := SignJWT("secret", test.identity, time.Minute, time.Now().UTC()); err == nil {
				t.Fatal("expected identity validation error")
			}
		})
	}
}
