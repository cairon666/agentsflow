# Agentsflow manifest: ownership, updates, and overwrite safety

Дата анализа: 2026-05-13.

## Контекст

`agentsflow use` генерирует файлы для разных CLI-инструментов:

- Codex: `AGENTS.md`, `.codex/config.toml`, `.codex/agents/*.toml`;
- Claude Code: `CLAUDE.md`, `.claude/settings.json`, `.claude/agents/*.md`;
- OpenCode: `AGENTS.md`, `opencode.json`, `.opencode/agents/*.md`.

Сейчас adapters собирают `desired` файлы и одновременно помечают все эти пути как
managed. Затем `internal/install` использует этот флаг для выбора между
`update` и `conflict`.

Ключевой риск: путь считается managed только потому, что текущий запуск хочет его
записать. Это не доказывает, что файл был создан `agentsflow` раньше.

## Что значит managed file

Managed file - это файл, которым `agentsflow` доказуемо управляет.

Практический смысл:

- если файл отсутствует, его можно создать;
- если файл был создан `agentsflow` раньше и не менялся вручную, его можно
  обновить при повторном запуске;
- если файл существует, но `agentsflow` не знает его происхождение, это конфликт;
- если файл был создан `agentsflow`, но пользователь изменил его вручную, это
  тоже конфликт или отдельный режим merge, но не молчаливое обновление.

Managed не должен означать "adapter хочет записать этот path". Это разные вещи:

- desired path - желаемый результат текущего render;
- managed path - существующий файл, владение которым подтверждено историей
  предыдущих записей.

## Текущая проблема в коде

В текущей реализации adapters делают примерно одно и то же:

```go
desired := map[string][]byte{}
managedPaths := map[string]struct{}{}
addDesired := func(path string, content []byte) {
    desired[path] = content
    managedPaths[path] = struct{}{}
}
```

После этого `BuildPlanWithManagedPaths` классифицирует существующий файл как
`update`, если path присутствует в `managedPaths`:

```go
case err == nil && managed:
    action.Kind = ActionUpdate
```

Следствие: если в проекте уже был пользовательский `AGENTS.md`, первый запуск
`agentsflow use` может запланировать `update`, а не `conflict`. Пользователь
увидит summary и подтвердит запись, но инструмент не подсветит, что это был
чужой файл.

Это особенно опасно для текстовых instruction-файлов:

- `AGENTS.md`;
- `CLAUDE.md`;
- agent markdown/toml files, если пользователь уже создал их руками;
- любые будущие файлы, где agentsflow не делает точечный merge.

## Нужен ли manifest инструменту

Manifest нужен только если `agentsflow` должен поддерживать безопасный повторный
запуск и обновление ранее сгенерированных файлов.

Если продуктовая модель проще, например "одноразовый генератор стартовых
конфигов", manifest не обязателен. Тогда безопасная стратегия такая:

- файла нет - create;
- файл есть и содержимое совпадает - skip;
- файл есть и содержимое отличается - conflict;
- никаких managed paths;
- config-файлы можно обрабатывать отдельным merge-режимом.

То есть manifest не является обязательной частью MVP. Но текущий код уже
использует идею managed-файлов, только без надежного источника правды. Поэтому
есть два корректных направления:

1. Убрать managed-поведение из MVP и всегда конфликтовать с неизвестными
   существующими файлами.
2. Реализовать manifest и считать managed только то, что подтверждено manifest.

## Где хранить manifest

Manifest принадлежит `agentsflow`, а не Codex, Claude Code или OpenCode. Поэтому
его не нужно класть в `.codex`, `.claude` или `.opencode`.

Для project scope:

```text
<project>/.agentsflow/manifest.json
```

Плюсы:

- manifest версионируется вместе с проектом, если команда этого хочет;
- пути можно хранить относительными;
- повторный запуск в этом же проекте видит историю записей;
- не смешивается с конфигами target-инструментов.

Минусы:

- появляется служебная директория `.agentsflow`;
- нужно решить, коммитить ее или добавлять в `.gitignore`.

Для global scope:

```text
$XDG_CONFIG_HOME/agentsflow/manifest.json
```

Fallback:

```text
$HOME/.config/agentsflow/manifest.json
```

Причина: global manifest описывает глобальные файлы пользователя, поэтому он
должен жить в глобальном config location самого `agentsflow`.

## Формат manifest

Manifest должен быть версионированным и достаточно простым для ручной проверки.
JSON подходит лучше, чем YAML, потому что это машинный state, а не авторский DSL.

Рекомендуемый формат:

```json
{
  "version": 1,
  "files": {
    ".codex/agents/reviewer.toml": {
      "target": "codex",
      "scope": "project",
      "strategy": "owned",
      "sha256": "sha256:3a4f...",
      "template_id": "default",
      "template_version": 1,
      "updated_at": "2026-05-13T12:00:00Z"
    },
    ".codex/config.toml": {
      "target": "codex",
      "scope": "project",
      "strategy": "merge",
      "managed_keys": [
        "model",
        "model_reasoning_effort",
        "plan_mode_reasoning_effort",
        "features.multi_agent",
        "agents.max_threads",
        "agents.max_depth"
      ],
      "template_id": "default",
      "template_version": 1,
      "updated_at": "2026-05-13T12:00:00Z"
    }
  }
}
```

Поля:

- `version` - версия формата manifest.
- `files` - map по нормализованному path.
- `target` - `codex`, `claude`, `opencode`.
- `scope` - `project` или `global`.
- `strategy` - способ управления файлом.
- `sha256` - checksum последнего содержимого, записанного `agentsflow`.
- `managed_keys` - список ключей для merge-файлов.
- `template_id` и `template_version` - источник генерации.
- `updated_at` - полезно для диагностики, но не должно участвовать в логике
  безопасности.

