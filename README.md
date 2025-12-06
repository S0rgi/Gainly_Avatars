# Gainly Avatars Service

REST API сервис для управления аватарками пользователей с хранением в Cloudflare R2 и метаданными в Upstash Redis. Аутентификация выполняется через gRPC сервис UserService.

## Функциональность

- **POST /api/avatar** - Загрузка аватарки (требует аутентификации)
- **POST /api/avatars** - Получение аватарок по списку username (без аутентификации)
- **GET /api/avatar/me** - Получение своей аватарки (требует аутентификации)
- **DELETE /api/avatar/me** - Удаление своей аватарки (требует аутентификации)

## Swagger UI

После запуска сервера Swagger UI доступен по адресу:
```
http://localhost:8080/swagger/
```

## Требования

- Go 1.25.1 или выше
- Protobuf compiler (protoc) - только для генерации proto файлов UserService
- Cloudflare R2 bucket
- Upstash Redis instance
- gRPC User Service (для аутентификации)

## Установка

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd Gainly_Avatars
```

2. Установите Protobuf compiler (опционально, только если нужно генерировать proto):
   - Windows: Скачайте с https://github.com/protocolbuffers/protobuf/releases
   - macOS: `brew install protobuf`
   - Linux: `sudo apt-get install protobuf-compiler` или `sudo yum install protobuf-compiler`

3. Установите Go плагины для protobuf (опционально):
```bash
make install-tools
```

4. Установите зависимости:
```bash
make deps
```

5. Сгенерируйте proto файлы (если нужно):
```bash
make proto
```

6. Настройте переменные окружения:
```bash
cp .env.example .env
# Отредактируйте .env файл с вашими настройками
```

## Запуск

### Разработка
```bash
make run
```

### Production
```bash
make build
./bin/server
```

## API Endpoints

### Загрузка аватарки
```bash
curl -X POST http://localhost:8080/api/avatar \
  -H "Authorization: Bearer <token>" \
  -F "avatar=@/path/to/avatar.jpg"
```

Ответ:
```json
{
  "guid": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Получение аватарок по списку username
```bash
curl -X POST http://localhost:8080/api/avatars \
  -H "Content-Type: application/json" \
  -d '{"usernames": ["user1", "user2"]}'
```

Ответ:
```json
{
  "user1": "https://r2.example.com/avatars/guid1",
  "user2": "https://r2.example.com/avatars/guid2"
}
```

### Получение своей аватарки
```bash
curl -X GET http://localhost:8080/api/avatar/me \
  -H "Authorization: Bearer <token>"
```

Ответ:
```json
{
  "url": "https://r2.example.com/avatars/550e8400-e29b-41d4-a716-446655440000"
}
```

### Удаление своей аватарки
```bash
curl -X DELETE http://localhost:8080/api/avatar/me \
  -H "Authorization: Bearer <token>"
```

Ответ: `204 No Content`

## Структура проекта

```
.
├── cmd/
│   └── server/
│       └── main.go          # Точка входа приложения
├── docs/
│   └── swagger.json        # Swagger документация
├── internal/
│   ├── clients/             # Клиенты для внешних сервисов
│   │   ├── grpc_client.go   # gRPC клиент для UserService (аутентификация)
│   │   ├── redis_client.go  # Redis клиент
│   │   └── r2_client.go     # Cloudflare R2 клиент
│   ├── config/
│   │   └── config.go        # Конфигурация
│   ├── handlers/
│   │   └── handlers.go      # REST API handlers
│   ├── middleware/
│   │   └── auth.go          # Middleware для аутентификации через gRPC
│   └── services/
│       └── avatar_service.go # Бизнес-логика
├── pkg/
│   └── proto/
│       └── user.proto        # Proto файл для UserService (только для аутентификации)
├── go.mod
├── Makefile
└── README.md
```

## Переменные окружения

- `SERVER_PORT` - Порт для HTTP сервера (по умолчанию: 8080)
- `R2_ACCOUNT_ID` - Cloudflare R2 Account ID
- `R2_ACCESS_KEY_ID` - Cloudflare R2 Access Key ID
- `R2_SECRET_KEY` - Cloudflare R2 Secret Key
- `R2_BUCKET_NAME` - Имя bucket в R2
- `R2_ENDPOINT` - Endpoint для R2 (опционально)
- `REDIS_URL` - URL для подключения к Upstash Redis
- `GRPC_USER_SERVICE_ADDR` - Адрес gRPC User Service (по умолчанию: localhost:50051)

## Хранение данных

### Redis структура:
- `username:<username>` -> `<guid>` - Связь username с GUID аватарки
- `avatar:<guid>` -> JSON метаданные - Метаданные аватарки (GUID, username, filename, size, mime_type, uploaded_at)

### R2 структура:
- `avatars/<guid>` - Файлы аватарок

## Аутентификация

Все методы, кроме `GetAvatarsByUsernames`, требуют аутентификации через Bearer token в заголовке `Authorization`:

```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

Токен валидируется через gRPC вызов к `UserService.ValidateToken`.

## Health Check

Сервис предоставляет health check endpoint:
```
GET /health
```

Возвращает `200 OK` если сервис работает.
