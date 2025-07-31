FROM golang:1.23-alpine AS builder 

ARG TARGET=delayed-notifier
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/${TARGET} ./cmd/${TARGET}

FROM alpine:latest

ARG TARGET
ENV TARGET=${TARGET}
COPY --from=builder /app/${TARGET} /usr/local/bin/${TARGET}
COPY .env /.env

EXPOSE 8080

ENTRYPOINT ["sh", "-c", "exec /usr/local/bin/${TARGET}"]