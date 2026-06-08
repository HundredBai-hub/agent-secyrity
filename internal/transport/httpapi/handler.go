// Package httpapi exposes the Agent Security Platform REST API.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/auth"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
)

type contextKey string

const apiKeyContextKey contextKey = "api_key"

// Options configures the HTTP API router dependencies.
type Options struct {
	Service     *runtimeSvc.Service
	Audit       audit.Store
	PolicyPacks policypack.Store
	Approvals   approval.Store
	APIKeys     auth.APIKeyConfig
}

// Handler owns HTTP route handlers and shared dependencies.
type Handler struct {
	service       *runtimeSvc.Service
	store         audit.Store
	packStore     policypack.Store
	approvalStore approval.Store
	apiKeys       auth.APIKeyConfig
}

// NewRouter creates a router with runtime service and audit store dependencies.
func NewRouter(service *runtimeSvc.Service, store audit.Store) http.Handler {
	return NewRouterWithOptions(Options{Service: service, Audit: store})
}

// NewRouterWithPolicyPacks creates a router with policy pack management enabled.
func NewRouterWithPolicyPacks(service *runtimeSvc.Service, store audit.Store, packStore policypack.Store) http.Handler {
	return NewRouterWithOptions(Options{Service: service, Audit: store, PolicyPacks: packStore})
}

// NewRouterWithStores creates a router with policy pack and approval stores enabled.
func NewRouterWithStores(service *runtimeSvc.Service, store audit.Store, packStore policypack.Store, approvalStore approval.Store) http.Handler {
	return NewRouterWithOptions(Options{Service: service, Audit: store, PolicyPacks: packStore, Approvals: approvalStore})
}

// NewRouterWithOptions creates a router from explicit dependencies.
func NewRouterWithOptions(opts Options) http.Handler {
	handler := &Handler{
		service:       opts.Service,
		store:         opts.Audit,
		packStore:     opts.PolicyPacks,
		approvalStore: opts.Approvals,
		apiKeys:       opts.APIKeys,
	}
	return handler.withAuth(handler.router())
}

// router registers API routes.
func (h *Handler) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /v1/evaluate", h.evaluate)
	mux.HandleFunc("GET /v1/audit/events", h.listAuditEvents)
	mux.HandleFunc("PUT /v1/tenants/{tenant_id}/policy-packs/{pack_id}", h.upsertPolicyPack)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/policy-packs", h.listPolicyPacks)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/policy-packs/{pack_id}", h.getPolicyPack)
	mux.HandleFunc("PATCH /v1/tenants/{tenant_id}/policy-packs/{pack_id}/enabled", h.setPolicyPackEnabled)
	mux.HandleFunc("POST /v1/tenants/{tenant_id}/policy-simulations", h.simulatePolicyPack)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/approvals", h.listApprovals)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/approvals/{approval_id}", h.getApproval)
	mux.HandleFunc("POST /v1/tenants/{tenant_id}/approvals/{approval_id}/decide", h.decideApproval)
	return mux
}

// healthz reports service liveness.
func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// evaluate evaluates one runtime event and records the audit result.
func (h *Handler) evaluate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var event domain.RuntimeEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if !h.authorizeTenant(w, r, event.TenantID) {
		return
	}
	result, err := h.service.Evaluate(r.Context(), event)
	if err != nil {
		status := http.StatusInternalServerError
		code := "evaluation_failed"
		var validationErr *domain.ValidationError
		if errors.Is(err, domain.ErrInvalidRuntimeEvent) {
			status = http.StatusUnprocessableEntity
			code = "invalid_runtime_event"
		}
		if errors.As(err, &validationErr) {
			writeErrorWithDetails(w, status, code, validationErr.Message, map[string][]domain.FieldError{"fields": validationErr.Fields})
			return
		}
		writeError(w, status, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// listAuditEvents queries audit events using tenant, actor, task, decision and event filters.
func (h *Handler) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	opts, err := parseAuditListOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_audit_query", err.Error())
		return
	}
	if h.apiKeys.Enabled() {
		if opts.TenantID == "" {
			writeError(w, http.StatusBadRequest, "tenant_id_required", "tenant_id query parameter is required when API key authentication is enabled")
			return
		}
		if !h.authorizeTenant(w, r, opts.TenantID) {
			return
		}
	}
	normalized := opts.Normalize()
	records, err := h.store.List(r.Context(), normalized)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_list_failed", err.Error())
		return
	}
	nextPageOpts := normalized
	nextPageOpts.Limit = 1
	nextPageOpts.Offset = normalized.Offset + normalized.Limit
	nextPageRecords, err := h.store.List(r.Context(), nextPageOpts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, auditEventsResponse{
		Events: records,
		Pagination: paginationResponse{
			Limit:   normalized.Limit,
			Offset:  normalized.Offset,
			Count:   len(records),
			HasMore: len(nextPageRecords) > 0,
		},
	})
}

