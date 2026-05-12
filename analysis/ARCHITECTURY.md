# Agentflow: архитектура Go CLI и выбор библиотек

Дата анализа: 2026-05-12.

Цель: спроектировать архитектуру CLI-инструмента на Go, который читает переносимый `template.yaml`, валидирует его, в интерактивном режиме собирает пользовательские решения и генерирует нативные конфиги для Codex, Claude Code, OpenCode и будущих CLI-инструментов.

## Исходные требования

Agentflow не должен превращать каждый template в набор vendor-specific блоков. Template должен описывать смысловую структуру workflow:

- логические model slots: `main`, `code`, `research`, произвольные слоты;
- роли агентов: `explorer_fast`, `planner`, `implementer`, `reviewer` и т.д.;
- permission profiles через capabilities, а не через конкретные поля Codex/Claude/OpenCode;
- общие инструкции, например `AGENTS.md`;
- минимальные tool-level настройки только там, где они действительно являются настройками выбранного target.

Маппинг capabilities, форматов файлов, permission-моделей и путей установки должен жить в коде Agentflow, в target adapters, а не дублироваться в каждом шаблоне.

## Архитектурный принцип

Рекомендуемая модель: `Template DSL -> Normalized IR -> Target Adapter -> Rendered Files -> Install Plan`.

Это ключевой слой защиты от vendor lock-in:

- `Template DSL` хранит пользовательский переносимый YAML.
- `Normalized IR` является проверенным внутренним представлением, уже без YAML-деталей и с разрешенными model bindings.
- `Target Adapter` знает, как конкретный CLI ожидает права, модели, инструкции и agent files.
- `Rendered Files` являются байтовыми артефактами, но еще не записаны на диск.
- `Install Plan` описывает, какие файлы будут созданы, изменены, пропущены или потребуют подтверждения.

Нельзя делать архитектуру как `template.yaml -> codex renderer` напрямую. Это быстро приведет к копированию правил между адаптерами и усложнит добавление новых инструментов.

## Рекомендуемая структура пакетов

```text
cmd/agentflow/
  main.go

internal/cli/
  root.go
  use.go

internal/app/
  app.go
  usecase_use.go

internal/builder/
  flow.go
  prompts.go
  summary.go

internal/template/
  model.go
  loader.go
  decoder.go

internal/schema/
  validate.go
  capabilities.go
  diagnostics.go

internal/binding/
  models.go
  targets.go
  scope.go

internal/ir/
  flow.go
  agent.go
  permissions.go

internal/adapter/
  adapter.go
  registry.go

internal/adapter/codex/
  adapter.go
  permissions.go
  render.go

internal/adapter/claude/
  adapter.go
  permissions.go
  render.go

internal/adapter/opencode/
  adapter.go
  permissions.go
  render.go

internal/render/
  files.go
  markdown.go
  toml.go
  json.go
  yaml.go

internal/install/
  plan.go
  writer.go
  conflict.go

internal/fsx/
  paths.go
  atomic.go
  permissions.go

internal/diagnostic/
  diagnostic.go
  format.go
```

`cmd/agentflow` должен быть тонким: создать root command, передать `context.Context`, вернуть exit code. Бизнес-логика не должна жить в Cobra handlers.

## Слои и ответственность

### `internal/template`

Отвечает только за чтение YAML и первичный decode.

Правила:

- использовать строгий decode, чтобы неизвестные поля не проходили молча;
- не выполнять target-specific маппинг;
- не подставлять default-модели;
- не писать на диск;
- сохранять позицию/контекст ошибок там, где это возможно.

Для YAML достаточно `gopkg.in/yaml.v3`: пакет поддерживает `Decoder.KnownFields(true)`, а `yaml.Node` можно использовать, если понадобятся более точные diagnostics по строкам.

### `internal/schema`

Содержит контракт Agentflow DSL, включая словарь capabilities. Именно здесь должны жить допустимые capability names:

```text
read_files
list_files
search_code
inspect_metadata
edit_files
run_shell
fetch_urls
web_search
spawn_agents
ask_user
```

