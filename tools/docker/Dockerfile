# Stage 1: Build the Go player binary
FROM golang:alpine AS builder

# Install build dependencies (git, etc.)
RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o imc-audio-local .

# Stage 2: Playout image
FROM savonet/liquidsoap:v2.2.4

# Switch to root to install dependencies and configure permissions
USER root

# Install standard utilities (curl for healthchecks, procps for pkill, sqlite3 for DB diagnostics)
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    procps \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Set up working directory
WORKDIR /app

# Copy Go player binary from builder
COPY --from=builder /src/imc-audio-local /app/imc-audio-local

# Make sure folders exist and set owner to liquidsoap user
RUN mkdir -p /app/cache /app/midias && \
    chown -R liquidsoap:liquidsoap /app

# Switch back to the liquidsoap user for safety
USER liquidsoap

# Set default environment variables
ENV SERVER_MODE=true
ENV SERVER_URL=http://192.168.15.250:8080
ENV STORE_TOKEN=""
ENV LOCAL_PORT=5005
ENV HARBOR_PORT=7000
ENV ICECAST_HOST=127.0.0.1
ENV ICECAST_PORT=8000
ENV ICECAST_PASSWORD=hackme
ENV ICECAST_MOUNT=/radio
ENV LIQUIDSOAP_PATH=liquidsoap
ENV SCRIPT_PATH=/app/player.liq
ENV LOG_PATH=/app/imcAudio.log

# Command to run the watchdog and player
ENTRYPOINT []
CMD ["/app/imc-audio-local"]
