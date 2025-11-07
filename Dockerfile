# ---- Builder Stage ----
FROM golang:1.25 AS builder

WORKDIR /app

COPY . .

# Ensure a statically linked binary (no glibc dependencies)
ENV CGO_ENABLED=0
RUN go build -o welcome-app

# ---- Runtime Stage ----
FROM ubuntu:24.04

# Install CA certificates (required for HTTPS requests)
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \ 
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/welcome-app /usr/local/bin/welcome-app

EXPOSE 8080

ENTRYPOINT ["welcome-app"]