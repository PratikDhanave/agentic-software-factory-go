# Multi-stage build: Go backend
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy source
COPY . .

# Build server
RUN CGO_ENABLED=0 GOOS=linux go build -o asf-web ./cmd/asf-web/

# Build CLI
RUN CGO_ENABLED=0 GOOS=linux go build -o asf-cli ./cmd/asf-cli/

# Final stage: Runtime
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binaries from builder
COPY --from=builder /app/asf-web .
COPY --from=builder /app/asf-cli .

# Copy static assets
COPY web-ui/public /app/web
COPY mcp.json .

# Create data directory
RUN mkdir -p /app/data

# Expose ports
EXPOSE 8080 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV GITHUB_TOKEN="" \
    ANTHROPIC_API_KEY="" \
    PORT=8080

# Run server
CMD ["./asf-web", "0.0.0.0:8080"]
