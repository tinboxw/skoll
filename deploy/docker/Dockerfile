FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/skoll ./cmd/skoll

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 app \
    && adduser -D -u 10001 -G app app \
    && mkdir -p /app/data/objects /app/log /app/plugins \
    && chown -R app:app /app
USER app
WORKDIR /app

COPY --from=builder --chown=app:app /out/skoll /app/skoll

EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=6 \
    CMD wget -qO- http://127.0.0.1:8080/skoll/health >/dev/null || exit 1
ENTRYPOINT ["/app/skoll"]
