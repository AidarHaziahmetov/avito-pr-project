# Стадия сборки - используем полный образ Go для компиляции
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей и загружаем их (кэшируется Docker)
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем приложение (статическая сборка без CGO для Alpine)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/api ./cmd/api/main.go

# Стадия runtime - минимальный образ для запуска
FROM alpine:latest


WORKDIR /root/

# Копируем только скомпилированный бинарник из стадии builder
COPY --from=builder /app/api .

# Открываем порт 8080 для HTTP сервера
EXPOSE 8080

# Запускаем приложение
CMD ["./api"]

