package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PlaywrightJSON struct {
	Suites []Suite   `json:"suites"`
	Errors []PWError `json:"errors"`
}

type PWError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

type Annotation struct {
	Type string `json:"type"`
}

type TestInstance struct {
	ProjectName string       `json:"projectName"`
	Annotations []Annotation `json:"annotations"`
}

type keymap struct {
	Submit, Remove, Select, ToggleRight, ToggleLeft key.Binding
}

type Spec struct {
	Title string         `json:"title"`
	Tags  []string       `json:"tags"`
	Tests []TestInstance `json:"tests"`
	File  string         `json:"file"`
	Line  int            `json:"line"`
}

type Suite struct {
	Title  string  `json:"title"`
	File   string  `json:"file"`
	Line   int     `json:"line"`
	Suites []Suite `json:"suites"`
	Specs  []Spec  `json:"specs"`
}

func tagStyleFor(tag string) lipgloss.Style {
	hash := sha256.Sum256([]byte(tag))
	r, g, b := hash[0], hash[1], hash[2]
	bgColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)

	fgColor := "#000000"
	if brightness := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b); brightness < 128 {
		fgColor = "#ffffff"
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
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
}

var keyMap = keymap{
	Submit:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	Remove:      key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "remove")),
	Select:      key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	ToggleRight: key.NewBinding(key.WithKeys("L", "shift+right"), key.WithHelp("L/Shift+Right", "toggle right")),
	ToggleLeft:  key.NewBinding(key.WithKeys("H", "shift+left"), key.WithHelp("H/Shift+Left", "toggle left")),
}

var (
	statusSelectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render // Green
	statusRemoveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render  // Red
)

var (
	ui, headed   bool
	configPath   string
	jsonDataPath string
)

func NewModel(pwData PlaywrightJSON, projects []string) model {
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
	}
}

func initData(projects []string, onlyChanged, lastFailed bool) (PlaywrightJSON, error) {
	args := []string{"playwright", "test", "--list", "--reporter=json"}
	if onlyChanged {
		args = append(args, "--only-changed")
	}
	if lastFailed {
		args = append(args, "--last-failed")
	}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	for _, p := range projects {
		args = append(args, "--project", p)
	}

	cmd := exec.Command("npx", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()

	var pwData PlaywrightJSON
	if jsonErr := json.Unmarshal(out.Bytes(), &pwData); jsonErr != nil {
		// If it's not even valid JSON, return the raw output + error
		return PlaywrightJSON{}, fmt.Errorf("failed to parse JSON output: %w\nOutput:\n%s", jsonErr, out.String())
	}

	if len(pwData.Errors) > 0 {
		fmt.Println("Playwright Errors:")
		for _, e := range pwData.Errors {
			fmt.Printf("- %s\n", e.Message)
		}
		// Still return the error to exit
		return pwData, fmt.Errorf("playwright returned %d error(s)", len(pwData.Errors))
	}

	if err != nil {
		// Non-zero exit code, but JSON parsed and no 'errors' array
		return pwData, fmt.Errorf("playwright failed: %w\nOutput:\n%s", err, out.String())
	}

	return pwData, nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "L", "shift+right":
			m.lists[m.focusedIdx].NewStatusMessage("")
			m.focusedIdx = (m.focusedIdx + 1) % len(m.lists)
			m.rightFocused = m.focusedIdx == 3
		case "H", "shift+left":
			m.lists[m.focusedIdx].NewStatusMessage("")
			m.focusedIdx = (m.focusedIdx + len(m.lists) - 1) % len(m.lists)
			m.rightFocused = m.focusedIdx == 3
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.lists[m.focusedIdx].FilterState() != list.Filtering {
				// If no items selected on right, and enter pressed on left list, run that single item
				if len(m.lists[3].Items()) == 0 && m.focusedIdx != 3 {
					selectedItem := m.lists[m.focusedIdx].SelectedItem()
					if selectedItem == nil {
						break
					}
					args := []string{"playwright", "test"}
					if ui {
						args = append(args, "--ui")
					}
					if headed {
						args = append(args, "--headed")
					}
					if configPath != "" {
						args = append(args, "--config", configPath)
					}
					it := selectedItem.(item)

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
				if ui {
					args = append(args, "--ui")
				}
				if headed {
					args = append(args, "--headed")
				}
				seen := map[string]struct{}{}
				var projects []string

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

// Recursive parsing
func collectData(
	suite Suite, suiteTitle string,
	testItems, fileItems *[]list.Item,
	tagSet map[string]struct{},
	tagToSpecs map[string][]item,
	seenTests map[string]struct{},
	fileTagMap map[string]map[string]struct{},
	fileToSpecs map[string][]item,
) {
	fullTitle := suiteTitle

	suiteTitleIsFile := suite.Title == suite.File || filepath.Base(suite.Title) == filepath.Base(suite.File)
	if suite.Title != "" && !suiteTitleIsFile {
		if fullTitle != "" {
			fullTitle += " › "
		}
		fullTitle += suite.Title
	}

	for _, spec := range suite.Specs {
		testTitle := fullTitle
		if testTitle != "" {
			testTitle += " › "
		}
		testTitle += spec.Title

		testKey := fmt.Sprintf("%s|%s|%d", spec.Title, spec.File, spec.Line)
		if _, exists := seenTests[testKey]; !exists {
			specItem := item{
				title:       testTitle,
				description: fmt.Sprintf("%s:%d", spec.File, spec.Line),
				line:        spec.Line,
				source:      "Tests",
				tags:        spec.Tags,
			}
			*testItems = append(*testItems, specItem)
			seenTests[testKey] = struct{}{}

			for _, tag := range spec.Tags {
				tagToSpecs[tag] = append(tagToSpecs[tag], specItem)
				tagSet[tag] = struct{}{}
			}

			fileToSpecs[spec.File] = append(fileToSpecs[spec.File], specItem)
		}

		if spec.File != "" {
			if _, ok := fileTagMap[spec.File]; !ok {
				fileTagMap[spec.File] = map[string]struct{}{}
			}
			for _, tag := range spec.Tags {
				fileTagMap[spec.File][tag] = struct{}{}
			}
		}
	}

	for _, child := range suite.Suites {
		collectData(child, fullTitle, testItems, fileItems, tagSet, tagToSpecs, seenTests, fileTagMap, fileToSpecs)
	}
}

func buildLists(pwData PlaywrightJSON) (
	list.Model, list.Model, list.Model,
	map[string][]item, map[string][]item,
) {
	var testItems, fileItems []list.Item
	tagSet := map[string]struct{}{}
	tagToSpecs := map[string][]item{}
	fileToSpecs := map[string][]item{}
	seenTests := map[string]struct{}{}
	fileTagMap := map[string]map[string]struct{}{}

	for _, suite := range pwData.Suites {
		collectData(suite, "", &testItems, &fileItems, tagSet, tagToSpecs, seenTests, fileTagMap, fileToSpecs)
	}

	uniqueFileMap := map[string]struct{}{}
	var uniqueFiles []list.Item
	for file, tagsMap := range fileTagMap {
		if _, exists := uniqueFileMap[file]; exists {
			continue
		}
		uniqueFileMap[file] = struct{}{}

		var tags []string
		for tag := range tagsMap {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		count := len(fileToSpecs[file])

		uniqueFiles = append(uniqueFiles, item{
			title:       file,
			source:      "Files",
			tags:        tags,
			description: fmt.Sprintf("%d test%s", count, plural(count)),
		})
	}

	var tagItems []list.Item
	for tag := range tagSet {
		count := len(tagToSpecs[tag])
		tagItems = append(tagItems, item{
			title:       tag,
			source:      "Tags",
			description: fmt.Sprintf("%d test%s", count, plural(count)),
		})
	}

	testList := list.New(testItems, list.NewDefaultDelegate(), 0, 0)
	fileList := list.New(uniqueFiles, list.NewDefaultDelegate(), 0, 0)
	tagList := list.New(tagItems, list.NewDefaultDelegate(), 0, 0)

	testList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Select}
	}
	testList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Select, keyMap.ToggleLeft, keyMap.ToggleRight}
	}

	fileList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Select}
	}
	fileList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Select, keyMap.ToggleLeft, keyMap.ToggleRight}
	}

	tagList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Select}
	}
	tagList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keyMap.Submit, keyMap.Select, keyMap.ToggleLeft, keyMap.ToggleRight}
	}

	testList.Title = "Tests"
	fileList.Title = "Files"
	tagList.Title = "Tags"

	return testList, fileList, tagList, tagToSpecs, fileToSpecs
}

