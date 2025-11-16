#!/bin/bash

set -e

API_URL="http://localhost:8080"

echo "PR Service Load Testing - $(date)"
echo ""

# Проверка API
if ! curl -s "$API_URL/health" > /dev/null; then
    echo "API не доступен на $API_URL"
    exit 1
fi
echo "API работает"

# Подготовка тестовых данных
echo "Настройка тестовых данных..."
curl -s -X POST "$API_URL/team/add" \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "loadtest",
    "members": [
      {"user_id": "lt1", "username": "User1", "is_active": true},
      {"user_id": "lt2", "username": "User2", "is_active": true},
      {"user_id": "lt3", "username": "User3", "is_active": true},
      {"user_id": "lt4", "username": "User4", "is_active": true},
      {"user_id": "lt5", "username": "User5", "is_active": true}
    ]
  }' > /dev/null 2>&1 || true

TOKEN=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"lt1"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

echo ""
echo "Запуск нагрузочных тестов (требования: 5 RPS, <300ms, >99.9% успеха)..."
echo ""

# Тест 1: Health Check
echo "[1/7] Health Check (1000 запросов, 10 параллельных)"
~/go/bin/hey -n 1000 -c 10 -m GET "$API_URL/health"
echo ""

# Тест 2: Аутентификация
echo "[2/7] Аутентификация (500 запросов, 5 RPS)"
~/go/bin/hey -n 500 -c 5 -q 5 -m POST \
    -H "Content-Type: application/json" \
    -d '{"user_id":"lt1"}' \
    "$API_URL/auth/login"
echo ""

# Тест 3: Получение команды
echo "[3/7] Получение команды (500 запросов, 5 RPS)"
~/go/bin/hey -n 500 -c 5 -q 5 -m GET \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/team/get?team_name=loadtest"
echo ""

# Тест 4: Создание PR
echo "[4/7] Создание PR (100 запросов)"
for i in {1..100}; do
    curl -s -X POST "$API_URL/pullRequest/create" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"pull_request_id\":\"pr-lt-${i}\",\"pull_request_name\":\"Load Test PR ${i}\",\"author_id\":\"lt1\"}" \
        > /dev/null 2>&1 &
    if (( i % 5 == 0 )); then wait; fi
done
wait

echo "Тестирование получения PR..."
~/go/bin/hey -n 300 -c 5 -q 5 -m GET \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/users/getReview?user_id=lt2"
echo ""

# Тест 5: Статистика
echo "[5/7] Статистика (200 запросов, 5 RPS)"
~/go/bin/hey -n 200 -c 5 -q 5 -m GET \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/stats"
echo ""

# Тест 6: Длительная нагрузка
echo "[6/7] Длительная нагрузка (5 RPS, 60 секунд)"
~/go/bin/hey -z 60s -c 5 -q 5 -m GET \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/team/get?team_name=loadtest"
echo ""

# Тест 7: Переназначение ревьюера
echo "[7/7] Переназначение ревьюера (50 запросов)"
for i in {201..250}; do
    curl -s -X POST "$API_URL/pullRequest/create" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"pull_request_id\":\"pr-reassign-${i}\",\"pull_request_name\":\"PR ${i}\",\"author_id\":\"lt1\"}" \
        > /dev/null 2>&1
done

~/go/bin/hey -n 50 -c 5 -m POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"pull_request_id":"pr-lt-5","old_user_id":"lt2"}' \
    "$API_URL/pullRequest/reassign"

echo ""
echo "Все тесты завершены!"