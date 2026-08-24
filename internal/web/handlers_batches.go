package web

import (
	"net/http"

	"citytree/internal/application"
)

func (s *Server) HandleBatches(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		view, err := s.service.Dashboard(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	key, err := idempotencyKey(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd application.CreateBatchCommand
	if err = decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.service.CreateBatch(r.Context(), cmd, key)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) HandleBatchDetail(w http.ResponseWriter, r *http.Request) {
	view, err := s.service.Batch(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleAddTree(w http.ResponseWriter, r *http.Request) {
	key, err := idempotencyKey(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd application.AddTreeCommand
	if err = decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	cmd.BatchID = r.PathValue("id")
	result, err := s.service.AddTree(r.Context(), cmd, key)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
