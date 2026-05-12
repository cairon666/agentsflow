# CLI Agent Configuration Analysis

Дата проверки: 2026-05-12.

Цель документа: собрать различия в конфигурации агентов для Codex, Claude Code и OpenCode, чтобы на этой базе проектировать общий template, из которого можно генерировать нативные конфиги каждого CLI.

## Источники

Основные страницы:

- Codex: https://developers.openai.com/codex/subagents
- Claude Code: https://code.claude.com/docs/en/sub-agents
- OpenCode: https://opencode.ai/docs/agents/

Дополнительные официальные страницы, использованные для проверки путей, прав и общих настроек:

- Codex AGENTS.md: https://developers.openai.com/codex/guides/agents-md
- Codex config basics: https://developers.openai.com/codex/config-basic
- Codex config reference: https://developers.openai.com/codex/config-reference
- Claude Code memory / CLAUDE.md: https://code.claude.com/docs/en/memory
- Claude Code settings: https://code.claude.com/docs/en/settings
- Claude Code agent teams: https://code.claude.com/docs/en/agent-teams
- OpenCode config: https://opencode.ai/docs/config/
- OpenCode rules / AGENTS.md: https://opencode.ai/docs/rules/
- OpenCode permissions: https://opencode.ai/docs/permissions/

## Короткая сравнительная таблица

| Область | Codex | Claude Code | OpenCode |
| --- | --- | --- | --- |
| Основной файл инструкций | `AGENTS.md` | `CLAUDE.md`; `AGENTS.md` не читается напрямую | `AGENTS.md`; fallback на `CLAUDE.md` |
| User-level инструкции | `~/.codex/AGENTS.md` или `~/.codex/AGENTS.override.md` | `~/.claude/CLAUDE.md` | `~/.config/opencode/AGENTS.md`; fallback `~/.claude/CLAUDE.md` |
| Project-level инструкции | `AGENTS.md`, `AGENTS.override.md`, fallback names из `project_doc_fallback_filenames` | `./CLAUDE.md`, `./.claude/CLAUDE.md`, `./CLAUDE.local.md`, `.claude/rules/*.md` | project root `AGENTS.md`; fallback `CLAUDE.md`; дополнительные `instructions` в `opencode.json` |
| Project agents | `.codex/agents/*.toml` | `.claude/agents/*.md` | `.opencode/agents/*.md` или `agent` в `opencode.json` |
| User agents | `~/.codex/agents/*.toml` | `~/.claude/agents/*.md` | `~/.config/opencode/agents/*.md` |
| Формат агента | TOML | Markdown + YAML frontmatter | JSON/JSONC в `opencode.json` или Markdown + YAML frontmatter |
| Имя агента | Поле `name` является source of truth | Поле `name`; filename не обязан совпадать | Для Markdown имя берется из filename; для JSON ключ в `agent` |
| Обязательные поля | `name`, `description`, `developer_instructions` | `name`, `description` | `description` требуется; `mode` по умолчанию `all` |
| Модель | `model`, `model_reasoning_effort`, `model_provider` | `model`: `sonnet`, `opus`, `haiku`, `inherit` или full model ID | `model: provider/model-id` |
| Права | `sandbox_mode`, `approval_policy`, `sandbox_workspace_write.*`; агент может переопределять обычные config keys | `tools`, `disallowedTools`, `permissionMode`, session `permissions.*`, sandbox settings | `permission` со значениями `allow`, `ask`, `deny`, включая object rules |
| Delegation limits | `[agents] max_threads`, `max_depth`, `job_max_runtime_seconds` | `maxTurns`; subagents не спавнят subagents; agent teams без hard limit, практично 3-5 teammates | `steps`; `permission.task`; явного общего max agents в docs не найдено |
| Plan mode | `plan_mode_reasoning_effort`; planning agents обычно через read-only sandbox | `permissionMode: plan`; built-in Plan subagent; `useAutoModeDuringPlan` | built-in primary `plan`; edit/bash по умолчанию `ask`; можно задать `deny` |

## Codex

### Инструкции: AGENTS.md

Codex нативно читает `AGENTS.md` перед началом работы. Discovery строит цепочку инструкций один раз на run/session:

