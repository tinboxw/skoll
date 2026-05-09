package main

import "net/http"

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/demo-backend/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	})
}
