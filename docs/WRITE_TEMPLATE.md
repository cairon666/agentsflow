# Writing Templates

Templates describe one reusable workflow and let each adapter render the native
files for Codex, Claude Code, OpenCode, and future targets.

## Git Repository Layout

When `agentsflow use` receives a Git repository URL, it looks for templates in
the repository's `.agentsflow` directory. Template files can be placed directly
under `.agentsflow` or one directory below it:

```text
.agentsflow/*.{yml,yaml}
.agentsflow/*/*.{yml,yaml}
```

Examples:

```text
.agentsflow/lightweight-engineering.yaml
.agentsflow/multi-agent-engineering.yml
.agentsflow/team/backend.yaml
```

The template selection prompt shows the path relative to `.agentsflow` without
the file extension, such as `lightweight-engineering` or `team/backend`.
Keep those extensionless paths unique; for example, do not include both
`.agentsflow/team/backend.yml` and `.agentsflow/team/backend.yaml` in the same
repository.

## Agent IDs

Use kebab-case for agent IDs:

```yaml
agents:
  explorer-fast:
    description: Fast read-heavy scout for quick repo discovery.
  docs-researcher:
    description: Researches external official documentation.
  unit-test-writer:
    description: Adds focused unit/integration tests.
```

Agent IDs should match this shape:

```text
^[a-z0-9]+(-[a-z0-9]+)*$
```

This means:

- lowercase letters and numbers only
- words separated by single hyphens
- no underscores
- no leading, trailing, or repeated hyphens

Kebab-case is the safest common denominator across supported tools. Claude Code
requires subagent `name` values to use lowercase letters and hyphens. OpenCode
uses markdown filenames as agent names and follows the same hyphenated naming
style in its config examples. Codex accepts the same names in `.codex/agents`
TOML files, so using one canonical kebab-case ID avoids target-specific aliases.

When project instructions mention agents, use the same IDs that appear under
`agents:`:

```yaml
instructions:
  AGENTS.md: |
    1. Start with `explorer-fast`.
    2. Use `docs-researcher` when external documentation is required.
```

Do not write `explorer_fast` in instructions if the agent ID is
`explorer-fast`; target adapters cannot reliably rewrite arbitrary prose without
changing user-authored text incorrectly.

## Other IDs

Model slots and permission profiles are internal template references. The
`main` model slot is built in and always available, so templates do not need to
declare `model_slots.main`. When an agent omits `model_slot`, it uses `main`.
Declare `model_slots` only for additional slots such as `scout`, `reasoning`,
`research`, `execution`, or `code`.

Every explicit `model_slot` other than `main`, and every `permission_profile`
used by an agent, must exist in the corresponding top-level section. Supported
targets automatically receive the selected `models.main` value.