1. Global scope: Codex home по умолчанию `~/.codex`, либо `$CODEX_HOME`. На этом уровне читается первый непустой файл из `AGENTS.override.md`, затем `AGENTS.md`.
2. Project scope: от project root, обычно git root, до текущей директории. В каждой директории проверяются `AGENTS.override.md`, затем `AGENTS.md`, затем имена из `project_doc_fallback_filenames`.
3. Merge order: файлы объединяются от корня к текущей директории. Более близкие к cwd инструкции идут позже и фактически имеют больший вес.

Важные параметры:

- `project_doc_fallback_filenames`: дополнительные имена файлов инструкций, если `AGENTS.md` отсутствует.
- `project_doc_max_bytes`: лимит суммарных project instructions; по умолчанию 32 KiB.
- `CODEX_HOME`: позволяет сменить home-профиль, например на проектный `.codex`.

Вывод для template: `AGENTS.md` можно считать каноническим общим файлом инструкций для Codex и OpenCode.

### Пути конфигурации и агентов

Codex config:

- User config: `~/.codex/config.toml`.
- Project config: `.codex/config.toml`; загружается только для trusted project.

Custom agents:

- User agents: `~/.codex/agents/*.toml`.
- Project agents: `.codex/agents/*.toml`.

Каждый TOML-файл описывает одного custom agent. Codex загружает agent file как config layer для spawned session, поэтому custom agent может переопределять те же настройки, что и обычный `config.toml`.

Built-in agents:

- `default`: общий fallback agent.
- `worker`: execution-focused agent.
- `explorer`: read-heavy exploration agent.

Если custom agent имеет то же `name`, что и built-in agent, custom agent имеет приоритет.

### Формат агента

Минимальный schema:

```toml
name = "reviewer"
description = "PR reviewer focused on correctness, security, and missing tests."
developer_instructions = """
Review code like an owner.
Prioritize correctness, security, behavior regressions, and missing test coverage.
"""
```

Обязательные поля:

- `name`: имя, по которому Codex спавнит агента. Это source of truth; filename может не совпадать, хотя совпадение проще для поддержки.
- `description`: human-facing guidance, когда использовать агента.
- `developer_instructions`: core behavior/instructions агента.

Опциональные поля:

- `nickname_candidates`: UI/display nicknames для нескольких экземпляров одного агента.
- Любые поддерживаемые `config.toml` keys, например `model`, `model_provider`, `model_reasoning_effort`, `sandbox_mode`, `approval_policy`, `mcp_servers`, `skills.config`, `web_search`.

### Права и sandbox

Codex использует config-level модель прав:

- `sandbox_mode`: `read-only`, `workspace-write`, `danger-full-access`.
- `approval_policy`: `untrusted`, `on-request`, `never`, либо granular object с флагами `sandbox_approval`, `rules`, `mcp_elicitations`, `request_permissions`, `skill_approval`.
- `sandbox_workspace_write.network_access`: разрешение outbound network внутри `workspace-write`.
- `sandbox_workspace_write.writable_roots`: дополнительные writable roots.
- `sandbox_workspace_write.exclude_slash_tmp`, `exclude_tmpdir_env_var`: исключения для `/tmp` и `$TMPDIR`.

Поскольку agent TOML является config layer, read-only агент обычно описывается так:

```toml
sandbox_mode = "read-only"
approval_policy = "never"
```

Для implementer-like агента:

```toml
sandbox_mode = "workspace-write"
approval_policy = "on-request"

[sandbox_workspace_write]
network_access = false
```

Вывод для template: права Codex лучше маппить не на отдельные tools, а на sandbox/approval profile. Для тонких tool permissions нужен passthrough, потому что Codex и OpenCode/Claude Code используют разные уровни абстракции.

### Модели

Основные model-related keys:

- `model`: строка модели, например `gpt-5.5`.
- `model_provider`: provider id из `model_providers`; по умолчанию `openai`.
- `model_reasoning_effort`: `minimal`, `low`, `medium`, `high`, `xhigh`.
- `model_reasoning_summary`: `auto`, `concise`, `detailed`, `none`.
- `model_verbosity`: `low`, `medium`, `high`.
- `model_context_window`, `model_auto_compact_token_limit`.

Для custom agent `model` и `model_reasoning_effort` можно указать в `.codex/agents/<agent>.toml`. Если опциональные поля не заданы, они наследуются от parent session.

