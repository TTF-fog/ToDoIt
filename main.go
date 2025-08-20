package main

import (
	"flag"
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.dalton.dog/bubbleup"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var docStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63"))

const (
	maxWidth       = 40
	minFrameWidth  = 200
	minFrameHeight = 200
	padding        = 1
	TASK_MESSAGE   = "Alt+T to switch modes, Enter to Save, Esc to leave"
)

var config_path = "config.json"
var last_pos int

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
		priorityStr := ""
		switch s.Priority {
		case 1:
			priorityStr = fmt.Sprintf("Priority: %s", lipgloss.NewStyle().Foreground(lipgloss.Color("70")).Render("LOW"))
		case 2:
			priorityStr = fmt.Sprintf("Priority: %s", lipgloss.NewStyle().Foreground(lipgloss.Color("202")).Render("MED"))
		case 3:
			priorityStr = fmt.Sprintf("Priority: %s", lipgloss.NewStyle().Foreground(lipgloss.Color("124")).Render("HIGH"))
		}
		str := fmt.Sprintf("%s%s", s.returnStatusString(), priorityStr)
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
	taskDueDateInput       textinput.Model
	shouldCreateTaskFolder bool
	creatingTask           bool
	taskPriorityInput      textinput.Model
	edit                   bool
}
type model struct {
	list          list.Model
	statusString  string
	currentFolder *TaskFolder
	alert         bubbleup.AlertModel
	rootFolder    *TaskFolder
	createNewUI   *CreateNewUI
	itemsToDelete []list.Item
	deletionMode  bool
	sortMode      bool
	help          help.Model
	showHelp      bool
}

