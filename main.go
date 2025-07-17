package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
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
type status struct {
	completed int
	total     int
	overdue   int
}

type TaskFolder struct {
	title                 string
	desc                  string
	progress              float64
	parent                *TaskFolder //so we can handle a folder of folders
	children_tasks        []Task
	children_task_folders []TaskFolder
	status                status
}

func (i TaskFolder) Title() string       { return i.title }
func (i TaskFolder) Description() string { return i.desc }
func (i TaskFolder) FilterValue() string { return i.title }

type model struct {
	list list.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func main() {
	items := []list.Item{
		TaskFolder{title: "Raspberry Pi’s", desc: "I have ’em all over my house"},
		TaskFolder{title: "Nutella", desc: "It's good on toast"},
		TaskFolder{title: "Bitter melon", desc: "It cools you down"},
		TaskFolder{title: "Nice socks", desc: "And by that I mean socks without holes"},
		TaskFolder{title: "Eight hours of sleep", desc: "I had this once"},
		TaskFolder{title: "Cats", desc: "Usually"},
		TaskFolder{title: "Plantasia, the album", desc: "My plants love it too"},
		TaskFolder{title: "Pour over coffee", desc: "It takes forever to make though"},
		TaskFolder{title: "VR", desc: "Virtual reality...what is there to say?"},
		TaskFolder{title: "Noguchi Lamps", desc: "Such pleasing organic forms"},
		TaskFolder{title: "Linux", desc: "Pretty much the best OS"},
		TaskFolder{title: "Business school", desc: "Just kidding"},
		TaskFolder{title: "Pottery", desc: "Wet clay is a great feeling"},
		TaskFolder{title: "Shampoo", desc: "Nothing like clean hair"},
		TaskFolder{title: "Table tennis", desc: "It’s surprisingly exhausting"},
		TaskFolder{title: "Milk crates", desc: "Great for packing in your extra stuff"},
		TaskFolder{title: "Afternoon tea", desc: "Especially the tea sandwich part"},
		TaskFolder{title: "Stickers", desc: "The thicker the vinyl the better"},
		TaskFolder{title: "20° Weather", desc: "Celsius, not Fahrenheit"},
		TaskFolder{title: "Warm light", desc: "Like around 2700 Kelvin"},
		TaskFolder{title: "The vernal equinox", desc: "The autumnal equinox is pretty good too"},
		TaskFolder{title: "Gaffer’s tape", desc: "Basically sticky fabric"},
		TaskFolder{title: "Terrycloth", desc: "In other words, towel fabric"},
	}

	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "My Fave Things"
	delegate := list.NewDefaultDelegate()
	delegate.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string
		if i, ok := m.SelectedItem().(TaskFolder); ok {
			title = i.Title()
		} else {
			return nil
		}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				return m.NewStatusMessage(title)
			}
		}
		return nil
	}
	m.list.SetDelegate(delegate)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

