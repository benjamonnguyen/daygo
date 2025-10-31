package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/benjamonnguyen/daygo"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const logo = `
	██████╗  █████╗ ██╗   ██╗ ██████╗  ██████╗ 
	██╔══██╗██╔══██╗╚██╗ ██╔╝██╔════╝ ██╔═══██╗
	██║  ██║███████║ ╚████╔╝ ██║  ███╗██║   ██║
	██║  ██║██╔══██║  ╚██╔╝  ██║   ██║██║   ██║
	██████╔╝██║  ██║   ██║   ╚██████╔╝╚██████╔╝
	╚═════╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝  ╚═════╝`

const programUsage = `Usage:
  daygo: start queued task
  daygo <task>: start new task
  daygo /a <task>: add task to queue
  daygo /review [date]: review tasks and notes for target date; accepts date formats "DD-MM"/"DD-MM-YYYY" or number of days ago
`

type model struct {
	// children
	vp        viewport.Model
	userinput textinput.Model

	// supplied
	l       *slog.Logger
	taskSvc TaskSvc

	// state
	tasks    []Task
	alerts   []string
	quitting bool
	h        int

	// configuration
	cmdTimeout time.Duration
	timeFormat string
}

var _ tea.Model = (*model)(nil)

func (m model) Init() tea.Cmd {
	cmd, err := m.parseProgramArgs()
	if err != nil {
		m.l.Error(err.Error())
		return tea.Quit
	}
	return tea.Batch(cmd, textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var tiCmd, vpCmd, cmd tea.Cmd

	// TODO extract
	m, cmd = func() (model, tea.Cmd) {
		switch msg := msg.(type) {
		case ErrorMsg:
			m = m.addAlert(colorize(colorRed, msg.err.Error()))
			return m, nil
		case NewTaskMsg:
			return m.handleNewTask(msg), nil
		case NewNoteMsg:
			return m.handleNewNote(msg), nil
		case QueueMsg:
			alert := fmt.Sprintf(`Queued up "%s"`, msg.task)
			m = m.addAlert(colorize(colorCyan, alert))
			return m, nil
		case SkipMsg:
			task := m.currentTask()
			if msg.id != task.ID {
				panic("skip msg: out of sync")
			}
			m.tasks = m.tasks[:len(m.tasks)-1]
			m.vp.SetContent(m.renderVisibleTasks())
			m.resizeViewport()
			return m, m.startNextTask()
		case DiscardPendingItemMsg:
			return m.handleDiscardPendingItem(msg), nil
		case tea.WindowSizeMsg:
			m.h = msg.Height
			m.userinput.Width = msg.Width
			m.vp.Width = msg.Width
			m.resizeViewport()
			return m, nil
		case tea.KeyMsg:
			// TODO consider a mutex to ignore input until state is consistent
			switch msg.Type {
			case tea.KeyEnter:
				return m.handleInput()
			case tea.KeyCtrlC:
				m.quitting = true
				if t := m.currentTask(); t != nil {
					t.IsTerminal = true
					if t.IsPending() {
						timeout, cancel := m.newTimeout()
						defer cancel()
						if err := m.endPendingTask(timeout); err != nil {
							logger.Error(err.Error())
						}
						t.EndedAt = time.Now().Local()
					}
					m.vp.SetContent(m.renderVisibleTasks())
					m.resizeViewport()
				}
				return m, tea.Quit
			}
		}
		return m, nil
	}()

	m.userinput, tiCmd = m.userinput.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// vp udpates on KeyMsg was causing a view flickering bug
	default:
		m.vp, vpCmd = m.vp.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd, cmd)
}

func (m model) handleDiscardPendingItem(msg DiscardPendingItemMsg) model {
	t := m.currentTask()
	if !t.IsPending() {
		panic("discardPendingItemMsg: out of sync")
	}
	if len(t.Notes) > 0 {
		if n := t.LastNote(); n == nil || n.ID != msg.id {
			panic("discardPendingItemMsg: out of sync")
		}
		t.Notes = t.Notes[:len(t.Notes)-1]
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
	} else {
		if t.ID != msg.id {
			panic("discardPendingItemMsg: out of sync")
		}
		m.tasks = m.tasks[:len(m.tasks)-1]
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
	}

	return m
}

func (m model) footerHeight() int {
	if m.quitting {
		return 1
	}
	// TODO magicnumber
	return 4 + len(m.alerts)
}

func (m model) View() string {
	// sections
	var content, footer strings.Builder

	// content
	content.WriteString(m.vp.View())

	// footer
	footer.WriteRune('\n')
	if !m.quitting {
		footer.WriteString(m.userinput.View())
		footer.WriteString("\n\n")
		if len(m.alerts) > 0 {
			footer.WriteString(strings.Join(m.alerts, "\n"))
		} else {
			footer.WriteString(faintStyle.Render("(ctrl+c to quit)"))
		}
		footer.WriteRune('\n')
	}

	return lipgloss.JoinVertical(0, content.String(), footer.String())
}

func (m model) currentTask() *Task {
	if len(m.tasks) == 0 {
		return nil
	}
	return &m.tasks[len(m.tasks)-1]
}

func (m model) newTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), m.cmdTimeout)
}

func (m model) addAlert(alert string) model {
	m.alerts = append(m.alerts, alert)
	return m
}

func (m model) endPendingTask(ctx context.Context) error {
	currentTask := m.currentTask()
	if !currentTask.IsPending() {
		return nil
	}
	if _, err := m.taskSvc.EndTask(ctx, currentTask.ID); err != nil {
		return err
	}
	return m.endPendingNote(ctx)
}

func (m model) endPendingNote(ctx context.Context) error {
	currentTask := m.currentTask()
	if !currentTask.IsPending() {
		return fmt.Errorf("failed to add note: no pending task")
	}
	if n := currentTask.LastNote(); n != nil {
		if _, err := m.taskSvc.EndTask(ctx, n.ID); err != nil {
			return err
		}
	}
	return nil
}

func (m *model) resizeViewport() {
	tasksHeight := 0
	for _, t := range m.tasks {
		tasksHeight += len(t.Notes) + 2
	}
	m.vp.Height = min(tasksHeight, m.h-m.footerHeight())
	m.vp.GotoBottom()
}

func (m model) handleNewNote(msg NewNoteMsg) model {
	t := m.currentTask()
	if !t.IsPending() {
		panic("handleNewNote: expecting pending task")
	}
	if n := t.LastNote(); n != nil {
		n.EndedAt = msg.note.StartedAt
	}
	t.Notes = append(t.Notes, msg.note)
	m.vp.SetContent(m.renderVisibleTasks())
	m.resizeViewport()
	m.l.Debug("handleNewNote", "tasks", m.tasks)
	return m
}

func (m model) renderVisibleTasks() string {
	if len(m.tasks) == 0 {
		return ""
	}
	var lines []string
	availableHeight := m.vp.Height
	for i := len(m.tasks) - 1; i >= 0 && availableHeight >= 0; i-- {
		line, h := m.tasks[i].Render(m.timeFormat)
		availableHeight -= h
		if i != len(m.tasks)-1 {
			line = faintStyle.Render(line)
		}
		lines = append(lines, line)
	}

	slices.Reverse(lines)
	return strings.Join(lines, "\n")
}

func (m model) handleNewTask(msg NewTaskMsg) model {
	if t := m.currentTask(); t.IsPending() {
		t.EndedAt = msg.task.StartedAt
		if n := t.LastNote(); n != nil {
			n.EndedAt = msg.task.StartedAt
		}
	}
	m.tasks = append(m.tasks, msg.task)
	m.vp.SetContent(m.renderVisibleTasks())
	m.resizeViewport()
	return m
}

func (m model) startNextTask() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		task, err := m.taskSvc.PeekNextTask(ctx)
		if err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		if task == (daygo.ExistingTaskRecord{}) {
			return errorMsg("task queue is empty!")
		}

		if err := m.endPendingTask(ctx); err != nil {
			return ErrorMsg{
				err: err,
			}
		}

		task, err = m.taskSvc.StartTask(ctx, startTaskRequest{
			ID: task.ID,
		})
		if err != nil {
			return ErrorMsg{
				err: err,
			}
		}

		return NewTaskMsg{
			task: Task{
				ExistingTaskRecord: task,
			},
		}
	}
}

func (m model) startNewTask(task string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		if err := m.endPendingTask(ctx); err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		task, err := m.taskSvc.StartTask(ctx, startTaskRequest{
			Name: task,
		})
		if err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		return NewTaskMsg{
			task: Task{
				ExistingTaskRecord: task,
			},
		}
	}
}

func (m model) queueTask(task string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		if _, err := m.taskSvc.QueueTask(ctx, queueTaskRequest{
			Name: task,
		}); err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		return QueueMsg{
			task: task,
		}
	}
}

func (m model) parseProgramArgs() (tea.Cmd, error) {
	if len(os.Args) == 1 {
		return m.startNextTask(), nil
	}

	var cmd, arg string
	if strings.HasPrefix(os.Args[1], "/") {
		cmd = os.Args[1]
		if len(os.Args) > 2 {
			arg = strings.Join(os.Args[2:], " ")
		}
	} else {
		arg = strings.Join(os.Args[1:], " ")
	}

	logger.Debug("parsed program args", "cmd", cmd, "arg", arg)
	switch cmd {
	case "/n", "":
		if arg == "" {
			return m.startNextTask(), nil
		}
		return m.startNewTask(arg), nil
	case "/a":
		_, err := m.taskSvc.QueueTask(context.Background(), queueTaskRequest{
			Name: arg,
		})
		if err != nil {
			m.l.Error(err.Error())
		}
		return tea.Quit, nil
	case "/review":
		// TODO /review
		return nil, fmt.Errorf("review command not implemented")
	default:
		fmt.Println(colorize(colorYellow, programUsage))
		return tea.Quit, nil
	}
}

func (m model) addNote(note string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		if err := m.endPendingNote(ctx); err != nil {
			return ErrorMsg{
				err: err,
			}
		}

		note, err := m.taskSvc.StartTask(ctx, startTaskRequest{
			Name:     note,
			ParentID: m.currentTask().ID,
		})
		if err != nil {
			return ErrorMsg{
				err: err,
			}
		}

		return NewNoteMsg{
			note: Note(note),
		}
	}
}

func (m model) discardLastPendingTaskItem() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		if len(m.tasks) == 0 {
			return errorMsg("nothing left to discard")
		}
		currentTask := m.currentTask()
		if !currentTask.IsPending() {
			return errorMsg("can't discard completed task")
		}

		lastItemID := currentTask.ID
		if n := currentTask.LastNote(); n != nil {
			lastItemID = n.ID
		}

		if _, err := m.taskSvc.DiscardTask(ctx, lastItemID); err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		return DiscardPendingItemMsg{
			id: lastItemID,
		}
	}
}

func (m model) displayHelp() model {
	const usage = `COMMANDS:
  /n [task]: end current task and start a new one; if task is not provided, one will be dequeued
  /k: skip current task
  /x: discard current task or note

  <note>: add a note to the current task
  /a <task>: add task to the queue
  /r <task>: rename current task or note
  /s: stash current task
