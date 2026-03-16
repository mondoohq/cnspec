// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// Option configures the TODO list progress UI.
type Option func(*modelTodoList)

// WithScore enables score display next to completed tasks.
func WithScore() Option {
	return func(m *modelTodoList) {
		m.includeScore = true
	}
}

type taskState int

const (
	taskStatePending taskState = iota
	taskStateInProgress
	taskStateCompleted
	taskStateErrored
	taskStateNotApplicable
)

type task struct {
	key       string
	name      string
	platform  string
	state     taskState
	score     string
	percent   float64
	startedAt time.Time
	duration  time.Duration
}

// Bubbletea messages for async updates.

type msgAddTask struct {
	key   string
	asset *inventory.Asset
}

type msgProgress struct {
	index   string
	percent float64
}

type msgScore struct {
	index string
	score string
}

type msgCompleted struct {
	index string
}

type msgErrored struct {
	index string
}

type msgNotApplicable struct {
	index string
}

type msgTick time.Time

// modelTodoList is the bubbletea model for the TODO-list progress UI.
type modelTodoList struct {
	tasks        []*task
	taskIndex    map[string]*task
	lock         sync.Mutex
	startTime    time.Time
	includeScore bool
	spinner      spinner.Model
}

func newTodoListModel(opts ...Option) *modelTodoList {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9"))

	m := &modelTodoList{
		taskIndex: make(map[string]*task),
		startTime: time.Now(),
		spinner:   s,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *modelTodoList) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return msgTick(t)
	})
}

func (m *modelTodoList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		return m, nil

	case msgAddTask:
		m.lock.Lock()
		added := false
		if _, exists := m.taskIndex[msg.key]; !exists {
			name := ""
			platform := ""
			if msg.asset != nil {
				name = msg.asset.Name
				if msg.asset.Platform != nil {
					platform = msg.asset.Platform.Name
				}
			}
			t := &task{key: msg.key, name: name, platform: platform, state: taskStatePending}
			m.tasks = append(m.tasks, t)
			m.taskIndex[msg.key] = t
			added = true
		}
		m.lock.Unlock()
		if added {
			// Force a full redraw since the view height changed.
			return m, tea.ClearScreen
		}
		return m, nil

	case msgProgress:
		m.lock.Lock()
		if t, ok := m.taskIndex[msg.index]; ok {
			if t.state == taskStatePending {
				t.state = taskStateInProgress
				t.startedAt = time.Now()
			}
			t.percent = msg.percent
		}
		m.lock.Unlock()
		return m, nil

	case msgScore:
		m.lock.Lock()
		if t, ok := m.taskIndex[msg.index]; ok {
			t.score = msg.score
		}
		m.lock.Unlock()
		return m, nil

	case msgCompleted:
		m.lock.Lock()
		if t, ok := m.taskIndex[msg.index]; ok {
			t.state = taskStateCompleted
			if !t.startedAt.IsZero() {
				t.duration = time.Since(t.startedAt)
			}
		}
		done := m.allDoneLocked()
		m.lock.Unlock()
		if done {
			return m, tea.Quit
		}
		return m, nil

	case msgErrored:
		m.lock.Lock()
		if t, ok := m.taskIndex[msg.index]; ok {
			t.state = taskStateErrored
			if !t.startedAt.IsZero() {
				t.duration = time.Since(t.startedAt)
			}
		}
		done := m.allDoneLocked()
		m.lock.Unlock()
		if done {
			return m, tea.Quit
		}
		return m, nil

	case msgNotApplicable:
		m.lock.Lock()
		if t, ok := m.taskIndex[msg.index]; ok {
			t.state = taskStateNotApplicable
			if !t.startedAt.IsZero() {
				t.duration = time.Since(t.startedAt)
			}
		}
		done := m.allDoneLocked()
		m.lock.Unlock()
		if done {
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case msgTick:
		return m, tickCmd()

	default:
		return m, nil
	}
}

// allDoneLocked returns true when there is at least one task and all tasks are
// in a terminal state. Must be called with m.lock held.
func (m *modelTodoList) allDoneLocked() bool {
	if len(m.tasks) == 0 {
		return false
	}
	for _, t := range m.tasks {
		if t.state == taskStatePending || t.state == taskStateInProgress {
			return false
		}
	}
	return true
}

var (
	styleHeader    = lipgloss.NewStyle().Bold(true)
	styleDim       = lipgloss.NewStyle().Faint(true)
	styleSuccess   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))
	styleInProgress = lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9"))

	// scoreColors maps score rating text to lipgloss colors, matching
	// the ScoreRatingLipglossColorMapping in cli/components/rating.go.
	scoreColors = map[string]lipgloss.Color{
		"UNRATED":  lipgloss.Color("231"),
		"NONE":     lipgloss.Color("78"),
		"LOW":      lipgloss.Color("117"),
		"MEDIUM":   lipgloss.Color("75"),
		"HIGH":     lipgloss.Color("212"),
		"CRITICAL": lipgloss.Color("204"),
		"ERROR":    lipgloss.Color("210"),
	}
)

