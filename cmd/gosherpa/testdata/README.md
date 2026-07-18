# GoSherpa CLI Test Fixtures

These fixtures are intentionally small module roots used by command-level JSON
golden tests and repository-shape tests.

| Fixture | Purpose |
| --- | --- |
| `json_project` | General command, diff, context, PR, affected-test, and agent-workflow golden JSON coverage. |
| `interface_project` | Interface, implementer, satisfied-interface, and relationship command coverage. |
| `possible_calls_project` | Bounded possible-runtime-call JSON coverage separate from direct caller and callee edges. |

The directories are nested below the main GoSherpa module on purpose. Root-level
analysis must report them as skipped nested modules instead of silently mixing
their packages into the main repository. Tests that need one fixture's packages
run commands with `--root cmd/gosherpa/testdata/<fixture>`.

Additional temporary fixtures for `go.work`, local `replace`, build tags,
partial package failures, generated files, and large-section truncation are
created inside focused Go tests. Keeping those cases generated in test temp
directories makes unsupported or boundary-sensitive repository shapes explicit
without adding large permanent fixture trees.
