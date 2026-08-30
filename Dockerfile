# Build stage
FROM golang:1.26.7-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/summario ./cmd/main.go

# Production stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy built binary from build stage
COPY --from=builder /app/summario /app/summario

# Expose port (default in Go code is 8080)
EXPOSE 8080

# Run the app
CMD ["/app/summario"]
