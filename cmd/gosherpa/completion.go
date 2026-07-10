package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type completionFlag struct {
	Name        string
	Description string
	TakesValue  bool
	ValueName   string
	Values      []string
}

type completionShell struct {
	Name        string
	Description string
}

var completionFlagDefinitions = map[string]completionFlag{
	"--root": {
		Name:        "--root",
		Description: "repository root",
		TakesValue:  true,
		ValueName:   "path",
	},
	"--tags": {
		Name:        "--tags",
		Description: "build tags for semantic package loading",
		TakesValue:  true,
		ValueName:   "tags",
	},
	"--json": {
		Name:        "--json",
		Description: "machine-readable output",
	},
	"--context": {
		Name:        "--context",
		Description: "show source context for human output",
	},
	"--version": {
		Name:        "--version",
		Description: "print GoSherpa version information",
	},
	"--tests": {
		Name:        "--tests",
		Description: "include test files where supported",
	},
	"--all": {
		Name:        "--all",
		Description: "include all packages",
	},
	"--use-snapshot": {
		Name:        "--use-snapshot",
		Description: "reuse a valid repository snapshot",
	},
	"--base": {
		Name:        "--base",
		Description: "git base reference",
		TakesValue:  true,
		ValueName:   "ref",
	},
	"--kind": {
		Name:        "--kind",
		Description: "filter by symbol or reference kind",
		TakesValue:  true,
		ValueName:   "kind",
		Values:      []string{"struct", "interface", "alias", "function", "method", "definition", "call", "type_usage", "field_access", "usage"},
	},
	"--scope": {
		Name:        "--scope",
		Description: "filter test scope",
		TakesValue:  true,
		ValueName:   "scope",
		Values:      []string{"direct", "related", "all"},
	},
	"--package": {
		Name:        "--package",
		Description: "filter by package",
		TakesValue:  true,
		ValueName:   "package",
	},
	"--max-files": {
		Name:        "--max-files",
		Description: "limit context files",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--max-references": {
		Name:        "--max-references",
		Description: "limit context references",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--max-symbols": {
		Name:        "--max-symbols",
		Description: "limit context symbols",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--max-tests": {
		Name:        "--max-tests",
		Description: "limit context tests",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--max-bytes": {
		Name:        "--max-bytes",
		Description: "limit context byte budget",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--source-radius": {
		Name:        "--source-radius",
		Description: "source lines around context targets",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--limit": {
		Name:        "--limit",
		Description: "limit results",
		TakesValue:  true,
		ValueName:   "n",
	},
	"--max-depth": {
		Name:        "--max-depth",
		Description: "limit path search depth",
		TakesValue:  true,
		ValueName:   "n",
	},
}

var completionShells = []completionShell{
	{Name: "zsh", Description: "zsh completion script"},
	{Name: "bash", Description: "bash completion script"},
	{Name: "fish", Description: "fish completion script"},
}

var completionTopLevelFlags = []string{"--root", "--tags", "--json", "--context", "--version"}

func runCompletionCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 1 {
		printCommandUsage(stderr, completionUsageLine)
		return exitUsage
	}

	switch strings.ToLower(invocation.CommandArgs[0]) {
	case "zsh":
		fmt.Fprint(stdout, formatZshCompletion())
	case "bash":
		fmt.Fprint(stdout, formatBashCompletion())
	case "fish":
		fmt.Fprint(stdout, formatFishCompletion())
	default:
		fmt.Fprintf(stderr, "error: unsupported completion shell: %s\n", invocation.CommandArgs[0])
		printCommandUsage(stderr, completionUsageLine)
		return exitUsage
	}

	return exitSuccess
}

