package main

import (
	"fmt"
	"net/http"
)

func main() {
	server := NewServer()
	// Start the server on port 8080
	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", server)
}
