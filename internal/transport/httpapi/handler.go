package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
)

type Handler struct {
	service *runtimeSvc.Service
	store   audit.Store
}

func NewRouter(service *runtimeSvc.Service, store audit.Store) http.Handler {
	handler := &Handler{service: service, store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.healthz)
	mux.HandleFunc("POST /v1/evaluate", handler.evaluate)
	mux.HandleFunc("GET /v1/audit/events", handler.listAuditEvents)
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
