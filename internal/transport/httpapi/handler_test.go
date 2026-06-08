package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
)

func TestHandlerEvaluateAndListAuditEvents(t *testing.T) {
	store := audit.NewMemoryStore()
	service := runtimeSvc.NewService(policy.NewEngine([]domain.Policy{
		{
			ID:       "redact-sensitive-response",
			TenantID: "tenant-a",
			Enabled:  true,
			Priority: 80,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeResponse},
				DataLabels: []string{"pii"},
			},
			Decision: domain.DecisionRedact,
			Reason:   "sensitive response must be redacted",
		},
	}), store)
	server := httptest.NewServer(NewRouter(service, store))
	defer server.Close()

	body := bytes.NewBufferString(`{
		"tenant_id":"tenant-a",
		"agent_id":"agent-support-001",
		"user_id":"user-001",
		"task_id":"ticket-001",
		"event_type":"response",
		"action":"write",
		"data_labels":["pii"]
	}`)
	resp, err := http.Post(server.URL+"/v1/evaluate", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	var result domain.EvaluationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if result.Decision != domain.DecisionRedact {
		t.Fatalf("Decision = %s, want redact", result.Decision)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/v1/audit/events", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	listResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", listResp.StatusCode)
	}
	var payload struct {
		Events []domain.AuditRecord `json:"events"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(payload.Events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(payload.Events))
	}
}