### Дополнительные параметры агентов

Global subagent settings находятся в `[agents]`:

```toml
[agents]
max_threads = 6
max_depth = 1
job_max_runtime_seconds = 1800
```

Поля:

- `agents.max_threads`: лимит одновременно открытых agent threads. По умолчанию `6`.
- `agents.max_depth`: глубина вложенного spawn; root session имеет depth `0`. По умолчанию `1`, что разрешает прямого child agent, но не более глубокую рекурсию.
- `agents.job_max_runtime_seconds`: default timeout per worker для `spawn_agents_on_csv`; если не задан, per-call default `1800` секунд.

Связанные параметры:

- `plan_mode_reasoning_effort`: reasoning override для Plan Mode.
- `profiles.<name>.*`: profile-scoped overrides почти любых config keys, включая model, plan mode, web search.
- `approvals_reviewer = "auto_review"`: eligible approval prompts может проверять reviewer subagent.
- `sqlite_home`: storage для SQLite-backed state agent jobs и exported CSV results.
- `skills.config`: per-skill enablement/path.
- `mcp_servers`: можно подключать MCP server прямо в agent config.

## Claude Code

### Инструкции: CLAUDE.md и совместимость с AGENTS.md

Claude Code нативно читает `CLAUDE.md`, а не `AGENTS.md`.

Релевантные пути:

- Managed policy:
  - macOS: `/Library/Application Support/ClaudeCode/CLAUDE.md`
  - Linux/WSL: `/etc/claude-code/CLAUDE.md`
  - Windows: `C:\Program Files\ClaudeCode\CLAUDE.md`
- Project instructions: `./CLAUDE.md` или `./.claude/CLAUDE.md`.
- User instructions: `~/.claude/CLAUDE.md`.
- Local instructions: `./CLAUDE.local.md`, обычно gitignored.
- Path-scoped project rules: `.claude/rules/*.md`.

Для совместимости с `AGENTS.md` официальный подход: создать `CLAUDE.md`, который импортирует общий файл:

```md
@AGENTS.md

## Claude Code

Use plan mode for risky changes.
```

Синтаксис imports: `@path/to/import`. Relative paths считаются относительно файла, который содержит import. Imports могут быть recursive, максимум 5 hops.

Вывод для template: общий project instruction source лучше хранить как `AGENTS.md`, а для Claude Code генерировать `CLAUDE.md` с `@AGENTS.md` и Claude-specific appendix.

### Пути конфигурации и агентов

Settings:

- User settings: `~/.claude/settings.json`.
- Project settings: `.claude/settings.json`, shared via git.
- Local project settings: `.claude/settings.local.json`, personal/gitignored.
- Managed settings: system-level `managed-settings.json`, plist/registry/server-managed.

Subagents:

- Managed settings directory: organization-wide, highest priority.
- `--agents` CLI flag: session-level JSON.
- Project agents: `.claude/agents/*.md`.
- User agents: `~/.claude/agents/*.md`.
- Plugin agents: plugin `agents/` directory, lowest priority.

Priority order for same-name agents:

1. Managed settings.
2. `--agents` CLI flag.
3. `.claude/agents/`.
4. `~/.claude/agents/`.
5. Plugin agents.

Project subagents are discovered by walking up from the current working directory. Directories passed with `--add-dir` grant file access but are not scanned for subagents.

### Формат агента

Subagent file: Markdown with YAML frontmatter. Body is the subagent system prompt.

```md
---
name: code-reviewer
description: Reviews code for quality and best practices
tools: Read, Glob, Grep
model: sonnet
permissionMode: plan
maxTurns: 8
---

You are a code reviewer. Focus on correctness, regressions, security, and tests.
```

Required frontmatter fields:

- `name`: unique identifier, lowercase letters and hyphens; filename does not have to match.
- `description`: when Claude should delegate to this subagent.

Important optional fields:

- `tools`: allowlist of tools. If omitted, inherits all tools from the main conversation.
- `disallowedTools`: denylist removed from inherited/specified tools.
- `model`.
- `permissionMode`.
- `maxTurns`.
- `skills`.
- `mcpServers`.
- `hooks`.
- `memory`.
- `background`.
- `effort`.
- `isolation`.
- `color`.
- `initialPrompt`.