func (m *model) Init() tea.Cmd {
	return m.alert.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var alertCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.deletionMode {
			switch msg.String() {
			case "c":
				var newTasks []*Task
				for _, task := range m.currentFolder.ChildrenTasks {
					isDeleting := false
					for _, toDelete := range m.itemsToDelete {
						if t, ok := toDelete.(*Task); ok && t == task {
							isDeleting = true
							break
						}
					}
					if !isDeleting {
						newTasks = append(newTasks, task)
					}
				}
				m.currentFolder.ChildrenTasks = newTasks

				var newFolders []*TaskFolder
				for _, folder := range m.currentFolder.ChildrenTaskFolders {
					isDeleting := false
					for _, toDelete := range m.itemsToDelete {
						if f, ok := toDelete.(*TaskFolder); ok && f == folder {
							isDeleting = true
							break
						}
					}
					if !isDeleting {
						newFolders = append(newFolders, folder)
					}
				}
				m.currentFolder.ChildrenTaskFolders = newFolders

				m.deletionMode = false
				m.itemsToDelete = nil
				m.statusString = "Deleted items."
				m.recreateList(m.currentFolder, 0)
				if err := m.rootFolder.DeepCopy(); err != nil {
					MarshalToFile("config_path", err)
				}

				return m, nil
			case "esc":
				for _, item := range m.itemsToDelete {
					switch v := item.(type) {
					case *Task:
						v.Name = strings.TrimSuffix(v.Name, " (queued for deletion)")
					case *TaskFolder:
						v.Name = strings.TrimSuffix(v.Name, " (queued for deletion)")
					}
				}
				m.deletionMode = false
				m.itemsToDelete = nil
				m.statusString = "Deletion cancelled."
				m.recreateList(m.currentFolder, m.list.Index())
				return m, nil
			}
		}
		if m.createNewUI.creatingTask {
			switch msg.String() {
			case "enter":
				if m.createNewUI.edit {
					switch selectedItem := m.list.SelectedItem().(type) {
					case *TaskFolder:
						selectedItem.Name, selectedItem.Desc = m.createNewUI.taskNameInput.Value(), m.createNewUI.taskDescInput.Value()
					case *Task:
						selectedItem.Name, selectedItem.Desc = m.createNewUI.taskNameInput.Value(), m.createNewUI.taskDescInput.Value()
						if m.createNewUI.taskDueDateInput.Value() != "" {
							dueDate, err := time.Parse("02/01/06 15:04", m.createNewUI.taskDueDateInput.Value())
							if err != nil {
								alertCmd = m.alert.NewAlertCmd(bubbleup.ErrorKey, "Invalid date format!")
								return m, alertCmd
							}
							selectedItem.DueDate = dueDate
							selectedItem.setTimeStatus()
						}
						if m.createNewUI.taskPriorityInput.Value() != "" {
							prio := m.createNewUI.taskPriorityInput.Value()
							if prio == "LOW" {
								selectedItem.Priority = 1
							} else if prio == "MED" {
								selectedItem.Priority = 2
							} else {
								selectedItem.Priority = 3
							}
						}
					}

					m.recreateList(m.currentFolder, m.list.GlobalIndex())
					m.createNewUI.creatingTask = false
					m.createNewUI.edit = false
					m.createNewUI.taskNameInput.Reset()
					m.createNewUI.taskDescInput.Reset()
					m.createNewUI.taskPriorityInput.Reset()
					if err := m.rootFolder.DeepCopy(); err != nil {
						MarshalToFile(config_path, err)
					}
					break
				}
				if m.createNewUI.shouldCreateTaskFolder {
					m.currentFolder.ChildrenTaskFolders = append(m.currentFolder.ChildrenTaskFolders, &TaskFolder{
						Name:     m.createNewUI.taskNameInput.Value(),
						Parent:   m.currentFolder,
						Desc:     m.createNewUI.taskDescInput.Value(),
						Progress: progress.New(),
					})
				} else {
					task := &Task{
						Name:         m.createNewUI.taskNameInput.Value(),
						ParentFolder: m.currentFolder,
						Desc:         m.createNewUI.taskDescInput.Value(),
					}
					if m.createNewUI.taskDueDateInput.Value() != "" {
						dueDate, err := time.Parse("02/01/06 15:04", m.createNewUI.taskDueDateInput.Value())
						if err != nil {
							alertCmd = m.alert.NewAlertCmd(bubbleup.ErrorKey, "Invalid date format!"+err.Error())
							return m, alertCmd
						}
						task.DueDate = dueDate
						task.setTimeStatus()
					}
					if m.createNewUI.taskPriorityInput.Value() != "" {
						if m.createNewUI.taskPriorityInput.Value() != "" {
							prio := m.createNewUI.taskPriorityInput.Value()
							if prio == "LOW" {
								task.Priority = 1
							} else if prio == "MED" {
								task.Priority = 2
							} else if prio == "HIGH" {
								task.Priority = 3
							} else {
								alertCmd = m.alert.NewAlertCmd(bubbleup.ErrorKey, "Invalid priority format!")
								return m, alertCmd
							}
						}
					}

					m.currentFolder.ChildrenTasks = append(m.currentFolder.ChildrenTasks, task)
					m.currentFolder.Status.Total++
				}
				m.recreateList(m.currentFolder, 0)
				if err := m.rootFolder.DeepCopy(); err != nil {
					MarshalToFile(config_path, err)
				}
				m.createNewUI.creatingTask = false
				m.createNewUI.taskNameInput.Reset()
				m.createNewUI.taskDescInput.Reset()
				m.createNewUI.taskDueDateInput.Reset()
				m.createNewUI.taskPriorityInput.Reset()
			case "esc":
				m.createNewUI.creatingTask = false
				m.createNewUI.status = ""
				m.createNewUI.edit = false
				m.createNewUI.shouldCreateTaskFolder = false
				m.createNewUI.taskNameInput.Reset()
				m.createNewUI.taskDescInput.Reset()
				m.createNewUI.taskDueDateInput.Reset()
				m.createNewUI.taskPriorityInput.Reset()

			case "down":
				if m.createNewUI.taskNameInput.Focused() {
					m.createNewUI.taskNameInput.Blur()
					m.createNewUI.taskDescInput.Focus()
				} else if m.createNewUI.taskDescInput.Focused() {
					m.createNewUI.taskDescInput.Blur()
					if !m.createNewUI.shouldCreateTaskFolder {
						m.createNewUI.taskDueDateInput.Focus()
					} else {
						m.createNewUI.taskNameInput.Focus()
					}
				} else if m.createNewUI.taskDueDateInput.Focused() {
					m.createNewUI.taskDueDateInput.Blur()
					m.createNewUI.taskPriorityInput.Focus()
				} else if m.createNewUI.taskPriorityInput.Focused() {
					m.createNewUI.taskPriorityInput.Blur()
					m.createNewUI.taskNameInput.Focus()
				}
			case "up":
				if m.createNewUI.taskNameInput.Focused() {
					m.createNewUI.taskNameInput.Blur()
					if m.createNewUI.shouldCreateTaskFolder {
						m.createNewUI.taskDescInput.Focus()
					} else {
						m.createNewUI.taskPriorityInput.Focus()
					}
				} else if m.createNewUI.taskDescInput.Focused() {
					m.createNewUI.taskDescInput.Blur()
					m.createNewUI.taskNameInput.Focus()
				} else if m.createNewUI.taskDueDateInput.Focused() {
					m.createNewUI.taskDueDateInput.Blur()
					m.createNewUI.taskDescInput.Focus()
				} else if m.createNewUI.taskPriorityInput.Focused() {
					m.createNewUI.taskPriorityInput.Blur()
					m.createNewUI.taskDueDateInput.Focus()
				}
			case "alt+t":
				if m.createNewUI.edit {
					break
				}
				m.createNewUI.shouldCreateTaskFolder = !m.createNewUI.shouldCreateTaskFolder
				m.createNewUI.taskNameInput.Focus()
				m.createNewUI.taskDescInput.Blur()
				m.createNewUI.taskDueDateInput.Blur()
				m.createNewUI.taskPriorityInput.Blur()
				if m.createNewUI.shouldCreateTaskFolder {
					m.createNewUI.status = "New Folder: " + TASK_MESSAGE
					alertCmd := m.alert.NewAlertCmd(bubbleup.InfoKey, "Creating TaskFolder")
					return m, alertCmd
				} else {
					m.createNewUI.status = "New Task: " + TASK_MESSAGE
					alertCmd := m.alert.NewAlertCmd(bubbleup.InfoKey, "Creating Task")
					return m, alertCmd
				}

			}
			var cmds []tea.Cmd
			var cmd tea.Cmd

			m.createNewUI.taskNameInput, cmd = m.createNewUI.taskNameInput.Update(msg)
			cmds = append(cmds, cmd)

			m.createNewUI.taskDescInput, cmd = m.createNewUI.taskDescInput.Update(msg)
			cmds = append(cmds, cmd)

			m.createNewUI.taskDueDateInput, cmd = m.createNewUI.taskDueDateInput.Update(msg)
			cmds = append(cmds, cmd)

			m.createNewUI.taskPriorityInput, cmd = m.createNewUI.taskPriorityInput.Update(msg)
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

		if m.sortMode {
			switch msg.String() {
			case "1":
				vm := make([]*Task, len(m.currentFolder.ChildrenTasks))
				copy(vm, m.currentFolder.ChildrenTasks)
				sort.Slice(vm, func(i, j int) bool {
					return vm[i].Priority > vm[j].Priority
				})
				m.currentFolder.ChildrenTasks = vm
				m.recreateList(m.currentFolder, 0)
				m.sortMode = false
				m.statusString = "Sorted by priority"
			case "2":
				vm := make([]*Task, len(m.currentFolder.ChildrenTasks))
				copy(vm, m.currentFolder.ChildrenTasks)
				sort.Slice(vm, func(i, j int) bool {
					return vm[i].Name > vm[j].Name
				})
				m.currentFolder.ChildrenTasks = vm
				m.recreateList(m.currentFolder, 0)
				m.sortMode = false
				m.statusString = "Sorted by name"
			case "3":
				vm := make([]*Task, len(m.currentFolder.ChildrenTasks))
				copy(vm, m.currentFolder.ChildrenTasks)
				sort.SliceStable(vm, func(i, j int) bool {
					return vm[i].Completed != vm[j].Completed
				})
				m.currentFolder.ChildrenTasks = vm
				m.recreateList(m.currentFolder, 0)
				m.sortMode = false
				m.statusString = "Sorted by completion status "
			case "esc":
				m.sortMode = false
				m.statusString = "Cancelled sort mode"
			}
			return m, nil
		}

		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			main()
		case "f":
			m.statusString = "In sort mode, sort by (1) Priority / (2) Name / (3) Completion Status"
			m.sortMode = true
			return m, nil
		case "enter":
			last_pos = m.list.Index()
			switch selectedItem := m.list.SelectedItem().(type) {
			case *TaskFolder:
				m.recreateList(selectedItem, 0)
			case *Task:
				selectedItem.setCompletionStatus(!selectedItem.Completed)
				m.recreateList(selectedItem.ParentFolder, m.list.GlobalIndex())
				if err := m.rootFolder.DeepCopy(); err != nil {
					MarshalToFile(config_path, err)
				}
			}
		case "e":
			m.createNewUI.creatingTask = true
			m.createNewUI.edit = true
			m.createNewUI.taskNameInput.Focus()
			m.createNewUI.taskDescInput.Blur()
			m.createNewUI.taskDueDateInput.Blur()
			m.createNewUI.taskPriorityInput.Blur()
			switch selectedItem := m.list.SelectedItem().(type) {
			case *TaskFolder:
				m.createNewUI.shouldCreateTaskFolder = true
				m.createNewUI.status = "Editing Folder: " + TASK_MESSAGE
				m.createNewUI.taskNameInput.SetValue(selectedItem.Name)
				m.createNewUI.taskDescInput.SetValue(selectedItem.Desc)
			case *Task:
				m.createNewUI.shouldCreateTaskFolder = false
				m.createNewUI.status = "Editing Task: " + TASK_MESSAGE
				m.createNewUI.taskNameInput.SetValue(selectedItem.Name)
				m.createNewUI.taskDescInput.SetValue(selectedItem.Desc)
				if !selectedItem.DueDate.IsZero() {
					m.createNewUI.taskDueDateInput.SetValue(selectedItem.DueDate.Format("02/01/06 15:04"))
				}
				if selectedItem.Priority != 0 {
					m.createNewUI.taskPriorityInput.SetValue(strconv.Itoa(selectedItem.Priority))
				} else {
					m.createNewUI.taskPriorityInput.SetValue("")
				}
			}
		case "b":
			if m.currentFolder != nil {
				m.recreateList(m.currentFolder.Parent, last_pos)
			}
			return m, nil
		case "p":
			switch v := m.list.SelectedItem().(type) {
			case *Task:
				alertCmd = m.alert.NewAlertCmd(bubbleup.ErrorKey, "Cannot preview a Task")
				return m, alertCmd
			case *TaskFolder:
				m.statusString = v.returnTree()

			}
		case "d":
			m.deletionMode = true
			selectedItem := m.list.SelectedItem()
			var exists bool
			for _, item := range m.itemsToDelete {
				if item == selectedItem {
					exists = true
					break
				}
			}
			if !exists {
				m.itemsToDelete = append(m.itemsToDelete, selectedItem)
			}

			var itemNames []string
			for _, item := range m.itemsToDelete {
				switch v := item.(type) {
				case *Task:
					itemNames = append(itemNames, v.Name)
				case *TaskFolder:
					itemNames = append(itemNames, v.Name)
				}
			}
			m.statusString = fmt.Sprintf("Deletions Pendnig: %d items queued \n [%s]'c' to confirm, 'esc' to escape. ", len(m.itemsToDelete), strings.Join(itemNames, "\n, "))

			switch item := selectedItem.(type) {
			case *Task:
				if !strings.HasSuffix(item.Name, " (queued for deletion)") {
					item.Name += " (queued for deletion)"
				}
			case *TaskFolder:
				if !strings.HasSuffix(item.Name, " (queued for deletion)") {
					item.Name += " (queued for deletion)"
				}
			}
			m.recreateList(m.currentFolder, m.list.Index())
			return m, nil
		case "n":
			m.createNewUI.creatingTask = true
			m.createNewUI.taskNameInput.Focus()
			m.createNewUI.taskDescInput.Blur()
			if m.createNewUI.shouldCreateTaskFolder {
				m.createNewUI.status = "New Folder: " + TASK_MESSAGE
				alertCmd := m.alert.NewAlertCmd(bubbleup.InfoKey, "Creating Task Folder")
				return m, alertCmd
			} else {
				m.createNewUI.status = "New Task: " + TASK_MESSAGE
				alertCmd := m.alert.NewAlertCmd(bubbleup.InfoKey, "Creating Task")
				return m, alertCmd
			}
		case "h":
			m.showHelp = !m.showHelp
			return m, nil

		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		listWidth := msg.Width - h
		listHeight := msg.Height - v
		m.list.SetSize(listWidth, listHeight)
		m.createNewUI.taskNameInput.Width = msg.Width - 20
		m.createNewUI.taskDescInput.SetWidth(msg.Width - 20)
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
		var s string
		if m.showHelp {
			s = lipgloss.JoinVertical(lipgloss.Left,
				m.createNewUI.status,
				m.createNewUI.taskNameInput.View(),
				m.createNewUI.taskDescInput.View(),
				m.createNewUI.taskDueDateInput.View(),
				m.createNewUI.taskPriorityInput.View(),
				"\n"+m.help.View(createKeys),
			)
		} else {
			s = lipgloss.JoinVertical(lipgloss.Right,
				m.createNewUI.status,
				m.createNewUI.taskNameInput.View(),
				"\n",
				m.createNewUI.taskDescInput.View(),
				"\n",
				m.createNewUI.taskDueDateInput.View(),
				"\n",
				m.createNewUI.taskPriorityInput.View(),
			)
		}
		return docStyle.Render(m.alert.Render(s))
	}

	var s string
	statusView := docStyle.Render(m.statusString)

	if m.showHelp {
		var helpView string
		if m.deletionMode {
			helpView = m.help.View(deleteKeys)
		} else {
			helpView = m.help.View(*keys)
		}
		s += lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Left, statusView, m.list.View()),
			"\n"+helpView,
		)
	} else {
		s += lipgloss.JoinHorizontal(lipgloss.Left, statusView, m.list.View())
	}

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
			key.NewBinding(key.WithKeys("? (shift+/)"), key.WithHelp("? (shift+/)", "show full help")),
		}
	}
	m.list.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "enable advanced sorting")),
			key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "go to upper level")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("d", "enter deletion mode")),
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "create new item")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit item")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "enter folder/toggle item")),
		}
	}
}