Template только использует эти names, но не описывает их.

Валидация должна проверять:

- `version` поддерживается;
- все `model_slot` существуют;
- все `permission_profile` существуют;
- все capability names известны;
- capability values входят в `allow | ask | deny`;
- agent id стабилен и переносим;
- target-specific настройки не попали в переносимый слой;
- `instructions` не дублируют одни и те же generated files конфликтующим образом.

JSON Schema можно генерировать для редакторов, но основную валидацию лучше держать в Go-коде. Причина: многие правила семантические и зависят от связей между секциями.

### `internal/ir`

Normalized IR должен быть уже очищенным от YAML-формы:

```go
type Flow struct {
    ID                 string
    Version            int
    ModelSlots          map[string]ModelSlot
    PermissionProfiles  map[string]PermissionProfile
    Agents              map[string]Agent
    Instructions        map[string]string
    ToolConfigs         map[string]map[string]any
}
```

IR не должен знать, что Codex использует TOML, Claude использует Markdown frontmatter, а OpenCode может использовать JSON/JSONC. Это задача adapters.

### `internal/binding`

Отвечает за решения, которые пользователь принимает в интерактивном builder-flow:

- выбор target: Codex, Claude Code, OpenCode или другой зарегистрированный adapter;
- ввод model binding для каждого slot из template;
- выбор installation scope: project или global;
- подтверждение install summary;
- aliases target names, например `claude`, `claudecode`, `claude-code`.

Важно: bindings не должны мутировать template. Они создают отдельный `ResolvedFlow`. В MVP эти bindings собираются только через вопросы CLI, а не через flags.

### `internal/builder`

Оркестрирует интерактивный сценарий `agentflow use <template>`.

Builder не должен знать, как рендерить Codex TOML или Claude Markdown. Его задача:

- показать пользователю понятную последовательность вопросов;
- получить target;
- получить model bindings для всех slots;
- получить installation scope;
- показать install summary;
- вернуть структурированные ответы в `internal/app`.

Это отдельный слой между Cobra command handler и use case. Так CLI остается тонким, а интерактивный UX можно тестировать отдельно от adapters и filesystem writer.

### `internal/adapter`

Основной extension point.

```go
type Adapter interface {
    Target() Target
    Validate(ctx context.Context, flow ir.Flow) []diagnostic.Diagnostic
    Render(ctx context.Context, input RenderInput) (install.Plan, []diagnostic.Diagnostic)
}
```

Target adapter отвечает за:

- поддержку target aliases;
- проверку target-specific ограничений;
- маппинг capabilities в нативные permissions;
- выбор generated paths для `project` и `global` scope;
- форматирование файлов;
- предупреждения о lossy mapping.

Новые CLI-инструменты должны добавляться через новый adapter package и регистрацию в `adapter.Registry`, а не через изменение всех templates.

## Маппинг target adapters

### Codex adapter

Generated outputs:

- `AGENTS.md`;
- `.codex/config.toml`;
- `.codex/agents/<agent>.toml`.

Codex permission mapping:

- read-only profiles -> `sandbox_mode = "read-only"`, `approval_policy = "never"`;
- workspace write -> `sandbox_mode = "workspace-write"`, `approval_policy = "on-request"`;
- network for workspace write -> `sandbox_workspace_write.network_access`;
- reasoning -> `model_reasoning_effort`;
- model slot binding -> `model`.

Codex-specific risk: permissions are sandbox/config based, not direct per-tool allow/deny. Adapter must warn when a profile asks for fine-grained behavior Codex cannot represent exactly.

### Claude Code adapter

Generated outputs:

- `CLAUDE.md` with `@AGENTS.md`;
- `.claude/settings.json`;
- `.claude/agents/<agent>.md`.

Claude adapter should avoid duplicating full project instructions in `CLAUDE.md`; keep `AGENTS.md` canonical and import it.

Claude permission mapping:

- read/search profiles -> `tools: [Read, Grep, Glob]`;
- planning profiles -> `permissionMode: plan`;
- denied write -> `disallowedTools: [Edit, Write]`;
- model slot binding -> `model`;
- optional iteration limits -> `maxTurns`.

