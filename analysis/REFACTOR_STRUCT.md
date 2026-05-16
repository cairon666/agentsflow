# Refactor structure plan

Цель документа: зафиксировать текущую связанность пакетов и дать последовательный
план снижения связанности маленькими этапами. План рассчитан на серию небольших
PR, где каждый шаг сохраняет текущее поведение `agentsflow use`.

Документ дополняет:

- `analysis/ARCHITECTURY.md` - целевая идея `Template DSL -> Normalized IR -> Target Adapter -> Rendered Files -> Install Plan`;
- `analysis/MANIFEST.md` - отдельная тема безопасного владения файлами и повторных запусков;
- `analysis/CLI_CONFIGS.md` - target-specific различия Codex, Claude Code и OpenCode.

## Текущее состояние

Основной execution path:

```text
cmd/agentsflow/main.go
  -> internal/cli.NewRootCommand
  -> internal/cli.newUseCommand
  -> internal/app.App.Use
  -> source resolver
  -> template loader
  -> schema validation
  -> schema.ToIR
  -> builder.Run
  -> adapter.Validate / adapter.Render
  -> install.Plan
  -> install.Writer.Apply
```

Ключевые файлы:

- `cmd/agentsflow/main.go` - точка входа.
- `internal/cli/root.go` - Cobra root, регистрация adapters, сборка `app.App`.
- `internal/cli/use.go` - flags и bridge к prompter.
- `internal/app/usecase_use.go` - главный use case и orchestration.
- `internal/app/source.go` - адаптация template chooser между `app`, `source` и `builder`.
- `internal/builder/flow.go` - сбор пользовательских choices, history output, install summary.
- `internal/source/resolver.go` - local/git source resolution, loading spinner, history output.
- `internal/template/*` - YAML форма flow.
- `internal/schema/*` - validation, capabilities, conversion to IR.
- `internal/ir/flow.go` - normalized flow.
- `internal/binding/*` - target/scope/model choices.
- `internal/adapter/*` - target contract and registry.
- `internal/adapter/{codex,claude,opencode}` - target renderers.
- `internal/install/*` - install plan classification and filesystem writer.
- `internal/console/*` - banner, loading spinner, persistent history output.

## Основные проблемы связанности

### 1. `internal/app` слишком много знает

`internal/app/usecase_use.go` одновременно:

- вызывает template source resolver;
- загружает YAML template;
- запускает schema validation;
- конвертирует template в IR;
- вызывает interactive builder;
- выбирает adapter;
- печатает diagnostics;
- строит render input;
- обрабатывает target diagnostics;
- печатает install summary;
- подтверждает запись;
- применяет install plan;
- пишет banner/history.

Это нарушает SRP: use case содержит и application orchestration, и terminal UI,
и formatting decisions. Также страдает DIP: `app` зависит от concrete packages
`console`, `schema`, `template`, `builder`, `source`, `install`, `adapter`.

### 2. Console/history протекли в доменную и инфраструктурную логику

Примеры:

- `internal/app/usecase_use.go` напрямую вызывает `console.WrintBanner` и
  `console.NewHistoryWriter`.
- `internal/builder/flow.go` принимает `io.Writer` и пишет history lines.
- `internal/source/resolver.go` напрямую использует `console.RunWithLoading` и
  пишет `Source: ...` в history.

Проблема не в самом `console`, а в направлении зависимости. `builder` должен
собирать решения пользователя, `source` должен резолвить source, а terminal
history является presentation concern.

### 3. Flow-схема размазана по нескольким пакетам

Сейчас одна концепция flow разделена так:

- YAML shape: `internal/template/model.go`;
- decode/load: `internal/template/decoder.go`, `internal/template/loader.go`;
- validation: `internal/schema/validate.go`;
- capabilities dictionary: `internal/schema/capabilities.go`;
- conversion: `internal/schema/convert.go`;
- normalized model: `internal/ir/flow.go`;
- model fallback resolution: частично `internal/binding/models.go`, частично
  `internal/render/files.go`.

Разделение template/spec и normalized IR само по себе нормальное, но сейчас нет
одного фасада, который отвечает за жизненный цикл flow: load -> validate ->
normalize -> resolve model. Из-за этого `app` вручную склеивает внутренние шаги.

### 4. Builder смешивает разные ответственности

`internal/builder/flow.go` делает три разных вида работы:

