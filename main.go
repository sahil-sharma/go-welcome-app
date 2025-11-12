package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	// --- Initialize OpenTelemetry (tracing + metrics) ---
	ctx := context.Background()
	tp := initTracer()
	defer func() { _ = tp.Shutdown(ctx) }()
	mp := initMetrics()

	http.Handle("/metrics", promHandler(mp))

	http.Handle("/", loggingMiddleware(otelhttp.NewHandler(http.HandlerFunc(rootHandler), "root")))
	http.Handle("/welcome", loggingMiddleware(otelhttp.NewHandler(http.HandlerFunc(welcomeHandler), "welcome")))
	http.Handle("/external", loggingMiddleware(otelhttp.NewHandler(http.HandlerFunc(externalHandler), "external")))

	port := "8080"
	address := "0.0.0.0:" + port
	fmt.Println("Server started on address:", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}

// --- Setup Tracing ---
func initTracer() *sdktrace.TracerProvider {
	exp, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	)
	otel.SetTracerProvider(tp)
	return tp
}

// --- Setup Metrics ---
func initMetrics() *metric.MeterProvider {
	exp, _ := prometheus.New()
	mp := metric.NewMeterProvider(metric.WithReader(exp))
	otel.SetMeterProvider(mp)
	return mp
}

func promHandler(mp *metric.MeterProvider) http.Handler {
	exp, _ := prometheus.New()
	return exp
}

// --- Middleware to log requests ---
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			next(w, r)
			return
		}
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
	appUsername := os.Getenv("APP_USERNAME")
	appPassword := os.Getenv("APP_PASSWORD")

	response := fmt.Sprintf(`
Hello, Welcome to the meetup!!!
Host: %s
Username: %s
Date & Time: %s
App Username: %s
App Password: %s
`, host, username, currentTime, appUsername, appPassword)

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

// --- Utilities ---
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
