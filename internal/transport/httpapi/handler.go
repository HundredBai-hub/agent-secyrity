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
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
)

type contextKey string

const apiKeyContextKey contextKey = "api_key"

type Options struct {
	Service     *runtimeSvc.Service
	Audit       audit.Store
	PolicyPacks policypack.Store
	Approvals   approval.Store
	APIKeys     auth.APIKeyConfig
}

type Handler struct {
	service       *runtimeSvc.Service
	store         audit.Store
	packStore     policypack.Store
	approvalStore approval.Store
	apiKeys       auth.APIKeyConfig
}

func NewRouter(service *runtimeSvc.Service, store audit.Store) http.Handler {
	return NewRouterWithOptions(Options{Service: service, Audit: store})
}

func NewRouterWithPolicyPacks(service *runtimeSvc.Service, store audit.Store, packStore policypack.Store) http.Handler {
	return NewRouterWithOptions(Options{Service: service, Audit: store, PolicyPacks: packStore})
}

func NewRouterWithStores(service *runtimeSvc.Service, store audit.Store, packStore policypack.Store, approvalStore approval.Store) http.Handler {
	return NewRouterWithOptions(Options{Service: service, Audit: store, PolicyPacks: packStore, Approvals: approvalStore})
}

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

func (h *Handler) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /v1/evaluate", h.evaluate)
	mux.HandleFunc("GET /v1/audit/events", h.listAuditEvents)
	mux.HandleFunc("PUT /v1/tenants/{tenant_id}/policy-packs/{pack_id}", h.upsertPolicyPack)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/policy-packs", h.listPolicyPacks)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/policy-packs/{pack_id}", h.getPolicyPack)
	mux.HandleFunc("PATCH /v1/tenants/{tenant_id}/policy-packs/{pack_id}/enabled", h.setPolicyPackEnabled)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/approvals", h.listApprovals)
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/approvals/{approval_id}", h.getApproval)
	mux.HandleFunc("POST /v1/tenants/{tenant_id}/approvals/{approval_id}/decide", h.decideApproval)
	return mux
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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
		if errors.Is(err, domain.ErrInvalidRuntimeEvent) {
			status = http.StatusBadRequest
			code = "invalid_runtime_event"
		}
		writeError(w, status, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if h.apiKeys.Enabled() {
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "tenant_id_required", "tenant_id query parameter is required when API key authentication is enabled")
			return
		}
		if !h.authorizeTenant(w, r, tenantID) {
			return
		}
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
			return
		}
		limit = parsed
	}
	records, err := h.store.List(r.Context(), audit.ListOptions{Limit: limit, TenantID: tenantID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.AuditRecord{"events": records})
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