- собирает `Choices`;
- пишет terminal history;
- форматирует summary для `install.Plan`.

Из-за этого builder зависит от `console` и `install`. Для SOLID лучше оставить
builder только как choice collector, а install summary перенести в presentation
или install-report слой.

### 5. Adapter зависит от install planning

Интерфейс:

```go
Render(context.Context, RenderInput) (install.Plan, []diagnostic.Diagnostic)
```

Target adapters сразу возвращают `install.Plan` и вызывают
`install.BuildPlanWithManagedPaths`. Это связывает target rendering с политикой
обновления файлов, conflict detection и будущим manifest behavior.

Более чистое разделение:

```text
target adapter -> desired artifacts
install planner -> plan create/update/skip/conflict
install writer  -> apply approved plan
```

### 6. Target registration зашита в CLI composition

`internal/cli/root.go` импортирует конкретные adapters:

- `internal/adapter/codex`;
- `internal/adapter/claude`;
- `internal/adapter/opencode`.

Для текущего размера проекта это приемлемо, но при росте target-ов лучше
выделить composition package, чтобы Cobra не знал target implementation details.

## Целевые принципы

### Dependency direction

Желаемое направление зависимостей:

```text
cmd
  -> cli
  -> composition
  -> application use cases
  -> domain contracts

terminal/ui adapters -> application ports
source adapters      -> application ports
target renderers     -> application/domain contracts
filesystem adapters  -> application ports
```

Domain/application не должны импортировать terminal UI, Cobra, huh, spinner,
filesystem writer implementation или конкретные target adapters.

### Разделение слоев

```text
CLI layer
  Cobra commands, flags, error usage formatting.

Terminal UI layer
  huh prompts, banner, spinner, history, user-facing formatting.

Application layer
  UseFlow use case, orchestration, ports, transaction boundaries.

Flow domain layer
  Template spec, validation, normalized flow, capabilities, model resolution.

Target rendering layer
  Codex/Claude/OpenCode mappers from flow to desired files.

Install layer
  Desired files, plan classification, conflict policy, writer.

Filesystem/source layer
  Local path resolution, git clone, config file reading, atomic writes.
```

### SOLID ориентиры

- SRP: каждый пакет имеет одну причину для изменения.
- OCP: новый target добавляется новым adapter package и регистрацией, без
  переписывания use case.
- LSP: все target renderers должны одинаково соблюдать contract: validate,
  render artifacts, return diagnostics.
- ISP: ports должны быть маленькими: `Reporter`, `ChoiceCollector`,
  `TemplateSource`, `TargetRenderer`, `InstallPlanner`, `InstallWriter`.
- DIP: use case зависит от interfaces, implementations находятся снаружи.

## Целевая структура после серии этапов

Это не нужно делать одним PR. Это ориентир, к которому этапы ниже постепенно
ведут проект.

```text
cmd/agentsflow/
  main.go

internal/cli/
  root.go
  use.go

internal/composition/
  app.go
  targets.go

internal/app/
  use_flow.go
  ports.go
  events.go

internal/flow/
  spec.go
  flow.go
  loader.go
  validate.go
  normalize.go
  capabilities.go
  models.go

internal/choices/
  choices.go
  targets.go
  scope.go

internal/ui/terminal/
  reporter.go
  prompter.go
  loading.go
  history.go
  summary.go

internal/source/
  resolver.go
  git.go

internal/target/
  renderer.go
  registry.go
  artifacts.go

internal/target/codex/
  adapter.go
  config.go
  permissions.go

internal/target/claude/
  adapter.go
  settings.go
  permissions.go

internal/target/opencode/
  adapter.go
  config.go
  permissions.go

internal/install/
  desired.go
  plan.go
  planner.go
  writer.go
  summary.go

internal/render/
  json.go
  toml.go
  markdown.go

internal/diagnostic/
  diagnostic.go
```

Названия можно уточнить в процессе. Важно не переименование само по себе, а
направление зависимостей.

## Последовательный план

### Этап 0. Зафиксировать baseline и границы поведения

Цель: перед refactor иметь быстрый способ понять, что поведение не изменилось.

Изменения:

- Не менять production code.
- Добавить или уточнить characterization tests только там, где уже видны риски:
  history output, source selection, install summary, target rendering.
- Зафиксировать текущее поведение `agentsflow use` с flags:
  `--target`, `--bind`, `--scope`, `--yes`.

Затрагиваемые файлы:

