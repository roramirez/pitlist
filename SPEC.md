# pitlist — Specification

## Purpose

pitlist is a personal CLI/TUI productivity tool for daily task planning and activity tracking. It answers two questions:

- **What do I plan to do?** (tasks, organized by day)
- **What did I actually do?** (activity log, linked to tasks or standalone)

Storage is plain YAML — human-editable, git-versionable, no database required.

---

## Core Concepts

### Task

A unit of work planned for a specific day. Tasks live in the day they were planned for, not the day they were completed.

- Has a title, labels, priority, status, optional notes, optional due date
- ID is date-prefixed: `t-YYYYMMDD-NNN` (human-readable, usable in CLI without copy-paste)
- Carries `activity_refs` — a list of `{id, date}` pointers to linked activity entries (avoids full-scan lookups)

**Status values:** `todo` | `in_progress` | `done` | `cancelled`

**Priority values:** `low` | `medium` | `high`

### Activity Entry

A record of something that was done. Can be linked to a task or standalone.

- Has a description, tags, optional duration (minutes), optional task reference
- Timestamp is calculated as `now - duration` so the start time is recorded, not the end time
- For past days: timestamp anchors to `end-of-day - duration`
- ID is date-prefixed: `a-YYYYMMDD-NNN`

### Carry

Moving a task from one day to another. Carry:

1. Removes the task from the source day file
2. Adds it to the destination day file with the **same ID**
3. Writes an activity entry on the source day: `"Carried to YYYY-MM-DD: <title>"` with tag `carried`
4. Adds the carry activity to the task's `activity_refs`

This preserves the full history — the YAML diffs show exactly when and why a task moved.

---

## Storage

### Layout

```
~/pitlist/                    # data_dir (configurable)
├── days/
│   └── YYYY-MM-DD.yaml       # one file per day, tasks only
├── activity/
│   └── YYYY-MM-DD.yaml       # one file per day, activity entries only
└── .git/                     # auto-initialized on first write
```

### Day plan file (`days/YYYY-MM-DD.yaml`)

```yaml
date: "2026-05-18"
tasks:
  - id: "t-20260518-001"
    title: "Write RFC for auth redesign"
    notes: "Focus on token rotation section."
    labels: [work, auth]
    status: todo
    priority: high
    created_at: "2026-05-18T09:00:00Z"
    updated_at: "2026-05-18T09:00:00Z"
    done_at: null
    due_date: "2026-05-19"
    carry_from: ""
    carry_to: ""
    activity_refs:
      - id: "a-20260518-001"
        date: "2026-05-18"
      - id: "a-20260519-002"
        date: "2026-05-19"
```

### Activity log file (`activity/YYYY-MM-DD.yaml`)

```yaml
date: "2026-05-18"
entries:
  - id: "a-20260518-001"
    timestamp: "2026-05-18T09:45:00Z"
    description: "Deep dive into token refresh bug"
    tags: [work, debugging, auth]
    task_ref: "t-20260518-001"
    duration_min: 45
```

### Git auto-commit

Every write triggers:
1. `git init` (idempotent) in `data_dir`
2. `git add <changed file>`
3. `git commit -m "tasks: save YYYY-MM-DD"` or `"activity: save YYYY-MM-DD"`

Push is never automatic. Use `pitlist sync --push` to push to a remote.

---

## ID Generation

IDs encode the date to make them human-readable and usable in CLI arguments:

- Tasks: `t-YYYYMMDD-NNN` (e.g. `t-20260518-001`)
- Activities: `a-YYYYMMDD-NNN` (e.g. `a-20260518-003`)

`NNN` is a zero-padded sequence, derived by counting existing entries + 1.

---

## Date Keywords

Wherever a date flag is accepted (`--date`, `--from`, `--to`) you can use either `YYYY-MM-DD` or a natural-language keyword:

