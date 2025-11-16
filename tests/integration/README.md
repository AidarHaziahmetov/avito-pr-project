# Интеграционные E2E тесты

Этот пакет содержит интеграционные E2E (end-to-end) тесты для PR Reviewer Assignment Service, использующие testcontainers-go для создания изолированной тестовой среды.

## Архитектура тестов

### Технологии

- **testcontainers-go** - автоматическое управление Docker контейнерами
- **testcontainers-go/modules/postgres** - специализированный модуль для PostgreSQL
- **stretchr/testify** - assertions и test helpers

### Структура

- `helpers.go` - вспомогательные функции и setup окружения
- `integration_test.go` - набор E2E тестов для различных сценариев

## Запуск тестов

### Предварительные требования

1. Docker daemon должен быть запущен и доступен
2. Go 1.25 установлен
3. Достаточно прав для работы с Docker (без sudo или с sudo)

### Команды запуска

```bash
# Из корня проекта
make test-integration

# Или напрямую
go test -v -race -timeout 5m ./tests/integration/...

# Запустить конкретный тест
go test -v -run TestE2E_CompleteWorkflow ./tests/integration/...

# С подробным выводом
go test -v -race -timeout 5m ./tests/integration/... -test.v
```

## Что тестируется

### TestE2E_CompleteWorkflow

Полный цикл работы с PR:
1. Создание команды с несколькими пользователями
2. Аутентификация (получение JWT токена)
3. Создание Pull Request с автоматическим назначением ревьюверов
4. Получение списка PR'ов для ревьювера
5. Переназначение ревьювера
6. Merge PR
7. Попытка переназначения после merge (должна провалиться)

### TestE2E_UserActivation

Работа с активацией/деактивацией пользователей:
1. Создание команды
2. Деактивация пользователя
3. Создание PR - проверка, что деактивированный пользователь не назначен
4. Реактивация пользователя

### TestE2E_MergeIdempotency

Проверка идемпотентности операции merge:
1. Создание и merge PR
2. Повторный merge того же PR
3. Проверка, что состояние не изменилось и операция успешна

### TestE2E_GetTeam

Получение информации о команде:
1. Создание команды
2. Запрос информации через API
3. Проверка корректности данных

### TestE2E_Stats

Работа со статистикой:
1. Создание команды и нескольких PR
2. Получение общей статистики
3. Получение статистики по конкретному пользователю

### TestE2E_SmallTeam

Работа с малыми командами:
1. Создание команды из 1 пользователя
2. Создание PR - должно быть 0 ревьюверов (автор не может быть ревьювером)

## Как работает TestEnvironment

### SetupTestEnvironment

1. **Запуск PostgreSQL контейнера**
   - Использует `postgres:16-alpine` image
   - Создает тестовую БД `pr_service_test`
   - Настраивает health check для ожидания готовности

2. **Применение миграций**
   - Читает SQL миграции из `migrations/`
   - Применяет к тестовой БД

3. **Запуск приложения**
   - Создает конфигурацию с параметрами тестовой БД
   - Инициализирует и запускает HTTP сервер на порту 18080
   - Ожидает готовности через health check

4. **Возврат TestEnvironment**
   - Содержит ссылки на все ресурсы
   - Предоставляет helper методы для HTTP запросов

### Cleanup

Автоматически очищает все ресурсы:
1. Graceful shutdown приложения
2. Закрытие DB connection pool
3. Остановка и удаление PostgreSQL контейнера

## Вспомогательные функции

### MakeRequest

```go
resp := env.MakeRequest(t, http.MethodPost, "/path", body, token)
```

Создает и выполняет HTTP запрос к тестовому серверу:
- Автоматически добавляет `Content-Type: application/json`
- Поддерживает JWT авторизацию
- Включает таймауты

### WaitForHealthCheck

```go
env.WaitForHealthCheck(t)
```

Ожидает готовности приложения через `/health` endpoint:
- Максимум 30 попыток с интервалом 100ms
- Fail теста при таймауте

## Изоляция тестов

Каждый тест:
- Использует свой собственный `TestEnvironment`
- Получает чистую БД с примененными миграциями
- Не влияет на другие тесты
- Автоматически очищает ресурсы через `defer env.Cleanup(t)`

## Пропуск в CI/CD

Для быстрого запуска unit-тестов без Docker:

```bash
go test -short ./...
```

Интеграционные тесты пропускаются при флаге `-short`:
```go
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}
```

## Отладка

### Просмотр логов контейнеров

Testcontainers автоматически выводит логи в stdout при ошибках.

### Ручная проверка

Тесты можно модифицировать для отладки:

```go
// В конце теста добавить задержку
time.Sleep(5 * time.Minute)
// Теперь можно подключиться к контейнеру вручную
```

### Увеличение таймаутов

Для медленных машин или CI:

```bash
go test -v -timeout 10m ./tests/integration/...
```

## Производительность

Типичное время выполнения:
- Запуск одного теста: ~3-5 секунд
- Все интеграционные тесты: ~15-30 секунд

Время включает:
- Загрузку и запуск PostgreSQL контейнера (первый раз медленнее из-за pull)
- Применение миграций
- Запуск приложения
- Выполнение HTTP запросов
- Cleanup

## Лучшие практики

1. **Всегда используйте `defer env.Cleanup(t)`**
   ```go
   env := SetupTestEnvironment(t)
   defer env.Cleanup(t)
   ```

2. **Ждите готовности приложения**
   ```go
   env.WaitForHealthCheck(t)
   ```

3. **Используйте уникальные ID**
   - Каждый тест должен использовать уникальные team_name, user_id, pull_request_id

4. **Проверяйте статус коды И тело ответа**
   ```go
   assert.Equal(t, http.StatusOK, resp.StatusCode)
   
   var result Response
   err := json.NewDecoder(resp.Body).Decode(&result)
   require.NoError(t, err)
   ```

5. **Закрывайте response body**
   ```go
   resp := env.MakeRequest(...)
   defer resp.Body.Close()
   ```

## Расширение тестов

Для добавления новых тестовых сценариев:

1. Создайте новую функцию `TestE2E_YourScenario` в `integration_test.go`
2. Следуйте структуре существующих тестов
3. Используйте `t.Run()` для подтестов
4. Добавьте описание в этот README

## Troubleshooting

### "Cannot connect to Docker daemon"

Убедитесь, что Docker запущен:
```bash
docker ps
```

### Порт 18080 занят

Измените порт в `helpers.go`:
```go
testPort := "18081" // или другой свободный порт
```

### Тесты падают с timeout

Увеличьте таймауты:
- В команде запуска: `-timeout 10m`
- В коде: `wait.ForLog(...).WithStartupTimeout(120*time.Second)`

### PostgreSQL контейнер не стартует

Проверьте логи Docker и доступную память:
```bash
docker logs <container_id>
docker system df
```

