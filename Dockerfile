FROM golang:1.25-alpine AS build

# Устанавливаем переменные окружения для сборки
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GO111MODULE=on

# Создаем непривилегированного пользователя для сборки
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

WORKDIR /app

# Копируем только файлы зависимостей для лучшего кэширования
COPY go.mod go.sum ./

# Проверяем целостность зависимостей и скачиваем их
RUN go mod verify && go mod download

# Копируем исходники
COPY . .

# Сборка приложения с оптимизацией
RUN go build -ldflags="-s -w" -o /app/subscription-service ./cmd/app


# Финальный образ
FROM alpine:3.21

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata

# Создаем непривилегированного пользователя для запуска
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Создаем рабочую директорию
WORKDIR /app

# Копируем бинарник из стадии сборки
COPY --from=build --chown=appuser:appgroup /app/subscription-service /usr/local/bin/subscription-service

# Копируем конфигурационный файл
COPY --chown=appuser:appgroup config.yaml /app/config.yaml

# Открываем порт
EXPOSE 8081

# Меняем на непривилегированного пользователя
USER appuser

# Запуск приложения
ENTRYPOINT ["/usr/local/bin/subscription-service"]