- `internal/app/usecase_use_test.go`;
- `internal/builder/flow_test.go`;
- `internal/source/resolver_test.go`;
- `internal/cli/use_test.go`;
- target adapter tests при необходимости.

Acceptance criteria:

- Тесты описывают существующее поведение, а не новую архитектуру.
- Нет package moves и broad renames.
- `task test` проходит.

Риск:

- Можно случайно закрепить неудачную внутреннюю реализацию. Тестировать нужно
  observable behavior, а не конкретный пакетный дизайн.

### Этап 1. Ввести порт Reporter для console/history

Цель: убрать прямой доступ к `console` из use case, builder и source.

Минимальный контракт:

```go
type Reporter interface {
    Banner() error
    Historyf(format string, args ...any) error
    HistorySpace() error
    Message(args ...any) error
    MessageLine(args ...any) error
    Diagnostics([]diagnostic.Diagnostic) error
}
```

Можно начать с меньшего интерфейса, если это уменьшает diff:

```go
type HistoryReporter interface {
    Historyf(format string, args ...any) error
    HistorySpace() error
}
```

Изменения:

- Создать terminal implementation поверх `internal/console`.
- `app.App` получает `Reporter` вместо того, чтобы напрямую вызывать
  `console.NewHistoryWriter`.
- `builder.Run` перестает принимать `io.Writer`; вместо этого:
  - либо возвращает choices events;
  - либо принимает маленький `HistoryReporter`;
  - предпочтительно первый вариант, но второй меньше по diff.
- `source.DefaultResolver` перестает импортировать `console`.
- Loading spinner вынести за порт:

```go
type LoadingRunner interface {
    Run(context.Context, string, func(context.Context) error) error
}
```

или оставить существующий `source.LoadingRunner`, но implementation передавать
из composition layer.

Затрагиваемые файлы:

- `internal/app/app.go`;
- `internal/app/usecase_use.go`;
- `internal/builder/flow.go`;
- `internal/source/resolver.go`;
- `internal/console/*`;
- tests для app/builder/source.

Acceptance criteria:

- `rg 'internal/console' internal/app internal/builder internal/source` показывает
  только временно допустимые места или ничего.
- History output в CLI остается прежним.
- `source` можно тестировать без terminal console.
- `task test` проходит.

Риск:

- Если сразу сделать слишком общий `Reporter`, он станет новым God interface.
  Держать методы маленькими и добавлять только реально используемые.

### Этап 2. Разделить choice collection и install summary

Цель: убрать зависимость `builder -> install`.

Текущая проблема:

- `builder.Summary(plan install.Plan)` форматирует install plan.
- Builder из-за этого знает install action kinds.

Изменения:

- Перенести summary formatter в `internal/install` или terminal UI layer.
- Например:

```go
package install

func FormatSummary(plan Plan) string
```

или:

```go
package terminal

func InstallSummary(plan install.Plan) string
```

Предпочтение:

- Если summary является neutral text representation of install plan, держать в
  `internal/install`.
- Если summary зависит от terminal UX, держать в `internal/ui/terminal`.

Затрагиваемые файлы:

- `internal/builder/flow.go`;
- `internal/builder/flow_test.go`;
- `internal/install/summary.go` или `internal/ui/terminal/summary.go`;
- `internal/app/usecase_use.go`.

Acceptance criteria:

- `builder` больше не импортирует `install`.
- `builder` отвечает только за choices.
- Existing summary text не меняется без отдельного решения.
- `task test` проходит.

### Этап 3. Ввести flow facade без массового переноса файлов

Цель: `app` должен вызывать один flow API вместо ручной склейки
`template.LoadFile -> schema.Validate -> schema.ToIR`.

Новый фасад:

```go
type LoadResult struct {
    Flow        ir.Flow
    Diagnostics []diagnostic.Diagnostic
}

func LoadFile(path string) (LoadResult, error)
```

На первом шаге фасад может жить в новом `internal/flow` и использовать старые
пакеты внутри:

```go
raw, err := template.LoadFile(path)
diags := schema.Validate(raw)
return LoadResult{Flow: schema.ToIR(raw), Diagnostics: diags}, err
```

Изменения:

- Создать `internal/flow/loader.go`.
- Перевести `app.Use` на `flow.LoadFile`.
- Не переносить сразу `template`, `schema`, `ir`. Это отдельный этап.

Затрагиваемые файлы:

