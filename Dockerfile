# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# Templates and static files are embedded into the binary (internal/web), so only the binaries are copied
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o seed ./cmd/seed

# Final stage
FROM alpine:3.18

# tzdata so the daily reminder job runs at local school time (set TZ)
RUN apk add --no-cache tzdata && adduser -D -H app
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/api /app/seed ./

# Do not run as root
USER app

EXPOSE 8080

CMD ["./api"]
