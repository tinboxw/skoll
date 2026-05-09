package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakeUserService struct {
	lastUpdate  usersvc.UpdateEmailInput
	lastDisable struct {
		id      string
		actorID string
	}
}

func (f *fakeUserService) Create(_ context.Context, _ usersvc.CreateUserInput) (*domainuser.User, error) {
	return nil, nil
}

func (f *fakeUserService) Get(_ context.Context, _ string) (*domainuser.User, error) {
	return nil, nil
}

func (f *fakeUserService) List(_ context.Context, _ usersvc.ListInput) ([]*domainuser.User, error) {
	return nil, nil
}

func (f *fakeUserService) UpdateEmail(_ context.Context, in usersvc.UpdateEmailInput) (*domainuser.User, error) {
	f.lastUpdate = in
	return nil, nil
}

func (f *fakeUserService) Disable(_ context.Context, id, actorID string) error {
	f.lastDisable.id = id
	f.lastDisable.actorID = actorID
	return nil
}

func TestUserHandlerUpdateEmailActorIDFallback(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	t.Run("use request actor id first", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/users/u-1/email", bytes.NewBufferString(`{"email":"a@b.com","actorId":"manual-actor"}`))
		req.SetPathValue("id", "u-1")
		req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-actor"}))
		resp := httptest.NewRecorder()

		h.updateEmail(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}
		if svc.lastUpdate.ActorID != "manual-actor" {
			t.Fatalf("expected manual actor id, got %q", svc.lastUpdate.ActorID)
		}
	})

	t.Run("fallback to jwt subject", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/users/u-2/email", bytes.NewBufferString(`{"email":"a@b.com","actorId":""}`))
		req.SetPathValue("id", "u-2")
		req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-actor"}))
		resp := httptest.NewRecorder()

		h.updateEmail(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}
		if svc.lastUpdate.ActorID != "jwt-actor" {
			t.Fatalf("expected jwt actor id, got %q", svc.lastUpdate.ActorID)
		}
	})
}

func TestUserHandlerDisableActorIDFallback(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/u-3/disable", bytes.NewBufferString(`{}`))
	req.SetPathValue("id", "u-3")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-disabler"}))
	resp := httptest.NewRecorder()

	h.disable(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if svc.lastDisable.actorID != "jwt-disabler" {
		t.Fatalf("expected jwt actor id, got %q", svc.lastDisable.actorID)
	}

	var body response
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "ok" {
		t.Fatalf("unexpected response code: %s", body.Code)
	}
}
