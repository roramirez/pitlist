package views

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

type SearchResultKind int

const (
	SearchResultTask SearchResultKind = iota
	SearchResultActivity
)

type SearchResult struct {
	Kind     SearchResultKind
	Task     *model.Task
	Activity *model.ActivityEntry
	Date     time.Time
}

type SearchResultsMsg struct{ Results []SearchResult }
type SearchNavigateTaskMsg struct{ Date time.Time }
type SearchNavigateActivityMsg struct{ Date time.Time }

type SearchView struct {
	store        *storage.YAMLStore
	query        string // plain string — no textinput widget
	results      []SearchResult
	cursor       int
	inputFocused bool
	width        int
	height       int
}

func NewSearchView(store *storage.YAMLStore) SearchView {
	return SearchView{store: store, inputFocused: true}
}

func (v SearchView) IsInputActive() bool { return v.inputFocused }
func (v SearchView) Query() string       { return v.query }

func (v SearchView) Update(msg tea.Msg) (SearchView, tea.Cmd) {
	switch msg := msg.(type) {
	case SearchResultsMsg:
		v.results = msg.Results
		if v.cursor >= len(v.results) {
			v.cursor = 0
		}
		return v, nil

	case tea.KeyMsg:
		if v.inputFocused {
			return v.updateInput(msg)
		}
		return v.updateResults(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

func (v SearchView) updateInput(msg tea.KeyMsg) (SearchView, tea.Cmd) {
	str := msg.String()
	switch {
	case str == "backspace" || str == "ctrl+h":
		if len(v.query) > 0 {
			_, size := utf8.DecodeLastRuneInString(v.query)
			v.query = v.query[:len(v.query)-size]
			return v, v.search()
		}
	case str == "esc":
		if len(v.results) > 0 {
			v.inputFocused = false
		}
	case str == "down":
		if len(v.results) > 0 {
			v.inputFocused = false
			v.cursor = 0
		}
	case str == "enter":
		if len(v.results) > 0 {
			v.inputFocused = false
			v.cursor = 0
		}
	case len(msg.Runes) > 0:
		v.query += string(msg.Runes)
		return v, v.search()
	}
	return v, nil
}

func (v SearchView) updateResults(msg tea.KeyMsg) (SearchView, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.results)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		} else {
			v.inputFocused = true
		}
	case "enter", "l", "right":
		return v, v.navigate()
	case "esc", "i", "/":
		v.inputFocused = true
	}
	return v, nil
}

