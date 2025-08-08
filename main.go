package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.dalton.dog/bubbleup"
	"io"
	"os"
	"strings"
)

var docStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63"))

const (
	maxWidth       = 40
	minFrameWidth  = 200
	minFrameHeight = 200
	padding        = 1
)

var index int

type itemDelegate struct{}

func (d itemDelegate) Height() int { return 6 }

func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	switch item := listItem.(type) {
	case *TaskFolder:
		s := item

		var p float64
		if s.Status.Total > 0 {
			p = float64(s.Status.Completed) / float64(s.Status.Total)
		}

		str := fmt.Sprintf("%s \n %s \n %s", s.Title(), s.Status.print(), s.Progress.ViewAs(p))
		fn := lipgloss.NewStyle().PaddingLeft(4).Render
		if index == m.Index() {
			fn = func(s ...string) string {
				return lipgloss.NewStyle().
					Padding(0, padding).
					Foreground(lipgloss.Color("201")).
					Background(lipgloss.Color("235")).
					Render("> " + strings.Join(s, " "))
			}
		}
		fmt.Fprint(w, fn(str))
		return
	case *Task:
		s := item
		if s.Overdue {
		}
		str := fmt.Sprintf("%s \n", s.returnStatusString())

		fn := lipgloss.NewStyle().PaddingLeft(4).Render
		if index == m.Index() {
			fn = func(s ...string) string {
				return lipgloss.NewStyle().Padding(0, padding).
					Foreground(lipgloss.Color("201")).
					Background(lipgloss.Color("235")).
					Render("> " + strings.Join(s, " "))
			}
		}

		fmt.Fprint(w, fn(str))
	}
}

type CreateNewUI struct {
	taskNameInput          textinput.Model
	status                 string
	taskDescInput          textarea.Model
	shouldCreateTaskFolder bool
	creatingTask           bool
}
type model struct {
	list          list.Model
	statusString  string
	currentFolder *TaskFolder
	alert         bubbleup.AlertModel
	rootFolder    *TaskFolder
	createNewUI   *CreateNewUI
}

