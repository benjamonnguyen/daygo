package main

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/benjamonnguyen/daygo"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
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
  daygo: start next queued task
  daygo <task>: start new task
  daygo /a <task>: add task to queue
  daygo /r [days_ago]: review tasks for date some number of days ago (default 0)`

const commandHelp = `COMMANDS:
  /n [task]: end current task and start a new one; if task is not provided, one will be dequeued
  /k: skip current task
  /x: delete current task or note

  <note>: add a note to the current task
  /a <task>: add task to the queue
  /e <edit>: edit text of current item
  /t <HHMM>: set a time to auto-end task

	/o: end program without saving
`

var timeRe = regexp.MustCompile(`^(?:[01]\d|2[0-3])[0-5]\d$`)

type model struct {
	// children
	vp        viewport.Model
	userinput textinput.Model
	tbTimer   timeBlockTimer

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

func (m model) Init() tea.Cmd {
	init := func() tea.Msg {
		return InitMsg{}
	}
	return tea.Batch(init, textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var tiCmd, vpCmd, tbCmd, cmd tea.Cmd

	m, cmd = m.updateParent(msg)

	// update children

	m.userinput, tiCmd = m.userinput.Update(msg)
	m.tbTimer.Model, tbCmd = m.tbTimer.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// vp udpates on KeyMsg was causing a view flickering bug
	default:
		m.vp, vpCmd = m.vp.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd, cmd, tbCmd)
}

func (m model) updateParent(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case ErrorMsg:
		m = m.addAlert(colorize(colorRed, msg.err.Error()))
		return m, tea.Quit
	case AlertMsg:
		if msg.color != colorNone {
			m = m.addAlert(colorize(msg.color, msg.message))
		} else {
			m = m.addAlert(msg.message)
		}
		return m, nil
	case NewTaskMsg:
		return m.handleNewTask(msg), nil
	case NewNoteMsg:
		return m.handleNewNote(msg), nil
	case QueueTaskMsg:
		alert := fmt.Sprintf(`Queued up "%s"`, msg.task)
		m = m.addAlert(colorize(colorCyan, alert))
		return m, nil
	case EditItemMsg:
		return m.handleEditItem(msg), nil
	case SkipTaskMsg:
		task := m.currentTask()
		if msg.id != task.ID {
			panic("skip msg: out of sync")
		}
		m.tasks = m.tasks[:len(m.tasks)-1]
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
		return m, m.startNextTask()
	case DeletePendingItemMsg:
		m = m.handleDeletePendingItem(msg)
		return m, nil
	case EndPendingTaskMsg:
		task := m.currentTask()
		if msg.id != task.ID {
			panic("EndPendingTaskMsg out of sync")
		}
		if !task.IsPending() {
			return m, nil
		}
		task.EndedAt = time.Now()
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
		return m, nil
	case TimeBlockMsg:
		task := m.currentTask()
		if msg.id != task.ID {
			panic("TimeBlockMsg out of sync")
		}
		m.tbTimer.Model = timer.New(msg.duration)
		return m, tea.Batch(m.tbTimer.Init())
	case timer.TimeoutMsg:
		if msg.ID == m.tbTimer.ID() {
			return m, func() tea.Msg {
				timeout, cancel := m.newTimeout()
				defer cancel()
				if err := m.endPendingTask(timeout); err != nil {
					return displayWarning(err.Error())
				}
				return EndPendingTaskMsg{
					id: m.currentTask().ID,
				}
			}
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.h = msg.Height
		m.userinput.Width = msg.Width
		m.vp.Width = msg.Width
		m.resizeViewport()
		return m, nil
	case InitMsg:
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
		return m, nil
	case EndProgramMsg:
		return m.endProgram(msg.discardPendingTask)
	case tea.KeyMsg:
		// TODO consider a mutex to ignore input until state is consistent
		switch msg.Type {
		case tea.KeyEnter:
			m.alerts = nil
			input := m.userinput.Value()
			m.userinput.Reset()
			if input == "" {
				return m, nil
			}

			return m, m.handleInput(input)
		case tea.KeyCtrlC:
			return m.endProgram(false)
		}
	}
	return m, nil
}

func (m model) handleEditItem(msg EditItemMsg) model {
	t := m.currentTask()
	id := t.ID
	n := t.LastNote()
	if n != nil {
		n.Name = msg.edit
		id = n.ID
	} else {
		t.Name = msg.edit
	}
	if msg.id != id {
		panic("skip msg: out of sync")
	}
	m.vp.SetContent(m.renderVisibleTasks())
	m.resizeViewport()
	return m
}

func (m model) endProgram(discardPendingTask bool) (model, tea.Cmd) {
	m.quitting = true
	if discardPendingTask && len(m.tasks) > 0 {
		m.tasks = m.tasks[:len(m.tasks)-1]
	}
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

func (m model) handleDeletePendingItem(msg DeletePendingItemMsg) model {
	t := m.currentTask()
	if !t.IsPending() {
		panic("deletePendingItemMsg: out of sync")
	}
	if len(t.Notes) > 0 {
		if n := t.LastNote(); n == nil || n.ID != msg.id {
			panic("deletePendingItemMsg: out of sync")
		}
		t.Notes = t.Notes[:len(t.Notes)-1]
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
	} else {
		if t.ID != msg.id {
			panic("deletePendingItemMsg: out of sync")
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
	h := 6
	if m.tbTimer.Timedout() {
		h += len(m.alerts)
	}
	return h
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
		if !m.tbTimer.Timedout() {
			footer.WriteString(m.tbTimer.View())
		} else if len(m.alerts) > 0 {
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
	m.tbTimer.Timeout = 0
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
			return displayWarning(err.Error())
		}
		if task == (daygo.ExistingTaskRecord{}) {
			return displayWarning("task queue is empty!")
		}

		if err := m.endPendingTask(ctx); err != nil {
			return displayWarning(err.Error())
		}

		task, err = m.taskSvc.StartTask(ctx, startTaskRequest{
			ID: task.ID,
		})
		if err != nil {
			return displayWarning(err.Error())
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
			return displayWarning(err.Error())
		}
		task, err := m.taskSvc.StartTask(ctx, startTaskRequest{
			Name: task,
		})
		if err != nil {
			return displayWarning(err.Error())
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
			return displayWarning(err.Error())
		}
		return QueueTaskMsg{
			task: task,
		}
	}
}

func (m model) addNote(note string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		if err := m.endPendingNote(ctx); err != nil {
			return displayWarning(err.Error())
		}

		note, err := m.taskSvc.StartTask(ctx, startTaskRequest{
			Name:     note,
			ParentID: m.currentTask().ID,
		})
		if err != nil {
			return displayWarning(err.Error())
		}

		return NewNoteMsg{
			note: Note(note),
		}
	}
}

func (m model) deleteLastPendingTaskItem() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.newTimeout()
		defer cancel()

		if len(m.tasks) == 0 {
			return warningAlert("nothing left to delete")
		}
		currentTask := m.currentTask()
		if !currentTask.IsPending() {
			return warningAlert("can't delete completed task")
		}

		lastItemID := currentTask.ID
		if n := currentTask.LastNote(); n != nil {
			lastItemID = n.ID
		}

		if _, err := m.taskSvc.DeleteTask(ctx, lastItemID); err != nil {
			return warningAlert(err.Error())
		}
		return DeletePendingItemMsg{
			id: lastItemID,
		}
	}
}

func (m model) skipPendingTask() tea.Cmd {
	return func() tea.Msg {
		t := m.currentTask()
		if !t.IsPending() {
			return displayWarning("no pending task to skip")
		}
		if len(t.Notes) > 0 {
			return displayWarning("can't skip a completed task")
		}

		ctx, cancel := m.newTimeout()
		defer cancel()

		next, err := m.taskSvc.PeekNextTask(ctx)
		if err != nil {
			return displayWarning(err.Error())
		}
		if next == (daygo.ExistingTaskRecord{}) {
			return displayWarning("task queue is empty")
		}
		if err := m.taskSvc.SkipTask(ctx, t.ID); err != nil {
			return displayWarning(err.Error())
		}
		return SkipTaskMsg{
			id: t.ID,
		}
	}
}

func (m model) editPendingItem(edit string) tea.Cmd {
	return func() tea.Msg {
		timeout, cancel := m.newTimeout()
		defer cancel()
		t := m.currentTask()
		if !t.IsPending() {
			return displayWarning("no pending item to edit")
		}
		id := t.ID
		if n := t.LastNote(); n != nil {
			id = n.ID
		}

		if _, err := m.taskSvc.RenameTask(timeout, id, edit); err != nil {
			return displayWarning(err.Error())
		}

		return EditItemMsg{
			id:   id,
			edit: edit,
		}
	}
}

func (m model) timeBlockPendingTask(arg string) tea.Cmd {
	return func() tea.Msg {
		if !timeRe.MatchString(arg) {
			return displayHelp("usage: /t <HHMM>")
		}
		task := m.currentTask()
		if !task.IsPending() {
			return displayWarning("no pending task to time block")
		}
		now, _ := time.Parse("1504", time.Now().Format("1504"))
		endTime, _ := time.Parse("1504", arg)

		logger.Debug("timeblockpendingtask", "endtime", endTime.String(), "now", now.String())
		if endTime.Compare(now) < 0 {
			endTime = endTime.Add(12 * time.Hour)
		}

		return TimeBlockMsg{
			id:       task.ID,
			duration: endTime.Sub(now),
		}
	}
}

func (m model) handleInput(input string) tea.Cmd {
	if strings.HasPrefix(input, "/") {
		parts := strings.SplitN(input, " ", 2)
		switch parts[0] {
		case "/n":
			if len(parts) < 2 {
				return m.startNextTask()
			}
			return m.startNewTask(parts[1])
		case "/x":
			return m.deleteLastPendingTaskItem()
		case "/h":
			return displayHelp(commandHelp)
		case "/e":
			if len(parts) < 2 {
				return displayHelp("usage: /e <edit>")
			}
			return m.editPendingItem(parts[1])
		case "/a":
			if len(parts) < 2 {
				return displayHelp("usage: /a <task>")
			}
			return m.queueTask(parts[1])
		case "/k":
			return m.skipPendingTask()
		case "/t":
			if len(parts) < 2 {
				return displayHelp("usage: /t <HHMM>")
			}
			return m.timeBlockPendingTask(parts[1])
		case "/o":
			return func() tea.Msg {
				return EndProgramMsg{
					discardPendingTask: true,
				}
			}
		}
	}

	if !m.currentTask().IsPending() {
		return m.startNewTask(input)
	}
	return m.addNote(input)
}