func (h *Handler) upsertPolicyPack(w http.ResponseWriter, r *http.Request) {
	if h.packStore == nil {
		writeError(w, http.StatusNotImplemented, "policy_pack_store_disabled", "policy pack store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	defer r.Body.Close()
	var pack domain.PolicyPack
	if err := json.NewDecoder(r.Body).Decode(&pack); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	pack.TenantID = r.PathValue("tenant_id")
	pack.ID = r.PathValue("pack_id")
	if err := h.packStore.Upsert(r.Context(), pack); err != nil {
		writeError(w, http.StatusBadRequest, "policy_pack_upsert_failed", err.Error())
		return
	}
	saved, err := h.packStore.Get(r.Context(), pack.TenantID, pack.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "policy_pack_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) listPolicyPacks(w http.ResponseWriter, r *http.Request) {
	if h.packStore == nil {
		writeError(w, http.StatusNotImplemented, "policy_pack_store_disabled", "policy pack store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	packs, err := h.packStore.List(r.Context(), r.PathValue("tenant_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "policy_pack_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.PolicyPack{"policy_packs": packs})
}

func (h *Handler) getPolicyPack(w http.ResponseWriter, r *http.Request) {
	if h.packStore == nil {
		writeError(w, http.StatusNotImplemented, "policy_pack_store_disabled", "policy pack store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	pack, err := h.packStore.Get(r.Context(), r.PathValue("tenant_id"), r.PathValue("pack_id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, policypack.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, "policy_pack_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pack)
}

func (h *Handler) setPolicyPackEnabled(w http.ResponseWriter, r *http.Request) {
	if h.packStore == nil {
		writeError(w, http.StatusNotImplemented, "policy_pack_store_disabled", "policy pack store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	defer r.Body.Close()
	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := h.packStore.SetEnabled(r.Context(), r.PathValue("tenant_id"), r.PathValue("pack_id"), payload.Enabled); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, policypack.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, "policy_pack_set_enabled_failed", err.Error())
		return
	}
	pack, err := h.packStore.Get(r.Context(), r.PathValue("tenant_id"), r.PathValue("pack_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "policy_pack_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pack)
}

func (h *Handler) simulatePolicyPack(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if !h.authorizeTenant(w, r, tenantID) {
		return
	}
	defer r.Body.Close()
	var request policy.SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if request.Event.TenantID != tenantID {
		writeError(w, http.StatusForbidden, "forbidden", "simulation event tenant_id must match path tenant_id")
		return
	}
	result, err := policy.Simulate(request)
	if err != nil {
		var validationErr *domain.ValidationError
		if errors.As(err, &validationErr) {
			writeErrorWithDetails(w, http.StatusUnprocessableEntity, "invalid_runtime_event", validationErr.Message, map[string][]domain.FieldError{"fields": validationErr.Fields})
			return
		}
		writeError(w, http.StatusInternalServerError, "policy_simulation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listApprovals(w http.ResponseWriter, r *http.Request) {
	if h.approvalStore == nil {
		writeError(w, http.StatusNotImplemented, "approval_store_disabled", "approval store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	approvals, err := h.approvalStore.List(r.Context(), r.PathValue("tenant_id"), approval.ListOptions{Limit: 100})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "approval_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.ApprovalRequest{"approvals": approvals})
}

func (h *Handler) getApproval(w http.ResponseWriter, r *http.Request) {
	if h.approvalStore == nil {
		writeError(w, http.StatusNotImplemented, "approval_store_disabled", "approval store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	request, err := h.approvalStore.Get(r.Context(), r.PathValue("tenant_id"), r.PathValue("approval_id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, approval.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, "approval_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (h *Handler) decideApproval(w http.ResponseWriter, r *http.Request) {
	if h.approvalStore == nil {
		writeError(w, http.StatusNotImplemented, "approval_store_disabled", "approval store is not configured")
		return
	}
	if !h.authorizeTenant(w, r, r.PathValue("tenant_id")) {
		return
	}
	defer r.Body.Close()
	var payload struct {
		Decision  domain.ApprovalStatus `json:"decision"`
		DecidedBy string                `json:"decided_by"`
		Reason    string                `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	request, err := h.approvalStore.Decide(r.Context(), r.PathValue("tenant_id"), r.PathValue("approval_id"), approval.DecisionInput{
		Status:    payload.Decision,
		DecidedBy: payload.DecidedBy,
		Reason:    payload.Reason,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, approval.ErrNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, approval.ErrApprovalExpired) || errors.Is(err, approval.ErrApprovalAlreadyDecided) || errors.Is(err, approval.ErrInvalidDecision) {
			status = http.StatusConflict
		}
		writeError(w, status, "approval_decide_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, request)
}

// auditEventsResponse is the response contract for audit list queries.
type auditEventsResponse struct {
	Events     []domain.AuditRecord `json:"events"`
	Pagination paginationResponse   `json:"pagination"`
}

// paginationResponse describes the returned audit page without requiring a total count query.
type paginationResponse struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Count   int  `json:"count"`
	HasMore bool `json:"has_more"`
}

// parseAuditListOptions validates external query parameters for audit listing.
func parseAuditListOptions(r *http.Request) (audit.ListOptions, error) {
	query := r.URL.Query()
	opts := audit.ListOptions{
		TenantID: query.Get("tenant_id"),
		AgentID:  query.Get("agent_id"),
		UserID:   query.Get("user_id"),
		TaskID:   query.Get("task_id"),
	}
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return audit.ListOptions{}, errors.New("limit must be an integer")
		}
		if parsed < 0 {
			return audit.ListOptions{}, errors.New("limit must be greater than or equal to 0")
		}
		opts.Limit = parsed
	}
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return audit.ListOptions{}, errors.New("offset must be an integer")
		}
		if parsed < 0 {
			return audit.ListOptions{}, errors.New("offset must be greater than or equal to 0")
		}
		opts.Offset = parsed
	}
	if raw := query.Get("decision"); raw != "" {
		decision := domain.Decision(raw)
		if !validDecision(decision) {
			return audit.ListOptions{}, errors.New("decision is unsupported")
		}
		opts.Decision = decision
	}
	if raw := query.Get("event_type"); raw != "" {
		eventType := domain.EventType(raw)
		if !eventType.Valid() {
			return audit.ListOptions{}, errors.New("event_type is unsupported")
		}
		opts.EventType = eventType
	}
	return opts, nil
}

// validDecision reports whether a decision query value is supported.
func validDecision(decision domain.Decision) bool {
	switch decision {
	case domain.DecisionAllow, domain.DecisionRecord, domain.DecisionRedact, domain.DecisionRequireApproval, domain.DecisionDeny:
		return true
	default:
		return false
	}
}

// withAuth applies API key authentication when configured.
func (h *Handler) withAuth(next http.Handler) http.Handler {
	if !h.apiKeys.Enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		key, err := h.apiKeys.AuthenticateRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid API key")
			return
		}
		ctx := context.WithValue(r.Context(), apiKeyContextKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authorizeTenant checks that the authenticated API key can access the tenant.
func (h *Handler) authorizeTenant(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	if !h.apiKeys.Enabled() {
		return true
	}
	key, ok := r.Context().Value(apiKeyContextKey).(auth.APIKey)
	if !ok || !key.AllowsTenant(tenantID) {
		writeError(w, http.StatusForbidden, "forbidden", "API key is not allowed to access tenant")
		return false
	}
	return true
}

// writeJSON writes a JSON response with a status code.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes a standard JSON error response.
func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

// writeErrorWithDetails writes a standard JSON error response with structured details.
func writeErrorWithDetails(w http.ResponseWriter, status int, code string, message string, details any) {
	writeJSON(w, status, map[string]any{
		"error":   code,
		"message": message,
		"details": details,
	})
}