- `internal/flow/loader.go`;
- `internal/app/usecase_use.go`;
- tests app/schema.

Acceptance criteria:

- `app` больше не импортирует `internal/template` и `internal/schema` напрямую.
- Validation behavior не меняется.
- `task test` проходит.

Риск:

- Новый фасад может стать просто еще одним слоем без пользы. Его задача
  конкретная: убрать знание pipeline из `app`.

### Этап 4. Сконцентрировать flow model и model resolution

Цель: убрать дублирование fallback logic и сделать flow единственным владельцем
своих правил.

Текущие места:

- `binding.Models.Resolve`;
- `render.ModelFor`;
- `render.Fallbacks`;
- `ir.Flow.ModelSlots`.

Изменения:

- Добавить метод или функцию в flow/domain layer:

```go
func ResolveModel(flow ir.Flow, models binding.Models, agent ir.Agent) string
```

или после переноса типов:

```go
func (f Flow) ResolveAgentModel(agentID string, models choices.Models) string
```

- Перевести adapters на единый resolver.
- Удалить дублирующие функции после миграции.

Затрагиваемые файлы:

- `internal/render/files.go`;
- `internal/binding/models.go`;
- `internal/adapter/{codex,claude,opencode}/adapter.go`;
- новый `internal/flow/models.go`.

Acceptance criteria:

- В проекте остается один implementation fallback resolution.
- Adapter tests остаются зелеными.
- `task test` проходит.

### Этап 5. Разделить target render artifacts и install plan

Цель: adapters должны возвращать желаемые файлы, а не `install.Plan`.

Новый контракт:

```go
type DesiredFile struct {
    Path     string
    Content  []byte
    Strategy install.Strategy
}

type ArtifactSet struct {
    Target string
    Scope  binding.Scope
    Files  []DesiredFile
}

type Renderer interface {
    Target() binding.Target
    Aliases() []string
    Validate(context.Context, ir.Flow) []diagnostic.Diagnostic
    Render(context.Context, RenderInput) (ArtifactSet, []diagnostic.Diagnostic)
}
```

Минимальный промежуточный вариант:

- Оставить старый `adapter.Adapter` interface.
- Добавить новый `Artifacts` type.
- Перевести один adapter, например OpenCode.
- После проверки перевести Codex и Claude.
- Затем поменять app orchestration:

```text
adapter.Render -> artifacts
install.Planner.Build(artifacts) -> plan
writer.Apply(plan)
```

Затрагиваемые файлы:

- `internal/adapter/adapter.go`;
- `internal/adapter/{codex,claude,opencode}/adapter.go`;
- `internal/install/planner.go`;
- `internal/install/plan.go`;
- `internal/app/usecase_use.go`;
- adapter/install tests.

Acceptance criteria:

- Target adapters не вызывают `install.BuildPlanWithManagedPaths`.
- Install planner единственный классифицирует create/update/skip/conflict.
- Behavior текущих generated files не меняется.
- `task test` проходит.

Риск:

- Этот этап затрагивает публичный внутренний contract adapters. Делать его после
  этапов 1-4, когда orchestration уже чище.

### Этап 6. Ввести file strategy и подготовить manifest behavior

Цель: убрать ложную семантику `managedPaths`, где desired path считается managed
только потому, что adapter хочет его записать.

Связанный документ: `analysis/MANIFEST.md`.

Стратегии:

```go
type Strategy string

const (
    StrategyOwned      Strategy = "owned"
    StrategyMerge      Strategy = "merge"
    StrategyCreateOnly Strategy = "create-only"
)
```

Изменения:

- Desired artifacts получают strategy.
- Planner принимает strategy и классифицирует файл соответствующим образом.
- На первом шаге можно не внедрять manifest, но убрать опасное поведение
  `managedPaths`.

Рекомендуемое поведение MVP:

- `owned`: пока нет manifest, трактовать как create-only для существующего
  отличающегося файла, либо явно назвать это temporary behavior.
- `merge`: adapter/config merger может читать existing config и изменять только
  свои keys.
- `create-only`: существующий отличающийся файл дает conflict.

Затрагиваемые файлы:

- `internal/install/plan.go`;
- `internal/install/planner.go`;
- `internal/adapter/*`;
- `internal/install/planner_test.go`;
- target adapter tests.

Acceptance criteria:

- Existing user file не становится update только из-за desired path.
- Tests явно покрывают unknown existing file conflict.
- `task test` проходит.

