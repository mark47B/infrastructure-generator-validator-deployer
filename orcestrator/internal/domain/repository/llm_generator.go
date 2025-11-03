package repository

import (
	"context"
	"orchestrator/internal/domain/entity"
)

// LLMGenerator интерфейс для генерации файлов через LLM
type LLMGenerator interface {
	// GenerateFiles генерирует файлы деплоя на основе описания и промпта
	GenerateInfrastructure(ctx context.Context, description string, prompt entity.Prompt) (entity.GenerateResponse, error)
	// RegenerateFileWithError регенерирует файл с учетом ошибки валидации
	RegenerateFileWithError(ctx context.Context, file entity.ConfigFile, errorMsg string, prompt entity.Prompt) (entity.ConfigFile, error)
}
