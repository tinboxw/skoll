package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakeUserService struct {
	createInputs []usersvc.CreateUserInput
	batchInput   usersvc.BatchCreateInput
	lastUpdate   usersvc.UpdateEmailInput
	lastDisable  struct {
		id      string
		actorID string
	}
}

func (f *fakeUserService) Create(_ context.Context, in usersvc.CreateUserInput) (*domainuser.User, error) {
	f.createInputs = append(f.createInputs, in)
	if strings.EqualFold(strings.TrimSpace(in.Account), "bad") {
		return nil, fmt.Errorf("invalid account")
	}
	return &domainuser.User{ID: shared.ID("1"), Account: in.Account, Name: in.Name, Email: domainuser.Email(in.Email)}, nil
}

func (f *fakeUserService) CreateBatch(_ context.Context, in usersvc.BatchCreateInput) ([]usersvc.BatchCreateResult, error) {
	f.batchInput = in
	out := make([]usersvc.BatchCreateResult, 0, len(in.Items))
	for idx, item := range in.Items {
		if strings.EqualFold(strings.TrimSpace(item.Account), "bad") {
			out = append(out, usersvc.BatchCreateResult{Index: idx, Account: item.Account, Success: false, Message: "invalid account"})
			if in.Atomic {
				return out, nil
			}
			continue
		}
		out = append(out, usersvc.BatchCreateResult{Index: idx, Account: item.Account, Success: true, Message: "ok", ID: "1"})
	}
	return out, nil
}

func (f *fakeUserService) Get(_ context.Context, _ string) (*domainuser.User, error) {
	return nil, nil
}

func (f *fakeUserService) List(_ context.Context, _ usersvc.ListInput) ([]*domainuser.User, error) {
	return nil, nil
}

func (f *fakeUserService) Update(_ context.Context, _ usersvc.UpdateUserInput) (*domainuser.User, error) {
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

func (f *fakeUserService) Delete(_ context.Context, _ string) error {
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

func TestUserHandlerCreateBatch(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"items":[{"account":"u1","name":"U1","email":"u1@example.com","passwordHash":"password-1234"},{"account":"bad","name":"Bad","email":"bad@example.com","passwordHash":"password-1234"}]}`))
	resp := httptest.NewRecorder()

	h.createBatch(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Code string `json:"code"`
		Data struct {
			SuccessCount int `json:"successCount"`
			FailureCount int `json:"failureCount"`
			Results      []struct {
				Account string `json:"account"`
				Success bool   `json:"success"`
			} `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "ok" {
		t.Fatalf("unexpected response code: %s", body.Code)
	}
	if body.Data.SuccessCount != 1 || body.Data.FailureCount != 1 {
		t.Fatalf("unexpected counters: success=%d failure=%d", body.Data.SuccessCount, body.Data.FailureCount)
	}
	if len(body.Data.Results) != 2 {
		t.Fatalf("unexpected results length: %d", len(body.Data.Results))
	}
}

func TestUserHandlerCreateBatchAtomicFlag(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"atomic":true,"items":[{"account":"u1","name":"U1","email":"u1@example.com","passwordHash":"password-1234"}]}`))
	resp := httptest.NewRecorder()

	h.createBatch(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !svc.batchInput.Atomic {
		t.Fatalf("expected atomic batch flag to be true")
	}
}