func (m *modelTodoList) View() string {
	var b strings.Builder

	m.lock.Lock()
	defer m.lock.Unlock()

	// Header
	b.WriteString("\n")
	b.WriteString(" " + styleHeader.Render("Scanning assets..."))
	b.WriteString("\n\n")

	// Count tasks by state and collect buckets for the rolling window.
	var (
		inProgress    []*task
		finished      []*task // completed, errored, n/a — in original order
		pending       []*task
		completed int
		errored   int
	)
	for _, t := range m.tasks {
		switch t.state {
		case taskStateInProgress:
			inProgress = append(inProgress, t)
		case taskStateCompleted:
			completed++
			finished = append(finished, t)
		case taskStateErrored:
			errored++
			finished = append(finished, t)
		case taskStateNotApplicable:
			finished = append(finished, t)
		default:
			pending = append(pending, t)
		}
	}

	// Rolling window: in-progress on top, then last 2 finished, then next 2 pending.
	const maxFinished = 2
	const maxPending = 2

	// In-progress tasks (typically 1 in sequential scanning)
	for _, t := range inProgress {
		b.WriteString(m.renderTask(t))
	}

	// Most recently finished tasks (tail of the finished slice)
	start := len(finished) - maxFinished
	if start < 0 {
		start = 0
	}
	for _, t := range finished[start:] {
		b.WriteString(m.renderTask(t))
	}

	// Next pending tasks
	shown := 0
	for _, t := range pending {
		if shown >= maxPending {
			break
		}
		b.WriteString(m.renderTask(t))
		shown++
	}

	remaining := len(pending) - shown
	if remaining > 0 {
		label := "asset"
		if remaining > 1 {
			label = "assets"
		}
		b.WriteString(styleDim.Render(fmt.Sprintf("  ... %d more %s ...", remaining, label)))
		b.WriteString("\n")
	}

	// Footer: completion stats with elapsed time.
	// All terminal states (completed, errored, n/a) count as "done" tasks.
	total := len(m.tasks)
	if total > 0 {
		done := len(finished)
		elapsed := time.Since(m.startTime).Truncate(time.Second)
		b.WriteString("\n")
		footer := fmt.Sprintf("  %d/%d completed", done, total)
		if errored > 0 {
			footer += styleError.Render(fmt.Sprintf(" · %d errored", errored))
		}
		footer += styleDim.Render(fmt.Sprintf(" · %s", formatDuration(elapsed)))
		b.WriteString(footer)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	return b.String()
}

func (m *modelTodoList) renderTask(t *task) string {
	var icon string
	var nameStr string
	suffix := ""

	switch t.state {
	case taskStatePending:
		icon = styleDim.Render("○")
		nameStr = styleDim.Render(t.name)
	case taskStateInProgress:
		icon = m.spinner.View()
		nameStr = styleInProgress.Render(t.name)
		suffix = styleDim.Render(fmt.Sprintf(" %d%%", int(t.percent*100)))
	case taskStateCompleted:
		icon = styleSuccess.Render("✓")
		nameStr = t.name
	case taskStateErrored:
		icon = styleError.Render("✗")
		nameStr = styleError.Render(t.name)
	case taskStateNotApplicable:
		icon = styleDim.Render("–")
		nameStr = styleDim.Render(t.name)
	}

	line := fmt.Sprintf("  %s %s", icon, nameStr)
	if t.platform != "" {
		line += styleDim.Render(" [" + t.platform + "]")
	}

	if m.includeScore && t.score != "" {
		scoreStr := t.score
		if c, ok := scoreColors[t.score]; ok {
			scoreStr = lipgloss.NewStyle().Foreground(c).Render(t.score)
		} else if t.state == taskStateErrored {
			scoreStr = styleError.Render(t.score)
		}
		line += "  " + scoreStr
	}

	// Show duration for finished tasks, or elapsed time for in-progress tasks.
	if t.duration > 0 {
		line += styleDim.Render(fmt.Sprintf(" (%s)", formatTaskDuration(t.duration)))
	} else if t.state == taskStateInProgress && !t.startedAt.IsZero() {
		line += styleDim.Render(fmt.Sprintf(" (%s)", formatTaskDuration(time.Since(t.startedAt))))
	}

	line += suffix
	return line + "\n"
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

// formatTaskDuration renders a per-task duration in a human-friendly way:
//
//	< 1s      → "0.3s"  (one decimal, shows ms-level detail)
//	1s–59s    → "12.4s" (one decimal)
//	1m–59m    → "2m 13s"
//	≥ 1h      → "1h 5m"
func formatTaskDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", h, m)
	}
}

// todoListProgress wraps a tea.Program and implements MultiProgress.
type todoListProgress struct {
	program *tea.Program
}

// NewTodoList creates a new TODO-list style multi-asset progress reporter.
func NewTodoList(opts ...Option) (*todoListProgress, error) {
	m := newTodoListModel(opts...)
	p := tea.NewProgram(m)
	return &todoListProgress{program: p}, nil
}

func newTodoListProgram(input io.Reader, output io.Writer, opts ...Option) (*todoListProgress, error) {
	m := newTodoListModel(opts...)
	p := tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output))
	return &todoListProgress{program: p}, nil
}

func (t *todoListProgress) Open() error {
	(logger.LogOutputWriter.(*logger.BufferedWriter)).Pause()
	defer (logger.LogOutputWriter.(*logger.BufferedWriter)).Resume()
	if _, err := t.program.Run(); err != nil {
		return err
	}
	return nil
}

func (t *todoListProgress) AddTask(index string, asset *inventory.Asset) {
	t.program.Send(msgAddTask{key: index, asset: asset})
}

func (t *todoListProgress) OnProgress(index string, percent float64) {
	t.program.Send(msgProgress{index: index, percent: percent})
}

func (t *todoListProgress) Score(index string, score string) {
	t.program.Send(msgScore{index: index, score: score})
}

func (t *todoListProgress) Errored(index string) {
	t.program.Send(msgErrored{index: index})
}

func (t *todoListProgress) NotApplicable(index string) {
	t.program.Send(msgNotApplicable{index: index})
}

func (t *todoListProgress) Completed(index string) {
	t.program.Send(msgCompleted{index: index})
}

func (t *todoListProgress) Close() {
	t.program.Quit()
}