func (m *model) Init() tea.Cmd {
	return m.alert.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var alertCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.createNewUI.creatingTask {
			switch msg.String() {
			case "enter":
				if m.createNewUI.shouldCreateTaskFolder {
					m.currentFolder.ChildrenTaskFolders = append(m.currentFolder.ChildrenTaskFolders, &TaskFolder{
						Name:   m.createNewUI.taskNameInput.Value(),
						Parent: m.currentFolder,
						Desc:   m.createNewUI.taskDescInput.Value(),
					})
					m.recreateList(m.currentFolder, 0)
					m.createNewUI.creatingTask = false
					m.createNewUI.taskNameInput.Reset()
					m.createNewUI.taskDescInput.Reset()
					break
				}
				m.currentFolder.ChildrenTasks = append(m.currentFolder.ChildrenTasks, &Task{
					Name:         m.createNewUI.taskNameInput.Value(),
					ParentFolder: m.currentFolder,
					Desc:         m.createNewUI.taskDescInput.Value(),
				})
				m.recreateList(m.currentFolder, 0)
				m.createNewUI.creatingTask = false
				m.createNewUI.taskNameInput.Reset()
				m.createNewUI.taskDescInput.Reset()
			case "esc":
				m.createNewUI.creatingTask = false
				m.createNewUI.taskNameInput.Reset()
				m.createNewUI.taskDescInput.Reset()
			case "down":
				m.createNewUI.taskNameInput.Blur()
				m.createNewUI.taskDescInput.Focus()
			case "up":
				m.createNewUI.taskDescInput.Blur()
				m.createNewUI.taskNameInput.Focus()
			case "t":
				m.createNewUI.shouldCreateTaskFolder = !m.createNewUI.shouldCreateTaskFolder
				if m.createNewUI.shouldCreateTaskFolder {
					m.createNewUI.status = "Enter to Save, Esc to leave, Creating TaskFolder"
				} else {
					m.createNewUI.status = "Enter to Save, Esc to leave, Creating Task"
				}

			}

			var cmds []tea.Cmd
			var cmd tea.Cmd

			m.createNewUI.taskNameInput, cmd = m.createNewUI.taskNameInput.Update(msg)
			cmds = append(cmds, cmd)

			m.createNewUI.taskDescInput, cmd = m.createNewUI.taskDescInput.Update(msg)
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

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
				selectedItem.setCompletionStatus(!selectedItem.Completed)
				m.recreateList(selectedItem.ParentFolder, m.list.GlobalIndex())
				if err := m.rootFolder.DeepCopy(); err != nil {
					MarshalToFile("person.json", err)
				}
			}

		case "b":
			if m.currentFolder != nil {
				m.recreateList(m.currentFolder.Parent, index)
			}
			return m, nil
		case "p":
			switch v := m.list.SelectedItem().(type) {
			case *Task:
				m.statusString = "Cannot preview item! "
			case *TaskFolder:
				m.statusString = v.returnTree()

			}
		case "d":
			switch m.list.SelectedItem().(type) {
			case *Task:
				m.currentFolder.ChildrenTasks, _ = SlicePop(m.currentFolder.ChildrenTasks, m.list.GlobalIndex())
				m.recreateList(m.currentFolder, 0)
			case *TaskFolder:
				m.currentFolder.ChildrenTaskFolders, _ = SlicePop(m.currentFolder.ChildrenTaskFolders, m.list.GlobalIndex())
				m.recreateList(m.currentFolder, 0)
			}
		case "n":
			m.createNewUI.creatingTask = true
			m.createNewUI.taskNameInput.Focus()
			m.createNewUI.taskDescInput.Blur()
			return m, nil

		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		listWidth := msg.Width - h
		listHeight := msg.Height - v
		m.list.SetSize(listWidth, listHeight)
		if listWidth < minFrameWidth || listHeight < minFrameHeight {
			m.alert.NewAlertCmd(bubbleup.ErrorKey, "Frame dimensions too small :(")
		}
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

func (m *model) View() string {
	if m.createNewUI.creatingTask {
		return docStyle.Render(lipgloss.JoinVertical(lipgloss.Right, m.createNewUI.status, m.createNewUI.taskNameInput.View(), "\n", m.createNewUI.taskDescInput.View()))
	}
	var s string

	statusView := docStyle.Copy().Render(m.statusString)

	s += lipgloss.JoinHorizontal(lipgloss.Left, statusView, m.list.View())

	return m.alert.Render(s)
}
func (m *model) recreateList(folder *TaskFolder, selectedItem int) {
	if folder == nil {
		return
	}
	m.currentFolder = folder
	var items []list.Item

	for _, child := range folder.ChildrenTaskFolders {
		if !strings.HasPrefix(child.Title(), "📁") {
			child.Name = "📁 " + child.Title()
		}
		items = append(items, child)
	}
	for _, child := range folder.ChildrenTasks {
		items = append(items, child)
	}
	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("%s \n %s", m.currentFolder.returnPath(), m.currentFolder.Status.print())
	m.list.Select(selectedItem)
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.goBack,
			keys.previewItem,
			keys.newTask,
		}
	}
}
func main() {

	delegate := itemDelegate{}
	err, ferr :=
		loadIntoTaskFolder("person.json")
	if ferr != nil {
		panic(ferr)
	}
	root := err
	root.Parent = nil
	reconstructFolderFromJSON(root)

	ti := textinput.New()
	t2 := textarea.New()
	ti.Placeholder = "New Task Name"
	ti.CharLimit = 156
	ti.Width = 20
	t2.Placeholder = "Task Description"
	m := model{list: list.New(nil, delegate, 80, 24), createNewUI: &CreateNewUI{taskDescInput: t2, taskNameInput: ti}}
	m.recreateList(root, m.list.GlobalIndex())
	m.statusString = "Press P to preview an Item!"
	m.list.Title = "Task View "
	m.createNewUI.status = "Enter to Save, Esc to leave, Creating Task"
	m.rootFolder = root

	p := tea.NewProgram(&m)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}

func SlicePop[T any](s []T, i int) ([]T, T) {
	elem := s[i]
	s = append(s[:i], s[i+1:]...)
	return s, elem
}
