package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// todo
// add break duration
// make durations configurable
// add progress bar
// add more key controls


// so far, basically a copy of https://github.com/charmbracelet/bubbletea/blob/master/examples/countdown/main.go

const duration = time.Minute * 25
const interval = time.Second

type model struct {
	timeout  time.Time
	lastTick time.Time
}

// transmits time from ticker to update function
type tickMsg time.Time

func (m model) Init() tea.Cmd {
	fmt.Println("Press Enter to start your first Pomodoro!\nDefault length is 25 minutes.\n")
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}

	case tickMsg:
		t := time.Time(msg)
		if t.After(m.timeout) {
			return m, tea.Quit
		}
		m.lastTick = t
		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	t := m.timeout.Sub(m.lastTick).Round(time.Second)
	return fmt.Sprintf("This program will quit in %v\n", t)
}

func main() {

	m := model{
		timeout:  time.Now().Add(duration),
		lastTick: time.Now(),
	}

	if err := tea.NewProgram(m).Start(); err != nil {
		fmt.Printf("Couldn't start program: %v\n", err)
		os.Exit(1)
	}

}

func tick() tea.Cmd {
	return tea.Tick(time.Duration(interval), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
