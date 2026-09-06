# ---- Builder Stage ----
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# Ensure a statically linked binary (no glibc dependencies)
=======
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o welcome-app .

# ---- Runtime Stage ----
FROM ubuntu:24.04

# Install CA certificates (required for HTTPS requests)
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    # Create non-root user
    useradd -m -u 1001 appuser

WORKDIR /app

COPY --from=builder /app/welcome-app /usr/local/bin/welcome-app

# Environment variables for OTEL
ENV OTEL_SERVICE_NAME="welcome-app" \
    OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4317" \
    OTEL_METRIC_EXPORT_INTERVAL=60000 \
    OTEL_TRACES_SAMPLER="always_on" \
    OTEL_RESOURCE_ATTRIBUTES="service.version=1.0.0,service.namespace=demo" \
    OTEL_EXPORTER_OTLP_INSECURE=true \
    APP_USERNAME=demo-user \
    APP_PASSWORD=some-secret

EXPOSE 8080

USER appuser

ENTRYPOINT ["welcome-app"]