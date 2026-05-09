package security

import (
	"testing"
	"time"
)

func TestSignAndParseJWT(t *testing.T) {
	now := time.Now().UTC()
	token, err := SignJWT("secret", "u-1", "admin", time.Minute, now)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	claims, err := ParseJWT("secret", token)
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	if claims.Subject != "u-1" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestParseJWTRejectsBadSignature(t *testing.T) {
	token, err := SignJWT("secret", "u-1", "admin", time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	if _, err := ParseJWT("wrong", token); err == nil {
		t.Fatalf("expected signature error")
	}
}
