package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingConfigReturnsDefaultsWithoutWarnings(t *testing.T) {
	root := t.TempDir()

	result := Load(root)

	if result.Exists {
		t.Fatal("expected missing config")
	}
	if !result.Valid {
		t.Fatal("expected missing config to be valid fallback")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
	if result.Config.UseSnapshot != true {
		t.Fatalf("expected snapshot default true, got %#v", result.Config)
	}
	if result.Config.BaseRef != "" {
		t.Fatalf("expected no base default, got %q", result.Config.BaseRef)
	}
	if result.Config.AgentContext.MaxFiles != 20 ||
		result.Config.AgentContext.MaxSymbols != 40 ||
		result.Config.AgentContext.MaxTests != 20 ||
		result.Config.AgentContext.MaxBytes != 12000 {
		t.Fatalf("unexpected defaults: %#v", result.Config.AgentContext)
	}
}

func TestLoadNormalizesConfig(t *testing.T) {
	root := t.TempDir()
	writeConfigTestFile(t, Path(root), `{
  "schemaVersion": 1,
  "baseRef": " main ",
  "useSnapshot": false,
  "buildTags": ["enterprise, integration", "enterprise"],
  "agentContext": {
    "maxFiles": 5
  }
}`)

	result := Load(root)

	if !result.Exists || !result.Valid {
		t.Fatalf("expected valid config, got %#v", result)
	}
	if result.Config.BaseRef != "main" {
		t.Fatalf("expected trimmed base, got %q", result.Config.BaseRef)
	}
	if result.Config.UseSnapshot {
		t.Fatal("expected stored useSnapshot false")
	}
	assertConfigTestStrings(t, result.Config.BuildTags, []string{"enterprise", "integration"})
	if result.Config.AgentContext.MaxFiles != 5 ||
		result.Config.AgentContext.MaxSymbols != 40 ||
		result.Config.AgentContext.MaxTests != 20 ||
		result.Config.AgentContext.MaxBytes != 12000 {
		t.Fatalf("unexpected agent context defaults: %#v", result.Config.AgentContext)
	}
}

func TestLoadInvalidJSONReturnsWarningFallback(t *testing.T) {
	root := t.TempDir()
	writeConfigTestFile(t, Path(root), `{`)

	result := Load(root)

	if !result.Exists {
		t.Fatal("expected config to exist")
	}
	if result.Valid {
		t.Fatal("expected invalid config")
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "invalid JSON") {
		t.Fatalf("expected invalid JSON warning, got %#v", result.Warnings)
	}
	if result.Config.AgentContext.MaxFiles != 20 {
		t.Fatalf("expected fallback defaults, got %#v", result.Config)
	}
}

func TestLoadInvalidAgentContextLimitReturnsWarning(t *testing.T) {
	root := t.TempDir()
	writeConfigTestFile(t, Path(root), `{
  "schemaVersion": 1,
  "baseRef": "main",
  "useSnapshot": true,
  "agentContext": {
    "maxFiles": 0
  }
}`)

	result := Load(root)

	if result.Valid {
		t.Fatal("expected invalid config")
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), "agentContext.maxFiles") {
		t.Fatalf("expected invalid limit warning, got %#v", result.Warnings)
	}
}

func TestSaveCreatesConfigAndPreservesUnknownFields(t *testing.T) {
	root := t.TempDir()
	writeConfigTestFile(t, Path(root), `{
  "schemaVersion": 1,
  "baseRef": "main",
  "useSnapshot": false,
  "buildTags": [],
  "agentContext": {
    "maxFiles": 10,
    "maxSymbols": 11,
    "maxTests": 12,
    "maxBytes": 13000,
    "extraAgent": "kept"
  },
  "extraTop": {
    "kept": true
  }
}`)
	loaded := Load(root)
	if !loaded.Valid {
		t.Fatalf("expected valid config, got %#v", loaded.Warnings)
	}

	cfg := loaded.Config
	cfg.BaseRef = "HEAD"
	cfg.AgentContext.MaxFiles = 21
	path, err := Save(root, cfg, &loaded)
	if err != nil {
		t.Fatal(err)
	}
	if path != Path(root) {
		t.Fatalf("expected path %s, got %s", Path(root), path)
	}

	var saved map[string]any
	data, err := os.ReadFile(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["extraTop"] == nil {
		t.Fatalf("expected unknown top-level field to be preserved: %#v", saved)
	}
	agentContext, ok := saved["agentContext"].(map[string]any)
	if !ok {
		t.Fatalf("expected agentContext object, got %#v", saved["agentContext"])
	}
	if agentContext["extraAgent"] != "kept" {
		t.Fatalf("expected unknown agentContext field to be preserved: %#v", agentContext)
	}
	if saved["baseRef"] != "HEAD" || agentContext["maxFiles"] != float64(21) {
		t.Fatalf("expected known fields to update, got %#v", saved)
	}
}

func TestSaveReplacesInvalidConfigWithoutPreservingUnknownFields(t *testing.T) {
	root := t.TempDir()
	writeConfigTestFile(t, Path(root), `{`)
	loaded := Load(root)

	_, err := Save(root, WithBase("HEAD"), &loaded)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("expected valid JSON, got %s", data)
	}
	if strings.Contains(string(data), "extraTop") {
		t.Fatalf("expected invalid config to be replaced, got %s", data)
	}
}

func writeConfigTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertConfigTestStrings(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %#v, got %#v", want, got)
		}
	}
}
