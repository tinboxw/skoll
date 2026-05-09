FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/skoll ./cmd/skoll

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
WORKDIR /app

COPY --from=builder /out/skoll /app/skoll

EXPOSE 8080
ENTRYPOINT ["/app/skoll"]