### Этап 7. Разделить config merge и target render

Цель: target adapters не должны напрямую читать filesystem там, где можно
разделить pure rendering и merge existing config.

Текущие примеры:

- `codex.mergedConfig` читает `config.toml`;
- `claude.mergedSettings` читает `settings.json`.

Возможные варианты:

1. Оставить merge в adapter, но передавать abstraction:

```go
type FileReader interface {
    ReadFile(path string) ([]byte, error)
}
```

2. Разделить:

```text
adapter declares desired managed keys
install/config merger reads existing file and applies patch
```

Практичный первый шаг:

- Вынести чтение файла из helper-функций в dependency.
- Сделать pure functions:

```go
func MergeCodexConfig(existing []byte, model string) ([]byte, error)
func MergeClaudeSettings(existing []byte, model string) ([]byte, error)
```

Затрагиваемые файлы:

- `internal/adapter/codex/adapter.go`;
- `internal/adapter/claude/adapter.go`;
- adapter tests.

Acceptance criteria:

- Merge functions тестируются без filesystem.
- Adapter filesystem access становится явно инжектируемым или ограниченным.
- `task test` проходит.

### Этап 8. Сжать `internal/app` до application use case с ports

Цель: `app.Use` зависит от маленьких interfaces, а concrete implementations
собираются снаружи.

Ports:

```go
type TemplateSource interface {
    Resolve(context.Context, string, TemplateChooser) (ResolvedSource, error)
}

type ChoiceCollector interface {
    Collect(context.Context, flow.Flow, []TargetOption) (Choices, error)
    Confirm(context.Context, string) (bool, error)
}

type TargetRegistry interface {
    Resolve(string) (binding.Target, error)
    Get(string) (TargetRenderer, error)
    All() []TargetRenderer
}

type InstallPlanner interface {
    Build(ArtifactSet) (install.Plan, []diagnostic.Diagnostic)
}

type InstallWriter interface {
    Apply(install.Plan) error
}
```

Изменения:

- `App` хранит ports, а не concrete packages.
- `internal/cli/root.go` или `internal/composition/app.go` собирает concrete
  implementations.
- `app.Use` остается главным orchestration, но без terminal/filesystem деталей.

Затрагиваемые файлы:

- `internal/app/app.go`;
- `internal/app/usecase_use.go`;
- `internal/cli/root.go`;
- возможно новый `internal/composition/app.go`.

Acceptance criteria:

- `internal/app` не импортирует `console`, `template`, `schema`, concrete target
  adapters, huh или os.
- App tests используют fakes ports.
- `task test` проходит.

Риск:

- Не вводить ports заранее "на будущее". Каждый port должен закрывать реальную
  текущую зависимость.

### Этап 9. Вынести composition root из CLI

Цель: Cobra layer не должен знать concrete target adapters и filesystem wiring.

Изменения:

- Создать `internal/composition`.
- Перенести туда `newApp` из `internal/cli/root.go`.
- CLI вызывает:

```go
application := composition.NewApp(composition.Config{Stdout: os.Stdout})
```

или:

```go
root.AddCommand(newUseCommand(composition.NewUseFlow(...)))
```

Затрагиваемые файлы:

- `internal/cli/root.go`;
- новый `internal/composition/app.go`;
- `internal/cli/root_test.go`.

Acceptance criteria:

- `internal/cli` больше не импортирует `internal/adapter/codex`,
  `internal/adapter/claude`, `internal/adapter/opencode`.
- Добавление нового target меняет composition и новый package, а не Cobra use
  command.
- `task test` проходит.

### Этап 10. Физически собрать flow packages

Цель: после того как `app` уже использует facade, можно безопасно привести
структуру flow к более понятной форме.

Изменения:

- Перенести `template/model.go` в `flow/spec.go`.
- Перенести `schema/validate.go` и `schema/capabilities.go` в `flow`.
- Перенести `schema/convert.go` в `flow/normalize.go`.
- Либо оставить `ir.Flow` как `flow.Flow`, либо сделать type alias на
  переходный период.

Переходный прием:

```go
type Flow = ir.Flow
```

или наоборот, чтобы не ломать все adapters одним diff.

Затрагиваемые файлы:

- `internal/template/*`;
- `internal/schema/*`;
- `internal/ir/*`;
- `internal/flow/*`;
- imports в adapters/render/tests.

