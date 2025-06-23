package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func writeTempJSON(t *testing.T, data PlaywrightJSON) string {
	t.Helper()

	file, err := os.CreateTemp("", "playwright_data_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		t.Fatalf("failed to write temp JSON data: %v", err)
	}

	return file.Name()
}

func TestPrepareData_WithJSONPath(t *testing.T) {
	sample := PlaywrightJSON{
		Suites: []Suite{{
			Title: "Root",
			File:  "test/file.spec.ts",
			Line:  1,
			Specs: []Spec{
				{
					Title: "does something",
					File:  "test/file.spec.ts",
					Line:  2,
					Tags:  []string{"smoke"},
					Tests: []TestInstance{{ProjectName: "chrome"}},
				},
			},
		}},
	}

	jsonPath := writeTempJSON(t, sample)
	defer os.Remove(jsonPath)

	// Simulate CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "--json-data-path", jsonPath, "-x"}

	result, projects, extraArgs, err := prepareData()
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}

	if len(result.Suites) != 1 {
		t.Errorf("expected 1 suite, got %d", len(result.Suites))
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
	if len(extraArgs) != 1 {
		t.Errorf("expected 1 extraArgs, got %d", len(extraArgs))
	}
}

func TestCollectData_Basic(t *testing.T) {
	testSuite := Suite{
		Title: "Root",
		File:  "spec.ts",
		Specs: []Spec{
			{
				Title: "should work",
				File:  "spec.ts",
				Line:  42,
				Tags:  []string{"unit", "ci"},
				Tests: []TestInstance{
					{ProjectName: "firefox"},
					{ProjectName: "chrome"},
				},
			},
		},
	}

	var testItems, fileItems []list.Item
	tagSet := map[string]struct{}{}
	tagToSpecs := map[string][]item{}
	fileToSpecs := map[string][]item{}
	fileToProjects := map[string]map[string]struct{}{}
	tagToProjects := map[string]map[string]struct{}{}
	fileTagMap := map[string]map[string]struct{}{}
	seenTests := map[string]struct{}{}

	collectData(testSuite, "", &testItems, &fileItems, tagSet, tagToSpecs, seenTests, fileTagMap, fileToSpecs, fileToProjects, tagToProjects)

	if len(testItems) != 1 {
		t.Errorf("expected 1 test item, got %d", len(testItems))
	}
	if len(tagSet) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tagSet))
	}
	if len(tagToSpecs["unit"]) != 1 {
		t.Errorf("expected tag 'unit' to have 1 spec")
	}
	if len(fileToSpecs["spec.ts"]) != 1 {
		t.Errorf("expected 1 spec for file")
	}
	if len(fileToProjects["spec.ts"]) != 2 {
		t.Errorf("expected 2 projects for file, got %d", len(fileToProjects["spec.ts"]))
	}
}

func TestBuildLists_SimpleSuite(t *testing.T) {
	pwData := PlaywrightJSON{
		Suites: []Suite{{
			Title: "Root",
			File:  "specs/test_a.spec.ts",
			Line:  10,
			Specs: []Spec{{
				Title: "runs properly",
				File:  "specs/test_a.spec.ts",
				Line:  11,
				Tags:  []string{"smoke", "regression"},
				Tests: []TestInstance{
					{ProjectName: "chromium"},
					{ProjectName: "firefox"},
				},
			}},
		}},
	}

	testList, fileList, tagList, tagToSpecs, fileToSpecs := buildLists(pwData)

	if len(testList.Items()) != 1 {
		t.Errorf("expected 1 test item, got %d", len(testList.Items()))
	}
	testItem := testList.Items()[0].(item)
	if testItem.title != "Root › runs properly" {
		t.Errorf("unexpected test title: %s", testItem.title)
	}

	if len(fileList.Items()) != 1 {
		t.Errorf("expected 1 file item, got %d", len(fileList.Items()))
	}
	fileItem := fileList.Items()[0].(item)
	if fileItem.title != "specs/test_a.spec.ts" {
		t.Errorf("unexpected file title: %s", fileItem.title)
	}
	expectedDesc := "2 tests across 2 projects"
	if fileItem.description != expectedDesc {
		t.Errorf("expected file description %q, got %q", expectedDesc, fileItem.description)
	}

	if len(tagList.Items()) != 2 {
		t.Errorf("expected 2 tag items, got %d", len(tagList.Items()))
	}
	tagTitles := map[string]bool{}
	for _, tag := range tagList.Items() {
		tagTitles[tag.(item).title] = true
	}
	for _, expected := range []string{"smoke", "regression"} {
		if !tagTitles[expected] {
			t.Errorf("expected tag %q to be present", expected)
		}
	}

	if len(tagToSpecs["smoke"]) != 1 {
		t.Errorf("expected 1 spec under 'smoke' tag, got %d", len(tagToSpecs["smoke"]))
	}

	if len(fileToSpecs["specs/test_a.spec.ts"]) != 1 {
		t.Errorf("expected 1 spec in file map, got %d", len(fileToSpecs["specs/test_a.spec.ts"]))
	}
}