func main() {
	projects := []string{}
	var onlyChanged, lastFailed bool

	// Parse command-line flags
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--project" && i+1 < len(os.Args):
			i++
			for i < len(os.Args) && !strings.HasPrefix(os.Args[i], "-") {
				projects = append(projects, os.Args[i])
				i++
			}
			i--
		case strings.HasPrefix(arg, "--project="):
			val := strings.TrimPrefix(arg, "--project=")
			for _, p := range strings.Fields(val) {
				projects = append(projects, p)
			}
		case arg == "--json-data-path":
			if i+1 < len(os.Args) {
				jsonDataPath = os.Args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--json-data-path="):
			jsonDataPath = strings.TrimPrefix(arg, "--json-data-path=")
		case arg == "-c" || arg == "--config":
			if i+1 < len(os.Args) {
				configPath = os.Args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")

		case arg == "--only-changed":
			onlyChanged = true
		case arg == "--last-failed":
			lastFailed = true
		case arg == "--ui":
			ui = true
		case arg == "--headed":
			headed = true
		}
	}

	var pwData PlaywrightJSON
	var err error

	if jsonDataPath != "" {
		data, readErr := os.ReadFile(jsonDataPath)
		if readErr != nil {
			fmt.Printf("Error reading JSON file at %s: %v\n", jsonDataPath, readErr)
			return
		}
		if jsonErr := json.Unmarshal(data, &pwData); jsonErr != nil {
			fmt.Printf("Error parsing JSON data from file: %v\n", jsonErr)
			return
		}
	} else {
		pwData, err = initData(projects, onlyChanged, lastFailed)
		if err != nil {
			fmt.Println("Error initializing data:", err)
			return
		}
	}
	if err != nil {
		fmt.Println("Error initializing data:", err)
		return
	}
	p := tea.NewProgram(NewModel(pwData, projects))
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
