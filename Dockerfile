# ======================
#       BUILD STAGE
# ======================
FROM golang:1.25 AS builder

WORKDIR /app

# Сначала копируем go.mod/go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарник из cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server

# ======================
#       RUN STAGE
# ======================
FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates

# Копируем бинарник
COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]
