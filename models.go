package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"slices"
	"strings"
	"time"
)

type listKeyMap struct {
	previewItem key.Binding
	reloadData  key.Binding
	goBack      key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		previewItem: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "preview task folder structure")),
		goBack:      key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "go to previous folder")),
		reloadData:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload data")),
	}
}

type TaskFolder struct {
	Name                string `json:"Name,omitempty"`
	Desc                string `json:"Desc,omitempty"`
	Progress            progress.Model
	Parent              *TaskFolder   `json:"Parent,omitempty"`
	ChildrenTasks       []*Task       `json:"children_tasks,omitempty"`
	ChildrenTaskFolders []*TaskFolder `json:"children_task_folders,omitempty"`
	Status              status        `json:"Status"`
}

func (i *TaskFolder) Title() string       { return "📁" + i.Name }
func (i *TaskFolder) Description() string { return i.Desc }
func (i *TaskFolder) FilterValue() string { return i.Name }
func (i *TaskFolder) returnTree() string {
	s := "Task View \n"
	//TODO make this recursive?

	for _, item := range i.ChildrenTaskFolders {
		s += item.Title() + "\n"
		for _, i := range item.ChildrenTasks {
			s += " - " + i.returnStatusString()

		}

	}
	for _, i := range i.ChildrenTasks {
		s += " -" + i.returnStatusString()

	}
	return s
}
func (i *TaskFolder) returnPath() string {
	var pathParts []string
	current := i
	pathParts = append(pathParts, "Root")
	for current != nil {
		pathParts = append(pathParts, current.Title())
		current = current.Parent
	}

	//otherwise it ends up being bottom-to-top
	slices.Reverse(pathParts)
	return strings.Join(pathParts, " > ")
}
func (t *Task) returnStatusString() string {
	var s string
	render_warning := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF593B")).Render
	if t.Completed {
		s += "📝 (✓Completed!) " + t.Title() + "\n"
		s += ""
		s += t.Description() + "\n"
	} else {
		s += "📝 " + t.Title() + "\n"
		if !t.DueDate.IsZero() {
			if t.Overdue {
				s += render_warning("📅 Overdue! %s\n", t.DueDate.Format("2006-01-02 15:04:05"))
			} else {
				s += "📅" + t.DueDate.Format("DD/MM/06 15:04:05 ") + "\n"
			}
		}
		s += t.Description() + "\n"
	}

	return s
}

func (i *TaskFolder) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		i.Progress.Width = msg.Width - padding*2 - 4
		if i.Progress.Width > maxWidth {
			i.Progress.Width = maxWidth
		}
		return nil
	default:
		return nil
	}
}

func (i *TaskFolder) View() string {

	pad := strings.Repeat(" ", padding)
	return "" + pad + i.Name + "" + pad + i.Progress.ViewAs(0.12) + "\n"
}

type Task struct {
	ParentFolder *TaskFolder
	Name         string
	Desc         string
	Completed    bool
	DueDate      time.Time
	Overdue      bool
}

var keys = newListKeyMap()

func (t *Task) FilterValue() string { return t.Name }
func (t *Task) Title() string       { return t.Name }
func (t *Task) Description() string { return t.Desc }
func (t *Task) setTimeStatus() {
	if time.Now().After(t.DueDate) {
		t.Overdue = true
		t.ParentFolder.Status.overdue += 1
	} else {
		t.Overdue = false
	}
}
func (t *Task) setCompletionStatus(status bool) {
	if status {
		t.Completed = true
		t.ParentFolder.Status.completed += 1
	} else {
		t.Completed = false
		t.ParentFolder.Status.completed -= 1
	}
}

type status struct {
	completed int
	total     int
	overdue   int
}

func (s *status) print() string {

	//render_info := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFED33")).Render
	var pp_string string
	if s.overdue > 0 {
		pp_string += renderWarning(fmt.Sprintf("%d overdue, ", s.overdue))
	} else {
		pp_string += fmt.Sprintf("%d overdue, ", s.overdue)
	}
	pp_string += fmt.Sprintf("%d/%d completed \n", s.completed, s.total)
	return pp_string
}
