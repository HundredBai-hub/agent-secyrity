package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/auth"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
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

func TestHandlerAPIKeyAuthentication(t *testing.T) {
	authConfig, err := auth.ParseAPIKeys("runtime:runtime-secret:tenant-a")
	if err != nil {
		t.Fatalf("ParseAPIKeys() error = %v", err)
	}
	store := audit.NewMemoryStore()
	service := runtimeSvc.NewService(policy.NewEngine(nil), store)
	server := httptest.NewServer(NewRouterWithOptions(Options{
		Service: service,
		Audit:   store,
		APIKeys: authConfig,
	}))
	defer server.Close()

	healthResp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("Get(healthz) error = %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", healthResp.StatusCode)
	}

	unauthorizedResp := postEvaluate(t, server.URL, "", "tenant-a")
	if unauthorizedResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorizedResp.StatusCode)
	}
	unauthorizedResp.Body.Close()

	forbiddenResp := postEvaluate(t, server.URL, "runtime-secret", "tenant-b")
	if forbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want 403", forbiddenResp.StatusCode)
	}
	forbiddenResp.Body.Close()

	allowedResp := postEvaluate(t, server.URL, "runtime-secret", "tenant-a")
	if allowedResp.StatusCode != http.StatusOK {
		t.Fatalf("allowed status = %d, want 200", allowedResp.StatusCode)
	}
	allowedResp.Body.Close()
}

func TestHandlerManagesApprovals(t *testing.T) {
	auditStore := audit.NewMemoryStore()
	approvalStore := approval.NewMemoryStore()
	request, err := approvalStore.Create(context.Background(), domain.ApprovalRequest{
		TenantID: "tenant-a",
		Event: domain.RuntimeEvent{
			TenantID:  "tenant-a",
			AgentID:   "agent-code-001",
			UserID:    "user-001",
			TaskID:    "task-001",
			EventType: domain.EventTypeToolCall,
			ToolName:  "shell",
			Action:    "execute",
		},
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		Reason:    "dangerous tool execution requires approval",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	service := runtimeSvc.NewServiceWithOptions(runtimeSvc.Options{
		Engine:        policy.NewEngine(nil),
		AuditStore:    auditStore,
		ApprovalStore: approvalStore,
		ApprovalTTL:   time.Hour,
	})
	server := httptest.NewServer(NewRouterWithStores(service, auditStore, nil, approvalStore))
	defer server.Close()

	listResp, err := http.Get(server.URL + "/v1/tenants/tenant-a/approvals")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", listResp.StatusCode)
	}
	var listPayload struct {
		Approvals []domain.ApprovalRequest `json:"approvals"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(listPayload.Approvals) != 1 {
		t.Fatalf("len(approvals) = %d, want 1", len(listPayload.Approvals))
	}

	decisionBody := bytes.NewBufferString(`{"decision":"approved","decided_by":"secops-001","reason":"approved for incident response"}`)
	decideResp, err := http.Post(server.URL+"/v1/tenants/tenant-a/approvals/"+request.ID+"/decide", "application/json", decisionBody)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer decideResp.Body.Close()
	if decideResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", decideResp.StatusCode)
	}
	var decided domain.ApprovalRequest
	if err := json.NewDecoder(decideResp.Body).Decode(&decided); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decided.Status != domain.ApprovalStatusApproved {
		t.Fatalf("Status = %s, want approved", decided.Status)
	}

	otherTenantResp, err := http.Get(server.URL + "/v1/tenants/tenant-b/approvals/" + request.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer otherTenantResp.Body.Close()
	if otherTenantResp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want 404 for other tenant", otherTenantResp.StatusCode)
	}
}

func postEvaluate(t *testing.T, serverURL string, apiKey string, tenantID string) *http.Response {
	t.Helper()
	body := bytes.NewBufferString(`{
		"tenant_id":"` + tenantID + `",
		"agent_id":"agent-support-001",
		"user_id":"user-001",
		"task_id":"ticket-001",
		"event_type":"response",
		"action":"write"
	}`)
	req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/evaluate", body)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	return resp
}

func TestHandlerManagesPolicyPacks(t *testing.T) {
	auditStore := audit.NewMemoryStore()
	packStore := policypack.NewMemoryStore()
	service := runtimeSvc.NewServiceWithPolicyPacks(packStore, auditStore)
	server := httptest.NewServer(NewRouterWithPolicyPacks(service, auditStore, packStore))
	defer server.Close()

	packJSON := `{
		"name":"Default Runtime",
		"version":"1.0.0",
		"enabled":true,
		"policies":[{
			"id":"deny-shell",
			"enabled":true,
			"priority":100,
			"conditions":{"event_types":["tool_call"],"tool_names":["shell"]},
			"decision":"deny",
			"reason":"shell is blocked"
		}]
	}`
	req, err := http.NewRequest(http.MethodPut, server.URL+"/v1/tenants/tenant-a/policy-packs/default-runtime", bytes.NewBufferString(packJSON))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	var saved domain.PolicyPack
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if saved.TenantID != "tenant-a" || saved.ID != "default-runtime" {
		t.Fatalf("saved pack = %#v", saved)
	}

	listResp, err := http.Get(server.URL + "/v1/tenants/tenant-a/policy-packs")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer listResp.Body.Close()
	var listPayload struct {
		PolicyPacks []domain.PolicyPack `json:"policy_packs"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(listPayload.PolicyPacks) != 1 {
		t.Fatalf("len(policy_packs) = %d, want 1", len(listPayload.PolicyPacks))
	}

	patchReq, err := http.NewRequest(http.MethodPatch, server.URL+"/v1/tenants/tenant-a/policy-packs/default-runtime/enabled", bytes.NewBufferString(`{"enabled":false}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", patchResp.StatusCode)
	}
}
