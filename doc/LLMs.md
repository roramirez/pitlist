# pitlist — LLM Quick Reference

CLI/TUI todo list and activity logger built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and YAML file storage.
Full spec: `SPEC.md`. This file is the dense implementation reference for LLMs.

## File Map

| File | What lives here |
|---|---|
| `cmd/pitlist/main.go` | Binary entry point; calls `cmd.NewRootCmd().Execute()` |
| `internal/cmd/root.go` | `NewRootCmd`; `PersistentPreRunE` wires config + store + scope; `--demo` flag |
| `internal/cmd/add.go` | `pitlist add` subcommand |
| `internal/cmd/done.go` | `pitlist done` subcommand |
| `internal/cmd/edit.go` | `pitlist edit` subcommand |
| `internal/cmd/delete.go` | `pitlist delete` subcommand |
| `internal/cmd/carry.go` | `pitlist carry` subcommand |
| `internal/cmd/list.go` | `pitlist list` subcommand |
| `internal/cmd/show.go` | `pitlist show` subcommand |
| `internal/cmd/log.go` | `pitlist log` (activity log) subcommand |
| `internal/cmd/agenda.go` | `pitlist agenda` subcommand |
| `internal/cmd/stats.go` | `pitlist stats` subcommand |
| `internal/cmd/sync.go` | `pitlist sync` subcommand |
| `internal/cmd/schedule.go` | `pitlist schedule` subcommand |
| `internal/cmd/demoseed.go` | `pitlist demo-seed` subcommand |
| `internal/cmd/util.go` | Shared CLI helpers |
| `internal/config/config.go` | `Config`, `Profile`, `GitConfig`, `TUIConfig`; `Load`/`ApplyScope` |
| `internal/model/task.go` | `Task`, `DayPlan`, `FutureList`, `TaskStatus`, `Priority`, `ActivityRef` |
| `internal/model/activity.go` | `ActivityEntry`, `ActivityLog` |
| `internal/storage/storage.go` | `Store` interface; `TaskFilter`, `ActivityFilter` |
| `internal/storage/yaml_store.go` | `YAMLStore` — YAML-based `Store` implementation |
| `internal/storage/git.go` | `gitHelper` — auto-commit data files after writes |
| `internal/tui/app.go` | `App` — root Bubble Tea model; tab routing; `Run` entry point |
| `internal/tui/keys.go` | `keyMap` — keybindings |
| `internal/tui/styles.go` | Top-level `lipgloss.Style` vars for the app shell |
| `internal/tui/views/tasks.go` | `TasksView` — tasks tab; add/edit/done/carry/notes/log-activity forms |
| `internal/tui/views/activity.go` | `ActivityView` — activity log tab; add/delete entries |
| `internal/tui/views/agenda.go` | `AgendaView` — multi-day agenda view |
| `internal/tui/views/future.go` | `FutureView` — backlog/future tasks tab |
| `internal/tui/views/search.go` | `SearchView` — full-text search across tasks and activity |
| `internal/tui/views/filter.go` | `FilterView` — overlay filter for the tasks tab |
| `internal/tui/views/styles.go` | Shared `lipgloss.Style` vars used across all views |
| `internal/demo/seed.go` | `Seed` — creates a temp dir with pre-seeded demo data |

## Core Types

```go
// App (internal/tui/app.go)
type App struct {
    store        *storage.YAMLStore
    activeTab    tab        // tabTasks | tabActivity | tabAgenda | tabSearch | tabFuture
    tasksView    views.TasksView
    activityView views.ActivityView
    agendaView   views.AgendaView
    searchView   views.SearchView
    futureView   views.FutureView
    filterView   views.FilterView
    filterMode   bool
    width        int
    height       int
}

// model.Task (internal/model/task.go)
type Task struct {
    ID           string
    Title        string
    Context      string        // work | personal | other (or custom)
    Notes        string
    Labels       []string
    Status       TaskStatus    // todo | in_progress | done | cancelled
    Priority     Priority      // low | medium | high
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DoneAt       *time.Time
    DueDate      string        // YYYY-MM-DD
    CarryFrom    string        // source date when carried
    CarryTo      string        // target date when carried
    ActivityRefs []ActivityRef
}

// model.ActivityEntry (internal/model/activity.go)
type ActivityEntry struct {
    ID          string
    Timestamp   time.Time
    Description string
    Tags        []string
    TaskRef     string
    DurationMin int
}

// storage.Store interface (internal/storage/storage.go)
type Store interface {
    GetDayPlan(date time.Time) (*model.DayPlan, error)
    SaveDayPlan(plan *model.DayPlan) error
    GetTaskByID(id string) (*model.Task, time.Time, error)
    ListTasks(filter TaskFilter) ([]*model.Task, error)
    GetActivityLog(date time.Time) (*model.ActivityLog, error)
    SaveActivityLog(log *model.ActivityLog) error
    ListActivity(filter ActivityFilter) ([]*model.ActivityEntry, error)
    GetActivitiesByRefs(refs []model.ActivityRef, fallbackDate time.Time) ([]*model.ActivityEntry, error)
    AddActivityRefToTask(taskID string, ref model.ActivityRef) error
}

// config.Config (internal/config/config.go)
type Config struct {
    DataDir   string
    Editor    string
    WeekStart string             // "monday" | "sunday"
    Contexts  []string
    Git       GitConfig          // AutoCommit bool
    TUI       TUIConfig          // ShowDoneTasks bool
    Profiles  map[string]Profile // named scopes
}
```

