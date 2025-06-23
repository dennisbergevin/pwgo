package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keymap struct {
	Submit, Remove, Select, ToggleRight, ToggleLeft key.Binding
}

type item struct {
	title       string
	description string
	line        int
	source      string
	tags        []string
}

type model struct {
	rightFocused bool
	tagToSpecs   map[string][]item
	list         list.Model
	lists        []list.Model
	focusedIdx   int
	quitting     bool
	projects     []string
	fileToSpecs  map[string][]item
	extraArgs    []string
}

var keyMap = keymap{
	Submit:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	Remove:      key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "remove")),
	Select:      key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	ToggleRight: key.NewBinding(key.WithKeys("L", "shift+right"), key.WithHelp("L/Shift+Right", "toggle right")),
	ToggleLeft:  key.NewBinding(key.WithKeys("H", "shift+left"), key.WithHelp("H/Shift+Left", "toggle left")),
}

func NewModel(pwData PlaywrightJSON, projects []string, extraArgs []string) model {
	selectedList := list.New([]list.Item{}, list.NewDefaultDelegate(), 40, 20)

	selectedList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Remove}
	}
	selectedList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Remove, keyMap.ToggleLeft, keyMap.ToggleRight}
	}
	selectedList.Title = "Selected"
	testList, fileList, tagList, tagToSpecs, fileToSpecs := buildLists(pwData)
	lists := []list.Model{testList, fileList, tagList, selectedList}

	const leftListWidth = 120

	for i := range lists {
		lists[i].SetWidth(leftListWidth)
		lists[i].SetHeight(30)
	}

	return model{
		lists:       lists,
		focusedIdx:  0,
		tagToSpecs:  tagToSpecs,
		fileToSpecs: fileToSpecs,
		projects:    projects,
		extraArgs:   extraArgs,
	}
}

