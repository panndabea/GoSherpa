package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/panndabea/GoSherpa/internal/semantics"
)

const (
	SchemaVersion = 1
	DirectoryName = ".gosherpa"
	FileName      = "config.json"
)

type Config struct {
	SchemaVersion int                `json:"schemaVersion"`
	BaseRef       string             `json:"baseRef,omitempty"`
	UseSnapshot   bool               `json:"useSnapshot"`
	BuildTags     []string           `json:"buildTags"`
	AgentContext  AgentContextConfig `json:"agentContext"`
}

type AgentContextConfig struct {
	MaxFiles   int `json:"maxFiles"`
	MaxSymbols int `json:"maxSymbols"`
	MaxTests   int `json:"maxTests"`
	MaxBytes   int `json:"maxBytes"`
}

type LoadResult struct {
	Config Config
	Exists bool
	Valid  bool
	Path   string

	Warnings []string

	rawTopLevel     map[string]json.RawMessage
	rawAgentContext map[string]json.RawMessage
}

type rawConfig struct {
	SchemaVersion *int             `json:"schemaVersion"`
	BaseRef       *string          `json:"baseRef"`
	UseSnapshot   *bool            `json:"useSnapshot"`
	BuildTags     *[]string        `json:"buildTags"`
	AgentContext  *rawAgentContext `json:"agentContext"`
}

type rawAgentContext struct {
	MaxFiles   *int `json:"maxFiles"`
	MaxSymbols *int `json:"maxSymbols"`
	MaxTests   *int `json:"maxTests"`
	MaxBytes   *int `json:"maxBytes"`
}

func Default() Config {
	return Config{
		SchemaVersion: SchemaVersion,
		UseSnapshot:   true,
		BuildTags:     []string{},
		AgentContext: AgentContextConfig{
			MaxFiles:   20,
			MaxSymbols: 40,
			MaxTests:   20,
			MaxBytes:   12000,
		},
	}
}

func WithBase(baseRef string) Config {
	cfg := Default()
	cfg.BaseRef = strings.TrimSpace(baseRef)
	return cfg
}

func Path(root string) string {
	return filepath.Join(root, DirectoryName, FileName)
}

func Load(root string) LoadResult {
	path := Path(root)
	cfg := Default()
	result := LoadResult{
		Config: cfg,
		Valid:  true,
		Path:   path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.Exists = true
		result.Valid = false
		result.Warnings = []string{fmt.Sprintf("gosherpa config could not be read from %s: %v", displayPath(root, path), err)}
		return result
	}

	result.Exists = true
	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		result.Valid = false
		result.Warnings = []string{fmt.Sprintf("gosherpa config is invalid JSON at %s: %v", displayPath(root, path), err)}
		return result
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(data, &topLevel); err == nil {
		result.rawTopLevel = topLevel
	}
	if raw.AgentContext != nil {
		if agentData, ok := result.rawTopLevel["agentContext"]; ok {
			var agentRaw map[string]json.RawMessage
			if err := json.Unmarshal(agentData, &agentRaw); err == nil {
				result.rawAgentContext = agentRaw
			}
		}
	}

	var warnings []string
	if raw.SchemaVersion == nil {
		warnings = append(warnings, "gosherpa config is missing schemaVersion")
	} else if *raw.SchemaVersion != SchemaVersion {
		warnings = append(warnings, fmt.Sprintf("gosherpa config schemaVersion %d is not supported", *raw.SchemaVersion))
	}

	if raw.BaseRef != nil {
		cfg.BaseRef = strings.TrimSpace(*raw.BaseRef)
	}
	if raw.UseSnapshot != nil {
		cfg.UseSnapshot = *raw.UseSnapshot
	}
	if raw.BuildTags != nil {
		cfg.BuildTags = semantics.NormalizeBuildTags(*raw.BuildTags)
	}
	if raw.AgentContext != nil {
		agent, agentWarnings := normalizeAgentContext(*raw.AgentContext, cfg.AgentContext)
		cfg.AgentContext = agent
		warnings = append(warnings, agentWarnings...)
	}

	result.Config = Normalize(cfg)
	if len(warnings) > 0 {
		result.Valid = false
		result.Warnings = prefixWarnings(displayPath(root, path), warnings)
	}

	return result
}

