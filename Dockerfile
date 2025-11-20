# Билд стадии
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Запускаем тесты
RUN go test ./internal/service/... -v

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Финальная стадия
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из builder стадии
COPY --from=builder /app/server .
# Копируем миграции
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
