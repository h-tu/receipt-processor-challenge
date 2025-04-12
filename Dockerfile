# Use the official Go image as the build environment
FROM golang:1.20-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy Go source files into the container
COPY . .

# Build the Go application
RUN go build -o receipt-processor main.go

# Start a new minimal image for running the binary
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/receipt-processor .

# Expose port 8080
EXPOSE 8080

# Run the service
CMD ["./receipt-processor"]