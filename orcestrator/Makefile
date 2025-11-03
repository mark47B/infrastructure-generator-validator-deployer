.PHONY: build test clean fmt lint run help

# Переменные
BINARY_NAME=orchestrator
BUILD_DIR=build
MAIN_PATH=./app/cmd/main.go

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

help: ## Показать справку
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

build: ## Собрать приложение
	@echo "$(GREEN)Сборка приложения...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

test: ## Запустить тесты
	@echo "$(GREEN)Запуск тестов...$(NC)"
	@go test ./... -v -race -cover
	@echo "$(GREEN)Тесты завершены$(NC)"

test-coverage: ## Запустить тесты с покрытием
	@echo "$(GREEN)Запуск тестов с покрытием...$(NC)"
	@go test ./... -v -race -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Отчет о покрытии создан: coverage.html$(NC)"

fmt: ## Форматировать код
	@echo "$(GREEN)Форматирование кода...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Форматирование завершено$(NC)"

lint: ## Запустить линтер
	@echo "$(GREEN)Запуск линтера...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint не установлен. Установите его: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

mod-tidy: ## Очистить зависимости
	@echo "$(GREEN)Очистка зависимостей...$(NC)"
	@go mod tidy
	@echo "$(GREEN)Зависимости очищены$(NC)"

mod-download: ## Скачать зависимости
	@echo "$(GREEN)Скачивание зависимостей...$(NC)"
	@go mod download
	@echo "$(GREEN)Зависимости скачаны$(NC)"

run: build ## Запустить приложение
	@echo "$(GREEN)Запуск приложения...$(NC)"
	@echo "$(YELLOW)Убедитесь, что установлена переменная окружения AMVERA_API_KEY$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME)

run-dev: ## Запустить приложение в режиме разработки
	@echo "$(GREEN)Запуск приложения в режиме разработки...$(NC)"
	@echo "$(YELLOW)Убедитесь, что установлена переменная окружения AMVERA_API_KEY$(NC)"
	@go run $(MAIN_PATH)

clean: ## Очистить артефакты сборки
	@echo "$(GREEN)Очистка артефактов...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf deployments
	@echo "$(GREEN)Очистка завершена$(NC)"

install-deps: ## Установить зависимости для разработки
	@echo "$(GREEN)Установка зависимостей для разработки...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/stretchr/testify@latest
	@echo "$(GREEN)Зависимости установлены$(NC)"

docker-build: ## Собрать Docker образ
	@echo "$(GREEN)Сборка Docker образа...$(NC)"
	@docker build -t $(BINARY_NAME) .
	@echo "$(GREEN)Docker образ собран: $(BINARY_NAME)$(NC)"

docker-run: ## Запустить в Docker
	@echo "$(GREEN)Запуск в Docker...$(NC)"
	@echo "$(YELLOW)Убедитесь, что установлена переменная окружения AMVERA_API_KEY$(NC)"
	@docker run --rm -p 8080:8080 -e AMVERA_API_KEY=$$AMVERA_API_KEY $(BINARY_NAME)

check: fmt lint test ## Проверить код (форматирование, линтер, тесты)

all: clean mod-tidy fmt lint test build ## Полная сборка (очистка, зависимости, проверки, сборка)

# Примеры использования
example-request: ## Показать пример запроса к API
	@echo "$(GREEN)Пример запроса к API:$(NC)"
	@echo "curl -X POST http://localhost:8080/job \\"
	@echo "  -H 'Content-Type: application/json' \\"
	@echo "  -d '{"
	@echo "    \"description\": \"Deploy a simple web application with nginx on AWS\","
	@echo "    \"target\": \"terraform\""
	@echo "  }'"

example-health: ## health check
	@echo "$(GREEN)Проверка здоровья сервиса:$(NC)"
	@echo "curl http://localhost:8080/health"