Subagents receive their own system prompt plus basic environment details, not the full default Claude Code system prompt. They start in the main conversation's current working directory. `cd` inside a subagent does not persist between shell calls and does not affect the main conversation.

### Права и permissions

Claude Code combines several layers:

1. Agent-level tool allow/deny:
   - `tools`: allowlist.
   - `disallowedTools`: denylist.
   - If both are set, `disallowedTools` is applied first, then `tools`.
2. Agent-level `permissionMode`.
3. Session/project/user settings `permissions`.
4. Optional sandbox settings.

`permissionMode` values:

- `default`: standard permission checking with prompts.
- `acceptEdits`: auto-accept file edits and common filesystem commands for allowed paths.
- `auto`: background classifier reviews commands and protected-directory writes.
- `dontAsk`: auto-deny permission prompts; explicitly allowed tools still work.
- `bypassPermissions`: skip permission prompts; dangerous.
- `plan`: read-only plan mode.

Inheritance caveats:

- Subagents inherit permission context from the main conversation.
- Parent `bypassPermissions` or `acceptEdits` takes precedence and cannot be overridden by a subagent.
- Parent `auto` mode makes the subagent inherit auto mode and ignores `permissionMode` in frontmatter.

Session settings permissions:

```json
{
  "permissions": {
    "allow": ["Bash(npm run lint)", "Read(~/.zshrc)"],
    "ask": ["Bash(git push *)"],
    "deny": ["Bash(curl *)", "Read(./.env)", "Read(./secrets/**)"],
    "defaultMode": "acceptEdits"
  }
}
```

Permission rules use `Tool` or `Tool(specifier)` syntax. Evaluation order: deny, then ask, then allow; first matching rule wins. Settings also support `additionalDirectories`.

Sandbox settings exist under `sandbox.*`, for example:

- `sandbox.enabled`.
- `sandbox.failIfUnavailable`.
- `sandbox.autoAllowBashIfSandboxed`.
- `sandbox.excludedCommands`.
- `sandbox.filesystem.allowWrite`, `denyWrite`, `denyRead`, `allowRead`.
- `sandbox.network.allowedDomains`, `deniedDomains`, proxy settings.

Subagent spawning restrictions:

- When an agent runs as main thread via `claude --agent`, it can spawn subagents only if `Agent` is included in `tools`.
- `Agent(worker, researcher)` allowlists specific subagent types.
- Omitting `Agent` prevents spawning.
- Subagents themselves cannot spawn other subagents, so `Agent(...)` matters only for agents running as the main thread.

### Модели

Subagent `model` accepts:

- Alias: `sonnet`, `opus`, `haiku`.
- Full model ID, for example `claude-opus-4-7` or `claude-sonnet-4-6`.
- `inherit`.
- Omitted means `inherit`.

Resolution order:

1. `CLAUDE_CODE_SUBAGENT_MODEL` env var.
2. Per-invocation `model` parameter.
3. Subagent frontmatter `model`.
4. Main conversation model.

`effort` can override session effort while the subagent is active:

- `low`, `medium`, `high`, `xhigh`, `max`.
- Available values depend on the model.

### Дополнительные параметры агентов

Subagent-specific:

- `maxTurns`: maximum agentic turns before the subagent stops.
- `background: true`: always run as background task.
- `isolation: worktree`: temporary git worktree; cleaned up if no changes.
- `memory: user | project | local`:
  - `user`: `~/.claude/agent-memory/<agent>/`.
  - `project`: `.claude/agent-memory/<agent>/`.
  - `local`: `.claude/agent-memory-local/<agent>/`.
- `mcpServers`: inline MCP servers or references to already-configured servers.
- `hooks`: lifecycle hooks scoped to this subagent.
- `skills`: preload Skills content into context.
- `initialPrompt`: auto-submitted when agent is main session via `--agent` or `agent` setting.
- `color`: UI color.

Settings-level:

- `agent`: run the main thread as a named subagent.
- `disableAgentView`: disables background agents / agent view.
- `teammateMode`: `auto`, `in-process`, `tmux`.
- `useAutoModeDuringPlan`: whether plan mode uses auto mode semantics when available.
- `worktree.baseRef`, `worktree.symlinkDirectories`, `worktree.sparsePaths`.

Agent teams:

