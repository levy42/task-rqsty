# Stage 1: Build the application
FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o ai-gateway ./cmd/ai-gateway

# Stage 2: Create the final image
FROM debian:bookworm-slim

# Install SQLite dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/ai-gateway .
COPY --from=builder /app/templates ./templates

RUN mkdir -p /app/data

ENV PORT=8080 \
    TARGET_URL="https://router.requesty.ai/v1" \
    DB_PATH="/app/data/ai-gateway.db"

VOLUME /app/data

EXPOSE 8080

CMD ["sh", "-c", "./ai-gateway -port=${PORT} -url=${TARGET_URL} -db=${DB_PATH}"]