## Key Invariants

- `PersistentPreRunE` in `internal/cmd/root.go` is the single place where config is loaded, scope is applied, and the store is opened — do not inline this in subcommands.
- All `tea.Msg` types are defined at the top of each view file (`TasksMsg`, `ActivityMsg`, `FutureMsg`, etc.); do not define them in `app.go`.
- All shared `lipgloss.Style` vars for views live in `internal/tui/views/styles.go`; app-shell styles live in `internal/tui/styles.go`. Do not create styles inline in `View()` functions.
- Data is stored as YAML under `DataDir/days/YYYY-MM-DD.yaml` (day plans) and `DataDir/activity/YYYY-MM-DD.yaml` (activity logs). Never hardcode these paths outside of `yaml_store.go`.
- After every `SaveDayPlan` or `SaveActivityLog`, `gitHelper.autoCommit` runs silently — callers must not git-commit themselves.
- `FutureList` is stored in a single `DataDir/future.yaml` file, not per-day.
- `ApplyScope` must be called after `config.Load()` before opening the store — the scope may change `DataDir`.
- Views communicate upward via `tea.Msg`; cross-view navigation messages (`AgendaNavigateMsg`, `SearchNavigateTaskMsg`, etc.) are handled in `App.Update`, not inside view `Update` methods.

## Code Style

- Run `gofmt -l -w .` before every commit — all code must be `gofmt`-clean.
- Never manually align struct fields, `case` blocks, or function args; let `gofmt` decide.
- All code must pass `go vet ./...` with zero errors.
- No exported symbols without a doc comment.
- Prefer early returns over deep nesting to keep cognitive complexity low.
- Named constants over magic literals — one constant per repeated value, even if it's just `2`.

## Testing

- Every feature or bug fix must include tests covering the new or changed behavior.
- Tests live in `*_test.go` files in the same package.
- Run `go test -race ./...` before reporting a task complete.
- Use table-driven tests for functions with multiple input variants.
- For TUI model changes: test `Update` with the relevant `tea.Msg`, not just `View`.

## Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/) with these scopes:

| Scope | When |
|---|---|
| `cli` | Changes under `cmd/` or `internal/cmd/` |
| `tui` | Changes under `internal/tui/` (excluding views) |
| `views` | Changes under `internal/tui/views/` |
| `model` | Changes under `internal/model/` |
| `storage` | Changes under `internal/storage/` |
| `config` | Changes under `internal/config/` |
| `demo` | Changes under `internal/demo/` |
| `pitlist` | Cross-cutting or repo-level changes |

Every commit that adds a user-visible change must include an entry in `CHANGELOG.md` under `## [Unreleased]`.

## Build & Check

```sh
make build    # compile binary
make test     # go test -race ./...
make fmt      # gofmt -w ./...
make vet      # go vet ./...
make check    # fmt + vet + verify + test (run before committing)
make lint     # golangci-lint run ./...
```

Use `/commit` skill to validate, compose, and push a commit — it runs `gofmt`, `go vet`, and `go test -race` automatically.

## Code Quality Gates (kimun)

`km` ([kimun](https://github.com/lnds/kimun)) — static + git analysis. Install: `cargo install --git https://github.com/lnds/kimun`. Config: `.kimun.toml` at repo root. Score target: see `fail_below` there.

Before every commit:
```sh
km score --trend origin/main --fail-if-worse
```

Before touching a file:
```sh
km hotspots    # high churn × complexity files
km knowledge   # bus-factor risk (>80% single author)
```

In PR context:
```sh
km score diff main   # per-dimension delta; negative deltas need justification
```

Do not let the score drop below **A+**.

Key drivers for regressions:
- **Cognitive complexity** — extract helpers when a function branches more than ~10 times.
- **Halstead effort** — replace repeated literals with named constants; reduce operand vocabulary.
- **Dead code** — remove unused vars and blank identifiers immediately.
