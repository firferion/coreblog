FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o devlog_app cmd/devlog/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/devlog_app .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/.env .env
# Папка для сокета
RUN mkdir -p /var/run/devlog
CMD ["./devlog_app"]