Claude-specific risk: parent session permissions can override or constrain subagent permissions. Adapter should document warnings in install summary.

### OpenCode adapter

Generated outputs:

- `AGENTS.md`;
- `opencode.json`;
- `.opencode/agents/<agent>.md` by default.

OpenCode permission mapping:

- `edit_files` -> `permission.edit`;
- `run_shell` -> `permission.bash`;
- `fetch_urls` -> `permission.webfetch`;
- `web_search` -> `permission.websearch`;
- `spawn_agents` -> `permission.task`;
- `mode` should eventually map to `primary`, `subagent`, or `all`.

OpenCode-specific risk: JSONC is common in config ecosystems, but generated output should be deterministic. Prefer JSON for generated `opencode.json`; support reading JSONC only if needed.

## CLI команды

В MVP нужна только одна пользовательская команда:

```sh
agentflow use <template>
```

`use` работает как последовательный builder:

1. Загружает и валидирует template.
2. Показывает краткую информацию: template id, version, agents, model slots, permission profiles.
3. Спрашивает, для какого CLI-инструмента сгенерировать конфигурацию.
4. Последовательно спрашивает модель для каждого `model_slot`.
5. Спрашивает installation scope: project или global.
6. Строит install plan.
7. Показывает summary: target, scope, generated files, conflicts, warnings adapter'а.
8. Просит подтверждение записи.
9. Записывает файлы или завершает без изменений.

`use` должен быть безопасным по умолчанию:

- сначала строит install plan;
- показывает summary;
- пишет только после явного подтверждения;
- позволяет отменить установку на шаге подтверждения;
- не перетирает пользовательские файлы без marker или явного managed-path правила.

Публичный CLI-контракт текущей архитектуры ограничен этим одним сценарием. Внутренние validation, render и target discovery остаются пакетами/функциями, но не отдельными пользовательскими командами.

## Install plan и безопасность записи

Не писать файлы напрямую из adapter. Adapter возвращает plan:

```go
type Plan struct {
    Target  string
    Scope   Scope
    Actions []Action
}

type Action struct {
    Path      string
    Kind      ActionKind // create, update, skip, conflict
    Content   []byte
    ManagedBy bool
}
```

`install.Writer` отвечает за:

- создание директорий;
- atomic write через temp file + rename;
- сохранение file mode;
- conflict detection;
- запрет записи вне выбранного scope;
- итоговый report.

Для generated files не стоит добавлять служебный marker в содержимое файла.

Безопасное обновление должно опираться на явные managed paths, checksum или metadata,
например `.agentflow/manifest.json`, чтобы понимать, что именно менялось прошлым запуском.

## Библиотеки

### CLI

Рекомендация: `github.com/spf13/cobra`.

Причины:

- зрелый стандарт для subcommand-based CLI;
- поддерживает help, aliases, shell completions и man pages без самописной CLI-инфраструктуры;
- хорошо подходит для одного стартового сценария `agentflow use <template>` с понятным help, positional arguments и shell completion.

Не стоит сразу брать `viper` как обязательную зависимость. Viper полезен для сложной runtime config, env и config file merging, но для Agentflow на старте достаточно одного positional argument и интерактивных prompts. Если появится user config (`~/.config/agentflow/config.yaml`), Viper можно добавить позже.

Источники:

- Cobra: https://pkg.go.dev/github.com/spf13/cobra
- Viper: https://pkg.go.dev/github.com/spf13/viper

### Интерактивные prompts

Рекомендация: `github.com/charmbracelet/huh`.

Подходит для:

- выбора target;
- ввода model slots;
- выбора installation scope;
- confirmation summary.

Плюс: есть accessible mode и нормальная модель forms. Для текущего MVP `huh` стоит добавить сразу, потому что основной UX должен быть builder-flow, а не набор flags.

Источник: https://pkg.go.dev/github.com/charmbracelet/huh

### YAML

Рекомендация: `gopkg.in/yaml.v3`.

