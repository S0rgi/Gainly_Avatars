.PHONY: proto generate build run deps install-tools

# Установка инструментов для разработки
install-tools:
	@echo "Installing protoc plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Генерация proto файлов
# Требует: protoc (https://grpc.io/docs/protoc-installation/)
# И установленные плагины: make install-tools
proto:
	@echo "Generating proto files..."
	@mkdir -p pkg/proto
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/user.proto
	@echo "Proto files generated successfully!"

# Установка зависимостей
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Сборка приложения
build:
	@echo "Building application..."
	@go build -o bin/server ./cmd/server

# Запуск приложения
run:
	@go run ./cmd/server

# Запуск с hot reload (требует air)
dev:
	@air

