package main

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	statusSelectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render
	statusRemoveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render
	rootStyle         = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
)

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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

func printHelp() {
	appName := rootStyle.Render("pwgo")

	sectionTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	description := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	fmt.Fprintf(&b, "\n%s - Multi-list CLI tool to run your Playwright suite\n\n", appName)

	fmt.Fprintln(&b, sectionTitle.Render("Usage"))
	fmt.Fprintln(&b, "  pwgo [options]\n")

	fmt.Fprintln(&b, sectionTitle.Render("Options"))

	options := []struct {
		flag string
		desc string
	}{
		{"--help, -h", "Show this help menu"},
		{"--project <name>...", "Specify project(s) to run tests for"},
		{"--grep, -g <pattern>", "Only include tests matching this pattern (for --list only)"},
		{"--grep-invert, -gv <pattern>", "Exclude tests matching this pattern (for --list only)"},
		{"--config, -c <path>", "Path to Playwright config file"},
		{"--json-data-path <path>", "Load Playwright test data from JSON file"},
		{"--only-changed", "Run only tests related to changed files"},
		{"--last-failed", "Run only last failed tests"},
	}

	const padding = 30
	for _, opt := range options {
		fmt.Fprintf(&b, "  %-*s %s\n", padding, opt.flag, description.Render(opt.desc))
	}

	fmt.Fprintln(&b, "\n"+sectionTitle.Render("Examples"))
	fmt.Fprintln(&b, "  pwgo --project=webkit --only-changed")
	fmt.Fprintln(&b, "  pwgo --config=playwright.config.ts --last-failed")
	fmt.Fprintln(&b, "  pwgo --json-data-path=./tests.json --ui")

	fmt.Fprintln(&b, "\n"+sectionTitle.Render("Additional Playwright Arguments"))
	fmt.Fprintln(&b, description.Render(
		"\nSee full list of Playwright CLI options at:\n  https://playwright.dev/docs/test-cli"))

	fmt.Println(b.String())
}
