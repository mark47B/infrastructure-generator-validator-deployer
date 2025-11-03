package usecase

import (
	"context"
	"fmt"

	"orchestrator/internal/domain/entity"
	"orchestrator/internal/domain/repository"
)

type JobUsecase interface {
	CreateJob(ctx context.Context, description, target string) error
	GetJob(ctx context.Context, id string) (*entity.Job, error)
	ListJobs(ctx context.Context) ([]*entity.Job, error)
	UpdateStatus(ctx context.Context, jobID string, status entity.JobStatus) error
	DeleteJob(ctx context.Context, jobID string) error
	DeployJob(ctx context.Context, jobID string) error
}

var _ JobUsecase = (*JobService)(nil)

type JobService struct {
	jobsRepo   repository.JobRepository
	configRepo repository.ConfgiFileRepository
	deployer   Deployer
}

func NewJobService(
	jr repository.JobRepository,
	cr repository.ConfgiFileRepository,
	d Deployer,
) *JobService {
	return &JobService{
		jobsRepo:   jr,
		configRepo: cr,
		deployer:   d,
	}
}

func (u *JobService) DeployJob(ctx context.Context, jobID string) error {
	job, err := u.jobsRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("err get job from store: %w", err)
	}
	_, err = u.deployer.Deploy(ctx, job)
	if err != nil {
		return fmt.Errorf("err deploy job: %w", err)
	}
	err = u.jobsRepo.UpdateStatus(ctx, jobID, entity.JobStatusDeployed)
	if err != nil {
		return fmt.Errorf("err update status: %w", err)
	}
	return nil
}

func (u *JobService) CreateJob(ctx context.Context, description, target string) error {
	job := entity.NewJob(description, target)

	if err := u.jobsRepo.Create(ctx, job); err != nil { // Заменить на событие в Kafka
		return fmt.Errorf("create job: %w", err)
	}

	return nil
}

func (u *JobService) GetJob(ctx context.Context, id string) (*entity.Job, error) {
	job, err := u.jobsRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, repositoryNotFoundError(id)
	}
	return job, nil
}

func (u *JobService) ListJobs(ctx context.Context) ([]*entity.Job, error) {
	return u.jobsRepo.List(ctx)
}

func (u *JobService) UpdateStatus(ctx context.Context, jobID string, status entity.JobStatus) error {
	return u.jobsRepo.UpdateStatus(ctx, jobID, status)
}

func (u *JobService) DeleteJob(ctx context.Context, jobID string) error {

	if err := u.configRepo.DeleteRequest(ctx, jobID); err != nil {
		return fmt.Errorf("delete config files: %w", err)
	}
	if err := u.jobsRepo.Delete(ctx, jobID); err != nil {
		return fmt.Errorf("delete job: %w", err)
	}

	return nil
}

func repositoryNotFoundError(id string) error {
	return fmt.Errorf("job %s not found", id)
}
