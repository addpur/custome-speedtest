# Build stage
FROM golang:1.27.4-alpine AS builder

WORKDIR /app

# Install git and ca-certificates if needed
RUN apk add --no-cache git ca-certificates tzdata

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/custome-speedtest main.go

# Final runtime image
FROM alpine:3.24

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata curl

# Copy binary from builder
COPY --from=builder /app/custome-speedtest /app/custome-speedtest

# Expose default HTTP port
EXPOSE 3000

# Environment variables
ENV LISTEN_ADDR=":3000" \
    BUFFER_SIZE="4194304" \
    DEFAULT_DURATION="10" \
    DEFAULT_PARALLEL="4" \
    DEFAULT_OVERHEAD="1.0"

# Run speedtest server by default
ENTRYPOINT ["/app/custome-speedtest"]
