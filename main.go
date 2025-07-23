package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.dalton.dog/bubbleup"
	"io"
	"os"
	"strings"
	"time"
)

var docStyle = lipgloss.NewStyle().
	Margin(1, 2).
	BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63"))

const (
	maxWidth       = 80
	minFrameWidth  = 200
	minFrameHeight = 200
	padding        = 2
)

var index int

type Task struct {
	parentFolder *TaskFolder
	name         string
	description  string
	completed    bool
	dueDate      time.Time
	overdue      bool
}

var keys = newListKeyMap()

func (t *Task) FilterValue() string { return t.name }
func (t *Task) Title() string       { return t.name }
func (t *Task) Description() string { return t.description }
func (t *Task) setTimeStatus() {
	if time.Now().After(t.dueDate) {
		t.overdue = true
		t.parentFolder.status.overdue += 1
	} else {
		t.overdue = false
	}
}
func (t *Task) setCompletionStatus(status bool) {
	if status {
		t.completed = true
		t.parentFolder.status.completed += 1
	} else {
		t.completed = false
		t.parentFolder.status.completed -= 1
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

type itemDelegate struct{}

func (d itemDelegate) Height() int { return 4 }

func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	switch item := listItem.(type) {
	case *TaskFolder:
		s := item

		var p float64
		if s.status.total > 0 {
			p = float64(s.status.completed) / float64(s.status.total)
		}

		str := fmt.Sprintf("%s\n %s \n%s", s.title, s.status.print(), s.progress.ViewAs(p))
		fn := lipgloss.NewStyle().PaddingLeft(4).Render
		if index == m.Index() {
			fn = func(s ...string) string {
				return lipgloss.NewStyle().
					PaddingLeft(2).
					Foreground(lipgloss.Color("201")).
					Background(lipgloss.Color("235")).
					Render("> " + strings.Join(s, " "))
			}
		}
		fmt.Fprint(w, fn(str))
		return
	case *Task:
		s := item
		if s.overdue {
		}
		str := fmt.Sprintf("%s \n", s.returnStatusString())

		fn := lipgloss.NewStyle().PaddingLeft(4).Render
		if index == m.Index() {
			fn = func(s ...string) string {
				return lipgloss.NewStyle().
					PaddingLeft(2).
					Foreground(lipgloss.Color("201")).
					Background(lipgloss.Color("235")).
					Render("> " + strings.Join(s, " "))
			}
		}

		fmt.Fprint(w, fn(str))
	}
}

type model struct {
	list          list.Model
	statusString  string
	currentFolder *TaskFolder
	alert         bubbleup.AlertModel
}

func (m *model) Init() tea.Cmd {
	return m.alert.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var alertCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r": //reload test data
			main()
		case "enter":
			index = m.list.GlobalIndex()
			switch selectedItem := m.list.SelectedItem().(type) {
			case *TaskFolder:
				m.recreateList(selectedItem, 0)
			case *Task:
				selectedItem.setCompletionStatus(!selectedItem.completed)

				m.recreateList(selectedItem.parentFolder, m.list.GlobalIndex())

			}

			return m, nil
		case "b":
			if m.currentFolder != nil {
				m.recreateList(m.currentFolder.parent, index)
			}
			return m, nil
		case "p":
			switch v := m.list.SelectedItem().(type) {
			case *Task:
				m.alert.NewAlertCmd(bubbleup.ErrorKey, "Cannot preview Task!")
			case *TaskFolder:
				m.statusString = v.returnTree()

			}

		}

	case tea.WindowSizeMsg:
		v, h := docStyle.GetFrameSize()
		if v > minFrameWidth && h > minFrameHeight {
			m.alert.NewAlertCmd(bubbleup.ErrorKey, "Frame dimensions too small :(")
		}
		statusWidth := lipgloss.Width(docStyle.Render(m.statusString))
		m.list.SetSize(msg.Width-statusWidth-h, msg.Height-v)
		childMsg := tea.WindowSizeMsg{Width: m.list.Width(), Height: m.list.Height()}
		for _, val := range m.list.Items() {
			if v, ok := val.(*TaskFolder); ok {
				v.update(childMsg)
			}
		}
	}
	var cmd tea.Cmd
	outAlert, outCmd := m.alert.Update(msg)
	m.alert = outAlert.(bubbleup.AlertModel)
	m.list, cmd = m.list.Update(msg)
	return m, tea.Batch(alertCmd, cmd, outCmd)
}

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
func (m *model) View() string {
	var s string

	s += lipgloss.JoinHorizontal(lipgloss.Left, docStyle.Render(m.statusString), docStyle.Render(m.list.View()))

	return m.alert.Render(s)
}
func (m *model) recreateList(folder *TaskFolder, selectedItem int) {
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

		items = append(items, child)
	}
	delegate := itemDelegate{}
	newList := list.New(items, delegate, 0, 0)
	newList.Title = fmt.Sprintf("%s \n %s", m.currentFolder.returnPath(), m.currentFolder.status.print())
	newList.SetSize(m.list.Width(), m.list.Height())

	m.list = newList
	m.list.Select(selectedItem)
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.goBack,
			keys.previewItem,
		}
	}
}
func main() {

	items := return_test_data()
	delegate := itemDelegate{}
	root := TaskFolder{title: "To-Dos"}
	for _, item := range items {
		if folder, ok := item.(*TaskFolder); ok {
			folder.parent = &root
			root.children_task_folders = append(root.children_task_folders, folder)
		}
	}
	m := model{list: list.New(nil, delegate, 0, 0)}
	m.recreateList(&root, m.list.GlobalIndex())
	m.statusString = "Press P to preview an Item!"
	m.list.Title = "Task View "
	p := tea.NewProgram(&m)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
