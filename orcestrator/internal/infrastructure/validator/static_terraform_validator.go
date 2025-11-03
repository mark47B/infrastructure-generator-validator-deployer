package validator

import (
	"fmt"
	"log"
	"orchestrator/internal/domain/entity"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type AnalysisResult struct {
	Passed bool
	Errors []*entity.ValidationConfigError
}

var SensitiveKeywords = []string{"password", "secret", "key", "token", "access_key", "secret_key"}

type Analyzer interface {
	Analyze(files []*entity.ConfigFile, outputDir string) (*AnalysisResult, error)
}

type TerraformAnalyzer struct{}

func NewTerraformAnalyzer() *TerraformAnalyzer {
	return &TerraformAnalyzer{}
}

func (a *TerraformAnalyzer) Analyze(files []*entity.ConfigFile, outputDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{Passed: true}

	parser := hclparse.NewParser()
	for _, file := range files {
		if file.Type != "terraform" || !strings.HasSuffix(file.Name, ".tf") {
			continue
		}

		hclFile, fileDiags := parser.ParseHCL([]byte(file.Content), file.Name)
		if fileDiags.HasErrors() {
			for _, diag := range fileDiags {
				result.Errors = append(result.Errors, &entity.ValidationConfigError{
					File:    file.Name,
					Message: fmt.Sprintf("%s: %s", diag.Summary, diag.Detail),
					Line:    diag.Subject.Start.Line,
					Column:  diag.Subject.Start.Column,
				})
			}
			result.Passed = false
			continue
		}

		fileDiags = a.analyzeFile(hclFile.Body, file.Name)
		for _, diag := range fileDiags {
			result.Errors = append(result.Errors, &entity.ValidationConfigError{
				File:    file.Name,
				Message: fmt.Sprintf("%s: %s", diag.Summary, diag.Detail),
				Line:    diag.Subject.Start.Line,
				Column:  diag.Subject.Start.Column,
			})
		}
		if fileDiags.HasErrors() {
			result.Passed = false
		}
	}

	outputDir = filepath.Join(outputDir, "static_validator")
	if err := a.saveResults(result, outputDir); err != nil {
		return nil, fmt.Errorf("save results: %w", err)
	}

	return result, nil
}

func (a *TerraformAnalyzer) analyzeFile(body hcl.Body, fileName string) hcl.Diagnostics {
	var diags hcl.Diagnostics

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "terraform"},
			{Type: "provider", LabelNames: []string{"name"}},
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
			{Type: "variable", LabelNames: []string{"name"}},
			{Type: "output", LabelNames: []string{"name"}},
			{Type: "module", LabelNames: []string{"name"}},
		},
	}

	content, _, contentDiags := body.PartialContent(schema)
	diags = append(diags, contentDiags...)

	diags = append(diags, a.analyzeTerraformBlocks(content, fileName)...)
	diags = append(diags, a.analyzeResourceBlocks(content, fileName)...)

	return diags
}

func (a *TerraformAnalyzer) analyzeTerraformBlocks(content *hcl.BodyContent, fileName string) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, block := range content.Blocks.OfType("terraform") {
		tfSchema := &hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "required_providers"},
			},
		}
		tfContent, _, tfDiags := block.Body.PartialContent(tfSchema)
		diags = append(diags, tfDiags...)

		for _, rpBlock := range tfContent.Blocks.OfType("required_providers") {
			attrs, attrsDiags := rpBlock.Body.JustAttributes()
			diags = append(diags, attrsDiags...)

			for providerName, attr := range attrs {
				val, valDiags := attr.Expr.Value(nil)
				diags = append(diags, valDiags...)
				if val.Type().IsObjectType() {
					obj := val.AsValueMap()
					if _, hasVersion := obj["version"]; !hasVersion {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  fmt.Sprintf("Provider %s missing version constraint in %s", providerName, fileName),
							Subject:  &attr.Range,
						})
					}
				} else {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  fmt.Sprintf("Provider %s has non-object requirement in %s", providerName, fileName),
						Subject:  &attr.Range,
					})
				}
			}
		}
	}
	return diags
}

func (a *TerraformAnalyzer) analyzeResourceBlocks(content *hcl.BodyContent, fileName string) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, block := range content.Blocks.OfType("resource") {
		resType, resName := block.Labels[0], block.Labels[1]

		resSchema := &hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "lifecycle"},
			},
			Attributes: []hcl.AttributeSchema{
				{Name: "tags"},
			},
		}
		resContent, _, resDiags := block.Body.PartialContent(resSchema)
		diags = append(diags, resDiags...)

		if len(resContent.Blocks.OfType("lifecycle")) == 0 {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Resource %s.%s missing lifecycle block in %s", resType, resName, fileName),
				Subject:  &block.DefRange,
			})
		}

		if _, hasTags := resContent.Attributes["tags"]; !hasTags {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Resource %s.%s missing tags attribute in %s", resType, resName, fileName),
				Subject:  &block.DefRange,
			})
		}

		allAttrs, allAttrsDiags := block.Body.JustAttributes()
		diags = append(diags, allAttrsDiags...)
		for attrName, attr := range allAttrs {
			for _, kw := range SensitiveKeywords {
				if strings.Contains(strings.ToLower(attrName), kw) {
					_, valDiags := attr.Expr.Value(nil)
					if !valDiags.HasErrors() {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  fmt.Sprintf("Potential hardcoded sensitive value in attribute %s of resource %s.%s in %s", attrName, resType, resName, fileName),
							Subject:  &attr.Range,
						})
					}
				}
			}
		}
	}
	return diags
}

func (a *TerraformAnalyzer) saveResults(result *AnalysisResult, outputDir string) error {
	if len(result.Errors) == 0 {
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "analysis_results.txt")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create results file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}()

	for _, err := range result.Errors {
		_, writeErr := fmt.Fprintf(file, "File: %s, Line: %d, Column: %d, Message: %s\n",
			err.File, err.Line, err.Column, err.Message)
		if writeErr != nil {
			return fmt.Errorf("write results: %w", writeErr)
		}
	}

	return nil
}