| Keyword | Resolves to |
|---------|-------------|
| `today` | current date |
| `tomorrow` | today + 1 day |
| `yesterday` | today − 1 day |
| `next_week` | Monday of next week |
| `last_week` | Monday of last week |
| `in_a_week` | today + 7 days (same weekday) |
| `next_month` | 1st of next month |
| `last_month` | 1st of last month |
| `in_a_month` | today + 30 days |
| `monday` … `sunday` | upcoming occurrence of that weekday (including today if it matches) |
| `next_monday` … `next_sunday` | strictly next occurrence (never today) |

Examples:

```bash
pitlist add "Prep for review" --date friday
pitlist carry t-20260519-001 --to next_monday
pitlist agenda --from today --to next_week
pitlist log list --date yesterday
```

---

## CLI Commands

All commands work without the TUI. The TUI launches when no subcommand is given.

### Tasks

```bash
pitlist add "Title" [--context work] [--label work] [--priority high] [--due YYYY-MM-DD] [--date <date>]
pitlist done <id>
pitlist list                          # today, todo + in_progress
pitlist list --label work             # all open work tasks across days
pitlist list --week                   # this week
pitlist list --status done
pitlist list --from <date> --to <date>
pitlist list --date <date>
pitlist show <id>
pitlist edit <id>                     # opens day file in $EDITOR
pitlist delete <id>                   # prompts confirmation
pitlist delete <id> --force
pitlist carry <id>                    # carries to tomorrow
pitlist carry <id> --to <date>
```

### Agenda

```bash
pitlist agenda                        # next 7 days, pending only
pitlist agenda -n 14                  # next N days
pitlist agenda --from <date> --to <date>
pitlist agenda --label work
```

### Activity log

```bash
pitlist log "Description" [--tag debugging] [--ref t-20260518-001] [--duration 45] [--date <date>]
pitlist log list                      # today
pitlist log list --date <date>
pitlist log list --tag debugging
pitlist log list --week
pitlist log link <activity-id> <task-id>   # link after the fact
```

### Utility

```bash
pitlist sync                          # git add . && git commit
pitlist sync --push                   # also git push
pitlist stats                         # today
pitlist stats --week
pitlist stats --month
```

---

## TUI

Launch with `pitlist` (no subcommand). Four tabs.

### Global keybindings

| Key | Action |
|---|---|
| `1` | Tasks tab |
| `2` | Activity tab |
| `3` | Agenda tab |
| `4` | Search tab |
| `q` | Quit |
| `ctrl+c` | Quit (always, even inside forms) |

Tab-switch keys (`1`–`4`, `q`) are disabled when a form or input is active in the current tab.

---

### Tab 1: Tasks

Split view: left pane (task list) | right pane (task detail).

**Left pane — task list**

| Key | Action |
|---|---|
| `h` / `l` | Previous / next day |
| `j` / `k` | Move cursor |
| `a` | Add task (inline input at top of list) |
| `d` | Toggle done / todo |
| `c` | Carry — opens date prompt in right pane |
| `D` | Delete task |
| `tab` | Switch focus to detail pane |
| `w` | Toggle week view |
| `/` | Open filter overlay (searches all days) |

Tasks with notes show `¶` indicator. Carried tasks show `↑`.

**Right pane — task detail**

Shows: title, status, priority, labels, due date, notes, and linked activity entries with date + duration.

Activity section header shows total time: `Activity:  ∑ 1h 15m`

| Key | Action |
|---|---|
| `n` | Edit notes (textarea, `ctrl+s` save, `esc` cancel) |
| `L` | Log activity linked to this task |
| `d` | Toggle done |
| `c` | Carry (opens date prompt) |
| `tab` | Switch focus back to list pane |

**Carry prompt (right pane)**

```
Carry task to…
→ Write RFC for auth redesign
────────────────────────────────────
  Date: 2026-05-19█

  enter to confirm  esc to cancel
```

Pre-filled with tomorrow. Validates `YYYY-MM-DD` format before confirming.

**Log activity form (right pane)**

```
Log activity
→ Write RFC for auth redesign
────────────────────────────────────

> Description:  What did you do?
  Tags:         tags space-separated
  Minutes:      minutes (optional)
  Date:         2026-05-18T14:30     ← auto-computed: now - minutes
  Task ref:     t-20260518-001       ← pre-filled, read-only

  tab next  ctrl+s save  esc cancel
```

