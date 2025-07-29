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
)

var docStyle = lipgloss.NewStyle().
	Margin(1, 1).
	BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63"))

const (
	maxWidth       = 80
	minFrameWidth  = 200
	minFrameHeight = 200
	padding        = 2
)

var index int

type itemDelegate struct{}

func (d itemDelegate) Height() int { return 4 }

func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	switch item := listItem.(type) {
	case *TaskFolder:
		s := item

		var p float64
		if s.Status.total > 0 {
			p = float64(s.Status.completed) / float64(s.Status.total)
		}

		str := fmt.Sprintf("%s\n %s \n%s", s.Title(), s.Status.print(), s.Progress.ViewAs(p))
		fn := lipgloss.NewStyle().PaddingLeft(4).Render
		if index == m.Index() {
			fn = func(s ...string) string {
				return lipgloss.NewStyle().
					Padding(padding).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).
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
				return lipgloss.NewStyle().
					Padding(padding).
					BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).
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
	rootFolder    *TaskFolder
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
				m.alert.NewAlertCmd(bubbleup.ErrorKey, "Cannot preview Task!")
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

func (m *model) View() string {
	var s string

	v, _ := docStyle.GetFrameSize()
	statusView := docStyle.Copy().MaxHeight(m.list.Height() + v).Render(m.statusString)
	s += lipgloss.JoinHorizontal(lipgloss.Left, statusView, docStyle.Render(m.list.View()))

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
	delegate := itemDelegate{}
	newList := list.New(items, delegate, 0, 0)
	newList.Title = fmt.Sprintf("%s \n %s", m.currentFolder.returnPath(), m.currentFolder.Status.print())
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

	delegate := itemDelegate{}
	err, ferr :=
		loadIntoTaskFolder("person.json")
	if ferr != nil {
		panic(ferr)
	}
	root := err
	root.Parent = nil
	reconstructFolderFromJSON(root)
	//for _, item := range items {
	//	if folder, ok := item.(*TaskFolder); ok {
	//		folder.Parent = &root
	//		root.ChildrenTaskFolders = append(root.ChildrenTaskFolders, folder)
	//	}
	//}
	m := model{list: list.New(nil, delegate, 0, 0)}
	m.recreateList(root, m.list.GlobalIndex())
	m.statusString = "Press P to preview an Item!"
	m.list.Title = "Task View "
	m.list.SetHeight(10)
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
