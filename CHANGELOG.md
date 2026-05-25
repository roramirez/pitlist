# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

### Fixed

- **Notes editor** — pressing `q` while typing a note no longer closes the app; only `ctrl+c` quits during active input

### Added

- **Named profiles / scopes** — define multiple profiles in `config.yaml` and activate one with `--scope <name>` or `PITLIST_SCOPE=<name>`; each scope has its own `data_dir`, `contexts`, and git history
- **Date keywords** — all `--date`, `--from`, and `--to` flags now accept natural-language words (`today`, `tomorrow`, `yesterday`, `next_week`, `last_week`, `in_a_week`, `next_month`, `last_month`, `in_a_month`, `monday`…`sunday`, `next_monday`…`next_sunday`) in addition to `YYYY-MM-DD`

## [0.1.0] - 2026-05-23

### Added

- **Task management** — create, edit, delete, and complete tasks from a TUI or CLI
- **Activity logging** — log time spent on tasks with duration and notes
- **YAML storage** — human-readable local data files, no database required
- **Fuzzy search** — filter tasks and activities with fuzzy matching
- **Demo mode** — `pitlist --demo` runs with seed data, no config required
- **`demo-seed` command** — generate a demo dataset for screenshots or testing
- **Config subcommand** — manage data directory and settings via `pitlist config`
- **Bubble Tea TUI** — keyboard-driven interface with tabs, list, and form views
- **Cobra CLI** — full command-line interface for scripting and automation

[Unreleased]: https://github.com/roramirez/pitlist/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/roramirez/pitlist/releases/tag/v0.1.0
