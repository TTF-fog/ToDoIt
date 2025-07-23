package main

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

type TaskFolder struct {
	title                 string
	desc                  string
	progress              progress.Model
	parent                *TaskFolder
	children_tasks        []*Task
	children_task_folders []*TaskFolder
	status                status
}

func (i *TaskFolder) Title() string       { return i.title }
func (i *TaskFolder) Description() string { return i.desc }
func (i *TaskFolder) FilterValue() string { return i.title }
func (i *TaskFolder) returnTree() string {
	s := "Task View \n"
	//TODO make this recursive?

	for _, item := range i.children_task_folders {
		s += item.Title() + "\n"
		for _, i := range item.children_tasks {
			s += " - " + i.returnStatusString()

		}

	}
	for _, i := range i.children_tasks {
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
		current = current.parent
	}

	//otherwise it ends up being bottom-to-top
	//slices.Reverse(pathParts)
	return strings.Join(pathParts, " > ")
}
func (t *Task) returnStatusString() string {
	var s string
	render_warning := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF593B")).Render
	if t.completed {
		s += "📝 (✓Completed!) " + t.Title() + "\n"
		s += ""
		s += t.Description() + "\n"
	} else {
		s += "📝 " + t.Title() + "\n"
		if !t.dueDate.IsZero() {
			if t.overdue {
				s += render_warning("📅 Overdue! %s\n", t.dueDate.Format("2006-01-02 15:04:05"))
			} else {
				s += "📅" + t.dueDate.Format("DD/MM/06 15:04:05 ") + "\n"
			}
		}
		s += t.Description() + "\n"
	}

	return s
}

func (i *TaskFolder) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		i.progress.Width = msg.Width - padding*2 - 4
		if i.progress.Width > maxWidth {
			i.progress.Width = maxWidth
		}
		return nil
	default:
		return nil
	}
}

func (i *TaskFolder) View() string {

	pad := strings.Repeat(" ", padding)
	return "" + pad + i.title + "" + pad + i.progress.ViewAs(0.12) + "\n"
}
