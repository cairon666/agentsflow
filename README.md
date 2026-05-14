# Agentsflow

Инструмент для удобного переиспользования разных agent workflows для разных cli инструментов(codex, claude, opencode, etc) с разными моделями(claude, gpt, deepseek, etc)

## Install

После первого npm-релиза CLI можно будет установить так:

```sh
npm install -g agentsflow
agentsflow --version 
```

## Release

Релизы управляются через GitHub Actions:

- `release-please` создает release PR, changelog, tag `vX.Y.Z` и GitHub Release из Conventional Commits.
- Версия релиза хранится в `version.txt`; release PR обновляет этот файл.
- Release workflow собирает Go binary для macOS, Linux и Windows на `x64`/`arm64` прямо в статические npm packages из `npm/`.
- npm публикует основной пакет `agentsflow` и platform packages `agentsflow-<os>-<arch>`.

Для первого npm-релиза добавьте repository secret `NPM_TOKEN` с правами публикации. После первой публикации настройте Trusted Publisher на npmjs.com для всех npm-пакетов:

- owner: `cairon666`
- repository: `agentsflow`
- workflow filename: `release.yml`

После проверки OIDC-публикации `NPM_TOKEN` можно удалить.
