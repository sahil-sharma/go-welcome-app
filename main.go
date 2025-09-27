package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"
)

// JSONLog is the structure for log entries
type JSONLog struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	Remote    string `json:"remote,omitempty"`
	Duration  string `json:"duration,omitempty"`
	Error     string `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("/", loggingMiddleware(rootHandler))
	http.HandleFunc("/welcome", loggingMiddleware(welcomeHandler))
	http.HandleFunc("/external", loggingMiddleware(externalHandler))

	port := "8080"
	address := "0.0.0.0:" + port
	fmt.Println("Server started on address:", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}

// --- Middleware to log requests ---
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logJSON("info", "Started request", r.Method, r.URL.Path, r.RemoteAddr, "", "", nil)

		next(w, r)

		duration := time.Since(start)
		logJSON("info", "Completed request", r.Method, r.URL.Path, r.RemoteAddr, duration.String(), "", nil)
	}
}

// --- Handlers ---

func rootHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Hello World!"))
	if err != nil {
		logJSON("error", "Error writing response", r.Method, r.URL.Path, r.RemoteAddr, "", "", err)
	}
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	username := getUsername()
	currentTime := time.Now().Format(time.RFC1123)

	response := fmt.Sprintf(`
Hello, Welcome to the meetup!
Host: %s
Username: %s
Date & Time: %s
`, host, username, currentTime)

	_, err := w.Write([]byte(response))
	if err != nil {
		logJSON("error", "Error writing response", r.Method, r.URL.Path, r.RemoteAddr, "", "", err)
	}
}

func externalHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("https://httpbin.org/get")
	if err != nil {
		http.Error(w, "Failed to reach httpbin.org", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward content type & response
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		logJSON("error", "Error forwarding response", r.Method, r.URL.Path, r.RemoteAddr, "", "", err)
	}
}

func getUsername() string {
	u, err := user.Current()
	if err == nil && u.Username != "" {
		return u.Username
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

// --- JSON logger ---
func logJSON(level, msg, method, path, remote, duration, extra string, err error) {
	entry := JSONLog{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Method:    method,
		Path:      path,
		Remote:    remote,
		Duration:  duration,
	}

	if err != nil {
		entry.Error = err.Error()
	}
	if extra != "" {
		entry.Message = entry.Message + " - " + extra
	}

	data, _ := json.Marshal(entry)
	log.Println(string(data))
}
