# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary
# -ldflags="-w -s" reduces binary size by stripping debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/server

# Runtime stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/server /app/server

# Expose port 8080 (Cloud Run will set PORT env var automatically)
EXPOSE 8080

# Use non-root user (distroless provides this)
USER nonroot:nonroot

# Run the server
# Cloud Run will automatically check the /healthz endpoint for health checks
ENTRYPOINT ["/app/server"]

