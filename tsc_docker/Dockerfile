# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
      -X github.com/tsc-forex/tsc_forex/internal/utils.Version=${VERSION} \
      -X github.com/tsc-forex/tsc_forex/internal/utils.Commit=${COMMIT} \
      -X github.com/tsc-forex/tsc_forex/internal/utils.BuildTime=${BUILD_TIME}" \
    -o /out/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl && \
    adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY configs/config.yaml /app/configs/config.yaml

ENV APP_CONFIG_FILE=/app/configs/config.yaml
EXPOSE 8080

USER appuser

HEALTHCHECK --interval=15s --timeout=3s --retries=5 \
  CMD curl -sf http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/server"]
