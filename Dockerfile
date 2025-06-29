# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

# GoReleaser builds the binary, so we only need the runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and git for vendor operations
RUN apk --no-cache add ca-certificates git

WORKDIR /workspace

# Copy the pre-built binary from GoReleaser context
COPY airuler /usr/local/bin/airuler

# Create directory for airuler files
RUN mkdir -p /workspace

# Set the binary as entrypoint
ENTRYPOINT ["airuler"]

# Default command
CMD ["--help"]