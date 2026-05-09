package v1

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	write(w, status, response{Code: "ok", Message: "ok", Data: data})
}

func writeMessage(w http.ResponseWriter, status int, code, message string) {
	write(w, status, response{Code: code, Message: message})
}

func writeError(w http.ResponseWriter, status int, err error) {
	msg := "internal server error"
	if err != nil {
		msg = err.Error()
	}
	write(w, status, response{Code: "error", Message: msg})
}

func write(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
