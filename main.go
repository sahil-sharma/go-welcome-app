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

// Global metrics
var (
	requestCount   metric.Int64Counter
	requestLatency metric.Float64Histogram
)

func main() {
	http.HandleFunc("/", loggingMiddleware(rootHandler))
	http.HandleFunc("/welcome", loggingMiddleware(welcomeHandler))
	http.HandleFunc("/external", loggingMiddleware(externalHandler))

	port := "8080"
	address := "0.0.0.0:" + port
	fmt.Println("Server started on", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal("Server error:", err)
	}
}

// --- OTEL setup ---
func setupOTel(ctx context.Context) func(context.Context) error {
	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("otel-go-demo"),
		),
	)

	// Trace exporter (stdout)
	traceExporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Metric exporter (stdout)
	metricExporter, _ := stdoutmetric.New()
	metricProcessor := basic.New(simple.NewWithHistogramDistribution())
	ctrl := basic.New(metricProcessor, metricExporter)
	ctrl.Start(ctx)
	otel.SetMeterProvider(ctrl.MeterProvider())

	return func(ctx context.Context) error {
		_ = ctrl.Stop(ctx)
		return tp.Shutdown(ctx)
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

// --- Middleware with tracing, metrics, and logs ---
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			next(w, r)
			return
		}

		ctx, span := otel.Tracer("otel-go-demo").Start(r.Context(), r.URL.Path)
		start := time.Now()

		logJSON("info", "Started request", r.Method, r.URL.Path, r.RemoteAddr, "", "", nil)

		next(w, r.WithContext(ctx))

		duration := time.Since(start)
		requestCount.Add(ctx, 1, metric.WithAttributes(attribute.String("path", r.URL.Path)))
		requestLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.String("path", r.URL.Path)))

		logJSON("info", "Completed request", r.Method, r.URL.Path, r.RemoteAddr, duration.String(), "", nil)
		span.SetAttributes(attribute.String("method", r.Method), attribute.String("path", r.URL.Path))
		span.End()
	}
}

// --- Handlers ---
func rootHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := traceFromContext(ctx, "rootHandler")
	defer span.End()

	_, err := w.Write([]byte("Hello World!"))
	if err != nil {
		span.RecordError(err)
		logJSON("error", "Error writing response", r.Method, r.URL.Path, r.RemoteAddr, "", "", err)
	}
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := traceFromContext(ctx, "welcomeHandler")
	defer span.End()

	host, _ := os.Hostname()
	username := getUsername()
	currentTime := time.Now().Format(time.RFC1123)
	appUsername := os.Getenv("APP_USERNAME")
	appPassword := os.Getenv("APP_PASSWORD")
	appEnvName := os.Getenv("APP_ENV_NAME")
	webhookTestKey := os.Getenv("WEBHOOK_TEST_KEY")

	response := fmt.Sprintf(`
Hello, Welcome to the meetup!!!
Host: %s
Username: %s
Date & Time: %s
App Username: %s
App Password: %s
App Environment Name: %s
Webhook Test Key: %s
`, host, username, currentTime, appUsername, appPassword, appEnvName, webhookTestKey)

	_, err := w.Write([]byte(response))
	if err != nil {
		span.RecordError(err)
		logJSON("error", "Error writing response", r.Method, r.URL.Path, r.RemoteAddr, "", "", err)
	}
}

func externalHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := traceFromContext(ctx, "externalHandler")
	defer span.End()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	resp, err := client.Get("https://httpbin.org/get")
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Failed to reach httpbin.org", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		span.RecordError(err)
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
		entry.Message += " - " + extra
	}
	data, _ := json.Marshal(entry)
	log.Println(string(data))
}
