// Package httpapi 提供 HTTP JSON API 与路由。
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/service"
)

// Server HTTP 服务。
type Server struct {
	app *service.App
	mux *http.ServeMux
}

// New 构建 HTTP 服务并注册全部路由。
func New(app *service.App) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回可挂载的 HTTP handler。
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)

	s.mux.HandleFunc("POST /api/snapshots", s.handleCreateSnapshot)
	s.mux.HandleFunc("GET /api/snapshots", s.handleListSnapshots)
	s.mux.HandleFunc("GET /api/snapshots/{id}", s.handleGetSnapshot)
	s.mux.HandleFunc("POST /api/snapshots/{id}/lock", s.handleLockSnapshot)
	s.mux.HandleFunc("POST /api/snapshots/{id}/archive", s.handleArchiveSnapshot)
	s.mux.HandleFunc("GET /api/snapshots/{id}/nodes", s.handleListNodes)
	s.mux.HandleFunc("POST /api/snapshots/{id}/nodes", s.handleAddNode)
	s.mux.HandleFunc("GET /api/snapshots/{id}/links", s.handleListLinks)
	s.mux.HandleFunc("POST /api/snapshots/{id}/links", s.handleAddLink)
	s.mux.HandleFunc("GET /api/snapshots/{id}/inspect", s.handleInspect)
	s.mux.HandleFunc("POST /api/snapshots/{id}/analyze", s.handleAnalyze)

	s.mux.HandleFunc("POST /api/samples", s.handleSubmitSample)
	s.mux.HandleFunc("GET /api/samples", s.handleListSamples)
	s.mux.HandleFunc("GET /api/samples/{id}", s.handleGetSample)

	s.mux.HandleFunc("GET /api/events", s.handleListEvents)
	s.mux.HandleFunc("GET /api/events/{id}", s.handleGetEvent)
	s.mux.HandleFunc("POST /api/events/{id}/localize", s.handleLocalize)
	s.mux.HandleFunc("GET /api/events/{id}/candidates", s.handleListCandidates)
	s.mux.HandleFunc("GET /api/events/{id}/evidence", s.handleListEvidence)
	s.mux.HandleFunc("POST /api/events/{id}/seal", s.handleSeal)

	s.mux.HandleFunc("GET /api/candidates/{id}", s.handleGetCandidate)
	s.mux.HandleFunc("POST /api/candidates/{id}/confirm", s.handleConfirm)
	s.mux.HandleFunc("POST /api/candidates/{id}/reject", s.handleReject)
	s.mux.HandleFunc("POST /api/candidates/{id}/untrusted", s.handleMarkUntrusted)
}

// --- 基础响应辅助 ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrDuplicate):
		status = http.StatusConflict
	case errors.Is(err, model.ErrSealed):
		status = http.StatusConflict
	case errors.Is(err, model.ErrInvalidState):
		status = http.StatusConflict
	case errors.Is(err, model.ErrSnapshotArchived):
		status = http.StatusConflict
	case errors.Is(err, model.ErrUnknownNode),
		errors.Is(err, model.ErrUnitMismatch),
		errors.Is(err, model.ErrTopologyCycle),
		errors.Is(err, model.ErrNegativeDelay),
		errors.Is(err, model.ErrOffsetOverflow):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func queryInt64(r *http.Request, key string, def int64) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// --- 基础端点 ---

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "network time synchronization root-cause localization service")
	fmt.Fprintln(w, "see /api/health or /api/stats")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
