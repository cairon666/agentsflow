# Writing Templates

Templates describe one reusable workflow and let each adapter render the native
files for Codex, Claude Code, OpenCode, and future targets.

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

Model slots and permission profiles are internal template references. Prefer
short, stable names such as `main`, `scout`, `reasoning`, `research`,
`execution`, `read_only`, and `workspace_write`. Keep references exact: every
`model_slot` and `permission_profile` used by an agent must exist in the
corresponding top-level section.
