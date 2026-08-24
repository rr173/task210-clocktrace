package httpapi

import (
	"net/http"
)

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.app.ComputeStats()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
