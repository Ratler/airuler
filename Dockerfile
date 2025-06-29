# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git (required for go modules)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o airuler .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and git for vendor operations
RUN apk --no-cache add ca-certificates git

WORKDIR /workspace

# Copy the binary from builder stage
COPY --from=builder /app/airuler /usr/local/bin/airuler

# Create directory for airuler files
RUN mkdir -p /workspace

# Set the binary as entrypoint
ENTRYPOINT ["airuler"]

# Default command
CMD ["--help"]