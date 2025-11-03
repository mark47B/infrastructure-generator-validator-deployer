package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"orchestrator/internal/domain/entity"
	"os"
	"path/filepath"
	"time"
)

type FileRepository struct {
	basePath string
}

func (fr *FileRepository) GetBasePath() string {
	return fr.basePath
}

func NewFileRepository(basePath string) (FileRepository, error) {
	info, err := os.Stat(basePath)
	if os.IsNotExist(err) {
		if mkErr := os.MkdirAll(basePath, 0755); mkErr != nil {
			return FileRepository{}, fmt.Errorf("failed to create directory %s: %w", basePath, mkErr)
		}
	} else if err != nil {
		return FileRepository{}, fmt.Errorf("failed to check directory %s: %w", basePath, err)
	} else if !info.IsDir() {
		return FileRepository{}, fmt.Errorf("path %s exists but is not a directory", basePath)
	}

	return FileRepository{
		basePath: basePath,
	}, nil
}

func (r *FileRepository) SaveFiles(ctx context.Context, files []*entity.ConfigFile, requestID string) error {
	requestDir := filepath.Join(r.basePath, requestID)
	if err := os.MkdirAll(requestDir, 0766); err != nil {
		return fmt.Errorf("failed to create request directory: %w", err)
	}

	for _, file := range files {
		filePath := filepath.Join(requestDir, file.Name)

		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Name, err)
		}
	}

	metadata := map[string]interface{}{
		"request_id":  requestID,
		"created_at":  time.Now(),
		"files_count": len(files),
		"files":       files,
	}

	metadataPath := filepath.Join(requestDir, "metadata.json")
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0744); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func (r *FileRepository) GetFiles(ctx context.Context, requestID string) ([]*entity.ConfigFile, error) {
	metadataPath := filepath.Join(r.basePath, requestID, "metadata.json")

	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("request not found: %s", requestID)
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata struct {
		Files []*entity.ConfigFile `json:"files"`
	}

	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	for i, file := range metadata.Files {
		filePath := filepath.Join(r.basePath, requestID, file.Name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file.Name, err)
		}
		metadata.Files[i].Content = string(content)
	}

	return metadata.Files, nil
}

func (r *FileRepository) ListRequests(ctx context.Context) ([]string, error) {
	var requests []string

	err := filepath.WalkDir(r.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && path != r.basePath {
			metadataPath := filepath.Join(path, "metadata.json")
			if _, err := os.Stat(metadataPath); err == nil {
				requests = append(requests, filepath.Base(path))
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return requests, nil
}

func (r *FileRepository) DeleteRequest(ctx context.Context, requestID string) error {
	requestDir := filepath.Join(r.basePath, requestID)

	if err := os.RemoveAll(requestDir); err != nil {
		return fmt.Errorf("failed to delete request directory: %w", err)
	}

	return nil
}

func (r *FileRepository) GetFilesByJobID(ctx context.Context, jobID string) ([]*entity.ConfigFile, error) {
	return []*entity.ConfigFile{}, fmt.Errorf("FileRepository does not support GetFilesByJobID functionality")
}