- Experimental; enabled with `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` in env/settings.
- One session is lead, teammates are independent Claude Code instances.
- No hard teammate limit documented, but official guidance recommends starting with 3-5 teammates.
- One team at a time; no nested teams.
- Teammates start with lead's permission mode; per-teammate modes cannot be set at spawn time.
- Teammates load project context such as `CLAUDE.md`, MCP servers, and skills, but do not inherit lead conversation history.

## OpenCode

### Инструкции: AGENTS.md

OpenCode нативно uses `AGENTS.md` for custom rules.

Paths:

- Project rules: `AGENTS.md` in project root; applies to the directory and subdirectories.
- Global rules: `~/.config/opencode/AGENTS.md`.
- Claude Code compatibility fallback:
  - Project `CLAUDE.md` if no project `AGENTS.md`.
  - Global `~/.claude/CLAUDE.md` if no `~/.config/opencode/AGENTS.md`.
  - `~/.claude/skills/` support for skills.

Disable Claude compatibility:

```sh
OPENCODE_DISABLE_CLAUDE_CODE=1
OPENCODE_DISABLE_CLAUDE_CODE_PROMPT=1
OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1
```

Rules precedence:

1. Local files by traversing up from current directory: `AGENTS.md`, then `CLAUDE.md`.
2. Global `~/.config/opencode/AGENTS.md`.
3. Claude global `~/.claude/CLAUDE.md`, unless disabled.

First matching file wins in each category. If both `AGENTS.md` and `CLAUDE.md` exist, only `AGENTS.md` is used.

Additional instruction files can be configured in `opencode.json`:

```json
{
  "instructions": ["CONTRIBUTING.md", "docs/guidelines.md", ".cursor/rules/*.md"]
}
```

Remote instruction URLs are supported and fetched with a 5 second timeout. All instruction files are combined with `AGENTS.md`.

### Пути конфигурации и агентов

Config format: JSON or JSONC.

Config precedence, lower to higher:

1. Remote config from `.well-known/opencode`.
2. Global config: `~/.config/opencode/opencode.json`.
3. Custom config via `OPENCODE_CONFIG`.
4. Project config: `opencode.json`.
5. `.opencode` directories: agents, commands, plugins.
6. Inline config via `OPENCODE_CONFIG_CONTENT`.
7. Managed config files, macOS `/Library/Application Support/opencode/`.
8. macOS managed preferences via MDM.

Configs are merged, not replaced; later conflicting keys override earlier ones.

Agent definitions:

- JSON/JSONC: `agent` object in `opencode.json`.
- Global Markdown agents: `~/.config/opencode/agents/*.md`.
- Project Markdown agents: `.opencode/agents/*.md`.

The `.opencode` and `~/.config/opencode` directories use plural subdirectory names such as `agents/`, `commands/`, `plugins/`, `skills/`, `tools/`, `themes/`. Singular names are still supported for backwards compatibility.

### Формат агента

JSON example:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "agent": {
    "code-reviewer": {
      "description": "Reviews code for best practices and potential issues",
      "mode": "subagent",
      "model": "anthropic/claude-sonnet-4-20250514",
      "prompt": "You are a code reviewer. Focus on security, performance, and maintainability.",
      "permission": {
        "edit": "deny"
      }
    }
  }
}
```

Markdown example:

```md
---
description: Reviews code for quality and best practices
mode: subagent
model: anthropic/claude-sonnet-4-20250514
temperature: 0.1
permission:
  edit: deny
  bash: deny
---