func (v SearchView) search() tea.Cmd {
	query := strings.TrimSpace(v.query)
	return func() tea.Msg {
		if query == "" {
			return SearchResultsMsg{}
		}

		// #tag  → strict tag/label search only
		// text  → search both text AND tags simultaneously
		tagOnly := strings.HasPrefix(query, "#")
		tag := strings.TrimPrefix(query, "#")

		var results []SearchResult

		// --- Tasks ---
		taskFilter := storage.TaskFilter{
			Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress, model.StatusDone},
		}
		if tagOnly {
			taskFilter.Labels = []string{tag}
		} else {
			// search by text; also do a separate label pass below
			taskFilter.Search = query
		}
		tasks, _ := v.store.ListTasks(taskFilter)
		seen := map[string]bool{}
		for _, t := range tasks {
			tc := *t
			date := dateFromID(t.ID)
			results = append(results, SearchResult{Kind: SearchResultTask, Task: &tc, Date: date})
			seen[t.ID] = true
		}

		// If free-text, also match by label and merge (dedup)
		if !tagOnly {
			labelFilter := storage.TaskFilter{
				Statuses: taskFilter.Statuses,
				Labels:   []string{query},
			}
			byLabel, _ := v.store.ListTasks(labelFilter)
			for _, t := range byLabel {
				if !seen[t.ID] {
					tc := *t
					date := dateFromID(t.ID)
					results = append(results, SearchResult{Kind: SearchResultTask, Task: &tc, Date: date})
				}
			}
		}

		// --- Activities ---
		actFilter := storage.ActivityFilter{}
		if tagOnly {
			actFilter.Tags = []string{tag}
		} else {
			actFilter.Search = query
		}
		entries, _ := v.store.ListActivity(actFilter)
		seenAct := map[string]bool{}
		for _, e := range entries {
			ec := *e
			date := time.Date(e.Timestamp.Year(), e.Timestamp.Month(), e.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
			results = append(results, SearchResult{Kind: SearchResultActivity, Activity: &ec, Date: date})
			seenAct[e.ID] = true
		}

		// Also match activities by tag in free-text mode
		if !tagOnly {
			tagFilter := storage.ActivityFilter{Tags: []string{query}}
			byTag, _ := v.store.ListActivity(tagFilter)
			for _, e := range byTag {
				if !seenAct[e.ID] {
					ec := *e
					date := time.Date(e.Timestamp.Year(), e.Timestamp.Month(), e.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
					results = append(results, SearchResult{Kind: SearchResultActivity, Activity: &ec, Date: date})
				}
			}
		}

		return SearchResultsMsg{Results: results}
	}
}

// dateFromID parses YYYY-MM-DD from t-YYYYMMDD-NNN or a-YYYYMMDD-NNN.
func dateFromID(id string) time.Time {
	if len(id) < 10 {
		return time.Now()
	}
	raw := id[2:10] // YYYYMMDD
	t, err := time.Parse("20060102", raw)
	if err != nil {
		return time.Now()
	}
	return t
}

func (v SearchView) navigate() tea.Cmd {
	if v.cursor >= len(v.results) {
		return nil
	}
	r := v.results[v.cursor]
	return func() tea.Msg {
		if r.Kind == SearchResultTask {
			return SearchNavigateTaskMsg{Date: r.Date}
		}
		return SearchNavigateActivityMsg{Date: r.Date}
	}
}

func (v SearchView) View(width, height int) string {
	cursor := " "
	if v.inputFocused {
		cursor = sAccent.Render("█")
	}
	inputLine := "  / " + v.query + cursor

	var lines []string
	lines = append(lines, sTitle.Render("Search")+"  "+sMuted.Render("keyword or #tag across all days"))
	lines = append(lines, inputLine)
	lines = append(lines, "")

	if len(v.results) == 0 && strings.TrimSpace(v.query) != "" {
		lines = append(lines, sMuted.Render("  No results."))
	} else {
		var prevKind SearchResultKind = -1
		for i, r := range v.results {
			selected := i == v.cursor && !v.inputFocused

			if SearchResultKind(prevKind) != r.Kind {
				if r.Kind == SearchResultTask {
					lines = append(lines, sMuted.Render("  ── Tasks ──"))
				} else {
					lines = append(lines, sMuted.Render("  ── Activity ──"))
				}
				prevKind = r.Kind
			}

			line := v.renderResult(r)
			if selected {
				line = sSelected.Render(line)
			}
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	if v.inputFocused {
		lines = append(lines, sMuted.Render("  ↓/enter → navigate results  1/2/3/4 tabs"))
	} else {
		lines = append(lines, sMuted.Render("  j/k navigate  enter → jump  esc/i → edit query  q quit"))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width-2).
		Height(height-2).
		Padding(0, 1).
		Render(content)
}

func (v SearchView) renderResult(r SearchResult) string {
	dateStr := sMuted.Render(r.Date.Format("Jan 02"))

	if r.Kind == SearchResultTask {
		check := "[ ]"
		title := r.Task.Title
		switch r.Task.Status {
		case model.StatusDone:
			check = "[x]"
			title = sMuted.Render(title)
		case model.StatusInProgress:
			check = "[~]"
		}
		labels := ""
		if len(r.Task.Labels) > 0 {
			labels = "  " + sAccent.Render("["+strings.Join(r.Task.Labels, ", ")+"]")
		}
		return fmt.Sprintf("    %s %s%s  %s", check, title, labels, dateStr)
	}

	dur := ""
	if r.Activity.DurationMin > 0 {
		dur = sCarried.Render(fmt.Sprintf(" %dm", r.Activity.DurationMin))
	}
	tags := ""
	if len(r.Activity.Tags) > 0 {
		tags = "  " + sAccent.Render("["+strings.Join(r.Activity.Tags, ", ")+"]")
	}
	ref := ""
	if r.Activity.TaskRef != "" {
		ref = "  " + sMuted.Render("→ "+r.Activity.TaskRef)
	}
	return fmt.Sprintf("    %s%s  %s%s%s  %s",
		sMuted.Render(r.Activity.Timestamp.Local().Format("15:04")),
		dur,
		r.Activity.Description,
		tags,
		ref,
		dateStr,
	)
}
