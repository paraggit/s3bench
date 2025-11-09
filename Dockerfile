# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.GitCommit=${GIT_COMMIT:-unknown} -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /build/s3-workload \
    ./cmd/s3-workload

# Runtime stage - distroless
FROM gcr.io/distroless/static:nonroot

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/s3-workload /usr/local/bin/s3-workload

# Use non-root user
USER 10001:10001

# Expose metrics port
EXPOSE 9090

ENTRYPOINT ["/usr/local/bin/s3-workload"]
CMD ["--help"]

