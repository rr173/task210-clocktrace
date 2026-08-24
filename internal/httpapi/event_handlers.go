package httpapi

import (
	"net/http"
)

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	snapshotID := queryInt64(r, "snapshot", 0)
	var events any
	var err error
	if snapshotID > 0 {
		events, err = s.app.DB.ListEventsBySnapshot(snapshotID)
	} else {
		events, err = s.app.DB.ListAllEvents()
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ev, err := s.app.DB.GetEvent(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

type localizeReq struct {
	JumpThresholdNs int64 `json:"jump_threshold_ns"`
}

func (s *Server) handleLocalize(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req localizeReq
	_ = decodeJSON(r, &req)
	cands, err := s.app.Diagnosis.Diagnose(id, req.JumpThresholdNs)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event_id": id, "candidates": cands})
}

func (s *Server) handleListCandidates(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	cands, err := s.app.ListCandidates(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cands)
}

func (s *Server) handleListEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	cands, err := s.app.ListCandidates(id)
	if err != nil {
		writeError(w, err)
		return
	}
	type evidenceBundle struct {
		CandidateID int64 `json:"candidate_id"`
		Paths       any   `json:"paths"`
	}
	var out []evidenceBundle
	for _, c := range cands {
		paths, err := s.app.EvidencePaths(c.ID)
		if err != nil {
			writeError(w, err)
			return
		}
		out = append(out, evidenceBundle{CandidateID: c.ID, Paths: paths})
	}
	writeJSON(w, http.StatusOK, out)
}

type sealReq struct {
	Note string `json:"note"`
}

func (s *Server) handleSeal(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req sealReq
	_ = decodeJSON(r, &req)
	if err := s.app.Verdict.Seal(id, req.Note); err != nil {
		writeError(w, err)
		return
	}
	ev, err := s.app.DB.GetEvent(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}