func TestBuildLists_MultipleSuites(t *testing.T) {
	pwData := PlaywrightJSON{
		Suites: []Suite{
			{
				Title: "Suite A",
				File:  "testA.spec.ts",
				Specs: []Spec{
					{
						Title: "A1",
						File:  "testA.spec.ts",
						Line:  5,
						Tags:  []string{"fast"},
						Tests: []TestInstance{{ProjectName: "webkit"}},
					},
				},
			},
			{
				Title: "Suite B",
				File:  "testB.spec.ts",
				Specs: []Spec{
					{
						Title: "B1",
						File:  "testB.spec.ts",
						Line:  10,
						Tags:  []string{"slow"},
						Tests: []TestInstance{{ProjectName: "firefox"}},
					},
				},
			},
		},
	}

	testList, fileList, tagList, _, _ := buildLists(pwData)

	if len(testList.Items()) != 2 {
		t.Errorf("expected 2 test items, got %d", len(testList.Items()))
	}
	if len(fileList.Items()) != 2 {
		t.Errorf("expected 2 file items, got %d", len(fileList.Items()))
	}
	if len(tagList.Items()) != 2 {
		t.Errorf("expected 2 tag items, got %d", len(tagList.Items()))
	}
}

func TestBuildListsDescriptions(t *testing.T) {
	pw := PlaywrightJSON{
		Suites: []Suite{
			{
				Title: "Root Suite",
				File:  "example.test.js",
				Specs: []Spec{
					{
						Title: "does something",
						Tags:  []string{"tagA"},
						File:  "example.test.js",
						Line:  12,
						Tests: []TestInstance{
							{ProjectName: "project1"},
						},
					},
					{
						Title: "does another thing",
						Tags:  []string{"tagA", "tagB"},
						File:  "example.test.js",
						Line:  20,
						Tests: []TestInstance{
							{ProjectName: "project1"},
							{ProjectName: "project2"},
						},
					},
				},
			},
		},
	}

	_, fileList, tagList, tagToSpecs, fileToSpecs := buildLists(pw)

	assertDescription := func(items []list.Item, title string, want string) {
		for _, it := range items {
			if it.(item).title == title {
				got := it.(item).description
				if got != want {
					t.Errorf("description mismatch for %q:\n  got:  %q\n  want: %q", title, got, want)
				}
			}
		}
	}

	assertDescription(fileList.Items(), "example.test.js", "4 tests across 2 projects")
	assertDescription(tagList.Items(), "tagA", "4 tests across 2 projects")
	assertDescription(tagList.Items(), "tagB", "2 tests across 2 projects")

	if len(tagToSpecs["tagA"]) != 2 {
		t.Errorf("expected 2 specs for tagA, got %d", len(tagToSpecs["tagA"]))
	}
	if len(fileToSpecs["example.test.js"]) != 2 {
		t.Errorf("expected 2 specs for file, got %d", len(fileToSpecs["example.test.js"]))
	}
}
