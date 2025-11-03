package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"orchestrator/internal/domain/entity"
	"orchestrator/internal/domain/repository"
	"orchestrator/internal/infrastructure/metrics"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AmveraGenerator struct {
	apiKey    string
	baseURL   string
	model     string
	client    *http.Client
	maxTokens int
	verbosity string
}

func NewAmveraGenerator(apiKey, baseURL, model string) repository.LLMGenerator {
	return &AmveraGenerator{
		apiKey:    apiKey,
		baseURL:   baseURL,
		model:     model,
		client:    &http.Client{Timeout: 2 * time.Minute},
		maxTokens: 4000,
		verbosity: "low",
	}
}

func (g *AmveraGenerator) GenerateInfrastructure(ctx context.Context, description string, prompt entity.Prompt) (entity.GenerateResponse, error) {
	metrics.IncLLMRequest(g.model)
	fullPrompt := prompt.Text + " " + description

	request := map[string]interface{}{
		"model": g.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": fullPrompt,
			},
		},
		"temperature": 1,
		"verbosity":   g.verbosity,
	}

	response, err := g.makeRequest(ctx, request)
	if err != nil {
		metrics.IncError("llm", "make_request")
		return entity.GenerateResponse{}, fmt.Errorf("failed to make Amvera request: %w", err)
	}

	files, err := g.parseResponse(response)
	if err != nil {
		metrics.IncError("llm", "parse_response")
		return entity.GenerateResponse{}, fmt.Errorf("failed to parse Amvera response: %w", err)
	}

	return entity.GenerateResponse{
		Files:     files,
		RequestID: uuid.NewString(),
		CreatedAt: time.Now().UTC(),
		Status:    "success",
	}, nil
}

func (g *AmveraGenerator) RegenerateFileWithError(ctx context.Context, file entity.ConfigFile, errorMsg string, prompt entity.Prompt) (entity.ConfigFile, error) {
	metrics.IncLLMRequest(g.model)

	regeneratePrompt := fmt.Sprintf("Please fix the following %s file based on the validation errors:\n\nOriginal file content:\n```\n%s\n```\n\nValidation errors:\n%s\n\nPlease provide the corrected file content only, without any explanations or markdown formatting.", file.Type, file.Content, errorMsg)

	request := map[string]interface{}{
		"model": g.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": regeneratePrompt,
			},
		},
		"temperature": 1,
		"verbosity":   "high",
	}

	response, err := g.makeRequest(ctx, request)
	if err != nil {
		metrics.IncError("llm", "make_request")
		return file, fmt.Errorf("failed to make Amvera request: %w", err)
	}

	correctedContent, err := g.parseSingleFileResponse(response)
	if err != nil {
		metrics.IncError("llm", "parse_single_response")
		return file, fmt.Errorf("failed to parse Amvera response: %w", err)
	}

	file.Content = correctedContent
	file.HasError = false
	file.ErrorMsg = nil

	return file, nil
}

func (g *AmveraGenerator) makeRequest(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		metrics.IncError("llm", "marshal_request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		metrics.IncError("llm", "create_request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", "Bearer "+g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		metrics.IncError("llm", "http_do")
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("close body err: %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		metrics.IncError("llm", fmt.Sprintf("api_error_%d", resp.StatusCode))
		return nil, fmt.Errorf("amvera api error: %d - %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		metrics.IncError("llm", "decode_response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}

func (g *AmveraGenerator) parseResponse(response map[string]interface{}) ([]*entity.ConfigFile, error) {
	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("invalid response format: no choices")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: invalid choice")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: no message")
	}

	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format: no content")
	}

	files := g.extractFilesFromContent(content)

	if len(files) == 0 {
		files = []*entity.ConfigFile{
			{
				JobID:    "",
				Name:     "main.tf",
				Content:  content,
				Type:     "terraform",
				HasError: false,
				ErrorMsg: nil,
			},
		}
	}

	return files, nil
}

func (g *AmveraGenerator) parseSingleFileResponse(response map[string]interface{}) (string, error) {
	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format: no choices")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: invalid choice")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: no message")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format: no content")
	}

	return strings.TrimSpace(content), nil
}

func (g *AmveraGenerator) extractFilesFromContent(content string) []*entity.ConfigFile {
	var files []*entity.ConfigFile

	lines := strings.Split(content, "\n")
	var currentFile *entity.ConfigFile
	var inCodeBlock bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "```") {
			if inCodeBlock && currentFile != nil {

				files = append(files, currentFile)
				currentFile = nil
			} else {
				fileName := strings.TrimPrefix(line, "```")
				if fileName == "" {
					fileName = "main.tf"
				}

				currentFile = &entity.ConfigFile{
					Name:     fileName,
					Content:  "",
					Type:     g.detectFileType(fileName),
					HasError: false,
					ErrorMsg: nil,
				}
			}
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock && currentFile != nil {
			if currentFile.Content != "" {
				currentFile.Content += "\n"
			}
			currentFile.Content += line
		}
	}

	if currentFile != nil {
		files = append(files, currentFile)
	}

	return files
}

func (g *AmveraGenerator) detectFileType(fileName string) string {
	fileName = strings.ToLower(fileName)

	if strings.HasSuffix(fileName, ".tf") {
		return "terraform"
	}
	if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
		return "kubernetes"
	}
	if strings.HasSuffix(fileName, ".yml") || strings.HasSuffix(fileName, ".yaml") {
		return "ansible"
	}

	return "unknown"
}