Для project scope `path` лучше хранить относительным от project root. Для global
scope можно хранить path относительно home/config root, но в runtime все равно
нужно нормализовать absolute path перед записью.

## Стратегии файлов

### owned

Файл полностью принадлежит `agentsflow`.

Подходит для:

- `.codex/agents/*.toml`;
- `.claude/agents/*.md`;
- `.opencode/agents/*.md`;
- возможно `AGENTS.md` или `CLAUDE.md`, если пользователь явно согласился, что
  agentsflow владеет этим файлом.

Алгоритм:

1. Если файла нет, создать и записать checksum в manifest.
2. Если файл есть и отсутствует в manifest, вернуть conflict.
3. Если файл есть в manifest и checksum текущего файла совпадает с manifest,
   разрешить update.
4. Если файл есть в manifest, но checksum отличается, вернуть conflict.

### merge

Файл не принадлежит `agentsflow` полностью. Инструмент управляет только
конкретными ключами.

Подходит для:

- `.codex/config.toml`;
- `.claude/settings.json`;
- `opencode.json`.

Алгоритм:

1. Прочитать существующий config, если он есть.
2. Распарсить формат.
3. Изменить только ключи, которыми управляет adapter.
4. Сохранить остальные пользовательские ключи.
5. Если файл невозможно распарсить, вернуть conflict.

Для merge-файлов checksum всего файла менее полезен, потому что пользователь
может менять соседние ключи. В manifest лучше хранить `managed_keys` и, при
необходимости, checksum только управляемого projection.

### create-only

Файл создается только если отсутствует.

Подходит для:

- `AGENTS.md`;
- `CLAUDE.md`;
- любые shared instruction-файлы, где риск перезаписать авторский текст выше
  пользы от автоматического update.

Алгоритм:

1. Если файла нет, создать.
2. Если файл есть и совпадает с desired, skip.
3. Если файл есть и отличается, conflict.

Для такого режима manifest не обязателен.

## Алгоритм планирования с manifest

Предлагаемый flow:

1. Adapter возвращает desired files и file strategy для каждого path.
2. App или install layer загружает manifest для выбранного scope.
3. Planner классифицирует каждый desired file:
   - create;
   - update;
   - skip;
   - conflict.
4. Writer применяет только create/update.
5. После успешной записи writer обновляет manifest.
6. Если хотя бы одна запись не удалась, manifest нельзя обновлять так, будто
   весь plan применен успешно.

Важно: manifest должен обновляться после фактической записи файлов, а не на
этапе render. Render не должен иметь side effects.

## Изменения в архитектуре

Сейчас adapters возвращают `install.Plan` напрямую. Для manifest-логики лучше
разделить render и planning:

```go
type RenderedFile struct {
    Path     string
    Content  []byte
    Strategy FileStrategy
}

type FileStrategy string

const (
    StrategyOwned      FileStrategy = "owned"
    StrategyMerge      FileStrategy = "merge"
    StrategyCreateOnly FileStrategy = "create_only"
)
```

Тогда adapter отвечает за target-specific render и выбор стратегии, а
`install.Planner` отвечает за безопасность записи с учетом manifest.

Минимальная альтернатива без большой перестройки:

- оставить `install.Plan`;
- убрать `managedPaths` из adapters;
- добавить отдельный тип или callback для merge-файлов;
- manifest отложить.

## MVP-рекомендация

Для текущего состояния инструмента лучше не начинать с manifest. Сначала стоит
исправить небезопасное поведение:

1. Убрать безусловное добавление каждого desired path в `managedPaths`.
2. Для generated agent files использовать create/skip/conflict.
3. Для `AGENTS.md` и `CLAUDE.md` использовать create-only.
4. Для `.codex/config.toml`, `.claude/settings.json`, `opencode.json` сохранить
   merge-поведение, но не называть это managed ownership.
5. Добавить тесты:
   - существующий пользовательский `AGENTS.md` дает conflict;
   - существующий пользовательский agent file дает conflict;
   - существующий config file сохраняет чужие ключи;
   - `--yes` не обходит conflict.

Manifest стоит добавлять следующим этапом, когда появится явный сценарий:

```text
agentsflow use template.yaml
# затем template изменился
agentsflow use template.yaml
```

и инструмент должен обновлять ранее установленные файлы без ручного удаления.

## Когда manifest точно понадобится

Manifest становится полезным, если появляются требования:

- обновлять сгенерированные файлы после изменения template;
- удалять файлы, которые больше не генерируются template;
- показывать пользователю, какие файлы принадлежат agentsflow;
- поддерживать команду `agentsflow uninstall`;
- поддерживать команду `agentsflow status`;
- различать "пользователь изменил файл" и "template изменился";
- безопасно управлять несколькими targets в одном проекте.

Без этих сценариев manifest может быть лишней сложностью.

## Итог

Главная проблема не в отсутствии `.agentsflow/manifest.json`. Главная проблема в
том, что текущий код уже использует managed-поведение без источника истины.

Для MVP лучше выбрать более простую и безопасную модель:

- не перезаписывать неизвестные существующие файлы;
- мержить только config-файлы с понятным списком ключей;
- конфликтовать с пользовательским текстом;
- не вводить ownership, пока нет сценария повторного обновления.

Если в продукте нужен повторный update/install/uninstall, тогда manifest нужен.
Хранить его стоит в `.agentsflow/manifest.json` для project scope и в
`$XDG_CONFIG_HOME/agentsflow/manifest.json` для global scope.
