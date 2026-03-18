# Stage 1: Build the custom k6 binary
FROM golang:1.26-alpine AS builder

# Install git (required by xk6 to fetch modules)
RUN apk add --no-cache git

# Install xk6
RUN go install go.k6.io/xk6/cmd/xk6@latest

# Set working directory
WORKDIR /app

# Copy the extension source code (the whole project)
COPY . .

# Build k6 with the local extension
# Use the exact module path from your go.mod
RUN xk6 build --with github.com/NoOneBoss/xk6-prometheus-query=. --output /k6

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Copy the custom k6 binary from the builder stage
COPY --from=builder /k6 /usr/bin/k6

# (Optional) Install ca-certificates if you need HTTPS to Prometheus
RUN apk add --no-cache ca-certificates

# Set the entrypoint
ENTRYPOINT ["k6"]