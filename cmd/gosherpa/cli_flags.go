package main

import (
	"fmt"
	"strconv"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type cliFlagSpec struct {
	Name       string
	TakesValue bool
	Apply      func(*cliInvocation, string, bool) error
}

var cliFlagSpecs = []cliFlagSpec{
	{
		Name: "--json",
		Apply: func(invocation *cliInvocation, _ string, _ bool) error {
			invocation.JSON = true
			return nil
		},
	},
	{
		Name: "--tests",
		Apply: func(invocation *cliInvocation, _ string, _ bool) error {
			invocation.IncludeTests = true
			invocation.HasTestsOption = true
			return nil
		},
	},
	{
		Name: "--all",
		Apply: func(invocation *cliInvocation, _ string, _ bool) error {
			invocation.All = true
			invocation.HasAllOption = true
			return nil
		},
	},
	{
		Name: "--context",
		Apply: func(invocation *cliInvocation, _ string, _ bool) error {
			invocation.ShowContext = true
			invocation.HasContextOption = true
			return nil
		},
	},
	{
		Name:       "--tags",
		TakesValue: true,
		Apply:      applyBuildTagsFlag,
	},
	{
		Name:       "--root",
		TakesValue: true,
		Apply:      applyRootFlag,
	},
	{
		Name:       "--base",
		TakesValue: true,
		Apply: applyStringCLIFlag("--base", func(invocation *cliInvocation, value string) {
			invocation.BaseRef = value
			invocation.HasBaseOption = true
		}),
	},
	{
		Name:       "--kind",
		TakesValue: true,
		Apply: applyStringCLIFlag("--kind", func(invocation *cliInvocation, value string) {
			invocation.KindFilter = value
			invocation.HasKindOption = true
		}),
	},
	{
		Name:       "--scope",
		TakesValue: true,
		Apply:      applyScopeFlag,
	},
	{
		Name:       "--package",
		TakesValue: true,
		Apply: applyStringCLIFlag("--package", func(invocation *cliInvocation, value string) {
			invocation.SearchPackage = value
			invocation.HasPackageOption = true
		}),
	},
	{
		Name:       "--max-files",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--max-files", func(invocation *cliInvocation, value int) {
			invocation.ContextLimits.MaxFiles = value
			invocation.HasContextLimit = true
		}),
	},
	{
		Name:       "--max-references",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--max-references", func(invocation *cliInvocation, value int) {
			invocation.ContextLimits.MaxReferences = value
			invocation.HasContextLimit = true
		}),
	},
	{
		Name:       "--max-symbols",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--max-symbols", func(invocation *cliInvocation, value int) {
			invocation.ContextLimits.MaxSymbols = value
			invocation.HasContextLimit = true
		}),
	},
	{
		Name:       "--max-tests",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--max-tests", func(invocation *cliInvocation, value int) {
			invocation.ContextLimits.MaxTests = value
			invocation.HasContextLimit = true
		}),
	},
	{
		Name:       "--max-bytes",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--max-bytes", func(invocation *cliInvocation, value int) {
			invocation.ContextLimits.MaxBytes = value
			invocation.HasContextLimit = true
		}),
	},
	{
		Name:       "--source-radius",
		TakesValue: true,
		Apply:      applySourceRadiusFlag,
	},
	{
		Name:       "--limit",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--limit", func(invocation *cliInvocation, value int) {
			invocation.CallPathLimit = value
			invocation.HasLimitOption = true
			invocation.HasCallPathOption = true
		}),
	},
	{
		Name:       "--max-depth",
		TakesValue: true,
		Apply: applyPositiveCLIFlag("--max-depth", func(invocation *cliInvocation, value int) {
			invocation.CallPathMaxDepth = value
			invocation.HasCallPathOption = true
			invocation.HasMaxDepthOption = true
		}),
	},
}

var cliFlagSpecIndex = indexCLIFlagSpecs(cliFlagSpecs)

func parseCLIArgsWithFlagSpecs(args []string) (cliInvocation, error) {
	invocation := cliInvocation{Root: "."}
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		flagName, inlineValue, hasInlineValue := splitCLIFlagArg(arg)

		if spec, ok := cliFlagSpecFor(flagName); ok {
			if !spec.TakesValue && hasInlineValue {
				return cliInvocation{}, fmt.Errorf("unknown flag: %s", arg)
			}

			value := inlineValue
			consumeNext := false
			if spec.TakesValue && !hasInlineValue {
				if i+1 >= len(args) {
					return cliInvocation{}, fmt.Errorf("missing value for %s", spec.Name)
				}
				value = args[i+1]
				consumeNext = true
			}

			if err := spec.Apply(&invocation, value, hasInlineValue); err != nil {
				return cliInvocation{}, err
			}

			if consumeNext {
				i++
			}
			continue
		}

		if strings.HasPrefix(arg, "-") {
			return cliInvocation{}, fmt.Errorf("unknown flag: %s", arg)
		}

		positionals = append(positionals, arg)
	}

	if len(positionals) > 0 {
		invocation.Command = positionals[0]
	}

	if len(positionals) > 1 {
		invocation.CommandArgs = positionals[1:]
	}

	if invocation.HasKindOption {
		if err := parseKindFilter(&invocation); err != nil {
			return cliInvocation{}, err
		}
	}

	return invocation, nil
}

