package errors

import (
	"encoding/json"
	"net/http"
)

type HTTPErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteHTTP(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	status := http.StatusInternalServerError
	code := "internal_error"
	msg := "internal server error"

	if ce, ok := err.(*CodeError); ok {
		code = ce.Code
		msg = ce.Message
		status = ToHTTPStatus(ce.Code)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(HTTPErrorResponse{Code: code, Message: msg})
}
