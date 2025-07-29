# Delayed Notifier

## Описание

**Delayed Notifier** — сервис для отложенной отправки email-уведомлений. Позволяет создавать уведомления, которые будут отправлены на указанный email в заданное время. Использует PostgreSQL, Redis, Kafka и email SMTP.

---

## Архитектура

- **API (cmd/delayed-notifier):** HTTP-сервер, принимает запросы на создание, получение и удаление уведомлений.
- **Worker (cmd/worker):** Фоновый воркер, который:
  - периодически ищет уведомления, готовые к отправке, и ставит их в очередь Kafka;
  - слушает Kafka и отправляет email через SMTP.
- **PostgreSQL:** Хранит уведомления.
- **Redis:** Кэширует уведомления для ускорения чтения.
- **Kafka:** Очередь для передачи уведомлений между API и воркером.
- **Email (SMTP):** Отправка email-сообщений.

### Взаимодействие компонентов

1. Пользователь создаёт уведомление через HTTP API.
2. API сохраняет уведомление в PostgreSQL и кэширует в Redis.
3. Worker периодически ищет уведомления, которые пора отправить, и помещает их в Kafka.
4. Worker слушает Kafka, отправляет email и обновляет статус уведомления.

---

## Быстрый старт

### Запуск через Docker Compose

```bash
git clone <repo-url>
cd delayed-notifier
# 1. Запустите инфраструктуру и сборку контейнеров
docker compose up --build
# 2. После запуска — примените миграции к базе данных
make migrate-up
# 3. Создайте топики Kafka (один раз)
make create-topics
```

- API будет доступен на `http://localhost:8080`
- PostgreSQL: порт 5435
- Redis: порт 6379
- Kafka: порт 9092

### Переменные окружения

Скопируйте файл `env.example` в `.env` и настройте переменные под ваше окружение:

```bash
cp .env.example .env
```

Пример настроек в `.env`:

```
SERVER_PORT=8080
DB_HOST=postgres
DB_PORT=5435
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=postgres
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
KAFKA_HOST=kafka
KAFKA_PORT=9092
KAFKA_TOPIC=notify-topic
MAIL_HOST=smtp.example.com
MAIL_PORT=465
MAIL_USER=notifier-app
MAIL_PASSWORD=yourpassword
LOG_LEVEL=debug
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

### Миграции

```bash
make migrate-up
```

## Примеры HTTP-запросов

### Создать уведомление

```bash
curl -X POST http://localhost:8080/notify \
  -H 'Content-Type: application/json' \
  -d '{
    "send_at": "2024-12-31T23:59:00Z",
    "message": "С Новым годом!",
    "email": "user@example.com"
  }'
```
**Ответ:**
```json
{
  "id": "<uuid>",
  "send_at": "2024-12-31T23:59:00Z",
  "message": "С Новым годом!",
  "status": "scheduled",
  "email": "user@example.com"
}
```

### Получить уведомление

```bash
curl http://localhost:8080/notify/<id>
```
**Ответ:**
```json
{
  "id": "<uuid>",
  "send_at": "2024-12-31T23:59:00Z",
  "message": "С Новым годом!",
  "status": "scheduled",
  "email": "user@example.com"
}
```

### Удалить уведомление

```bash
curl -X DELETE http://localhost:8080/notify/<id>
```
**Ответ:** HTTP 204 No Content

---

## Формат уведомления

```json
{
  "id": "string (uuid)",
  "send_at": "RFC3339 datetime",
  "message": "string",
  "status": "scheduled|queued|sent|failed",
  "email": "string"
}
```

---

## Тесты и линтинг

- Запуск тестов:
  ```bash
  go test ./...
  ```
- Линтинг:
  ```bash
  make lint
  ```

---

## CI

В проекте настроен GitHub Actions для автоматической проверки кода: линтинг и тесты при каждом push/pull request.
