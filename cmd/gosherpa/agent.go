package main

import (
	"fmt"
	"io"

	agentworkflow "github.com/panndabea/GoSherpa/internal/agentworkflow"
)

func runAgentCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) == 0 {
		printAgentUsage(stderr)
		return exitUsage
	}

	switch invocation.CommandArgs[0] {
	case "context":
		if len(invocation.CommandArgs) != 1 {
			printAgentContextUsage(stderr)
			return exitUsage
		}
		if !hasBaseRef(invocation) {
			printAgentContextUsage(stderr)
			printConfigBaseUsageHint(stderr)
			return exitUsage
		}

		root, ok := resolveInvocationRootPath(invocation, stderr)
		if !ok {
			return exitFailure
		}

		report, err := agentworkflow.AnalyzeContext(root, invocation.BaseRef, agentworkflow.AnalyzeOptions{
			BuildTags:   invocation.BuildTags,
			UseSnapshot: invocation.UseSnapshot,
			Limits:      invocation.ContextLimits,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "agent context", invocation.BaseRef, stderr, err)
		}
		report.Warnings = uniqueStringsInOrder(append(invocation.ConfigWarnings, report.Warnings...))

		if invocation.JSON {
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"agent context",
				report.Target,
				report.Warnings,
				report,
			))
		}

		fmt.Fprint(stdout, agentworkflow.Format(report))
		return exitSuccess
	default:
		printAgentUsage(stderr)
		return exitUsage
	}
}

func printAgentUsage(writer io.Writer) {
	printUsageLines(writer, agentUsageLines)
}

func printAgentContextUsage(writer io.Writer) {
	printCommandUsage(writer, agentContextUsageLine)
}
