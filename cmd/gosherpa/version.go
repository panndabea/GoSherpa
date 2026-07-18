package main

import (
	"fmt"
	"io"
	"runtime"
)

var version = "v0.8.0"

type versionJSONData struct {
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
}

func runVersionCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printCommandUsage(stderr, versionUsageLine)
		return exitUsage
	}

	info := currentVersionInfo()
	if invocation.JSON {
		return writeJSON(stdout, stderr, jsonResponse[versionJSONData]{
			SchemaVersion: jsonSchemaVersion,
			Command:       "version",
			Target:        "",
			Root:          "",
			ModulePath:    "",
			Warnings:      []string{},
			Data:          info,
		})
	}

	fmt.Fprintf(stdout, "GoSherpa %s\n", info.Version)
	fmt.Fprintf(stdout, "go: %s\n", info.GoVersion)
	return exitSuccess
}

func currentVersionInfo() versionJSONData {
	return versionJSONData{
		Version:   version,
		GoVersion: runtime.Version(),
	}
}
