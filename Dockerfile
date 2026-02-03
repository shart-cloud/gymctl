# Multi-stage build for gymctl
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gymctl ./cmd/gymctl

# Final stage - minimal image with just the binary and exercises
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    docker-cli \
    kubectl \
    curl \
    bash \
    git

# Create non-root user
RUN addgroup -g 1000 -S gymuser && \
    adduser -u 1000 -S gymuser -G gymuser

# Copy binary from builder
COPY --from=builder /build/gymctl /usr/local/bin/gymctl

# Copy exercise files
COPY --from=builder /build/tasks /usr/share/gymctl/tasks

# Copy entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Create working directories
RUN mkdir -p /home/gymuser/.gym/workdir && \
    chown -R gymuser:gymuser /home/gymuser/.gym

# Switch to non-root user
USER gymuser
WORKDIR /home/gymuser

# Set environment variables
ENV GYMCTL_TASKS_DIR=/usr/share/gymctl/tasks

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD gymctl list > /dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD []