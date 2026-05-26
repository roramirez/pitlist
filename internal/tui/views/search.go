package views

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
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
		v.inputFocused = false
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
	store := v.store
	return func() tea.Msg {
		if query == "" {
			return SearchResultsMsg{}
		}
		tagOnly := strings.HasPrefix(query, "#")
		tag := strings.TrimPrefix(query, "#")
		results := searchTaskResults(store, query, tagOnly, tag)
		results = append(results, searchActivityResults(store, query, tagOnly, tag)...)
		return SearchResultsMsg{Results: results}
	}
}

// searchTaskResults queries tasks by text or tag and deduplicates label matches.
func searchTaskResults(store *storage.YAMLStore, query string, tagOnly bool, tag string) []SearchResult {
	statuses := []model.TaskStatus{model.StatusTodo, model.StatusInProgress, model.StatusDone}
	f := storage.TaskFilter{Statuses: statuses}
	if tagOnly {
		f.Labels = []string{tag}
	} else {
		f.Search = query
	}
	tasks, _ := store.ListTasks(f)
	seen := map[string]bool{}
	var results []SearchResult
	for _, t := range tasks {
		tc := *t
		results = append(results, SearchResult{Kind: SearchResultTask, Task: &tc, Date: dateFromID(t.ID)})
		seen[t.ID] = true
	}
	if !tagOnly {
		byLabel, _ := store.ListTasks(storage.TaskFilter{Statuses: statuses, Labels: []string{query}})
		for _, t := range byLabel {
			if !seen[t.ID] {
				tc := *t
				results = append(results, SearchResult{Kind: SearchResultTask, Task: &tc, Date: dateFromID(t.ID)})
			}
		}
	}
	return results
}

// searchActivityResults queries activity entries by text or tag and deduplicates.
func searchActivityResults(store *storage.YAMLStore, query string, tagOnly bool, tag string) []SearchResult {
	f := storage.ActivityFilter{}
	if tagOnly {
		f.Tags = []string{tag}
	} else {
		f.Search = query
	}
	entries, _ := store.ListActivity(f)
	seen := map[string]bool{}
	var results []SearchResult
	for _, e := range entries {
		ec := *e
		date := time.Date(e.Timestamp.Year(), e.Timestamp.Month(), e.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
		results = append(results, SearchResult{Kind: SearchResultActivity, Activity: &ec, Date: date})
		seen[e.ID] = true
	}
	if !tagOnly {
		byTag, _ := store.ListActivity(storage.ActivityFilter{Tags: []string{query}})
		for _, e := range byTag {
			if !seen[e.ID] {
				ec := *e
				date := time.Date(e.Timestamp.Year(), e.Timestamp.Month(), e.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
				results = append(results, SearchResult{Kind: SearchResultActivity, Activity: &ec, Date: date})
			}
		}
	}
	return results
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

	lines = append(lines, v.renderResultLines()...)

	lines = append(lines, "")
	if v.inputFocused {
		lines = append(lines, sMuted.Render("  ↓/enter → navigate results  esc → stop typing  q quit  1/2/3/4 tabs"))
	} else {
		lines = append(lines, sMuted.Render("  j/k navigate  enter → jump  esc/i → edit query  q quit"))
	}

	content := strings.Join(lines, "\n")
	return sPaneActive.Width(width - 2).Height(height - 2).Render(content)
}

func (v SearchView) renderResultLines() []string {
	if len(v.results) == 0 && strings.TrimSpace(v.query) != "" {
		return []string{sMuted.Render("  No results.")}
	}
	var prevKind SearchResultKind = -1
	var lines []string
	for i, r := range v.results {
		if prevKind != r.Kind {
			if r.Kind == SearchResultTask {
				lines = append(lines, sMuted.Render("  ── Tasks ──"))
			} else {
				lines = append(lines, sMuted.Render("  ── Activity ──"))
			}
			prevKind = r.Kind
		}
		line := v.renderResult(r)
		if i == v.cursor && !v.inputFocused {
			line = sSelected.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (v SearchView) renderResult(r SearchResult) string {
	if r.Kind == SearchResultTask {
		return v.renderTaskResult(r)
	}
	return v.renderActivityResult(r)
}

func (v SearchView) renderTaskResult(r SearchResult) string {
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
	return fmt.Sprintf("    %s %s%s  %s", check, title, labels, sMuted.Render(r.Date.Format("Jan 02")))
}

func (v SearchView) renderActivityResult(r SearchResult) string {
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
		sMuted.Render(r.Date.Format("Jan 02")),
	)
}
