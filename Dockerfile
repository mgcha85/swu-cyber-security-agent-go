# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies (CGO required for go-sqlite3)
RUN apk add --no-cache git gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled
RUN CGO_ENABLED=1 go build -o agent-server ./cmd/server

# Runtime Stage
FROM alpine:latest

WORKDIR /app

# Install basic runtime dependencies (ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/agent-server .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/gnn_results ./gnn_results

# Create doc directory for volume mount
RUN mkdir doc

# Set environment variables
ENV QDRANT_ADDR=qdrant:6334
ENV OLLAMA_BASE=http://host.containers.internal:11434

# Expose API port
EXPOSE 8080

# Command to run (default to server mode)
ENTRYPOINT ["./agent-server"]
CMD ["--server"]
