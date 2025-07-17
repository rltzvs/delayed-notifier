FROM golang:1.23-alpine AS builder 
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api ./cmd/delayed-notifier
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/worker ./cmd/worker

FROM alpine:latest

COPY --from=builder /app/api /usr/local/bin/api
COPY --from=builder /app/worker /usr/local/bin/worker
COPY .env /.env

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]