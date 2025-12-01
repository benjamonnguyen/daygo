package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	"github.com/google/uuid"
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
  /f [tag]: filter task queue by tag; if no tag provided, clear filter

  /o: end program without saving
`

var timeRe = regexp.MustCompile(`^(?:[01]\d|2[0-3])[0-5]\d$`)

type model struct {
	// children
	vp        viewport.Model // TODO refactor into taskLogModel
	userinput textinput.Model
	tbTimer   timeBlockTimer

	// supplied
	l       daygo.Logger
	taskSvc TaskSvc
	opts    modelOptions

	// state
	taskQueue TaskQueue
	taskLog   []Task
	alerts    []string
	quitting  bool
	h         int
}

type modelOptions struct {
	cmdTimeout    time.Duration
	timeFormat    string
	syncServerURL string
	syncRate      time.Duration
}

func NewModel(taskSvc TaskSvc, initialTasks []Task, logger daygo.Logger, opts modelOptions) model {
	userinput := textinput.New()
	userinput.Focus()
	userinput.CharLimit = 280
	userinput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))

	return model{
		taskSvc: taskSvc,
		taskLog: initialTasks,
		l:       logger,
		opts:    opts,

		vp:        viewport.New(0, 0),
		userinput: userinput,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.initTaskQueue,
		textinput.Blink,
		m.checkSyncServerURL,
	)
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
		m.addAlert(colorRed, msg.err.Error())
		m.l.Error(msg.err)
		var cmd tea.Cmd
		if msg.isFatal {
			cmd = tea.Quit
		}
		return m, cmd
	case timer.TimeoutMsg:
		if msg.ID == m.tbTimer.ID() {
			ended, err := m.endPendingTask()
			return m, func() tea.Msg {
				if err != nil {
					return ErrorMsg{
						err: err,
					}
				}
				timeout, cancel := m.newTimeout()
				defer cancel()
				_, err := m.taskSvc.UpsertTask(timeout, ended)
				if err != nil {
					return ErrorMsg{
						err: err,
					}
				}
				return nil
			}
		}
		return m, nil
	case SyncMsg:
		if msg.err != "" {
			m.addAlert(colorRed, msg.err)
		}
		if msg.toServerSyncCount > 0 {
			m.addAlert(colorCyan, "Synced %d tasks to server", msg.toServerSyncCount)
		}
		if len(msg.tasksToQueue) > 0 {
			m.taskQueue.Sync(msg.tasksToQueue)
			m.addAlert(colorCyan, "Queued %d tasks from sync server", len(msg.tasksToQueue))
		}
		return m, func() tea.Msg {
			time.Sleep(m.opts.syncRate)
			return m.sync()
		}
	case QueueMsg:
		m.taskQueue.Queue(msg.task)
		m.addAlert(colorCyan, "Queued \"%s\"", msg.task.Name)
		return m, nil
	case tea.WindowSizeMsg:
		m.h = msg.Height
		m.userinput.Width = msg.Width
		m.vp.Width = msg.Width
		m.resizeViewport()
		return m, nil
	case InitTaskQueueMsg:
		m.taskQueue = NewTaskQueue(msg.tasks)

		if len(m.taskLog) == 0 && m.taskQueue.Size() > 0 {
			t := m.taskQueue.Dequeue()
			t.StartedAt = time.Now()
			m.taskLog = append(m.taskLog, t)
		}

		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
		return m, nil
	case EndProgramMsg:
		return m.endProgram(msg.discardPendingTask)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			input := m.userinput.Value()
			m.userinput.Reset()
			if input == "" {
				return m, nil
			}

			var cmd tea.Cmd
			m.alerts = nil
			m, cmd = m.handleInput(input)
			m.vp.SetContent(m.renderVisibleTasks())
			m.resizeViewport()
			return m, cmd
		case tea.KeyCtrlC:
			return m.endProgram(false)
		}
	}
	return m, nil
}

func (m model) initTaskQueue() tea.Msg {
	timeout, cancel := m.newTimeout()
	defer cancel()

	tasks, err := m.taskSvc.GetPendingTasks(timeout)
	if err != nil {
		return ErrorMsg{
			err: err,
		}
	}

	return InitTaskQueueMsg{
		tasks: tasks,
	}
}

func (m model) endProgram(discardPendingTask bool) (model, tea.Cmd) {
	m.quitting = true
	if t := m.currentTask(); t.IsPending() {
		if discardPendingTask {
			_ = m.removeCurrentTask()
		} else {
			timeout, cancel := m.newTimeout()
			defer cancel()
			t.EndedAt = time.Now()
			if _, err := m.taskSvc.UpsertTask(timeout, *t); err != nil {
				logger.Error(err.Error())
			}
		}
		if curr := m.currentTask(); curr != nil {
			curr.IsTerminal = true
		}
		m.vp.SetContent(m.renderVisibleTasks())
		m.resizeViewport()
	}
	m.sync()
	return m, tea.Quit
}

func (m model) checkSyncServerURL() tea.Msg {
	if m.opts.syncServerURL == "" {
		return nil
	}
	_, err := http.Head(m.opts.syncServerURL)
	if err != nil {
		return ErrorMsg{
			err: fmt.Errorf("failed to connect to syncServerURL: %s", m.opts.syncServerURL),
		}
	}
	return SyncMsg{}
}

func (m model) sync() tea.Msg {
	if m.opts.syncServerURL == "" {
		return nil
	}

	// hardcoded timeout
	timeout, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()

	lastSync, err := m.taskSvc.GetLastSuccessfulSync(timeout, m.opts.syncServerURL)
	if err != nil {
		return ErrorMsg{
			err: err,
		}
	}

	tasksToSync, err := m.taskSvc.GetTasksToSync(timeout, m.opts.syncServerURL)
	if err != nil {
		return ErrorMsg{
			err: err,
		}
	}

	req := daygo.SyncRequest{
		LastSyncTime: lastSync.CreatedAt,
		ClientTasks:  tasksToSync,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return ErrorMsg{err: fmt.Errorf("failed to marshal sync request: %w", err)}
	}

	resp, err := http.Post(m.opts.syncServerURL+"/sync", "application/json", bytes.NewReader(reqData))
	if err != nil {
		return ErrorMsg{err: fmt.Errorf("failed to make sync request: %w", err)}
	}
	defer resp.Body.Close() //nolint:errcheck

	session := daygo.SyncSessionRecord{
		ServerURL: m.opts.syncServerURL,
	}
	if resp.StatusCode != http.StatusOK {
		session.Status = daygo.SyncStatusError
		session.Error = "sync request failed: " + resp.Status
		if _, err := m.taskSvc.UpsertSyncSession(timeout, 0, session); err != nil {
			return ErrorMsg{
				err: err,
			}
		}
		return SyncMsg{
			err: session.Error,
		}
	}

	var syncResp daygo.SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return ErrorMsg{err: fmt.Errorf("failed to decode sync response: %w", err)}
	}

	session.Status = daygo.SyncStatusPartial
	toServerSyncCnt := syncResp.ToServerSyncCount
	session.ToServerSyncCount = &toServerSyncCnt
	created, err := m.taskSvc.UpsertSyncSession(timeout, 0, session)
	if err != nil {
		return ErrorMsg{
			err: err,
		}
	}

	upserted, errs := m.taskSvc.SyncTasks(timeout, syncResp.ServerTasks)
	if len(errs) > 0 {
		// has error but still partial sync status
		session.Status = daygo.SyncStatusPartial
		session.Error = errors.Join(errs...).Error()
	} else {
		session.Status = daygo.SyncStatusSuccess
	}
	fromServerSyncCnt := len(upserted)
	session.FromServerSyncCount = &fromServerSyncCnt
	if _, err := m.taskSvc.UpsertSyncSession(timeout, created.ID, session); err != nil {
		return ErrorMsg{
			err: err,
		}
	}

	return SyncMsg{
		tasksToQueue:      upserted,
		toServerSyncCount: toServerSyncCnt,
	}
}

func (m model) renderFooter() string {
	if m.quitting {
		return ""
	}

	var footer strings.Builder
	footer.WriteRune('\n')
	footer.WriteString(m.userinput.View())
	footer.WriteString("\n\n")

	showQuit := true
	if !m.tbTimer.Timedout() {
		footer.WriteString(m.tbTimer.View())
		footer.WriteString("\n\n")
		showQuit = false
	}

	if len(m.alerts) > 0 {
		footer.WriteString(strings.Join(m.alerts, "\n"))
		footer.WriteString("\n\n")
		showQuit = false
	}

	if m.taskQueue != nil && len(m.taskQueue.AllTags()) > 0 {
		footer.WriteString(m.renderTags())
		footer.WriteString("\n\n")
	}

	if showQuit {
		footer.WriteString(faintStyle.Render("(ctrl+c to quit)"))
		footer.WriteRune('\n')
	}

	return footer.String()
}

func (m model) renderTags() string {
	tags := m.taskQueue.AllTags()
	if len(tags) == 0 {
		return ""
	}

	// sort to get consistent ordering
	slices.Sort(tags)

	// Format tags with # prefix
	var tagLines []string
	var currentLine []string
	lineWidth := 0

	for _, tag := range tags {
		tagText := "#" + tag

		// Apply styling
		var styledTag string
		if tag == m.taskQueue.FilterTag() {
			styledTag = colorize(colorCyan, tagText)
		} else {
			styledTag = faintStyle.Render(tagText)
		}

		tagWidth := lipgloss.Width(styledTag)

		// Check if adding this tag would exceed line width
		// Assume reasonable max width of 80 characters for the footer
		if lineWidth > 0 && lineWidth+tagWidth+1 >= m.vp.Width {
			// Start new line
			tagLines = append(tagLines, strings.Join(currentLine, " "))
			currentLine = []string{styledTag}
			lineWidth = tagWidth
		} else {
			// Add to current line with space separator
			currentLine = append(currentLine, styledTag)
			lineWidth += tagWidth + 1
		}
	}

	// Add the last line
	if len(currentLine) > 0 {
		tagLines = append(tagLines, strings.Join(currentLine, " "))
	}

	return strings.Join(tagLines, "\n")
}

func (m model) View() string {
	return lipgloss.JoinVertical(0, m.vp.View(), m.renderFooter())
}

func (m model) newTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), m.opts.cmdTimeout)
}

func (m *model) addAlert(c color, format string, args ...any) {
	m.alerts = append(m.alerts, colorize(c, fmt.Sprintf(format, args...)))
}

func (m model) currentTask() *Task {
	if len(m.taskLog) == 0 {
		return nil
	}
	return &m.taskLog[len(m.taskLog)-1]
}

// endPendingTask returns error if no pending task
func (m *model) endPendingTask() (Task, error) {
	now := time.Now()
	t := m.currentTask()
	if !t.IsPending() {
		return Task{}, fmt.Errorf("no pending task")
	}

	t.EndedAt = now
	if n := t.LastNote(); n != nil {
		n.EndedAt = now
	}

	return *t, nil
}

func (m *model) resizeViewport() {
	tasksHeight := lipgloss.Height(m.renderVisibleTasks())
	footerHeight := lipgloss.Height(m.renderFooter())
	m.vp.Height = min(tasksHeight, m.h-footerHeight)
	m.vp.GotoBottom()
}

func (m model) renderVisibleTasks() string {
	if len(m.taskLog) == 0 {
		return ""
	}
	var lines []string
	availableHeight := m.vp.Height
	for i := len(m.taskLog) - 1; i >= 0 && availableHeight >= 0; i-- {
		line, h := m.taskLog[i].Render(m.opts.timeFormat)
		availableHeight -= h
		if i != len(m.taskLog)-1 {
			line = faintStyle.Render(line)
		}
		lines = append(lines, line)
	}

	slices.Reverse(lines)
	return strings.Join(lines, "\n")
}

func (m *model) startNewTask(task string) Task {
	_, _ = m.endPendingTask()
	t := TaskFromName(task)
	t.StartedAt = time.Now()
	m.taskLog = append(m.taskLog, t)
	return t
}

func (m *model) addNote(note string) {
	now := time.Now()
	parent := m.currentTask()
	if n := parent.LastNote(); n != nil {
		n.EndedAt = now
	}

	n := daygo.TaskRecord{
		Name:      note,
		ParentID:  parent.ID,
		StartedAt: now,
	}
	parent.Notes = append(parent.Notes, Note(n))
}

func (m *model) deleteLastPendingTaskItem() daygo.ExistingTaskRecord {
	currentTask := m.currentTask()
	if !currentTask.IsPending() {
		return daygo.ExistingTaskRecord{}
	}

	note := m.removeLastNote()
	// using StartedAt to determine if IsZero - addNote() sets it
	if note.StartedAt.IsZero() {
		return m.removeCurrentTask().ExistingTaskRecord
	}
	return daygo.ExistingTaskRecord{}
}

func (m *model) removeCurrentTask() Task {
	if t := m.currentTask(); t != nil {
		tmp := *t
		m.taskLog = m.taskLog[:len(m.taskLog)-1]
		return tmp
	}
	return Task{}
}

func (m *model) removeLastNote() Note {
	if t := m.currentTask(); t != nil {
		if n := t.LastNote(); n != nil {
			t.Notes = t.Notes[:len(t.Notes)-1]
			return *n
		}
	}
	return Note{}
}

func (m *model) timeBlockPendingTask(arg string) {
	if !timeRe.MatchString(arg) {
		m.addAlert(colorYellow, "usage: /t <HHMM>")
		return
	}
	task := m.currentTask()
	if !task.IsPending() {
		m.addAlert(colorRed, "no pending task to time block")
		return
	}
	now, _ := time.Parse("1504", time.Now().Format("1504"))
	endTime, _ := time.Parse("1504", arg)

	if endTime.Compare(now) < 0 {
		endTime = endTime.Add(12 * time.Hour)
	}

	m.tbTimer.Model = timer.New(endTime.Sub(now))
}

func (m *model) editPendingItem(edit string) *Task {
	t := m.currentTask()
	if n := t.LastNote(); n != nil {
		n.Name = edit
	} else {
		t.Name = edit
		t.Tags = extractTags(edit)
		return t
	}
	return nil
}

func (m model) handleInput(input string) (model, tea.Cmd) {
	if strings.HasPrefix(input, "/") {
		parts := strings.SplitN(input, " ", 2)
		switch parts[0] {
		case "/n":
			var started Task
			if len(parts) < 2 {
				if m.taskQueue.Size() == 0 {
					m.addAlert(colorRed, "task queue is empty")
					return m, nil
				}

				started = m.taskQueue.Dequeue()
			} else {
				started = TaskFromName(parts[1])
			}

			var persistEnded tea.Cmd
			ended, err := m.endPendingTask()
			if err == nil {
				persistEnded = func() tea.Msg {
					timeout, cancel := m.newTimeout()
					defer cancel()
					if _, err := m.taskSvc.UpsertTask(timeout, ended); err != nil {
						return ErrorMsg{
							err: err,
						}
					}
					return nil
				}
			}
			started.StartedAt = time.Now()
			m.taskLog = append(m.taskLog, started)
			return m, persistEnded
		case "/x":
			if !m.currentTask().IsPending() {
				m.addAlert(colorRed, "nothing left to delete")
				return m, nil
			}
			deleted := m.deleteLastPendingTaskItem()
			if !m.currentTask().IsPending() && m.taskQueue.Size() > 0 {
				started := m.taskQueue.Dequeue()
				started.StartedAt = time.Now()
				m.taskLog = append(m.taskLog, started)
			}
			var cmd tea.Cmd
			if !deleted.CreatedAt.IsZero() {
				cmd = func() tea.Msg {
					timeout, c := m.newTimeout()
					defer c()

					if _, err := m.taskSvc.DeleteTask(timeout, deleted.ID); err != nil {
						return ErrorMsg{
							err: err,
						}
					}
					return nil
				}
			}
			return m, cmd
		case "/h":
			m.addAlert(colorYellow, commandHelp)
			return m, nil
		case "/e":
			if len(parts) < 2 {
				m.addAlert(colorYellow, "usage: /e <edit>")
				return m, nil
			}
			var cmd tea.Cmd
			if t := m.editPendingItem(input); t != nil && t.ID != uuid.Nil {
				// update existing task
				cmd = func() tea.Msg {
					timeout, c := m.newTimeout()
					defer c()
					if _, err := m.taskSvc.UpsertTask(timeout, *t); err != nil {
						return ErrorMsg{
							err: err,
						}
					}
					return nil
				}
			}
			return m, cmd
		case "/a":
			if len(parts) < 2 {
				m.addAlert(colorYellow, "usage: /a <task>")
				return m, nil
			}
			t := TaskFromName(parts[1])
			return m, func() tea.Msg {
				timeout, c := m.newTimeout()
				defer c()
				inserted, err := m.taskSvc.UpsertTask(timeout, t)
				if err != nil {
					return ErrorMsg{
						err: err,
					}
				}
				return QueueMsg{
					task: inserted,
				}
			}
		case "/k":
			if !m.currentTask().IsPending() {
				m.addAlert(colorRed, "no pending task to skip")
				return m, nil
			}
			if m.taskQueue.Size() == 0 {
				m.addAlert(colorRed, "task queue is empty")
				return m, nil

			}

			curr := m.removeCurrentTask()
			t := m.taskQueue.Dequeue()
			t.StartedAt = time.Now()
			m.taskLog = append(m.taskLog, t)

			return m, func() tea.Msg {
				timeout, c := m.newTimeout()
				defer c()
				updated, err := m.taskSvc.UpsertTask(timeout, curr)
				if err != nil {
					return ErrorMsg{
						err: err,
					}
				}
				return QueueMsg{
					task: updated,
				}
			}
		case "/t":
			if len(parts) < 2 {
				m.addAlert(colorYellow, "usage: /t <HHMM>")
				return m, nil
			}
			m.timeBlockPendingTask(parts[1])
			return m, m.tbTimer.Init()
		case "/f":
			if len(parts) < 2 {
				m.taskQueue.SetFilter("")
			} else {
				m.taskQueue.SetFilter(parts[1])
			}
			return m, nil
		case "/o":
			return m, func() tea.Msg {
				return EndProgramMsg{
					discardPendingTask: true,
				}
			}
		}
	}

	if !m.currentTask().IsPending() {
		m.startNewTask(input)
		return m, nil
	}
	m.addNote(input)
	return m, nil
}
