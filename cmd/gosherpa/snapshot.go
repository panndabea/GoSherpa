package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

type snapshotCommandData struct {
	Status   string                 `json:"status"`
	Path     string                 `json:"path"`
	Snapshot snapshotstore.Snapshot `json:"snapshot"`
}

func runSnapshotCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printCommandUsage(stderr, snapshotUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	builtSnapshot, err := snapshotstore.Build(root, snapshotstore.BuildOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "snapshot", ".", stderr, err)
	}

	path, err := snapshotstore.Write(root, builtSnapshot)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "snapshot", ".", stderr, err)
	}
	displayPath := snapshotDisplayPath(root, path)

	if invocation.JSON {
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"snapshot",
			".",
			nil,
			snapshotCommandData{
				Status:   snapshotstore.StatusValid,
				Path:     displayPath,
				Snapshot: builtSnapshot,
			},
		))
	}

	fmt.Fprint(stdout, formatSnapshotCommand(builtSnapshot, displayPath))
	return exitSuccess
}

func formatSnapshotCommand(snapshot snapshotstore.Snapshot, path string) string {
	var builder strings.Builder
	builder.WriteString("SNAPSHOT\n\n")
	fmt.Fprintf(&builder, "Status: %s\n", snapshotstore.StatusValid)
	fmt.Fprintf(&builder, "Path: %s\n", path)
	fmt.Fprintf(&builder, "Module: %s\n", valueOrNone(snapshot.ModulePath))
	fmt.Fprintf(&builder, "Created: %s\n", valueOrNone(snapshot.CreatedAt))
	fmt.Fprintf(&builder, "Format: %d\n", snapshot.FormatVersion)
	fmt.Fprintf(&builder, "Fingerprint: %s\n", snapshot.Fingerprint)
	builder.WriteString("\n")

	builder.WriteString("INPUTS\n")
	fmt.Fprintf(&builder, "  Go: %s %s/%s\n", snapshot.GoVersion, snapshot.GOOS, snapshot.GOARCH)
	writeDoctorValues(&builder, "  Build tags", snapshot.BuildTags)
	if strings.TrimSpace(snapshot.GitState) != "" {
		fmt.Fprintf(&builder, "  Git state: %s\n", snapshot.GitState)
	} else {
		builder.WriteString("  Git state: none\n")
	}
	builder.WriteString("\n")

	builder.WriteString("CONTENTS\n")
	fmt.Fprintf(&builder, "  Files: %d\n", len(snapshot.Files))
	fmt.Fprintf(&builder, "  Packages: %d\n", len(snapshot.Packages))
	fmt.Fprintf(&builder, "  Symbols: %d\n", len(snapshot.Symbols))

	return builder.String()
}

func snapshotDisplayPath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}

	return filepath.Clean(path)
}
