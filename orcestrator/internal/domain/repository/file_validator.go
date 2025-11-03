package repository

import (
	"context"
	"orchestrator/internal/domain/entity"
)

// FileValidator интерфейс для валидации сгенерированных файлов
type ConfigFileValidator interface {
	ValidateFile(ctx context.Context, file entity.ConfigFile) ([]entity.ValidationConfigError, error)
}