Date auto-updates to `now - minutes` when tabbing from Minutes to Date. User can edit it manually (`YYYY-MM-DDTHH:MM`). The activity is saved to the day matching the Date field.

**Filter overlay (`/`)**

Searches across ALL days when labels or text are specified.

```
─── Filter ───
> Search:   auth
  Labels:   work auth
  Status:   [x]todo  [x]in_progress  [ ]done

  enter apply  esc cancel  tab next field
```

---

### Tab 2: Activity Log

| Key | Action |
|---|---|
| `h` / `l` | Previous / next day |
| `j` / `k` | Move cursor |
| `a` | Add activity entry |
| `D` | Delete selected entry |

**Add activity form:**

```
─── New Activity ───
> Description:  What did you do?
  Tags:         work debugging
  Duration:     30
  Date:         2026-05-18T14:30    ← auto-computed: now - duration
  Task ref:     (optional)

  tab next  ctrl+s save  esc cancel
```

Same timestamp logic as the task detail log form.

---

### Tab 3: Agenda

Shows pending tasks from the last 7 days through the next 7 days (14-day window). Days with no pending tasks are hidden. Past-day tasks are marked `overdue`.

| Key | Action |
|---|---|
| `j` / `k` | Navigate tasks |
| `d` | Mark done directly |
| `enter` | Jump to that day in Tasks tab |
| `r` | Refresh |

---

### Tab 4: Search

Full-text and tag search across all tasks and activity entries.

**Two modes:**

- **Input mode** (default): type to search. `↓` or `enter` switches to navigate mode.
- **Navigate mode**: `j`/`k` to move cursor, `enter` to jump to the day in Tasks or Activity tab. `i`, `esc`, or `/` returns to input mode.

Single-word queries without `#` search both text and tag/label simultaneously. `#tag` searches strictly by tag/label. Multi-word queries are text-only.

Results are grouped: Tasks first, then Activity entries. Each result shows its date.

---

## Configuration

`~/.config/pitlist/config.yaml`

```yaml
data_dir: "~/pitlist"      # where YAML files are stored
editor: ""                  # falls back to $EDITOR
week_start: monday
contexts: [work, personal, other]
git:
  auto_commit: true
tui:
  show_done_tasks: false
```

Override data dir at runtime: `PITLIST_DATA_DIR=/path/to/dir pitlist`

---

## Scopes (named profiles)

Scopes let you maintain completely separate data directories for different areas of your life. Each scope has its own YAML files, git history, and context list.

Define profiles in `~/.config/pitlist/config.yaml`:

```yaml
data_dir: "~/pitlist"           # default when no --scope is given
contexts: [work, personal]

profiles:
  work:
    data_dir: "~/pitlist-work"
    contexts: [work, meetings, reviews]
  personal:
    data_dir: "~/pitlist-personal"
    contexts: [personal, health, home]
```

Activate a scope:

```bash
pitlist --scope work              # flag (any subcommand)
pitlist --scope work list
pitlist --scope personal log "ran 5k"

PITLIST_SCOPE=work pitlist        # environment variable
```

- Only `data_dir` and `contexts` can be overridden per profile; all other settings (`git`, `tui`, `editor`) are inherited from the base config.
- If `--scope` names a profile that does not exist, pitlist exits with an error listing the available scopes.
- Scopes are independent — they have separate YAML files and separate git repos under their respective `data_dir`.

---

## Activity Lookup Efficiency

Each task stores `activity_refs: [{id, date}]` — direct pointers to its linked activity entries. Loading linked activities for a task reads only the specific files that contain those entries (O(N refs), typically 1–5 files).

If a task has no refs yet (e.g., tasks created before this feature), the fallback loads the task's own day file.

---

## Invariants

- Tasks never duplicate on carry — same ID moves between files
- Activity entries are immutable after creation (no edit UI — edit the YAML directly)
- `activity_refs` on a task is a soft index — stale refs are silently ignored
- Git auto-commit never pushes, never force-pushes, never configures remotes
- The TUI never blocks on slow operations — all storage I/O is async via Bubble Tea commands
