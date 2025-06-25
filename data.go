package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
)

var (
	configPath   string
	jsonDataPath string
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

func initData(projects []string, onlyChanged, lastFailed bool, grep, grepInvert string) (PlaywrightJSON, error) {
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
	if grep != "" {
		args = append(args, "--grep", grep)
	}
	if grepInvert != "" {
		args = append(args, "--grep-invert", grepInvert)
	}
	for _, p := range projects {
		args = append(args, "--project", p)
	}

	cmd := exec.Command("npx", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	err := cmd.Run()

	var pwData PlaywrightJSON
	if jsonErr := json.Unmarshal(out.Bytes(), &pwData); jsonErr != nil {
		// If it's not even valid JSON, return the raw output + error
		return PlaywrightJSON{}, fmt.Errorf("failed to parse JSON output: %w\nOutput:\n%s", jsonErr, out.String())
	}

	if len(pwData.Suites) == 0 {
		return pwData, fmt.Errorf("No tests found")
	}

	if len(pwData.Errors) > 0 {
		fmt.Println("Playwright Errors:")
		for _, e := range pwData.Errors {
			fmt.Printf("- %s\n", e.Message)
		}
		return pwData, fmt.Errorf("Playwright returned %d error(s)", len(pwData.Errors))
	}

	if err != nil {
		return pwData, fmt.Errorf("Playwright failed: %w\nOutput:\n%s", err, out.String())
	}

	return pwData, nil
}

func collectData(
	suite Suite, suiteTitle string,
	testItems, fileItems *[]list.Item,
	tagSet map[string]struct{},
	tagToSpecs map[string][]item,
	seenTests map[string]struct{},
	fileTagMap map[string]map[string]struct{},
	fileToSpecs map[string][]item,
	fileToProjects map[string]map[string]struct{},
	tagToProjects map[string]map[string]struct{},
) {
	fullTitle := suiteTitle
	if suite.Title != "" && suite.Title != suite.File && filepath.Base(suite.Title) != filepath.Base(suite.File) {
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

		for _, test := range spec.Tests {
			// Track projects per tag
			for _, tag := range spec.Tags {
				if _, ok := tagToProjects[tag]; !ok {
					tagToProjects[tag] = map[string]struct{}{}
				}
				tagToProjects[tag][test.ProjectName] = struct{}{}
			}
			// Track projects per file
			if _, ok := fileToProjects[spec.File]; !ok {
				fileToProjects[spec.File] = map[string]struct{}{}
			}
			fileToProjects[spec.File][test.ProjectName] = struct{}{}
		}

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
		collectData(child, fullTitle, testItems, fileItems, tagSet, tagToSpecs, seenTests, fileTagMap, fileToSpecs, fileToProjects, tagToProjects)
	}
}

func prepareData() (PlaywrightJSON, []string, []string, error) {
	projects := []string{}
	var onlyChanged, lastFailed bool
	var extraArgs []string
	var grep, grepInvert string

	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			printHelp()
			os.Exit(0)
		}
	}

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
		case arg == "-g" || arg == "--grep":
			if i+1 < len(os.Args) {
				grep = os.Args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--grep="):
			grep = strings.TrimPrefix(arg, "--grep=")
		case arg == "-gv" || arg == "--grep-invert":
			if i+1 < len(os.Args) {
				grepInvert = os.Args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--grep-invert="):
			grepInvert = strings.TrimPrefix(arg, "--grep-invert=")
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
		default:
			extraArgs = append(extraArgs, arg)
		}
	}

	var pwData PlaywrightJSON

	if jsonDataPath != "" {
		data, readErr := os.ReadFile(jsonDataPath)
		if readErr != nil {
			return pwData, nil, nil, fmt.Errorf("error reading JSON file at %s: %w", jsonDataPath, readErr)
		}
		if jsonErr := json.Unmarshal(data, &pwData); jsonErr != nil {
			return pwData, nil, nil, fmt.Errorf("error parsing JSON data from file: %w", jsonErr)
		}
	} else {
		var err error
		pwData, err = initData(projects, onlyChanged, lastFailed, grep, grepInvert)
		if err != nil {
			return pwData, nil, nil, fmt.Errorf("error initializing data: %w", err)
		}
	}

	return pwData, projects, extraArgs, nil
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
	fileToProjects := map[string]map[string]struct{}{}
	tagToProjects := map[string]map[string]struct{}{}

	for _, suite := range pwData.Suites {
		collectData(suite, "", &testItems, &fileItems, tagSet, tagToSpecs, seenTests, fileTagMap, fileToSpecs, fileToProjects, tagToProjects)
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
		projectCount := len(fileToProjects[file]) // Use pre-collected projects count

		uniqueFiles = append(uniqueFiles, item{
			title:       file,
			source:      "Files",
			tags:        tags,
			description: fmt.Sprintf("%d test%s across %d project%s", count*projectCount, plural(count*projectCount), projectCount, plural(projectCount)),
		})
	}

	var tagItems []list.Item
	for tag := range tagSet {
		count := len(tagToSpecs[tag])
		projectCount := len(tagToProjects[tag]) // Use pre-collected projects count

		tagItems = append(tagItems, item{
			title:       tag,
			source:      "Tags",
			description: fmt.Sprintf("%d test%s across %d project%s", count*projectCount, plural(count*projectCount), projectCount, plural(projectCount)),
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
