package main

import (
	"encoding/json"
	"fmt"
	"ledis/storage"
	"net/http"
	"strings"
)

type Server struct {
	redis *storage.Ledis
}

// NewServer creates a new instance of Server
func NewServer() *Server {
	return &Server{
		redis: storage.NewLedis(),
	}
}

// ServeHTTP is the method on the Server struct that will handle incoming requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check the URL path and handle accordingly
	fmt.Println("Received request:", r.URL.Path)
	switch {
	case r.URL.Path == "/":
		// Serve the main HTML page
		fmt.Println("Serving main page")
		http.ServeFile(w, r, "frontend/views/index.html")
	case r.URL.Path == "/execute":
		// Handle execute command logic
		s.handleExecute(w, r)
	case strings.HasPrefix(r.URL.Path, "/static/"):
		// Serve static files from the frontend directory
		staticPath := "frontend" + r.URL.Path
		fmt.Println("Serving static file:", staticPath)
		http.ServeFile(w, r, staticPath)
	default:
		// Return 404 Not Found for unsupported routes
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// executeCommand simulates executing a Redis-like command
func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	// Check if the HTTP method is POST
	if r.Method != http.MethodPost {
		// If the method is not POST, return a 405 Method Not Allowed
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.executeCommand(w, r)
}

func (s *Server) executeCommand(w http.ResponseWriter, r *http.Request) {
	// Get the command from the request body
	decoder := json.NewDecoder(r.Body)
	var decoded map[string]string
	err := decoder.Decode(&decoded)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	command, ok := decoded["command"]
	if !ok {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	responseValues, err := s.redis.HandleCommand(command)
	if err != nil {
		fmt.Println("Error executing command:", err)

		// Send the error message as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // Set HTTP status code
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	if responseValues == nil {
		responseValues = []string{"OK"}
	}
	response := map[string]interface{}{"response": responseValues}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK

	_ = json.NewEncoder(w).Encode(response)
}
