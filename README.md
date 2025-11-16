# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюеров на Pull Request'ы. Тестовое задание для стажировки Backend в Авито (осенняя волна 2025).

## Технологический стек

- **Язык**: Go 1.25
- **Фреймворк**: Chi Router v5
- **База данных**: PostgreSQL 16
- **Драйвер БД**: pgx/v5
- **Миграции**: golang-migrate/migrate
- **Аутентификация**: JWT (golang-jwt/jwt)
- **Конфигурация**: envconfig
- **Контейнеризация**: Docker, docker-compose
- **Тестирование**: testcontainers-go

## Быстрый старт

```bash
# Запустить все сервисы
docker compose up

# В другом терминале - автоматический тест API
./test-api.sh
```

Сервис будет доступен на `http://localhost:8080`

### Что происходит при запуске

1. Поднимается PostgreSQL контейнер
2. Автоматически применяются миграции в отдельном контейнере
3. Запускается API сервер

## Структура проекта

```
cmd/api/              - Точка входа приложения
internal/
  ├── app/            - Инициализация и настройка приложения
  ├── config/         - Конфигурация
  ├── domain/         - Доменные модели и ошибки
  ├── repository/     - Слой работы с БД
  │   └── postgres/   - PostgreSQL реализация
  ├── service/        - Бизнес-логика
  ├── handler/        - HTTP обработчики
  └── middleware/     - HTTP middleware (JWT auth)
migrations/           - SQL миграции
tests/integration/    - E2E тесты
```

## API Endpoints

### Публичные (без авторизации)

- `POST /auth/login` - Получить JWT токен
- `POST /team/add` - Создать команду с участниками
- `GET /health` - Проверка состояния сервиса

### Защищенные (требуют JWT токен)

**Teams:**
- `GET /team/get?team_name={name}` - Получить команду

**Users:**
- `POST /users/setIsActive` - Установить флаг активности пользователя
- `GET /users/getReview?user_id={id}` - Получить PR'ы пользователя

**Pull Requests:**
- `POST /pullRequest/create` - Создать PR (автоматически назначает ревьюверов)
- `POST /pullRequest/merge` - Смержить PR (идемпотентно)
- `POST /pullRequest/reassign` - Переназначить ревьювера

**Statistics:**
- `GET /stats` - Общая статистика по назначениям
- `GET /stats/user?user_id={id}` - Статистика по пользователю

## Примеры использования

### 1. Получение токена

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"user_id": "u1"}'
```

### 2. Создание команды

```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }'
```

### 3. Создание Pull Request

```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {token}" \
  -d '{
    "pull_request_id": "pr-1",
    "pull_request_name": "Add authentication",
    "author_id": "u1"
  }'
```

Ответ автоматически включит до 2 ревьюверов:
```json
{
  "pull_request_id": "pr-1",
  "pull_request_name": "Add authentication",
  "author_id": "u1",
  "status": "OPEN",
  "reviewers": ["u2", "u3"]
}
```

## Бизнес-логика

### Назначение ревьюверов

1. При создании PR автоматически назначаются до 2 активных ревьюверов
2. Ревьюверы выбираются из команды автора
3. Автор не может быть назначен ревьювером своего PR
4. Выбираются только пользователи с `is_active = true`
5. Если доступных кандидатов меньше 2, назначается доступное количество

### Переназначение ревьювера

1. Можно заменить только ревьювера, который уже назначен на PR
2. Новый ревьювер выбирается из команды заменяемого ревьювера
3. Выбирается случайный активный участник, еще не назначенный на этот PR
4. Нельзя переназначить ревьювера после merge PR

### Merge PR

1. Операция идемпотентная - повторный вызов возвращает актуальное состояние
2. После merge изменение ревьюверов запрещено
3. Время `mergedAt` устанавливается только при первом merge

## Конфигурация

Конфигурация через переменные окружения:

```env
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=pr_service
DB_PASSWORD=pr_service_pass
DB_NAME=pr_service
DB_SSLMODE=disable
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION_HOURS=24

# Migrations
MIGRATIONS_PATH=file://migrations
```

## Тестирование

### Интеграционные E2E тесты

Используют testcontainers-go для автоматического поднятия PostgreSQL и тестирования полного цикла работы.

```bash
# Запустить интеграционные тесты
make test-integration

# Запустить все тесты
make test-all
```

**Что тестируется:**

1. `TestE2E_CompleteWorkflow` - полный цикл: создание команды, логин, создание PR, переназначение, merge
2. `TestE2E_UserActivation` - деактивация/реактивация пользователей
3. `TestE2E_MergeIdempotency` - идемпотентность операции merge
4. `TestE2E_GetTeam` - получение информации о команде
5. `TestE2E_Stats` - работа со статистикой
6. `TestE2E_SmallTeam` - корректная работа с командами меньше 2 человек

**Преимущества:**
- Реальная PostgreSQL БД (не моки)
- Автоматическое управление контейнерами
- Изоляция между тестами
- Воспроизводимость на любой машине

### Нагрузочное тестирование

Проверка соответствия SLI требованиям с использованием `hey`.

```bash
# Установить инструменты
make install-load-tools