func formatZshCompletion() string {
	var builder strings.Builder

	builder.WriteString("#compdef gosherpa\n\n")
	builder.WriteString("_gosherpa() {\n")
	writeZshArray(&builder, "commands", zshCommandEntries())
	writeZshArray(&builder, "shells", zshShellEntries())
	writeZshArray(&builder, "context_subcommands", []string{
		"symbol:context for one symbol",
		"file:context for one file",
		"package:context for one package",
		"diff:context for a git diff",
	})
	writeZshArray(&builder, "impact_subcommands", []string{
		"file:impact for one file",
		"package:impact for one package",
		"symbol:impact for one symbol",
		"diff:impact for a git diff",
	})
	writeZshArray(&builder, "test_subcommands", []string{
		"affected:tests affected by a git diff",
	})
	builder.WriteString("  local command=\"\"\n")
	builder.WriteString("  local skip_next=0\n")
	builder.WriteString("  local index=1\n")
	builder.WriteString("  local word\n")
	builder.WriteString("  local value_flags=\"")
	builder.WriteString(strings.Join(completionValueFlagNames(), " "))
	builder.WriteString(" \"\n")
	builder.WriteString("  for word in \"${words[@]}\"; do\n")
	builder.WriteString("    if (( index == CURRENT )); then\n")
	builder.WriteString("      break\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if (( index == 1 )); then\n")
	builder.WriteString("      (( index++ ))\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if (( skip_next )); then\n")
	builder.WriteString("      skip_next=0\n")
	builder.WriteString("      (( index++ ))\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if [[ \"$word\" == --*=* ]]; then\n")
	builder.WriteString("      (( index++ ))\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if [[ \"$word\" == --* ]]; then\n")
	builder.WriteString("      case \" $value_flags\" in\n")
	builder.WriteString("        *\" $word \"*) skip_next=1 ;;\n")
	builder.WriteString("      esac\n")
	builder.WriteString("      (( index++ ))\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    command=\"$word\"\n")
	builder.WriteString("    break\n")
	builder.WriteString("  done\n\n")
	builder.WriteString("  if [[ -z \"$command\" ]]; then\n")
	builder.WriteString("    _describe -t commands 'gosherpa command' commands\n")
	builder.WriteString("    _values 'global options' \\\n")
	builder.WriteString(zshFlagValueLines(flagsByNames(completionTopLevelFlags), "      "))
	builder.WriteString("    return\n")
	builder.WriteString("  fi\n\n")
	builder.WriteString("  case \"$command\" in\n")
	builder.WriteString("    completion)\n")
	builder.WriteString("      _describe -t shells 'shell' shells\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    context)\n")
	builder.WriteString("      if [[ \"${words[CURRENT-1]}\" == \"context\" ]]; then\n")
	builder.WriteString("        _describe -t context-subcommands 'context target' context_subcommands\n")
	builder.WriteString("        return\n")
	builder.WriteString("      fi\n")
	builder.WriteString("      _values 'context options' \\\n")
	builder.WriteString(zshFlagValueLines(completionFlagsForCommandName("context"), "        "))
	builder.WriteString("      ;;\n")
	builder.WriteString("    impact)\n")
	builder.WriteString("      if [[ \"${words[CURRENT-1]}\" == \"impact\" ]]; then\n")
	builder.WriteString("        _describe -t impact-subcommands 'impact target' impact_subcommands\n")
	builder.WriteString("        return\n")
	builder.WriteString("      fi\n")
	builder.WriteString("      _values 'impact options' \\\n")
	builder.WriteString(zshFlagValueLines(completionFlagsForCommandName("impact"), "        "))
	builder.WriteString("      ;;\n")
	builder.WriteString("    tests)\n")
	builder.WriteString("      if [[ \"${words[CURRENT-1]}\" == \"tests\" ]]; then\n")
	builder.WriteString("        _describe -t test-subcommands 'tests target' test_subcommands\n")
	builder.WriteString("      fi\n")
	builder.WriteString("      _values 'tests options' \\\n")
	builder.WriteString(zshFlagValueLines(completionFlagsForCommandName("tests"), "        "))
	builder.WriteString("      ;;\n")

	for _, spec := range commandSpecs {
		switch spec.Name {
		case "completion", "context", "impact", "tests":
			continue
		}
		flags := completionFlagsForSpec(spec)
		if len(flags) == 0 {
			continue
		}
		fmt.Fprintf(&builder, "    %s)\n", spec.Name)
		fmt.Fprintf(&builder, "      _values '%s options' \\\n", spec.Name)
		builder.WriteString(zshFlagValueLines(flags, "        "))
		builder.WriteString("      ;;\n")
	}

	builder.WriteString("  esac\n")
	builder.WriteString("}\n\n")
	builder.WriteString("_gosherpa \"$@\"\n")

	return builder.String()
}

