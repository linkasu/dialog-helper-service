package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var (
	ErrEmptyBody    = errors.New("empty body")
	ErrInvalidJSON  = errors.New("invalid json")
	ErrBodyTooLarge = errors.New("body too large")
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{
		Error:   code,
		Message: message,
	})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}, maxBytes int64) error {
	if r.Body == nil {
		return ErrEmptyBody
	}
	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrEmptyBody
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}
		return ErrInvalidJSON
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}

func JSONError(err error) (status int, code, message string) {
	switch {
	case errors.Is(err, ErrEmptyBody):
		return http.StatusBadRequest, "empty_body", "request body is required"
	case errors.Is(err, ErrBodyTooLarge):
		return http.StatusRequestEntityTooLarge, "body_too_large", "request body is too large"
	default:
		return http.StatusBadRequest, "invalid_json", "invalid json"
	}
}
