package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"orchestrator/internal/domain/entity"
	"orchestrator/internal/domain/repository"
	"orchestrator/internal/infrastructure/store/filesystem"
	"orchestrator/internal/infrastructure/validator"
)

type Validator interface {
	Validate(ctx context.Context, files []entity.ConfigFile) (ValidationResult, error)
	Name() string
}

// ValidationResult — результат валидатора.
type ValidationResult struct {
	Passed bool
	Errors []entity.ValidationConfigError
	Notes  string
}

type ConfigGeneratorService struct {
	jobsRepo       repository.JobRepository
	configRepo     repository.ConfgiFileRepository
	configFileRepo filesystem.FileRepository
	llm            repository.LLMGenerator

	staticVal validator.TerraformAnalyzer // Dependency on external circles
	// sandboxVal  Validator
	// securityVal Validator

	logger *slog.Logger

	pollInterval      time.Duration
	validationTimeout time.Duration
	maxRetries        int

	// control
	stop    chan struct{}
	stopped chan struct{}
}

func NewConfigGeneratorService(
	jr repository.JobRepository,
	cr repository.ConfgiFileRepository,
	cfr filesystem.FileRepository,
	llm repository.LLMGenerator,
	staticVal validator.TerraformAnalyzer,
	sandboxVal Validator,
	securityVal Validator,
	logger *slog.Logger,
) *ConfigGeneratorService {
	pi := 5 * time.Second
	return &ConfigGeneratorService{
		jobsRepo:       jr,
		configRepo:     cr,
		configFileRepo: cfr,
		llm:            llm,
		staticVal:      staticVal,
		// sandboxVal:        sandboxVal,
		// securityVal:       securityVal,
		logger:            logger,
		pollInterval:      pi,
		validationTimeout: 30 * time.Minute,
		maxRetries:        3,
		stop:              make(chan struct{}),
		stopped:           make(chan struct{}),
	}
}

func (s *ConfigGeneratorService) Start(ctx context.Context) {
	go func() {
		defer close(s.stopped)
		ticker := time.NewTicker(s.pollInterval)
		defer ticker.Stop()

		s.logger.Info("ConfigGeneratorService started", "interval", s.pollInterval)

		if err := s.runOnce(ctx); err != nil {
			s.logger.Warn("initial runOnce failed", "err", err)
		}

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("ConfigGeneratorService context canceled")
				return
			case <-s.stop:
				s.logger.Info("ConfigGeneratorService stopped by Stop()")
				return
			case <-ticker.C:
				if err := s.runOnce(ctx); err != nil {
					s.logger.Warn("runOnce failed", "err", err)
				}
			}
		}
	}()
}

func (s *ConfigGeneratorService) Stop() {
	close(s.stop)
	<-s.stopped
	s.logger.Info("ConfigGeneratorService fully stopped")
}

func (s *ConfigGeneratorService) runOnce(ctx context.Context) error {
	jobs, err := s.jobsRepo.ListByStatus(ctx, entity.JobStatusPending)
	if err != nil {
		return fmt.Errorf("list pending jobs: %w", err)
	}
	if len(jobs) == 0 {
		return nil
	}

	s.logger.Debug("found pending jobs", "count", len(jobs))

	for _, job := range jobs {
		if err := s.jobsRepo.UpdateStatus(ctx, job.ID, entity.JobStatusRunning); err != nil {
			// если не можем обновить статус — пропускаем и логируем
			s.logger.Warn("failed to set job running; skip", "job_id", job.ID, "err", err)
			continue
		}

		procCtx, cancel := context.WithTimeout(ctx, s.validationTimeout)
		func() {
			defer cancel()
			if err := s.processJob(procCtx, job); err != nil {
				s.logger.Error("processJob failed", "job_id", job.ID, "err", err)
			}
		}()
	}

	return nil
}

// processJob — полный pipeline для отдельного job:
// 1) Generate via LLM
// 2) Save files
// 3) Static validator
// 4) Set final status and publish events
func (s *ConfigGeneratorService) processJob(ctx context.Context, job *entity.Job) error {
	startTime := time.Now()
	jobID := job.ID

	s.logger.Info("start processing job", "job_id", jobID)

	// 1) Generate via LLM
	generatedResponse, err := s.llm.GenerateInfrastructure(ctx, job.Description, entity.TerraformPrompt)
	if err != nil {
		_ = s.jobsRepo.UpdateStatus(ctx, jobID, entity.JobStatusFailed)
		s.logger.Error("llm generation failed", "job_id", jobID, "err", err)
		return fmt.Errorf("llm generate: %w", err)
	}
	for i := range generatedResponse.Files {
		generatedResponse.Files[i].JobID = jobID
	}

	// 2) Save generated files
	if err := s.configRepo.SaveFiles(ctx, generatedResponse.Files); err != nil {
		_ = s.jobsRepo.UpdateStatus(ctx, jobID, entity.JobStatusFailed)
		s.logger.Error("save files failed", "job_id", jobID, "err", err)
		return fmt.Errorf("save files: %w", err)
	}

	if err := s.configFileRepo.SaveFiles(ctx, generatedResponse.Files, jobID); err != nil {
		s.logger.Error("save files to local failed", "job_id", jobID, "err", err)
	}

	// 3) Static validation
	workDir := filepath.Join(s.configFileRepo.GetBasePath(), jobID)

	staticRes, err := s.staticVal.Analyze(generatedResponse.Files, workDir)
	if err != nil {
		_ = s.jobsRepo.UpdateStatus(ctx, jobID, entity.JobStatusFailed)
		s.logger.Error("static validator error", "job_id", jobID, "err", err)
	}

	markFilesWithErrors(generatedResponse.Files, staticRes.Errors)
	if err := s.configRepo.SaveFiles(ctx, generatedResponse.Files); err != nil {
		s.logger.Error("resave files with errors failed", "job_id", jobID, "err", err)
	}

	// 6) Всё прошло - помечаем ready_to_deploy
	if err := s.jobsRepo.UpdateStatus(ctx, jobID, entity.JobStatusReady2Deploy); err != nil {
		s.logger.Warn("failed to update job to ready_to_deploy", "job_id", jobID, "err", err)
	}

	s.logger.Info("job processed", "job_id", jobID, "duration", time.Since(startTime))
	return nil
}

func markFilesWithErrors(files []*entity.ConfigFile, errors []*entity.ValidationConfigError) {
	for _, file := range files {
		file.HasError = false
		file.ErrorMsg = nil
		for _, err := range errors {
			if err.File == file.Name {
				file.HasError = true
				file.ErrorMsg = err
			}
		}
	}
}