func parseStringFlag(flag string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return "", fmt.Errorf("missing value for %s", flag)
	}

	return trimmed, nil
}

func parseBuildTagsFlag(flag string, value string) ([]string, error) {
	tags := semantics.NormalizeBuildTags([]string{value})
	if len(tags) == 0 {
		return nil, fmt.Errorf("missing value for %s", flag)
	}

	return tags, nil
}

func parsePositiveInteger(flag string, value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid value for %s: %s", flag, trimmed)
	}

	return parsed, nil
}

func parseNonNegativeInteger(flag string, value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid value for %s: %s", flag, trimmed)
	}

	return parsed, nil
}

func parseKindFilter(invocation *cliInvocation) error {
	switch invocation.Command {
	case "search", "symbols":
		kind, err := parseSymbolKindFlag("--kind", invocation.KindFilter)
		if err != nil {
			return err
		}
		invocation.SearchKind = kind
	case "refs":
		kind, err := parseReferenceKindFlag("--kind", invocation.KindFilter)
		if err != nil {
			return err
		}
		invocation.ReferenceKind = kind
	}

	return nil
}

func parseSymbolKindFlag(flag string, value string) (sherpa.SymbolKind, error) {
	trimmed, err := parseStringFlag(flag, value)
	if err != nil {
		return "", err
	}

	kind := sherpa.SymbolKind(strings.ToLower(trimmed))
	if isSupportedSearchKind(kind) {
		return kind, nil
	}

	return "", fmt.Errorf("invalid value for %s: %s", flag, trimmed)
}

func isSupportedSearchKind(kind sherpa.SymbolKind) bool {
	switch kind {
	case sherpa.SymbolKindStruct, sherpa.SymbolKindInterface, sherpa.SymbolKindFunction, sherpa.SymbolKindMethod:
		return true
	default:
		return false
	}
}

func parseReferenceKindFlag(flag string, value string) (sherpa.ReferenceKind, error) {
	trimmed, err := parseStringFlag(flag, value)
	if err != nil {
		return "", err
	}

	if kind, ok := sherpa.ParseReferenceKind(trimmed); ok {
		return kind, nil
	}

	return "", fmt.Errorf("invalid value for %s: %s", flag, trimmed)
}

func parseTestScopeFlag(flag string, value string) (sherpa.TestScope, error) {
	trimmed, err := parseStringFlag(flag, value)
	if err != nil {
		return "", err
	}

	if scope, ok := sherpa.ParseTestScope(trimmed); ok {
		return scope, nil
	}

	return "", fmt.Errorf("invalid value for %s: %s", flag, trimmed)
}

func indexCLIFlagSpecs(specs []cliFlagSpec) map[string]cliFlagSpec {
	index := make(map[string]cliFlagSpec, len(specs))
	for _, spec := range specs {
		index[spec.Name] = spec
	}

	return index
}

func cliFlagSpecFor(name string) (cliFlagSpec, bool) {
	spec, ok := cliFlagSpecIndex[name]
	return spec, ok
}

func splitCLIFlagArg(arg string) (string, string, bool) {
	if !strings.HasPrefix(arg, "--") {
		return arg, "", false
	}

	name, value, ok := strings.Cut(arg, "=")
	return name, value, ok
}

func applyRootFlag(invocation *cliInvocation, value string, _ bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("missing value for --root")
	}

	invocation.Root = value
	return nil
}

func applyBuildTagsFlag(invocation *cliInvocation, value string, inline bool) error {
	if !inline {
		parsed, err := parseStringFlag("--tags", value)
		if err != nil {
			return err
		}
		value = parsed
	}

	tags, err := parseBuildTagsFlag("--tags", value)
	if err != nil {
		return err
	}

	invocation.BuildTags = tags
	invocation.HasTagsOption = true
	return nil
}

func applyStringCLIFlag(flag string, assign func(*cliInvocation, string)) func(*cliInvocation, string, bool) error {
	return func(invocation *cliInvocation, value string, _ bool) error {
		parsed, err := parseStringFlag(flag, value)
		if err != nil {
			return err
		}

		assign(invocation, parsed)
		return nil
	}
}

func applyScopeFlag(invocation *cliInvocation, value string, _ bool) error {
	scope, err := parseTestScopeFlag("--scope", value)
	if err != nil {
		return err
	}

	invocation.TestScope = scope
	invocation.HasTestScopeOption = true
	return nil
}

func applyPositiveCLIFlag(flag string, assign func(*cliInvocation, int)) func(*cliInvocation, string, bool) error {
	return func(invocation *cliInvocation, value string, _ bool) error {
		parsed, err := parsePositiveInteger(flag, value)
		if err != nil {
			return err
		}

		assign(invocation, parsed)
		return nil
	}
}

func applySourceRadiusFlag(invocation *cliInvocation, value string, _ bool) error {
	parsed, err := parseNonNegativeInteger("--source-radius", value)
	if err != nil {
		return err
	}

	invocation.ContextLimits.SourceRadius = agentcontext.NewSourceRadius(parsed)
	invocation.HasContextLimit = true
	invocation.HasSourceRadius = true
	return nil
}