func Save(root string, cfg Config, previous *LoadResult) (string, error) {
	path := Path(root)
	cfg = Normalize(cfg)

	topLevel := map[string]json.RawMessage{}
	agentContext := map[string]json.RawMessage{}
	if previous != nil && previous.Valid {
		for key, value := range previous.rawTopLevel {
			topLevel[key] = append(json.RawMessage(nil), value...)
		}
		for key, value := range previous.rawAgentContext {
			agentContext[key] = append(json.RawMessage(nil), value...)
		}
	}

	mustSetRaw(topLevel, "schemaVersion", cfg.SchemaVersion)
	mustSetRaw(topLevel, "baseRef", cfg.BaseRef)
	mustSetRaw(topLevel, "useSnapshot", cfg.UseSnapshot)
	mustSetRaw(topLevel, "buildTags", nonNilStrings(cfg.BuildTags))

	mustSetRaw(agentContext, "maxFiles", cfg.AgentContext.MaxFiles)
	mustSetRaw(agentContext, "maxSymbols", cfg.AgentContext.MaxSymbols)
	mustSetRaw(agentContext, "maxTests", cfg.AgentContext.MaxTests)
	mustSetRaw(agentContext, "maxBytes", cfg.AgentContext.MaxBytes)
	mustSetRaw(topLevel, "agentContext", agentContext)

	data, err := json.MarshalIndent(topLevel, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return path, nil
}

func Normalize(cfg Config) Config {
	defaults := Default()
	cfg.SchemaVersion = SchemaVersion
	cfg.BaseRef = strings.TrimSpace(cfg.BaseRef)
	cfg.UseSnapshot = cfg.UseSnapshot
	cfg.BuildTags = semantics.NormalizeBuildTags(cfg.BuildTags)
	cfg.AgentContext.MaxFiles = defaultPositive(cfg.AgentContext.MaxFiles, defaults.AgentContext.MaxFiles)
	cfg.AgentContext.MaxSymbols = defaultPositive(cfg.AgentContext.MaxSymbols, defaults.AgentContext.MaxSymbols)
	cfg.AgentContext.MaxTests = defaultPositive(cfg.AgentContext.MaxTests, defaults.AgentContext.MaxTests)
	cfg.AgentContext.MaxBytes = defaultPositive(cfg.AgentContext.MaxBytes, defaults.AgentContext.MaxBytes)
	return cfg
}

func normalizeAgentContext(raw rawAgentContext, defaults AgentContextConfig) (AgentContextConfig, []string) {
	cfg := defaults
	var warnings []string
	apply := func(name string, value *int, assign func(int)) {
		if value == nil {
			return
		}
		if *value <= 0 {
			warnings = append(warnings, fmt.Sprintf("agentContext.%s must be a positive integer", name))
			return
		}
		assign(*value)
	}

	apply("maxFiles", raw.MaxFiles, func(value int) { cfg.MaxFiles = value })
	apply("maxSymbols", raw.MaxSymbols, func(value int) { cfg.MaxSymbols = value })
	apply("maxTests", raw.MaxTests, func(value int) { cfg.MaxTests = value })
	apply("maxBytes", raw.MaxBytes, func(value int) { cfg.MaxBytes = value })

	return cfg, warnings
}

func defaultPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func mustSetRaw(values map[string]json.RawMessage, key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	values[key] = data
}

func prefixWarnings(path string, warnings []string) []string {
	prefixed := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		prefixed = append(prefixed, "gosherpa config warning at "+path+": "+warning)
	}
	return prefixed
}

func displayPath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}
	return filepath.Clean(path)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
