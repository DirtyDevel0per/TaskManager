# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies and ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

# Build optimized binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/main .

# Create non-root user for security
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["./main"]