package v1

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	write(w, status, Response{Code: "ok", Message: "ok", Data: data})
}

func WriteMessage(w http.ResponseWriter, status int, code, message string) {
	write(w, status, Response{Code: code, Message: message})
}

func WriteError(w http.ResponseWriter, status int, err error) {
	msg := "internal server error"
	if err != nil {
		msg = err.Error()
	}
	write(w, status, Response{Code: "error", Message: msg})
}

func write(w http.ResponseWriter, status int, body Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
