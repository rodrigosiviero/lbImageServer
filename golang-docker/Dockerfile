# Use a minimal Go image
FROM golang:1.21-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o image_server main.go

# Use a minimal runtime image
FROM alpine:latest

# Set up working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/image_server .

# Expose the server port
EXPOSE 8089

# Start the server
CMD ["/app/image_server"]