func formatBashCompletion() string {
	var builder strings.Builder

	builder.WriteString("_gosherpa()\n")
	builder.WriteString("{\n")
	builder.WriteString("  local cur prev command flags\n")
	builder.WriteString("  COMPREPLY=()\n")
	builder.WriteString("  cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	builder.WriteString("  prev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n\n")
	builder.WriteString("  case \"$prev\" in\n")
	builder.WriteString("    --root)\n")
	builder.WriteString("      COMPREPLY=( $(compgen -d -- \"$cur\") )\n")
	builder.WriteString("      return\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    --kind)\n")
	builder.WriteString("      command=$(__gosherpa_command)\n")
	builder.WriteString("      case \"$command\" in\n")
	builder.WriteString("        search|symbols) COMPREPLY=( $(compgen -W \"struct interface alias function method\" -- \"$cur\") ) ;;\n")
	builder.WriteString("        refs) COMPREPLY=( $(compgen -W \"definition call type_usage field_access usage\" -- \"$cur\") ) ;;\n")
	builder.WriteString("      esac\n")
	builder.WriteString("      return\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    --scope)\n")
	builder.WriteString("      COMPREPLY=( $(compgen -W \"direct related all\" -- \"$cur\") )\n")
	builder.WriteString("      return\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    --tags|--base|--package|--max-files|--max-references|--max-symbols|--max-tests|--max-bytes|--source-radius|--limit|--max-depth)\n")
	builder.WriteString("      return\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("  esac\n\n")
	builder.WriteString("  command=$(__gosherpa_command)\n")
	builder.WriteString("  if [[ -z \"$command\" ]]; then\n")
	fmt.Fprintf(&builder, "    COMPREPLY=( $(compgen -W %s -- \"$cur\") )\n", shellSingleQuote(strings.Join(append(completionCommandNames(), completionTopLevelFlags...), " ")))
	builder.WriteString("    return\n")
	builder.WriteString("  fi\n\n")
	builder.WriteString("  if [[ \"$cur\" == --* ]]; then\n")
	builder.WriteString("    flags=$(__gosherpa_flags \"$command\")\n")
	builder.WriteString("    COMPREPLY=( $(compgen -W \"$flags\" -- \"$cur\") )\n")
	builder.WriteString("    return\n")
	builder.WriteString("  fi\n\n")
	builder.WriteString("  case \"$command\" in\n")
	builder.WriteString("    completion)\n")
	builder.WriteString("      COMPREPLY=( $(compgen -W \"zsh bash fish\" -- \"$cur\") )\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    context)\n")
	builder.WriteString("      if ! __gosherpa_seen_any \"symbol file package diff\"; then\n")
	builder.WriteString("        COMPREPLY=( $(compgen -W \"symbol file package diff\" -- \"$cur\") )\n")
	builder.WriteString("      fi\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    impact)\n")
	builder.WriteString("      if ! __gosherpa_seen_any \"file package symbol diff\"; then\n")
	builder.WriteString("        COMPREPLY=( $(compgen -W \"file package symbol diff\" -- \"$cur\") )\n")
	builder.WriteString("      fi\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("    tests)\n")
	builder.WriteString("      if ! __gosherpa_seen_any \"affected\"; then\n")
	builder.WriteString("        COMPREPLY=( $(compgen -W \"affected\" -- \"$cur\") )\n")
	builder.WriteString("      fi\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("  esac\n")
	builder.WriteString("}\n\n")
	builder.WriteString("__gosherpa_command()\n")
	builder.WriteString("{\n")
	builder.WriteString("  local i word skip_next\n")
	builder.WriteString("  skip_next=0\n")
	builder.WriteString("  for (( i=1; i<COMP_CWORD; i++ )); do\n")
	builder.WriteString("    word=\"${COMP_WORDS[i]}\"\n")
	builder.WriteString("    if [[ $skip_next -eq 1 ]]; then\n")
	builder.WriteString("      skip_next=0\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if [[ \"$word\" == --*=* ]]; then\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    case \"$word\" in\n")
	builder.WriteString("      --root|--tags|--base|--kind|--scope|--package|--max-files|--max-references|--max-symbols|--max-tests|--max-bytes|--source-radius|--limit|--max-depth)\n")
	builder.WriteString("        skip_next=1\n")
	builder.WriteString("        continue\n")
	builder.WriteString("        ;;\n")
	builder.WriteString("      --*)\n")
	builder.WriteString("        continue\n")
	builder.WriteString("        ;;\n")
	builder.WriteString("    esac\n")
	builder.WriteString("    printf '%s\\n' \"$word\"\n")
	builder.WriteString("    return\n")
	builder.WriteString("  done\n")
	builder.WriteString("}\n\n")
	builder.WriteString("__gosherpa_seen_any()\n")
	builder.WriteString("{\n")
	builder.WriteString("  local choices word choice\n")
	builder.WriteString("  choices=\"$1\"\n")
	builder.WriteString("  for word in \"${COMP_WORDS[@]:1:COMP_CWORD-1}\"; do\n")
	builder.WriteString("    for choice in $choices; do\n")
	builder.WriteString("      if [[ \"$word\" == \"$choice\" ]]; then\n")
	builder.WriteString("        return 0\n")
	builder.WriteString("      fi\n")
	builder.WriteString("    done\n")
	builder.WriteString("  done\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("}\n\n")
	builder.WriteString("__gosherpa_flags()\n")
	builder.WriteString("{\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, spec := range commandSpecs {
		flags := completionFlagsForSpec(spec)
		fmt.Fprintf(&builder, "    %s)\n", spec.Name)
		fmt.Fprintf(&builder, "      printf '%%s\\n' %s\n", shellSingleQuote(strings.Join(completionFlagNames(flags), " ")))
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("  esac\n")
	builder.WriteString("}\n\n")
	builder.WriteString("complete -F _gosherpa gosherpa\n")

	return builder.String()
}

func formatFishCompletion() string {
	var builder strings.Builder

	builder.WriteString("function __fish_gosherpa_seen_command\n")
	builder.WriteString("    set -l words (commandline -opc)\n")
	builder.WriteString("    set -e words[1]\n")
	builder.WriteString("    for word in $words\n")
	builder.WriteString("        switch $word\n")
	builder.WriteString("            case ")
	builder.WriteString(strings.Join(completionCommandNames(), " "))
	builder.WriteString("\n")
	builder.WriteString("                return 0\n")
	builder.WriteString("        end\n")
	builder.WriteString("    end\n")
	builder.WriteString("    return 1\n")
	builder.WriteString("end\n\n")

	for _, spec := range commandSpecs {
		fmt.Fprintf(
			&builder,
			"complete -c gosherpa -f -n 'not __fish_gosherpa_seen_command' -a %s -d %s\n",
			shellSingleQuote(spec.Name),
			shellSingleQuote(completionCommandDescription(spec)),
		)
	}
	builder.WriteString("\n")

	for _, shell := range completionShells {
		fmt.Fprintf(
			&builder,
			"complete -c gosherpa -f -n '__fish_seen_subcommand_from completion' -a %s -d %s\n",
			shellSingleQuote(shell.Name),
			shellSingleQuote(shell.Description),
		)
	}
	builder.WriteString("\n")

	for _, subcommand := range []struct {
		Command     string
		Subcommand  string
		Description string
	}{
		{"context", "symbol", "context for one symbol"},
		{"context", "file", "context for one file"},
		{"context", "package", "context for one package"},
		{"context", "diff", "context for a git diff"},
		{"impact", "file", "impact for one file"},
		{"impact", "package", "impact for one package"},
		{"impact", "symbol", "impact for one symbol"},
		{"impact", "diff", "impact for a git diff"},
		{"tests", "affected", "tests affected by a git diff"},
	} {
		fmt.Fprintf(
			&builder,
			"complete -c gosherpa -f -n '__fish_seen_subcommand_from %s' -a %s -d %s\n",
			subcommand.Command,
			shellSingleQuote(subcommand.Subcommand),
			shellSingleQuote(subcommand.Description),
		)
	}
	builder.WriteString("\n")

	for _, flag := range flagsByNames([]string{"--root", "--version"}) {
		writeFishFlagLine(&builder, "", flag)
	}
	for _, spec := range commandSpecs {
		for _, flag := range completionFlagsForSpec(spec) {
			if flag.Name == "--root" {
				continue
			}
			writeFishFlagLine(&builder, spec.Name, flag)
		}
	}

	return builder.String()
}

func completionFlagsForCommandName(command string) []completionFlag {
	spec, ok := commandSpecFor(command)
	if !ok {
		return nil
	}

	return completionFlagsForSpec(spec)
}

func completionFlagsForSpec(spec commandSpec) []completionFlag {
	var names []string
	names = append(names, "--root")
	if spec.JSON {
		names = append(names, "--json")
	}
	if spec.Tests {
		names = append(names, "--tests")
	}
	if spec.All {
		names = append(names, "--all")
	}
	if spec.Context {
		names = append(names, "--context")
	}
	if spec.Snapshot {
		names = append(names, "--use-snapshot")
	}
	if spec.Tags || spec.TagsWhen != nil {
		names = append(names, "--tags")
	}
	if spec.BaseWhen != nil {
		names = append(names, "--base")
	}
	if spec.Kind {
		names = append(names, "--kind")
	}
	if spec.Package {
		names = append(names, "--package")
	}
	if spec.Limit {
		names = append(names, "--limit")
	}
	if spec.MaxDepth {
		names = append(names, "--max-depth")
	}
	if spec.ContextLimits {
		names = append(names, "--max-files", "--max-references", "--max-symbols", "--max-tests", "--max-bytes", "--source-radius")
	}
	if spec.Name == "tests" {
		names = append(names, "--scope")
	}

	return flagsByNames(names)
}

func flagsByNames(names []string) []completionFlag {
	flags := make([]completionFlag, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		flag, ok := completionFlagDefinitions[name]
		if !ok {
			continue
		}
		flags = append(flags, flag)
		seen[name] = struct{}{}
	}

	return flags
}

func completionFlagNames(flags []completionFlag) []string {
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		names = append(names, flag.Name)
	}

	return names
}

func completionValueFlagNames() []string {
	var names []string
	for _, flag := range completionFlagDefinitions {
		if flag.TakesValue {
			names = append(names, flag.Name)
		}
	}
	sort.Strings(names)

	return names
}

func completionCommandNames() []string {
	names := make([]string, 0, len(commandSpecs))
	for _, spec := range commandSpecs {
		names = append(names, spec.Name)
	}

	return names
}

func completionCommandDescription(spec commandSpec) string {
	return strings.Join(spec.Usage, " / ")
}

func zshCommandEntries() []string {
	entries := make([]string, 0, len(commandSpecs))
	for _, spec := range commandSpecs {
		entries = append(entries, spec.Name+":"+completionCommandDescription(spec))
	}

	return entries
}

func zshShellEntries() []string {
	entries := make([]string, 0, len(completionShells))
	for _, shell := range completionShells {
		entries = append(entries, shell.Name+":"+shell.Description)
	}

	return entries
}

func writeZshArray(builder *strings.Builder, name string, values []string) {
	fmt.Fprintf(builder, "  local -a %s\n", name)
	fmt.Fprintf(builder, "  %s=(\n", name)
	for _, value := range values {
		fmt.Fprintf(builder, "    %s\n", shellSingleQuote(value))
	}
	builder.WriteString("  )\n")
}

func zshFlagValueLines(flags []completionFlag, indent string) string {
	lines := make([]string, 0, len(flags))
	for _, flag := range flags {
		lines = append(lines, indent+shellSingleQuote(zshFlagSpec(flag)))
	}

	return strings.Join(lines, " \\\n") + "\n"
}

func zshFlagSpec(flag completionFlag) string {
	spec := flag.Name + "[" + flag.Description + "]"
	if flag.TakesValue {
		spec += ":" + flag.ValueName + ":"
		if len(flag.Values) > 0 {
			spec += "(" + strings.Join(flag.Values, " ") + ")"
		}
	}

	return spec
}

func writeFishFlagLine(builder *strings.Builder, command string, flag completionFlag) {
	var parts []string
	parts = append(parts, "complete", "-c", "gosherpa")
	if command != "" {
		parts = append(parts, "-n", shellSingleQuote("__fish_seen_subcommand_from "+command))
	}
	parts = append(parts, "-l", strings.TrimPrefix(flag.Name, "--"))
	if flag.TakesValue {
		parts = append(parts, "-r")
	}
	if len(flag.Values) > 0 {
		parts = append(parts, "-a", shellSingleQuote(strings.Join(flag.Values, " ")))
	}
	parts = append(parts, "-d", shellSingleQuote(flag.Description))
	builder.WriteString(strings.Join(parts, " "))
	builder.WriteString("\n")
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
