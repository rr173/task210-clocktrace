package httpapi

import (
	"net/http"
)

func (s *Server) handleGetCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := s.app.DB.GetCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	paths, err := s.app.EvidencePaths(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidate": c, "evidence_paths": paths})
}

type noteReq struct {
	Note string `json:"note"`
}

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := s.app.DB.GetCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	var req noteReq
	_ = decodeJSON(r, &req)
	if err := s.app.Verdict.Confirm(c.EventID, id, req.Note); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidate_id": id, "status": "confirmed"})
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := s.app.DB.GetCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	var req noteReq
	_ = decodeJSON(r, &req)
	if err := s.app.Verdict.Reject(c.EventID, id, req.Note); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidate_id": id, "status": "rejected"})
}

type untrustedReq struct {
	SampleID int64  `json:"sample_id"`
	Note     string `json:"note"`
}

func (s *Server) handleMarkUntrusted(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := s.app.DB.GetCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	var req untrustedReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.app.Verdict.MarkUntrusted(c.EventID, req.SampleID, req.Note); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sample_id": req.SampleID, "status": "untrusted"})
}
