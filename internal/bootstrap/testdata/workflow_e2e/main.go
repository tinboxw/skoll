package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type workflowCall struct {
	Operation  string                              `json:"operation"`
	ID         string                              `json:"id,omitempty"`
	Definition pluginsdk.WorkflowDefinitionInput   `json:"definition,omitempty"`
	Start      pluginsdk.WorkflowStartInput        `json:"start,omitempty"`
	Task       pluginsdk.WorkflowTaskActionInput   `json:"task,omitempty"`
	Target     pluginsdk.WorkflowTargetActionInput `json:"target,omitempty"`
}

func main() {
	client, err := pluginclient.FromEnvironment()
	if err != nil {
		panic(err)
	}
	host, err := client.HostServices()
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /v1/workflows", func(w http.ResponseWriter, r *http.Request) {
		var call workflowCall
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&call); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		result, err := invokeWorkflow(ctx, host.Workflows, call)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	server := &http.Server{
		Addr:              os.Getenv(pluginclient.EnvironmentPluginAddress),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = server.ListenAndServe() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	_ = server.Shutdown(context.Background())
}

func invokeWorkflow(ctx context.Context, workflows pluginsdk.WorkflowService, call workflowCall) (any, error) {
	switch strings.TrimSpace(call.Operation) {
	case "create-definition":
		return workflows.CreateDefinition(ctx, call.Definition)
	case "publish-definition":
		return workflows.PublishDefinition(ctx, call.ID)
	case "start":
		return workflows.Start(ctx, call.Start)
	case "get-instance":
		return workflows.GetInstance(ctx, call.ID)
	case "approve":
		return workflows.Approve(ctx, call.Task)
	case "reject":
		return workflows.Reject(ctx, call.Task)
	case "delegate":
		return workflows.Delegate(ctx, call.Target)
	default:
		return nil, errors.New("unsupported workflow acceptance operation")
	}
}

func bearerToken(r *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
