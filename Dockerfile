# Build stage
FROM golang:alpine AS builder

# Install build dependencies including Node.js for Tailwind
RUN apk add --no-cache git gcc musl-dev sqlite-dev nodejs npm

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (including package.json for tailwind)
COPY . .

# Install dependencies, generate templates, and build
RUN ["/bin/sh", "-c", "\
    npm install && \
    npm install -D @tailwindcss/cli@latest && \
    go install github.com/a-h/templ/cmd/templ@latest && \
    templ generate && \
    npx @tailwindcss/cli -i static/css/input.css -o static/css/tailwind.css --minify && \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server"]
# Runtime stage
FROM alpine:3

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs tzdata

# Set timezone to Europe/Bucharest (EEST)
ENV TZ=Europe/Bucharest

# Create app user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy static files and migrations
COPY --from=builder /app/static ./static
COPY --from=builder /app/migrations ./migrations

# Create data directory for SQLite database
RUN mkdir -p /app/data && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/bin/sh", "-c", "wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1"]

# Run the application
CMD ["./server"]

