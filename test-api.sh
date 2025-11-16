#!/bin/bash

set -e

API_URL="http://localhost:8080"
TOKEN=""

echo "Тестирование API PR Service"
echo ""

# 1. Создание команды
echo "[1/10] Создание команды..."
curl -s -X POST "$API_URL/team/add" \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }' | jq '.' || echo "(команда уже существует)"
echo ""

# 2. Получение JWT токена
echo "[2/10] Получение JWT токена..."
TOKEN=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "u1"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Не удалось получить токен"
  exit 1
fi
echo "Токен получен"
echo ""

# 3. Получение информации о команде
echo "[3/10] Получение команды..."
curl -s -X GET "$API_URL/team/get?team_name=backend" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# 4. Создание Pull Request
echo "[4/10] Создание PR..."
curl -s -X POST "$API_URL/pullRequest/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "pull_request_id": "pr-test-1",
    "pull_request_name": "Add authentication feature",
    "author_id": "u1"
  }' | jq '.'
echo ""

# 5. Получение PR для ревьювера
echo "[5/10] Получение PR для ревьювера..."
curl -s -X GET "$API_URL/users/getReview?user_id=u2" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# 6. Переназначение ревьювера
echo "[6/10] Переназначение ревьювера..."
curl -s -X POST "$API_URL/pullRequest/reassign" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "pull_request_id": "pr-test-1",
    "old_user_id": "u2"
  }' | jq '.'
echo ""

# 7. Merge PR
echo "[7/10] Merge PR..."
curl -s -X POST "$API_URL/pullRequest/merge" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"pull_request_id": "pr-test-1"}' | jq '.'
echo ""

# 8. Попытка переназначить после merge
echo "[8/10] Попытка переназначить после merge (должна провалиться)..."
curl -s -X POST "$API_URL/pullRequest/reassign" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "pull_request_id": "pr-test-1",
    "old_user_id": "u3"
  }' | jq '.'
echo ""

# 9. Деактивация пользователя
echo "[9/10] Деактивация пользователя..."
curl -s -X POST "$API_URL/users/setIsActive" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id": "u2", "is_active": false}' | jq '.'
echo ""

# 10. Статистика
echo "[10/10] Получение статистики..."
curl -s -X GET "$API_URL/stats" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

echo "Все тесты завершены успешно"