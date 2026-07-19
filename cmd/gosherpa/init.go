package main

import (
	"fmt"
	"io"
	"strings"

	sherpaconfig "github.com/panndabea/GoSherpa/internal/config"
	gitrepo "github.com/panndabea/GoSherpa/internal/git"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

type initCommandData struct {
	ConfigPath      string                `json:"configPath"`
	ConfigWritten   bool                  `json:"configWritten"`
	SnapshotWritten bool                  `json:"snapshotWritten"`
	Config          sherpaconfig.Config   `json:"config"`
	BaseDetection   gitrepo.BaseDetection `json:"baseDetection"`
	Snapshot        initSnapshotData      `json:"snapshot"`
	NextCommands    []string              `json:"nextCommands"`
}

type initSnapshotData struct {
	Status  string                `json:"status"`
	Path    string                `json:"path"`
	Summary snapshotstore.Summary `json:"summary"`
}

func runInitCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printCommandUsage(stderr, initUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	loadedConfig := sherpaconfig.Load(root)
	cfg := sherpaconfig.Default()
	if loadedConfig.Exists && loadedConfig.Valid {
		cfg = loadedConfig.Config
	}

	baseDetection, err := initBaseDetection(root, invocation, loadedConfig)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "init", ".", stderr, err)
	}
	cfg.BaseRef = baseDetection.Selected
	if invocation.HasTagsOption {
		cfg.BuildTags = append([]string{}, invocation.BuildTags...)
	}
	if invocation.HasMaxFilesOption {
		cfg.AgentContext.MaxFiles = invocation.ContextLimits.MaxFiles
	}
	if invocation.HasMaxSymbolsOption {
		cfg.AgentContext.MaxSymbols = invocation.ContextLimits.MaxSymbols
	}
	if invocation.HasMaxTestsOption {
		cfg.AgentContext.MaxTests = invocation.ContextLimits.MaxTests
	}
	if invocation.HasMaxBytesOption {
		cfg.AgentContext.MaxBytes = invocation.ContextLimits.MaxBytes
	}
	cfg = sherpaconfig.Normalize(cfg)

	builtSnapshot, err := snapshotstore.Build(root, snapshotstore.BuildOptions{
		BuildTags: cfg.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "init", ".", stderr, err)
	}

	configPath, err := sherpaconfig.Save(root, cfg, &loadedConfig)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "init", ".", stderr, err)
	}
	snapshotPath, err := snapshotstore.Write(root, builtSnapshot)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "init", ".", stderr, err)
	}

	configDisplayPath := snapshotDisplayPath(root, configPath)
	snapshotDisplay := snapshotDisplayPath(root, snapshotPath)
	warnings := initWarnings(loadedConfig, baseDetection)
	data := initCommandData{
		ConfigPath:      configDisplayPath,
		ConfigWritten:   true,
		SnapshotWritten: true,
		Config:          cfg,
		BaseDetection:   baseDetection,
		Snapshot: initSnapshotData{
			Status:  snapshotstore.StatusValid,
			Path:    snapshotDisplay,
			Summary: snapshotstore.Summarize(builtSnapshot),
		},
		NextCommands: initNextCommands(),
	}

	if invocation.JSON {
		return writeJSON(stdout, stderr, newJSONResponse(root, "init", ".", warnings, data))
	}

	fmt.Fprint(stdout, formatInitCommand(root, data, warnings))
	return exitSuccess
}

func initBaseDetection(root string, invocation cliInvocation, loadedConfig sherpaconfig.LoadResult) (gitrepo.BaseDetection, error) {
	if invocation.HasBaseOption {
		return gitrepo.DetectBase(root, invocation.BaseRef)
	}

	if loadedConfig.Exists && loadedConfig.Valid && strings.TrimSpace(loadedConfig.Config.BaseRef) != "" {
		detected, err := gitrepo.DetectBase(root, loadedConfig.Config.BaseRef)
		if err == nil {
			return detected, nil
		}

		fallback, fallbackErr := gitrepo.DetectBase(root, "")
		if fallbackErr != nil {
			return fallback, fallbackErr
		}
		fallback.Warnings = append([]string{
			fmt.Sprintf("configured base ref %q could not be resolved locally; selected a new default base", loadedConfig.Config.BaseRef),
		}, fallback.Warnings...)
		return fallback, nil
	}

	return gitrepo.DetectBase(root, "")
}

func initWarnings(loadedConfig sherpaconfig.LoadResult, baseDetection gitrepo.BaseDetection) []string {
	var warnings []string
	warnings = append(warnings, loadedConfig.Warnings...)
	if loadedConfig.Exists && !loadedConfig.Valid {
		warnings = append(warnings, "invalid gosherpa config was replaced with a normalized config")
	}
	warnings = append(warnings, baseDetection.Warnings...)
	return uniqueStringsInOrder(warnings)
}

func initNextCommands() []string {
	return []string{
		"gosherpa doctor",
		"gosherpa agent context",
		"gosherpa pr",
	}
}

func formatInitCommand(root string, data initCommandData, warnings []string) string {
	var builder strings.Builder
	builder.WriteString("INIT\n\n")
	fmt.Fprintf(&builder, "Repository: %s\n", root)
	fmt.Fprintf(&builder, "Config: %s (written)\n", data.ConfigPath)
	fmt.Fprintf(&builder, "Base: %s (%s)\n", data.Config.BaseRef, valueOrNone(data.BaseDetection.Source))
	fmt.Fprintf(&builder, "Snapshot: refreshed (%s)\n", data.Snapshot.Path)
	builder.WriteString("\n")

	builder.WriteString("Defaults\n")
	fmt.Fprintf(&builder, "  use snapshot: %t\n", data.Config.UseSnapshot)
	writeInitValues(&builder, "  build tags", data.Config.BuildTags)
	fmt.Fprintf(&builder, "  max files: %d\n", data.Config.AgentContext.MaxFiles)
	fmt.Fprintf(&builder, "  max symbols: %d\n", data.Config.AgentContext.MaxSymbols)
	fmt.Fprintf(&builder, "  max tests: %d\n", data.Config.AgentContext.MaxTests)
	fmt.Fprintf(&builder, "  max bytes: %d\n", data.Config.AgentContext.MaxBytes)
	builder.WriteString("\n")

	if len(warnings) > 0 {
		builder.WriteString("WARNINGS\n")
		for _, warning := range warnings {
			fmt.Fprintf(&builder, "  %s\n", warning)
		}
		builder.WriteString("\n")
	}

	builder.WriteString("NEXT COMMANDS\n")
	for _, command := range data.NextCommands {
		fmt.Fprintf(&builder, "  %s\n", command)
	}

	return builder.String()
}

func writeInitValues(builder *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(builder, "%s: none\n", label)
		return
	}
	fmt.Fprintf(builder, "%s: %s\n", label, strings.Join(values, ", "))
}
