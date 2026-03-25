# ---- Stage 1: Build frontend ----
FROM oven/bun:1 AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lockb* ./
RUN bun install --frozen-lockfile

COPY frontend/ ./
RUN bun run build

# ---- Stage 2: Build Go binary ----
FROM golang:1.26-alpine AS go-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

COPY --from=frontend-builder /app/frontend/dist ./internal/static/dist

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w -X stalkarr/internal/version.Version=${VERSION}" \
  -o stalkarr ./cmd/server

# ---- Stage 3: Final image ----
FROM alpine:3.19

RUN apk add --no-cache su-exec tzdata

WORKDIR /app
COPY --from=go-builder /app/stalkarr .

COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

VOLUME ["/config"]

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
