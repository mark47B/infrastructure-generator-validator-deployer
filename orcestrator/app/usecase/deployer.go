package usecase

import (
	"context"
	"fmt"
	"orchestrator/internal/domain/entity"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Deployer interface {
	Deploy(ctx context.Context, job *entity.Job) (string, error)
}

type TerraformDeployer struct {
}

func NewTerraformDeployer() *TerraformDeployer {
	return &TerraformDeployer{}
}

func (t *TerraformDeployer) Deploy(parent context.Context, job *entity.Job) (string, error) {
	if job.ID == "" {
		return "", fmt.Errorf("job id is empty")
	}

	relPath := filepath.Join("./deployments", job.ID)

	info, err := os.Stat(relPath)
	if err != nil {
		return "", fmt.Errorf("deployment directory not found %q: %w", relPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("deployment path is not a directory: %s", relPath)
	}

	logDir := filepath.Join(relPath, "terraform-deployer-logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", fmt.Errorf("create logs dir: %w", err)
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", job.ID))
	f, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}
	defer func() {
		_ = f.Sync()
		_ = f.Close()
	}()

	header := fmt.Sprintf("job_id: %s\nstarted_at: %s\ndir: %s\n\n--- COMMAND OUTPUT ---\n\n",
		job.ID, time.Now().Format(time.RFC3339), relPath)
	if _, err := f.WriteString(header); err != nil {
		return logPath, fmt.Errorf("write header to log: %w", err)
	}

	runCmd := func(ctx context.Context, name string, args ...string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = relPath

		cmd.Stdout = f
		cmd.Stderr = f

		if err := cmd.Start(); err != nil {
			return err
		}

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			<-done
			return ctx.Err()
		case err := <-done:
			return err
		}
	}

	initCtx, cancelInit := context.WithTimeout(parent, 2*time.Minute)
	defer cancelInit()

	if _, err := f.WriteString("\n--- terraform init ---\n"); err != nil {
		return logPath, fmt.Errorf("write log: %w", err)
	}
	if err := runCmd(initCtx, "terraform", "init", "-input=false"); err != nil {
		if initCtx.Err() != nil {
			return logPath, fmt.Errorf("terraform init canceled or timed out: %w", initCtx.Err())
		}
		return logPath, fmt.Errorf("terraform init failed: %w", err)
	}

	applyCtx, cancelApply := context.WithTimeout(parent, 5*time.Minute)
	defer cancelApply()

	if _, err := f.WriteString("\n--- terraform apply ---\n"); err != nil {
		return logPath, fmt.Errorf("write log: %w", err)
	}
	if err := runCmd(applyCtx, "terraform", "apply", "-auto-approve", "-input=false"); err != nil {
		if applyCtx.Err() != nil {
			return logPath, fmt.Errorf("terraform apply canceled or timed out: %w", applyCtx.Err())
		}
		return logPath, fmt.Errorf("terraform apply failed: %w", err)
	}

	if _, err := f.WriteString("\n--- SUCCESS ---\nended_at: " + time.Now().Format(time.RFC3339) + "\n"); err != nil {
		return logPath, fmt.Errorf("write footer to log: %w", err)
	}

	return logPath, nil
}
