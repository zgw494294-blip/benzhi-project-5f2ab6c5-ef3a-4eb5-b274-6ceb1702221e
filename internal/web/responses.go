package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"citytree/internal/domain"
)

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}
type errorBody struct {
	Code    string                  `json:"code"`
	Message string                  `json:"message"`
	Fields  domain.ValidationErrors `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: value})
}

func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, domain.ErrValidation):
		status, code = http.StatusUnprocessableEntity, "validation_error"
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusConflict, "version_conflict"
	case errors.Is(err, domain.ErrIdempotency):
		status, code = http.StatusConflict, "idempotency_conflict"
	case errors.Is(err, domain.ErrTransition):
		status, code = http.StatusConflict, "invalid_transition"
	}
	body := &errorBody{Code: code, Message: err.Error()}
	var fields domain.ValidationErrors
	if errors.As(err, &fields) {
		body.Fields = fields
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: body})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.ValidationErrors{{Field: "body", Message: "JSON 请求无效: " + err.Error()}}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.ValidationErrors{{Field: "body", Message: "只能包含一个 JSON 对象"}}
	}
	return nil
}

func idempotencyKey(r *http.Request) (string, error) {
	key := r.Header.Get("Idempotency-Key")
	if !domain.ValidIdempotencyKey(key) {
		return "", domain.ValidationErrors{{Field: "Idempotency-Key", Message: "请求头必填，长度须为 8 到 128"}}
	}
	return key, nil
}
