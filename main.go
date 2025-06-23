package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	pwData, projects, extraArgs, err := prepareData()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	p := tea.NewProgram(NewModel(pwData, projects, extraArgs))
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
