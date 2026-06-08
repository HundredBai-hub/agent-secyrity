package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
)

type Handler struct {
	service   *runtimeSvc.Service
	store     audit.Store
	packStore policypack.Store
}

func NewRouter(service *runtimeSvc.Service, store audit.Store) http.Handler {
	handler := &Handler{service: service, store: store}
	return handler.router()
}

func NewRouterWithPolicyPacks(service *runtimeSvc.Service, store audit.Store, packStore policypack.Store) http.Handler {
	handler := &Handler{service: service, store: store, packStore: packStore}
	return handler.router()
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
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
			return
		}
		limit = parsed
	}
	records, err := h.store.List(r.Context(), audit.ListOptions{Limit: limit})
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
