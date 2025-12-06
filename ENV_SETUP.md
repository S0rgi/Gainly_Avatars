# Настройка переменных окружения

## Расположение файла

Создайте файл `.env` в **корне проекта** (там же, где находится `go.mod`):

```
Gainly_Avatars/
├── .env          ← Создайте здесь
├── go.mod
├── cmd/
├── internal/
└── ...
```

## Содержимое .env файла

Скопируйте следующий шаблон и заполните своими значениями:

```env
# Server Configuration
SERVER_PORT=8080

# Cloudflare R2 Configuration
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY_ID=your_access_key_id
R2_SECRET_KEY=your_secret_key
R2_BUCKET_NAME=your_bucket_name
R2_ENDPOINT=https://your_account_id.r2.cloudflarestorage.com

# Upstash Redis Configuration
# Формат: redis://default:password@host:port
# Или для TLS: rediss://default:password@host:port
REDIS_URL=redis://default:your_password@your_redis_endpoint.upstash.io:6379

# gRPC User Service Configuration
GRPC_USER_SERVICE_ADDR=localhost:50051
```

## Как получить значения

### Cloudflare R2

1. Войдите в Cloudflare Dashboard
2. Перейдите в R2 → Manage R2 API Tokens
3. Создайте API Token с правами на чтение/запись
4. Скопируйте:
   - **Account ID** → `R2_ACCOUNT_ID`
   - **Access Key ID** → `R2_ACCESS_KEY_ID`
   - **Secret Access Key** → `R2_SECRET_KEY`
5. Создайте bucket и укажите его имя в `R2_BUCKET_NAME`
6. **R2_ENDPOINT** обычно: `https://<account_id>.r2.cloudflarestorage.com`

### Upstash Redis

1. Войдите в Upstash Console
2. Создайте базу данных Redis
3. На странице базы данных найдите **REST URL**
4. Формат URL: `redis://default:<password>@<endpoint>:<port>`
5. Скопируйте полный URL в `REDIS_URL`

Пример:
```
REDIS_URL=redis://default:AbCdEf123456@usw1-example-12345.upstash.io:6379
```

### gRPC User Service

Укажите адрес вашего gRPC сервиса аутентификации:
- Локально: `localhost:50051`
- Удаленно: `your-service.example.com:50051`

## Важно

⚠️ **НЕ коммитьте файл `.env` в git!** Он уже добавлен в `.gitignore`.

Файл `.env` содержит секретные данные и должен храниться локально или в безопасном хранилище секретов (например, в CI/CD переменных окружения).

## Проверка

После создания `.env` файла запустите сервер:

```bash
go run ./cmd/server
```

Если все настроено правильно, сервер запустится без ошибок.

