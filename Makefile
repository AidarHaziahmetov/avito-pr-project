.PHONY: help build run test test-integration test-all clean docker-build docker-up docker-down docker-logs migrate-up migrate-down lint

# Переменные
APP_NAME=pr-service-api
DOCKER_COMPOSE=docker compose

help: ## Показать список доступных команд
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Собрать Go приложение
	@echo "Сборка приложения..."
	go build -o bin/api cmd/api/main.go
	@echo "Сборка завершена: bin/api"

run: ## Запустить приложение локально
	@echo "Запуск приложения..."
	go run cmd/api/main.go

test: ## Запустить unit-тесты (примечание: unit-тестов в проекте пока нет, используйте test-integration)
	@echo "Запуск unit-тестов..."
	@echo "Примечание: Unit-тесты в проекте отсутствуют, используйте 'make test-integration'"
	go test -v -race -short -coverprofile=coverage.out ./...
	@echo "Покрытие тестами:"
	go tool cover -func=coverage.out

test-integration: ## Запустить интеграционные/E2E тесты (требуется Docker)
	@echo "Запуск интеграционных тестов..."
	@echo "Примечание: Docker-контейнеры будут запущены автоматически"
	go test -v -race -timeout 5m ./tests/integration/...

test-all: ## Запустить все тесты (unit + интеграционные)
	@echo "Запуск всех тестов..."
	go test -v -race -timeout 5m ./...

test-load: ## Запустить нагрузочные тесты (требуется запущенный сервис)
	@echo "Запуск нагрузочных тестов..."
	@if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then \
		echo "Ошибка: Сервис не запущен. Запустите его через 'docker compose up' или 'make run'"; \
		exit 1; \
	fi
	./load-test.sh

install-load-tools: ## Установить инструменты нагрузочного тестирования (hey, vegeta)
	@echo "Установка инструментов нагрузочного тестирования..."
	go install github.com/rakyll/hey@latest
	go install github.com/tsenart/vegeta@latest
	@echo "Инструменты установлены в ~/go/bin/"

clean: ## Очистить артефакты сборки
	@echo "Очистка..."
	rm -rf bin/
	rm -f coverage.out
	@echo "Очистка завершена"

docker-build: ## Собрать Docker-образ
	@echo "Сборка Docker-образа..."
	$(DOCKER_COMPOSE) build

docker-up: ## Запустить все сервисы через docker-compose
	@echo "Запуск сервисов..."
	$(DOCKER_COMPOSE) up -d
	@echo "Сервисы запущены. API доступен по http://localhost:8080"

docker-down: ## Остановить все сервисы
	@echo "Остановка сервисов..."
	$(DOCKER_COMPOSE) down
	@echo "Сервисы остановлены"

docker-logs: ## Показать логи всех сервисов
	$(DOCKER_COMPOSE) logs -f

docker-logs-api: ## Показать логи API-сервиса
	$(DOCKER_COMPOSE) logs -f api

docker-logs-db: ## Показать логи базы данных
	$(DOCKER_COMPOSE) logs -f postgres

docker-restart: docker-down docker-up ## Перезапустить все сервисы

migrate-up: ## Применить миграции базы данных
	@echo "Применение миграций..."
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/pr_service?sslmode=disable" up
	@echo "Миграции применены"

migrate-down: ## Откатить миграции базы данных
	@echo "Откат миграций..."
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/pr_service?sslmode=disable" down
	@echo "Миграции откачены"

migrate-create: ## Создать новый файл миграции (использование: make migrate-create NAME=имя_миграции)
	@if [ -z "$(NAME)" ]; then echo "Ошибка: требуется NAME. Использование: make migrate-create NAME=имя_миграции"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(NAME)

lint: ## Запустить линтер (требуется golangci-lint)
	@echo "Запуск линтера..."
	golangci-lint run ./...

fmt: ## Форматировать код
	@echo "Форматирование кода..."
	go fmt ./...
	gofmt -s -w .
	goimports -w -local github.com/aidar/avito-pr-project .

tidy: ## Упорядочить go-модули
	@echo "Упорядочивание модулей..."
	go mod tidy

deps: ## Загрузить зависимости
	@echo "Загрузка зависимостей..."
	go mod download

install-tools: ## Установить инструменты разработки
	@echo "Установка инструментов..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Рабочий процесс разработки
dev: docker-up ## Запустить окружение для разработки
	@echo "Окружение для разработки готово!"
	@echo "API: http://localhost:8080"
	@echo "База данных: localhost:5432"

all: clean deps build test ## Выполнить все шаги: очистка, загрузка зависимостей, сборка, тесты

.DEFAULT_GOAL := help