package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	registerRoutes(mux)

	fmt.Println("demo monolith plugin entry")
	fmt.Println("demo routes: /demo-monolith/health, /demo-monolith/widgets, /demo-monolith/manifest")
	fmt.Println("listen on :18083")
	if err := http.ListenAndServe(":18083", mux); err != nil {
		fmt.Println("demo monolith plugin stopped:", err)
	}
}