Acceptance criteria:

- Есть один публичный для internal-пакетов flow API.
- Старые пакеты удалены или оставлены только как временные aliases.
- `rg 'internal/schema|internal/template|internal/ir' internal` показывает
  ожидаемый переходный минимум.
- `task test` проходит.

Риск:

- Это самый шумный этап по imports. Делать только после того, как behavior уже
  стабилен и фасад используется.

### Этап 11. Переименовать builder в terminal choice collector

Цель: убрать неоднозначность слова `builder`.

Сейчас `builder` не строит flow и не строит files. Он собирает пользовательские
решения. Более точные имена:

- `internal/ui/terminal`;
- `internal/prompt`;
- `internal/choices/terminal`.

Изменения:

- Перенести `HuhPrompter` и related options в terminal UI package.
- Domain types `Choices`, `TargetOption`, `TemplateOption` держать отдельно от
  huh implementation.

Acceptance criteria:

- huh imports живут только в terminal UI package.
- Use case зависит от `ChoiceCollector`, а не от concrete `HuhPrompter`.
- `task test` проходит.

### Этап 12. Уточнить target extension API

Цель: новый CLI target добавляется предсказуемо и без изменений flow/usecase.

Target package должен реализовать:

```go
func New() target.Renderer
```

Renderer contract:

- metadata: canonical name, aliases, supported scopes;
- validate target-specific limitations;
- render desired artifacts;
- return warnings for lossy permission mapping.

Проверить:

- target package не импортирует `cli`, `console`, `cobra`, `huh`;
- target package не пишет файлы;
- target package не решает confirmation UX;
- target package не должен знать history.

Acceptance criteria:

- Добавление fake target в тесте требует только registration.
- `app` и `flow` не меняются при добавлении target.
- `task test` проходит.

## Рекомендуемый порядок PR

1. PR 1: baseline tests, если текущего покрытия недостаточно.
2. PR 2: Reporter/History port, убрать `console` из `builder` и `source`.
3. PR 3: перенести install summary из `builder`.
4. PR 4: добавить `internal/flow.LoadFile` facade и перевести `app`.
5. PR 5: единый model resolution.
6. PR 6: adapters возвращают artifacts, install planner строит plan.
7. PR 7: file strategy вместо `managedPaths`.
8. PR 8: pure config merge helpers для Codex/Claude.
9. PR 9: application ports и сжатие `internal/app`.
10. PR 10: composition root вне CLI.
11. PR 11: физическая сборка `flow` пакета.
12. PR 12: переименование `builder` в terminal choice collector.

Если хочется быстрее убрать самое болезненное:

1. Reporter/History port.
2. Summary out of builder.
3. Flow facade.
4. Artifacts before install plan.

Это даст основную пользу без большого package move.

## Проверки после каждого этапа

Минимально:

```sh
task test
```

Для meaningful structural changes:

```sh
task check
```

Точечные grep-проверки:

```sh
rg 'internal/console' internal/app internal/builder internal/source
rg 'BuildPlanWithManagedPaths' internal/adapter
rg 'internal/adapter/(codex|claude|opencode)' internal/cli
rg 'internal/schema|internal/template|internal/ir' internal/app
```

Ожидаемый тренд:

- `console` остается только в terminal UI implementation.
- `adapter` не вызывает install planner.
- `cli` не импортирует concrete target adapters.
- `app` не импортирует low-level UI, template/schema internals и concrete
  target packages.

## Что не делать одним этапом

- Не переносить все пакеты сразу в новую структуру.
- Не менять format generated files одновременно с архитектурным refactor.
- Не добавлять manifest в тот же PR, где меняется adapter contract.
- Не переименовывать `template/schema/ir` до появления flow facade.
- Не вводить большие interfaces заранее. Ports должны появляться из реальных
  текущих зависимостей.
- Не менять public CLI flags без отдельного продуктового решения.

## Итоговое целевое состояние

После выполнения плана:

- `internal/app` становится тонким application use case.
- Terminal output/history/loading живут в одном presentation adapter.
- Flow lifecycle имеет единый API.
- Model fallback resolution реализован в одном месте.
- Target adapters отвечают только за target-specific mapping и artifacts.
- Install layer единственный отвечает за conflict/update/write policy.
- CLI layer отвечает только за Cobra commands, flags и composition handoff.
- Новый target добавляется без изменений flow schema и без знания о console или
  install writer.
