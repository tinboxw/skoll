package main

import (
	"fmt"
	"net/http"
)

func main() {
	// The separated plugin backend can run independently for demo and contract tests.
	mux := http.NewServeMux()
	registerRoutes(mux)

	fmt.Println("demo separated backend entry")
	fmt.Println("demo routes: /demo-separated/health, /demo-separated/overview, /demo-separated/recommendations")
	fmt.Println("listen on :18081")
	if err := http.ListenAndServe(":18081", mux); err != nil {
		fmt.Println("demo separated backend stopped:", err)
	}
}
