# PR Reviewer Service

Микросервис для автоматического назначения ревьюверов на Pull Request'ы.

## Старт

```bash
git clone https://github.com/Chamistery/Test_task.git
cd Test_task

# Запуск
docker-compose up --build

# Сервис доступен на http://localhost:8080
```

## Функциональность

### Основные задания (OpenAPI)
1. **POST /team/add** - Создание команды с участниками
2. **GET /team/get** - Получение команды
3. **POST /users/setIsActive** - Установка активности пользователя
4. **GET /users/getReview** - PR'ы где пользователь ревьювер
5. **POST /pullRequest/create** - Создание PR с автоназначением (до 2 ревьюверов)
6. **POST /pullRequest/merge** - Merge PR (идемпотентный)
7. **POST /pullRequest/reassign** - Переназначение ревьювера

### Дополнительные задания
1. **GET /statistics** - Статистика по назначениям и PR
2. **POST /team/deactivate** - Массовая деактивация команды (< 100ms)
3. **tests/integration_test.go** - Интеграционные тесты
4. **tests/load_test.go** - Нагрузочное тестирование
5. **.golangci.yml** - Конфигурация линтера

## API Endpoints

### Teams

#### POST /team/add
```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true}
    ]
  }'
```

#### POST /team/deactivate
Массовая деактивация всех членов команды и переназначение их открытых PR.

```bash
curl -X POST http://localhost:8080/team/deactivate \
  -H "Content-Type: application/json" \
  -d '{"team_name": "backend"}'
```

**Ответ:**
```json
{
  "deactivated_users": 5,
  "reassigned_prs": 12,
  "duration": "85.234ms"
}
```

### Statistics

#### GET /statistics
```bash
curl http://localhost:8080/statistics
```

**Ответ:**
```json
{
  "total_prs": 150,
  "open_prs": 45,
  "merged_prs": 105,
  "reviewer_assignments": {
    "Alice": 87,
    "Bob": 92,
    "Charlie": 65
  },
  "prs_by_author": {
    "Alice": 42,
    "Bob": 38,
    "Charlie": 70
  },
  "average_reviewers_per_pr": 1.63
}
```

## Тестирование

### Интеграционные тесты
```bash
make integration-test
```

**Базовые сценарии (integration_test.go):**
- Создание и получение команды
- Создание PR с автоназначением ревьюверов
- Идемпотентность операции merge

**Крайние случаи (edge_cases_test.go):**
- Назначение 0 ревьюверов при отсутствии активных участников команды
- Назначение 1 ревьювера при наличии только одного доступного кандидата
- Запрет переназначения ревьювера после merge PR
- Обработка попытки переназначения неназначенного ревьювера
- Проверка исключения неактивных пользователей из назначения
- Переназначение ревьювера из команды заменяемого участника

### Нагрузочное тестирование
```bash
make load-test
```

## Команды

```bash
# Разработка
make build          # Сборка
make run            # Локальный запуск
make test           # Все тесты
make lint           # Линтер
make format         # Форматирование кода

# Docker
make docker-up      # Запуск в Docker
make docker-down    # Остановка
make docker-logs    # Логи

# Тестирование
make integration-test  # Интеграционные тесты
make load-test        # Нагрузочное тестирование

# Очистка
make clean
```

## Структура проекта

```
github.com/Chamistery/Test_task/
├── cmd/server/              # Entry point
├── internal/
│   ├── models/             # Модели (OpenAPI схемы)
│   ├── storage/            # БД слой
│   ├── service/            # Бизнес-логика
│   └── handlers/           # HTTP handlers
│       ├── teams.go
│       ├── users.go
│       ├── pull_requests.go
│       └── statistics.go
├── tests/
│   ├── integration_test.go # Базовые интеграционные тесты
│   ├── edge_cases_test.go  # Тесты граничных условий
│   └── load_test.go        # Нагрузочные тесты
├── docker-compose.yml
├── Dockerfile
├── .golangci.yml           # Конфигурация линтера
├── load_test_results.md    # Результаты тестов
├── Makefile
└── README.md
```

## Технологии

- **Go 1.24**
- **PostgreSQL 15**
- **Docker & Docker Compose**
- **golangci-lint**
- **Clean Architecture**

## Соответствие требованиям

### Основные требования
- Автоматическое назначение до 2 активных ревьюверов из команды автора
- Исключение автора из списка потенциальных ревьюверов
- Переназначение ревьювера на участника из той же команды
- Запрет изменения состава ревьюверов после merge
- Назначение доступного количества ревьюверов (0/1/2)
- Идемпотентность операции merge
- Запуск через docker-compose up на порту 8080
- Соответствие OpenAPI спецификации

### Дополнительные задания
- Эндпоинт статистики
- Нагрузочное тестирование
- Массовая деактивация команды
- Интеграционные тесты
- Конфигурация линтера
