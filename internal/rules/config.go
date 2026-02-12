package rules

import (
	"fmt"
	"os"

	"github.com/DanielTso/pixshift/internal/codec"
	"gopkg.in/yaml.v3"
)

// Config represents the top-level rules configuration file.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// Rule defines a single conversion rule.
type Rule struct {
	// Name is an optional label for the rule.
	Name string `yaml:"name,omitempty"`

	// Match conditions (all specified conditions must match).
	Format string `yaml:"format,omitempty"` // input format (e.g., "heic", "cr2")
	Glob   string `yaml:"glob,omitempty"`   // filename glob pattern (e.g., "*.heic")

	// Output settings.
	Output  string `yaml:"output"`            // output format (e.g., "webp", "jpg")
	Quality int    `yaml:"quality,omitempty"`  // quality 1-100 (0 = default)
	Dir     string `yaml:"dir,omitempty"`      // output directory override
}

// ParsedRule is a Rule with parsed format fields.
type ParsedRule struct {
	Rule         Rule
	InputFormat  codec.Format // parsed from Rule.Format (empty = match any)
	OutputFormat codec.Format // parsed from Rule.Output
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("config has no rules")
	}

	return &cfg, nil
}

// ParseRules validates and parses all rules in a config.
func ParseRules(cfg *Config) ([]ParsedRule, error) {
	var parsed []ParsedRule

	for i, rule := range cfg.Rules {
		pr := ParsedRule{Rule: rule}

		if rule.Output == "" {
			return nil, fmt.Errorf("rule %d: missing output format", i+1)
		}

		outFmt, err := codec.ParseFormat(rule.Output)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i+1, err)
		}
		pr.OutputFormat = outFmt

		if rule.Format != "" {
			inFmt, err := codec.ParseFormat(rule.Format)
			if err != nil {
				return nil, fmt.Errorf("rule %d: %w", i+1, err)
			}
			pr.InputFormat = inFmt
		}

		parsed = append(parsed, pr)
	}

	return parsed, nil
}