`
	m = m.addAlert(colorize(colorYellow, usage))
	return m
}

func (m model) skipPendingTask() tea.Cmd {
	return func() tea.Msg {
		t := m.currentTask()
		if !t.IsPending() {
			return errorMsg("no pending task to skip")
		}
		if len(t.Notes) > 0 {
			return errorMsg("this task has been started - use /s to stash it instead")
		}

		ctx, cancel := m.newTimeout()
		defer cancel()

		next, err := m.taskSvc.PeekNextTask(ctx)
		if err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		if next == (daygo.ExistingTaskRecord{}) {
			return errorMsg("task queue is empty")
		}
		if err := m.taskSvc.SkipTask(ctx, t.ID); err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		return SkipMsg{
			id: t.ID,
		}
	}
}

// TODO parseInput only return Cmd
func (m model) handleInput() (model, tea.Cmd) {
	m.alerts = nil
	input := m.userinput.Value()
	m.userinput.Reset()
	if input == "" {
		return m, nil
	}
	if strings.HasPrefix(input, "/") {
		parts := strings.SplitN(input, " ", 2)
		switch parts[0] {
		case "/n":
			if len(parts) < 2 {
				return m, m.startNextTask()
			}
			return m, m.startNewTask(parts[1])
		case "/x":
			return m, m.discardLastPendingTaskItem()
		case "/h":
			return m.displayHelp(), nil
		case "/r":
			// TODO /r rename
			panic("rename not implemented")
			// if len(parts) < 2 {
			// 	return fmt.Errorf(`usage: /r <task>`)
			// }
			// return RenameTaskCommand{
			// 	TaskName: parts[1],
			// }, nil
		case "/a":
			if len(parts) < 2 {
				m = m.addAlert("usage: /a <task>")
				return m, nil
			}
			return m, m.queueTask(parts[1])
		case "/k":
			return m, m.skipPendingTask()
		case "/s":
			// TODO /s stash
			panic("stash not implemeneted")
		}
	}

	if !m.currentTask().IsPending() {
		return m, m.startNewTask(input)
	}
	return m, m.addNote(input)
}
