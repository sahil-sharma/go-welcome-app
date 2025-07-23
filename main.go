package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"time"
)

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/welcome", welcomeHandler)
	http.HandleFunc("/external", externalHandler)

	port := "8080"
	address := "0.0.0.0:" + port
	fmt.Println("Server started on address:", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	username := getUsername()
	currentTime := time.Now().Format(time.RFC1123)

	response := fmt.Sprintf(`
Hello, Welcome to the Golang App: V1
Host: %s
Username: %s
Date & Time: %s
`, host, username, currentTime)

	w.Write([]byte(response))
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
	io.Copy(w, resp.Body)
}

func getUsername() string {
	u, err := user.Current()
	if err == nil && u.Username != "" {
		return u.Username
	}
	// Fallbacks
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}
