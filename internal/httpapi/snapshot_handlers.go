package httpapi

import (
	"net/http"
)

// --- 快照 ---

type createSnapshotReq struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req createSnapshotReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	snap, err := s.app.Topology.CreateSnapshot(req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	snaps, err := s.app.Topology.ListSnapshots()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	snap, err := s.app.Topology.GetSnapshot(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) handleLockSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	snap, err := s.app.Topology.LockSnapshot(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) handleArchiveSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	snap, err := s.app.Topology.ArchiveSnapshot(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// --- 节点 ---

type addNodeReq struct {
	NodeKey    string `json:"node_key"`
	Role       string `json:"role"`
	Hostname   string `json:"hostname"`
	ClockClass int    `json:"clock_class"`
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	nodes, err := s.app.Topology.Nodes(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleAddNode(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req addNodeReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	node, err := s.app.Topology.AddNode(id, req.NodeKey, req.Role, req.Hostname, req.ClockClass)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, node)
}

// --- 边 ---

type addLinkReq struct {
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	Protocol string `json:"protocol"`
	Declared *bool  `json:"declared"`
}

func (s *Server) handleListLinks(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	links, err := s.app.Topology.Links(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func (s *Server) handleAddLink(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req addLinkReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	declared := true
	if req.Declared != nil {
		declared = *req.Declared
	}
	link, err := s.app.Topology.AddLink(id, req.FromNode, req.ToNode, req.Protocol, declared)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

// --- 拓扑诊断与一键分析 ---

func (s *Server) handleInspect(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	report, err := s.app.Topology.Inspect(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

type analyzeReq struct {
	JumpThresholdNs int64 `json:"jump_threshold_ns"`
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req analyzeReq
	_ = decodeJSON(r, &req) // body 可为空
	res, err := s.app.Analyze(id, req.JumpThresholdNs)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
