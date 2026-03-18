FROM golang:1.26 AS builder

RUN go install go.k6.io/xk6/cmd/xk6@latest

WORKDIR /app

# копируем твой extension
COPY . .

# билдим k6 с расширением
RUN xk6 build \
    --with github.com/you/xk6-prometheus=.

# финальный образ
FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/k6 /usr/bin/k6

ENTRYPOINT ["k6"]