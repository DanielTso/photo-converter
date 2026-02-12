package rules

import (
	"path/filepath"

	"github.com/DanielTso/pixshift/internal/codec"
	"github.com/DanielTso/pixshift/internal/pipeline"
)

const defaultQuality = 92

// Engine matches files against rules to generate conversion jobs.
type Engine struct {
	Rules    []ParsedRule
	OutputDir string // global output directory override
	Quality   int    // global quality default
	Metadata  bool   // preserve metadata globally
}

// NewEngine creates a rules engine from parsed rules.
func NewEngine(rules []ParsedRule) *Engine {
	return &Engine{
		Rules:   rules,
		Quality: defaultQuality,
	}
}

// Match finds the first rule that matches the given file and returns a Job.
// Returns nil if no rule matches.
func (e *Engine) Match(filePath string, inputFormat codec.Format) *pipeline.Job {
	for _, rule := range e.Rules {
		if !e.ruleMatches(rule, filePath, inputFormat) {
			continue
		}

		quality := rule.Rule.Quality
		if quality == 0 {
			quality = e.Quality
		}

		outDir := rule.Rule.Dir
		if outDir == "" {
			outDir = e.OutputDir
		}

		outputPath := buildOutputPath(filePath, outDir, rule.OutputFormat)

		return &pipeline.Job{
			InputPath:        filePath,
			OutputPath:       outputPath,
			InputFormat:      inputFormat,
			OutputFormat:     rule.OutputFormat,
			Quality:          quality,
			PreserveMetadata: e.Metadata,
		}
	}
	return nil
}

func (e *Engine) ruleMatches(rule ParsedRule, filePath string, inputFormat codec.Format) bool {
	// Check format match
	if rule.InputFormat != "" && rule.InputFormat != inputFormat {
		return false
	}

	// Check glob pattern match
	if rule.Rule.Glob != "" {
		matched, err := filepath.Match(rule.Rule.Glob, filepath.Base(filePath))
		if err != nil || !matched {
			return false
		}
	}

	return true
}

func buildOutputPath(inputPath, outputDir string, format codec.Format) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + codec.DefaultExtension(format)

	if outputDir != "" {
		return filepath.Join(outputDir, name)
	}
	return filepath.Join(filepath.Dir(inputPath), name)
}
