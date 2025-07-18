package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Task struct {
	name        string
	description string
	completed   bool
	dueDate     time.Time
}

func (t Task) FilterValue() string {
	return t.name
}

func (t Task) Title() string {
	return t.name
}
func (t Task) Description() string {
	return t.description
}

type status struct {
	completed int
	total     int
	overdue   int
}

func (s *status) print() string {
	return fmt.Sprintf("%d/%d completed, %d overdue", s.completed, s.total, s.overdue)
}

type TaskFolder struct {
	title                 string
	desc                  string
	progress              float64
	parent                *TaskFolder //so we can handle a folder of folders
	children_tasks        []Task
	children_task_folders []*TaskFolder
	status                status
}

func (i *TaskFolder) Title() string       { return i.title }
func (i *TaskFolder) Description() string { return i.desc }
func (i *TaskFolder) FilterValue() string { return i.title }
func (i *TaskFolder) returnPath() string {
	s := "Task View\n"
	for _, item := range i.children_task_folders { //once for nested tasks
		s += item.Title() + "\n"
		for _, i := range item.children_tasks {
			s += "- 📝 " + i.Title() + "\n"
		}

	}
	for _, i := range i.children_tasks {
		s += "📝 " + i.Title() + "\n"
	}
	return s
}

type model struct {
	list          list.Model
	statusString  string
	currentFolder *TaskFolder
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if selectedItem, ok := m.list.SelectedItem().(*TaskFolder); ok {
				m.recreateList(selectedItem)
			}
			return m, nil
		case "b":
			if m.currentFolder != nil {
				m.recreateList(m.currentFolder.parent)

			}
			return m, nil
		case "p":
			switch v := m.list.SelectedItem().(type) {
			case Task:
				m.list.NewStatusMessage("Cannot preview Task!")
			case *TaskFolder:
				m.statusString = v.returnPath()
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	var s string

	s += lipgloss.JoinHorizontal(lipgloss.Left, docStyle.Render(m.statusString), docStyle.Render(m.list.View()))
	return s
}
func (m *model) recreateList(folder *TaskFolder) {
	if folder == nil {
		return
	}
	m.currentFolder = folder
	var items []list.Item

	for _, child := range folder.children_task_folders {
		if !strings.HasPrefix(child.title, "📁") {
			child.title = "📁 " + child.title
		}
		items = append(items, child)
	}
	for _, child := range folder.children_tasks {
		if !strings.HasPrefix(child.name, "📝") {
			child.name = "📝 " + child.name
		}
		items = append(items, child)
	}
	delegate := list.NewDefaultDelegate()
	newList := list.New(items, delegate, 0, 0)
	newList.Title = fmt.Sprintf("%s, %s", m.currentFolder.Title(), m.currentFolder.status.print())
	newList.SetSize(m.list.Width(), m.list.Height())
	m.list = newList
}
func main() {
	items := return_test_data()
	delegate := list.NewDefaultDelegate()
	root := TaskFolder{title: "To-Dos"}
	for _, item := range items {
		if folder, ok := item.(*TaskFolder); ok {
			folder.parent = &root
			root.children_task_folders = append(root.children_task_folders, folder)
		}
	}
	m := model{list: list.New(nil, delegate, 0, 0)}
	m.recreateList(&root)
	m.statusString = "Press P to preview an Item!"
	m.list.Title = "Task View "
	p := tea.NewProgram(&m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
