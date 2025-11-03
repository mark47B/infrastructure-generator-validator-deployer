package usecase

import (
	"context"
	"fmt"

	"orchestrator/internal/domain/entity"
	"orchestrator/internal/domain/repository"
)

type ConfigFilesUseCase interface {
	SaveFiles(ctx context.Context, files []*entity.ConfigFile, requestID string) error
	GetFiles(ctx context.Context, requestID string) ([]*entity.ConfigFile, error)
	GetFilesByJobID(ctx context.Context, jobID string) ([]*entity.ConfigFile, error)
	ListRequests(ctx context.Context) ([]string, error)
	DeleteRequest(ctx context.Context, requestID string) error
}

type ConfigService struct {
	repo repository.ConfgiFileRepository
}

func NewConfigService(repo repository.ConfgiFileRepository) ConfigFilesUseCase {
	return &ConfigService{repo: repo}
}

var _ ConfigFilesUseCase = (*ConfigService)(nil)

func (s *ConfigService) SaveFiles(ctx context.Context, files []*entity.ConfigFile, JobID string) error {
	if len(files) == 0 {
		return nil
	}
	if JobID == "" {
		return fmt.Errorf("requestID is required")
	}
	if err := s.repo.SaveFiles(ctx, files); err != nil {
		return fmt.Errorf("save files for JobID %s: %w", JobID, err)
	}
	return nil
}

func (s *ConfigService) GetFiles(ctx context.Context, requestID string) ([]*entity.ConfigFile, error) {
	if requestID == "" {
		return nil, fmt.Errorf("requestID is required")
	}
	files, err := s.repo.GetFiles(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("get files for request %s: %w", requestID, err)
	}
	return files, nil
}

func (s *ConfigService) GetFilesByJobID(ctx context.Context, jobID string) ([]*entity.ConfigFile, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID is required")
	}
	files, err := s.repo.GetFilesByJobID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get files for job %s: %w", jobID, err)
	}
	return files, nil
}

func (s *ConfigService) ListRequests(ctx context.Context) ([]string, error) {
	reqs, err := s.repo.ListRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	return reqs, nil
}

func (s *ConfigService) DeleteRequest(ctx context.Context, requestID string) error {
	if requestID == "" {
		return fmt.Errorf("requestID is required")
	}
	if err := s.repo.DeleteRequest(ctx, requestID); err != nil {
		return fmt.Errorf("delete request %s: %w", requestID, err)
	}
	return nil
}
