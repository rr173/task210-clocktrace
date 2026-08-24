package httpapi

import (
	"net/http"
	"time"
)

type submitSampleReq struct {
	SnapshotID  int64  `json:"snapshot_id"`
	NodeKey     string `json:"node_key"`
	Sequence    int64  `json:"sequence"`
	Offset      int64  `json:"offset"`
	Roundtrip   int64  `json:"roundtrip"`
	Unit        string `json:"unit"`
	SourceID    string `json:"source_id"`
	CollectedAt string `json:"collected_at"`
}

func (s *Server) handleSubmitSample(w http.ResponseWriter, r *http.Request) {
	var req submitSampleReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var collectedAt time.Time
	if req.CollectedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, req.CollectedAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid collected_at"})
			return
		}
		collectedAt = t
	}
	smp, err := s.app.Samples.Submit(req.SnapshotID, req.NodeKey, req.Sequence, req.Offset, req.Roundtrip, req.Unit, req.SourceID, collectedAt)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, smp)
}

func (s *Server) handleListSamples(w http.ResponseWriter, r *http.Request) {
	snapshotID := queryInt64(r, "snapshot", 0)
	nodeKey := r.URL.Query().Get("node")

	var samples any
	var err error
	if nodeKey != "" {
		samples, err = s.app.Samples.ListByNode(snapshotID, nodeKey)
	} else {
		samples, err = s.app.Samples.ListBySnapshot(snapshotID)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, samples)
}

func (s *Server) handleGetSample(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	smp, err := s.app.Samples.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, smp)
}
