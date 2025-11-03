package repository

import (
	"context"
	"orchestrator/internal/domain/entity"
)

// JobRepository определяет интерфейс доступа к хранилищу задач (Job).
type JobRepository interface {
	Create(ctx context.Context, job *entity.Job) error
	GetByID(ctx context.Context, id string) (*entity.Job, error)
	List(ctx context.Context) ([]*entity.Job, error)
	ListByStatus(ctx context.Context, status entity.JobStatus) ([]*entity.Job, error)
	Update(ctx context.Context, job *entity.Job) error
	UpdateStatus(ctx context.Context, id string, status entity.JobStatus) error
	Delete(ctx context.Context, id string) error
	CountByStatus(ctx context.Context, status entity.JobStatus) (int, error)
}