You are in code review mode.
```

For Markdown agents, the filename becomes the agent name: `.opencode/agents/review.md` creates `review`.

Important options:

- `description`: required.
- `temperature`.
- `steps`.
- `disable`.
- `prompt`: inline text or file reference like `{file:./prompts/build.txt}`; path is relative to the config file.
- `model`.
- `permission`.
- `mode`.
- `hidden`.
- `color`.
- `top_p`.
- Additional provider-specific options.

### Типы агентов

OpenCode has primary agents and subagents:

- Primary agents: main assistants users interact with directly; can be cycled with Tab / configured keybind.
- Subagents: specialized assistants primary agents can invoke automatically or manually via `@mention`.

Built-in primary agents:

- `build`: default primary agent with all tools enabled.
- `plan`: restricted primary agent for analysis/planning. By default, file edits and bash are set to `ask`.

Built-in subagents:

- `general`: general-purpose, multi-step tasks, full tool access except todo.
- `explore`: fast, read-only codebase exploration.
- `scout`: read-only external docs/dependency research.

Hidden system agents:

- `compaction`.
- `title`.
- `summary`.

### Права и permissions

OpenCode uses the `permission` config. Each rule resolves to:

- `allow`: run without approval.
- `ask`: prompt for approval.
- `deny`: block action.

Global shorthand:

```json
{
  "permission": "allow"
}
```

Global object:

```json
{
  "permission": {
    "*": "ask",
    "bash": "allow",
    "edit": "deny"
  }
}
```

Agent-level permissions override global permissions:

```json
{
  "permission": {
    "edit": "deny"
  },
  "agent": {
    "build": {
      "permission": {
        "edit": "ask"
      }
    }
  }
}
```

Available permission keys:

- `read`: `read`.
- `edit`: `write`, `edit`, `apply_patch`.
- `glob`: `glob`.
- `grep`: `grep`.
- `list`: `list`.
- `bash`: `bash`.
- `task`: `task`.
- `external_directory`: tools touching paths outside project worktree.
- `todowrite`: `todowrite`, `todoread`.
- `webfetch`: `webfetch`.
- `websearch`: `websearch`.
- `lsp`: `lsp`.
- `skill`: `skill`.
- `question`: `question`.
- `doom_loop`: recovery prompts when an agent appears stuck.

Granular object syntax:

```json
{
  "permission": {
    "bash": {
      "*": "ask",
      "git *": "allow",
      "npm *": "allow",
      "rm *": "deny"
    },
    "edit": {
      "*": "deny",
      "packages/web/src/content/docs/*.mdx": "allow"
    }
  }
}
```

Pattern behavior:

- Simple wildcard matching: `*` and `?`.
- Last matching rule wins, so put catch-all first and more specific rules later.
- `~` and `$HOME` are expanded at the start of patterns.
- `external_directory` is required for paths outside the worktree.

Agent invocation permissions:

```json
{
  "agent": {
    "orchestrator": {
      "mode": "primary",
      "permission": {
        "task": {
          "*": "deny",
          "orchestrator-*": "allow",
          "code-reviewer": "ask"
        }
      }
    }
  }
}
```

If a subagent is denied via `permission.task`, it is removed from the Task tool description, so the model should not try to call it. Users can still invoke subagents directly via `@` autocomplete even when task permissions deny model invocation.

Legacy `tools` boolean config is deprecated as of OpenCode `v1.1.1`. New configs should use `permission`.

### Модели

Agent model override:

```json
{
  "agent": {
    "plan": {
      "model": "anthropic/claude-haiku-4-20250514"
    }
  }
}
```

Rules:

- Model ID format is `provider/model-id`.
- If no model is specified:
  - primary agents use the globally configured model.
  - subagents use the model of the primary agent that invoked them.
- `opencode models` lists available models.

Provider-specific additional options are passed through to the provider:

```json
{
  "agent": {
    "deep-thinker": {
      "description": "Agent that uses high reasoning effort for complex problems",
      "model": "openai/gpt-5",
      "reasoningEffort": "high",
      "textVerbosity": "low"
    }
  }
}
```

### Дополнительные параметры агентов

- `mode`: `primary`, `subagent`, `all`; default `all`.
- `steps`: max agentic iterations before forced text response. Legacy `maxSteps` is deprecated.
- `disable`: disables an agent.
- `hidden`: hides `mode: subagent` agents from `@` autocomplete; model can still invoke them through Task if permissions allow.
- `permission.task`: controls which subagents an agent can invoke.
- `temperature`: generally 0.0-1.0; if omitted, OpenCode uses model-specific defaults.
- `top_p`: response diversity, 0.0-1.0.
- `color`: hex color or theme color: `primary`, `secondary`, `accent`, `success`, `warning`, `error`, `info`.
- Additional provider-specific options, such as OpenAI reasoning parameters.

No hard max-subagents setting was found in the OpenCode agent/config docs. The available control for cost/runaway behavior is mainly `steps`, `permission.task`, `hidden`, and the permission system.

## Implications for a common template

### Canonical instruction file

Use project `AGENTS.md` as the canonical cross-tool instruction file:

- Codex reads it natively.
- OpenCode reads it natively.
- Claude Code can read it through generated `CLAUDE.md`:

```md
@AGENTS.md

