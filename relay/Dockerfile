FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY relay.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o relay-server .

# Create a minimal image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/relay-server .

# Expose ports
EXPOSE 42421/tcp
EXPOSE 80/tcp
EXPOSE 443/tcp

# Run the application
ENTRYPOINT ["/app/relay-server"] 