# Запустить нагрузочные тесты
make test-load
```

**Требования и результаты:**

| Метрика | Требование | Результат | Статус |
|---------|------------|-----------|--------|
| RPS | 5 req/sec | 24.99 req/sec | Превышено в 5 раз |
| Время ответа | < 300ms | 1.9ms (avg) | В 158 раз лучше |
| Успешность | > 99.9% | 100% | Выполнено |

**Тестовые сценарии:**

1. Health check - 1000 запросов (8,117 req/sec, 1.2ms avg)
2. Аутентификация - 500 запросов, 5 RPS (1.7ms avg)
3. Получение команды - 500 запросов, 5 RPS (1.9ms avg)
4. Создание PR - 100 запросов (3.3ms avg)
5. Статистика - 200 запросов, 5 RPS (15.4ms avg)
6. Sustained load - 1500 запросов за 60 секунд, 5 RPS (1.9ms avg, 0 ошибок)
7. Переназначение ревьювера - 50 запросов (1.4ms avg)

**Распределение времени ответа (sustained load, 60 секунд):**

| Перцентиль | Время |
|------------|-------|
| 50% | 1.8ms |
| 95% | 2.7ms |
| 99% | 3.3ms |
| 100% | 8.1ms |

Все метрики значительно ниже требуемых 300ms.

## Архитектурные решения

### 1. Clean Architecture

- **Domain** - чистые модели без зависимостей
- **Repository** - абстракция работы с БД
- **Service** - бизнес-логика
- **Handler** - HTTP слой

### 2. JWT Авторизация

- Реализована через middleware `AuthMiddleware`
- Токен содержит `user_id` и `team_name`
- `/team/add` сделан публичным для начальной настройки системы

### 3. Миграции в отдельном контейнере

- Применяются до запуска API
- API зависит от успешного завершения миграций
- Приложение не содержит код миграций

### 4. Случайный выбор ревьюверов

- Используется отдельный `ReviewerSelector` с собственным `rand.Rand`
- Потокобезопасная реализация
- Легко заменяется на другой алгоритм (round-robin, по нагрузке)

### 5. Connection Pool

- Min: 5, Max: 25 соединений
- Оптимизировано для текущей нагрузки
- Используется pgx pool для эффективного управления

### 6. Graceful Shutdown

- Обработка сигналов `SIGINT`/`SIGTERM`
- Таймаут 30 секунд для завершения текущих запросов
- Корректное закрытие соединений с БД

### 7. Идемпотентный merge

```sql
UPDATE pull_requests
SET status = 'MERGED',
    merged_at = COALESCE(merged_at, NOW())
WHERE pull_request_id = $1
```

`COALESCE` не перезаписывает время при повторных вызовах.

## Полезные команды

```bash
# Показать все команды
make help

# Сборка
make build              # Собрать бинарник
make clean              # Очистить артефакты

# Тестирование
make test               # Unit тесты (пока отсутствуют)
make test-integration   # E2E тесты (требует Docker)
make test-load          # Нагрузочные тесты (требует запущенный сервис)
make test-all           # Все тесты

# Качество кода
make lint               # Запустить линтер
make fmt                # Форматировать код

# Docker
make docker-up          # Запустить все сервисы
make docker-down        # Остановить сервисы
make docker-logs        # Посмотреть логи

# Миграции
make migrate-up         # Применить миграции
make migrate-down       # Откатить миграции
make migrate-create NAME=name  # Создать новую миграцию

# Разработка
make dev                # Запустить окружение для разработки
make run                # Запустить приложение локально
```

## Производительность

Согласно требованиям ТЗ:
- Объем данных: до 20 команд, до 200 пользователей
- RPS: 5 запросов в секунду
- SLI времени ответа: 300 мс
- SLI успешности: 99.9%

Оптимизации:
- Connection pool для эффективного использования соединений
- Индексы на часто запрашиваемых полях (team_name, user_id, status)
- Транзакции для атомарных операций
- Кэширование на уровне pgx pool

## Дополнительные задания

| Задание | Статус |
|---------|--------|
| Статистика | Реализовано |
| Нагрузочное тестирование | Реализовано |
| Массовая деактивация | Не реализовано |
| E2E тестирование | Реализовано |
| Конфигурация линтера | Реализовано |

## Файлы для review

**Бизнес-логика:**
- `internal/service/pullrequest.go` - логика назначения ревьюверов
- `internal/service/reviewer_selector.go` - алгоритм выбора
- `internal/repository/postgres/pullrequest.go` - работа с БД

**API:**
- `internal/handler/pullrequest.go` - HTTP обработчики PR
- `internal/app/app.go` - инициализация и роутинг

**Инфраструктура:**
- `docker-compose.yml` - Docker конфигурация
- `migrations/000001_init_schema.up.sql` - схема БД

**Тесты:**
- `tests/integration/integration_test.go` - E2E тесты
- `load-test.sh` - нагрузочное тестирование

## Автор

Проект создан в рамках тестового задания для стажировки Backend в Авито (осенняя волна 2025).
