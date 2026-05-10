package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	registerRoutes(mux)

	fmt.Println("demo backend-only plugin entry")
	fmt.Println("demo routes: /demo-backend/metrics, /demo-backend/audit/report")
	fmt.Println("listen on :18082")
	if err := http.ListenAndServe(":18082", mux); err != nil {
		fmt.Println("demo backend-only plugin stopped:", err)
	}
}