func (i item) Title() string {
	base := i.title

	// If it's already got tags (e.g., File or Test), use them
	if len(i.tags) > 0 {
		var styledTags []string
		for _, tag := range i.tags {
			styledTags = append(styledTags, tagStyleFor(tag).Render(tag))
		}
		return fmt.Sprintf("%s  %s", base, lipgloss.JoinHorizontal(lipgloss.Left, styledTags...))
	}

	// If it's a tag item, show how it appears in files/tests by rendering the tag itself
	if i.source == "Tags" {
		styled := tagStyleFor(i.title).Render(i.title)
		return fmt.Sprintf("%s  %s", i.title, styled)
	}

	return base
}
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "L", "shift+right":
			if m.lists[m.focusedIdx].FilterState() != list.Filtering {
				m.lists[m.focusedIdx].NewStatusMessage("")
				m.focusedIdx = (m.focusedIdx + 1) % len(m.lists)
				m.rightFocused = m.focusedIdx == 3
			}
		case "H", "shift+left":
			if m.lists[m.focusedIdx].FilterState() != list.Filtering {
				m.lists[m.focusedIdx].NewStatusMessage("")
				m.focusedIdx = (m.focusedIdx + len(m.lists) - 1) % len(m.lists)
				m.rightFocused = m.focusedIdx == 3
			}
		case "ctrl+c", "q":
			if m.lists[m.focusedIdx].FilterState() != list.Filtering {
				return m, tea.Quit
			}
		case "enter":
			if m.lists[m.focusedIdx].FilterState() != list.Filtering {
				// If no items selected on right, and enter pressed on left list, run that single item
				if len(m.lists[3].Items()) == 0 && m.focusedIdx != 3 {
					selectedItem := m.lists[m.focusedIdx].SelectedItem()
					if selectedItem == nil {
						break
					}
					args := []string{"playwright", "test"}
					if configPath != "" {
						args = append(args, "--config", configPath)
					}
					it := selectedItem.(item)
					args = append(args, m.extraArgs...)

					if specs, ok := m.tagToSpecs[it.title]; ok && it.source == "Tags" {
						seen := map[string]struct{}{}
						for _, specItem := range specs {
							arg := specItem.description // "file:line"
							if _, exists := seen[arg]; !exists {
								args = append(args, arg)
								seen[arg] = struct{}{}
							}
						}
					} else {
						// Normal case for Tests or Files
						var arg string
						switch it.source {
						case "Tests":
							arg = it.description // file:line
						case "Files":
							arg = it.title
						default:
							arg = it.title
						}
						args = append(args, m.extraArgs...)
						args = append(args, arg)
					}

					for _, p := range m.projects {
						args = append(args, "--project", p)
					}

					m.quitting = true
					return m, tea.ExecProcess(exec.Command("npx", args...), nil)
				}

				// Else fallback: use the selected list as before
				args := []string{"playwright", "test"}
				seen := map[string]struct{}{}
				var projects []string
				args = append(args, m.extraArgs...)

				for _, li := range m.lists[3].Items() {
					it := li.(item)

					// Handle tag item properly by expanding to matching tests
					if it.source == "Tags" {
						if specs, ok := m.tagToSpecs[it.title]; ok {
							for _, specItem := range specs {
								arg := specItem.description // "file:line"
								if _, exists := seen[arg]; !exists {
									args = append(args, arg)
									seen[arg] = struct{}{}
								}
							}
						}
						continue
					}

					// Handle project-only selection
					if it.source == "Projects" {
						projects = append(projects, it.title)
						continue
					}

					// Handle file or test
					var arg string
					switch it.source {
					case "Tests":
						arg = it.description // file:line
					case "Files":
						arg = it.title
					default:
						arg = it.title
					}
					if _, exists := seen[arg]; !exists {
						args = append(args, arg)
						seen[arg] = struct{}{}
					}
				}

				for _, p := range m.projects {
					args = append(args, "--project", p)
				}

				m.quitting = true
				return m, tea.ExecProcess(exec.Command("npx", args...), nil)
			}
		case " ":
			if m.lists[m.focusedIdx].FilterState() != list.Filtering {
				if m.rightFocused {
					// Remove from selected list and re-add to left
					selectedItem := m.lists[3].SelectedItem()
					if selectedItem == nil {
						break
					}
					var updated []list.Item
					for _, it := range m.lists[3].Items() {
						if it.FilterValue() != selectedItem.FilterValue() {
							updated = append(updated, it)
						} else {
							// Put back into matching left list
							for i := range m.lists {
								if m.lists[i].Title == selectedItem.(item).source {
									m.lists[i].InsertItem(len(m.lists[i].Items()), selectedItem)
								}
							}
						}
					}
					m.lists[3].SetItems(updated)

					// Reset filtering
					m.lists[m.focusedIdx].ResetFilter()

					removedMsg := fmt.Sprintf("Removed %s", strings.ToLower(selectedItem.(item).source[:len(selectedItem.(item).source)-1]))
					return m, m.lists[3].NewStatusMessage(statusRemoveStyle(removedMsg))
				} else {
					// Add to selected list and remove from left list
					selectedItem := m.lists[m.focusedIdx].SelectedItem()
					if selectedItem == nil {
						break
					}
					for _, it := range m.lists[3].Items() {
						if it.FilterValue() == selectedItem.FilterValue() {
							return m, nil // already selected
						}
					}
					m.lists[3].InsertItem(len(m.lists[3].Items()), selectedItem)

					// Remove from the left list
					var newItems []list.Item
					for _, it := range m.lists[m.focusedIdx].Items() {
						if it.FilterValue() != selectedItem.FilterValue() {
							newItems = append(newItems, it)
						}
					}
					m.lists[m.focusedIdx].SetItems(newItems)

					// Reset filtering
					m.lists[m.focusedIdx].ResetFilter()

					addedMsg := fmt.Sprintf("Selected %s", strings.ToLower(selectedItem.(item).source[:len(selectedItem.(item).source)-1]))
					return m, m.lists[m.focusedIdx].NewStatusMessage(statusSelectStyle(addedMsg))
				}
			}
		}
	}
	var commd tea.Cmd
	m.lists[m.focusedIdx], commd = m.lists[m.focusedIdx].Update(msg)
	return m, commd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	activeTitle := lipgloss.NewStyle().Bold(true).Underline(true).Render()

	left := m.lists[m.focusedIdx]
	leftView := lipgloss.JoinVertical(lipgloss.Left,
		activeTitle,
		left.View(),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView)
}