## Claude Code

<claude-specific instructions>
```

Avoid duplicating the full instructions in `CLAUDE.md`; duplication risks drift.

### Normalized shared fields

Recommended normalized agent fields:

- `id`: stable machine id.
- `description`: delegation/use guidance.
- `prompt`: system/developer instructions/body.
- `mode`: `primary`, `subagent`, `all`, plus higher-level semantic roles like `planner`, `reviewer`, `implementer`.
- `model`: abstract model alias resolved per tool.
- `reasoning_effort`: abstract effort enum.
- `permissions`: normalized shape around `allow | ask | deny`.
- `tools`: optional high-level allow/deny for tools where supported.
- `limits`: `max_threads`, `max_depth`, `max_turns`, `steps`, `runtime_seconds`.
- `memory`: user/project/local/off.
- `mcp`: references or inline definitions.
- `hooks`: lifecycle hooks.
- `ui`: color, hidden, nickname candidates.
- `provider_options`: model/provider-specific passthrough.
- `tool_overrides`: per-tool blocks for native config that cannot be normalized.

### Why passthrough blocks are necessary

Permissions are not isomorphic:

- Codex centers on sandbox and approval policy.
- Claude Code combines tool allow/deny, permission modes, settings permissions, and optional sandbox.
- OpenCode has direct per-tool/per-pattern `permission` rules.

The common template should normalize the simple case:

```yaml
permissions:
  edit: deny
  bash: ask
  webfetch: allow
```

But keep native escape hatches:

```yaml
codex:
  sandbox_mode: read-only
  approval_policy: never

claude_code:
  permissionMode: plan
  tools: [Read, Grep, Glob]
  disallowedTools: [Edit, Write]

opencode:
  permission:
    edit: deny
    bash:
      "*": ask
      "git diff*": allow
```

### Recommended generated outputs

Codex:

- `AGENTS.md`
- `.codex/config.toml`
- `.codex/agents/<agent>.toml`

Claude Code:

- `CLAUDE.md` with `@AGENTS.md`
- `.claude/settings.json`
- `.claude/agents/<agent>.md`

OpenCode:

- `AGENTS.md`
- `opencode.json`
- `.opencode/agents/<agent>.md`, or JSON-only agent definitions when a single-file config is preferred.

### Suggested mapping defaults

Read-only explorer/reviewer:

- Codex: `sandbox_mode = "read-only"`, `approval_policy = "never"`.
- Claude Code: `permissionMode: plan` or `tools: Read, Grep, Glob` plus `disallowedTools: Write, Edit`.
- OpenCode: `permission.edit = deny`, `permission.bash = ask` or `deny`.

Planner:

- Codex: read-only sandbox plus `model_reasoning_effort` / `plan_mode_reasoning_effort`.
- Claude Code: `permissionMode: plan`, possibly `maxTurns`.
- OpenCode: primary/subagent `plan`-style mode with `edit: deny` or `ask`, `bash: ask`.

Implementer:

- Codex: `workspace-write`, `approval_policy = "on-request"`, network disabled unless explicitly needed.
- Claude Code: constrained `tools` plus `permissionMode: default` or `acceptEdits` only when safe.
- OpenCode: `edit: allow` or `ask`, `bash` granular allowlist, `task` restricted if it can spawn helpers.

Docs researcher:

- Codex: read-only, `web_search = "live"` or scoped MCP docs server.
- Claude Code: read/search/web tools, optional `mcpServers`, no write tools.
- OpenCode: `webfetch`/`websearch` allowed, `edit` denied, optional `scout`-like profile.

### Open design questions for the future template

- Whether model aliases should be global, per tool, or per agent role.
- Whether permissions should default to conservative read-only unless role is explicitly implementer/build.
- Whether to generate Markdown agents for OpenCode or keep OpenCode agents in `opencode.json`.
- Whether to support Claude Code agent teams as a first-class template concept, or treat them as a runtime orchestration feature outside static agent definitions.
- Whether Codex `[agents].max_threads` / `max_depth` should be global template fields or codex-only passthrough.
