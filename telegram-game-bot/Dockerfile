# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bot \
    ./cmd/bot/main.go

# Final stage - minimal image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bot /app/bot

# Copy config directory
COPY --from=builder /app/config /app/config

# Copy migrations (for reference, migrations run in code)
COPY --from=builder /app/migrations /app/migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set timezone (can be overridden by TZ env var)
ENV TZ=Asia/Shanghai

# Expose no ports (bot uses long polling, not webhooks)

# Health check (optional - bot doesn't expose HTTP)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#     CMD pgrep bot || exit 1

# Run the bot
CMD ["/app/bot"]
