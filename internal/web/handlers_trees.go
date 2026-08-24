package web

import (
	"net/http"

	"citytree/internal/application"
	"citytree/internal/domain"
)

func (s *Server) HandleTreeDetail(w http.ResponseWriter, r *http.Request) {
	view, err := s.service.Tree(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleCertificate(w http.ResponseWriter, r *http.Request) {
	view, err := s.service.Tree(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if view.Certificate == nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, view.Certificate)
}

func (s *Server) HandleTreeAction(w http.ResponseWriter, r *http.Request) {
	id, action := r.PathValue("id"), r.PathValue("action")
	key, err := idempotencyKey(r)
	if err != nil {
		writeError(w, err)
		return
	}
	switch action {
	case "evidence":
		var cmd application.SubmitEvidenceCommand
		if err = decodeJSON(w, r, &cmd); err == nil {
			cmd.TreeID = id
			var result application.SubmitEvidenceResult
			result, err = s.service.SubmitEvidence(r.Context(), cmd, key)
			if err == nil {
				writeJSON(w, http.StatusCreated, result)
				return
			}
		}
	case "assess":
		var cmd application.AssessCommand
		if err = decodeJSON(w, r, &cmd); err == nil {
			cmd.TreeID = id
			var result application.AssessResult
			result, err = s.service.Assess(r.Context(), cmd, key)
			if err == nil {
				writeJSON(w, http.StatusOK, result)
				return
			}
		}
	case "remediation":
		var cmd application.CompleteRemediationCommand
		if err = decodeJSON(w, r, &cmd); err == nil {
			cmd.TreeID = id
			var result application.CompleteRemediationResult
			result, err = s.service.CompleteRemediation(r.Context(), cmd, key)
			if err == nil {
				writeJSON(w, http.StatusOK, result)
				return
			}
		}
	case "recheck":
		var cmd application.RecheckCommand
		if err = decodeJSON(w, r, &cmd); err == nil {
			cmd.TreeID = id
			var result application.RecheckResult
			result, err = s.service.Recheck(r.Context(), cmd, key)
			if err == nil {
				writeJSON(w, http.StatusOK, result)
				return
			}
		}
	default:
		writeError(w, domain.ErrNotFound)
		return
	}
	writeError(w, err)
}