func main() {

	flag.StringVar(&config_path, "c", config_path, "config file path")
	flag.Parse()
	delegate := itemDelegate{}
	err, ferr := loadIntoTaskFolder(config_path)
	if ferr != nil {
		panic(ferr)
	}
	root := err
	root.Parent = nil
	reconstructFolderFromJSON(root)
	ti := textinput.New()
	t2 := textarea.New()
	ti.Placeholder = "New Task Name (Mandatory)"
	ti.CharLimit = 156
	ti.Width = 100
	t2.Placeholder = "Task Description (Mandatory)"
	t2.SetWidth(100)
	t3 := textinput.New()
	t3.Placeholder = "DD/MM/YY HH:MM (Optional)"
	t3.Width = 100
	t4 := textinput.New()
	t4.Placeholder = "Priority (LOW/MED/HIGH) (Optional)"
	t4.Width = 100
	m := model{
		list:        list.New(nil, delegate, 80, 24),
		createNewUI: &CreateNewUI{taskDescInput: t2, taskNameInput: ti, taskDueDateInput: t3, taskPriorityInput: t4},
		help:        help.New(),
		alert:       *bubbleup.NewAlertModel(20, true),
	}
	m.recreateList(root, m.list.GlobalIndex())
	m.statusString = "Press P to preview an Item!"
	m.list.Title = "Task View "
	m.createNewUI.status = TASK_MESSAGE
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
