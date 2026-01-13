# Build stage
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev curl

# Set working directory
WORKDIR /app

# Download Tailwind CSS standalone CLI
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 && \
    chmod +x tailwindcss-linux-x64 && \
    mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

# Generate templ templates
RUN templ generate

# Build Tailwind CSS with DaisyUI
RUN tailwindcss -i static/css/input.css -o static/css/tailwind.css --minify

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:latest

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
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
CMD ["./server"]