Причины:

- стабильный API;
- поддержка `KnownFields`;
- можно работать с `yaml.Node`, если понадобятся line-aware diagnostics;
- достаточно для `template.yaml`.

Источник: https://pkg.go.dev/gopkg.in/yaml.v3

### TOML

Рекомендация: `github.com/pelletier/go-toml/v2`.

Нужно для генерации Codex `.toml` файлов. Библиотека поддерживает TOML v1.0.0.

Источник: https://pkg.go.dev/github.com/pelletier/go-toml/v2

### JSON и JSONC

Для записи generated JSON использовать стандартный `encoding/json`.

Для чтения/нормализации JSONC, если понадобится совместимость с human-written OpenCode configs, можно использовать `github.com/tailscale/hujson`. Он поддерживает JSON with comments and trailing commas и умеет стандартизировать input в обычный JSON.

Источник: https://pkg.go.dev/github.com/tailscale/hujson

### JSON Schema

Разделить две задачи:

- `github.com/invopop/jsonschema` для генерации JSON Schema из Go structs, чтобы дать пользователям editor autocomplete;
- `github.com/santhosh-tekuri/jsonschema/v6` только если понадобится runtime validation внешней JSON Schema.

Для MVP можно обойтись Go-валидатором. Генерацию JSON Schema для редакторов стоит отложить до стабилизации DSL.

Источники:

- https://pkg.go.dev/github.com/invopop/jsonschema
- https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6

### Terminal styling

Для MVP лучше минимальный plain text.

Если нужен аккуратный summary в терминале:

- `github.com/charmbracelet/lipgloss` для layout/styling;
- либо совсем без styling, чтобы CI output был максимально чистым.

Источник: https://pkg.go.dev/github.com/charmbracelet/lipgloss

## Тестовая стратегия

Минимальные тесты:

- table-driven tests для template decode;
- validation tests для unknown capabilities, missing model slots, missing permission profiles;
- adapter golden tests: input template + bindings -> generated files;
- install plan tests без реальной записи на диск;
- path safety tests для project/global scope;
- snapshot/golden tests для Codex TOML, Claude Markdown frontmatter, OpenCode JSON.
- CLI contract tests: наружу доступен только `agentflow use <template>`, а target, model bindings и scope собираются через prompts, не через `--target`, `--bind`, `--scope`.

Для файловых тестов на старте достаточно `t.TempDir()` и стандартного `os`/`io/fs`. `afero` добавлять только если появится реальная боль с filesystem abstraction.

## MVP порядок реализации

1. Описать Go structs для template DSL.
2. Добавить loader YAML с strict known fields.
3. Добавить semantic validator.
4. Добавить adapter interface и registry.
5. Добавить интерактивный builder для `agentflow use <template>` через `huh`.
6. Добавить model binding resolver, который принимает ответы builder'а.
7. Реализовать первый target adapter.
8. Добавить install plan и summary перед записью.
9. Добавить install writer с conflict handling.
10. Добавить остальные target adapters.
11. Покрыть builder-flow и adapters тестами.

## Главные риски

- Слишком ранние target-specific overrides в template. Решение: capabilities и adapters в core.
- Потеря семантики permissions при маппинге. Решение: adapter diagnostics и warnings.
- Небезопасная перезапись пользовательских файлов. Решение: install plan, managed markers/paths и обязательное подтверждение.
- Разрастание CLI config. Решение: один builder-flow и один публичный сценарий `agentflow use <template>`.

## Рекомендуемый стартовый набор зависимостей

Для MVP:

```text
github.com/spf13/cobra
github.com/charmbracelet/huh
gopkg.in/yaml.v3
github.com/pelletier/go-toml/v2
```

Добавить после появления соответствующей необходимости:

```text
github.com/invopop/jsonschema
github.com/tailscale/hujson
github.com/charmbracelet/lipgloss
```

Не добавлять сразу:

```text
github.com/spf13/viper
github.com/santhosh-tekuri/jsonschema/v6
```

Они полезны, но на старте увеличат поверхность проекта без обязательной пользы.
