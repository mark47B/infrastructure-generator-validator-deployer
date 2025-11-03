package repository

import (
	"context"
	"orchestrator/internal/domain/entity"
)

type ConfgiFileRepository interface {
	SaveFiles(ctx context.Context, files []*entity.ConfigFile) error
	GetFiles(ctx context.Context, requestID string) ([]*entity.ConfigFile, error)
	ListRequests(ctx context.Context) ([]string, error)
	DeleteRequest(ctx context.Context, requestID string) error
	GetFilesByJobID(ctx context.Context, requestID string) ([]*entity.ConfigFile, error)
}
