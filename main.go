package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO
// start timer on enter
// add break duration
// make durations configurable (with restrictions)
// add more key controls (e.g. pause timer)
// track across multiple pomodoros
// altscreen?
// how to notify (sound)?
// understand implications of WindowSizeMsg better
// show that progress had started even for long pomodoros (time until first segment of progress bar appears)

// so far, heavily based on https://github.com/charmbracelet/bubbletea/blob/master/examples/countdown/main.go
// and https://github.com/charmbracelet/bubbletea/blob/master/examples/progress/main.go

const durationMinutes = 7
const duration = time.Minute * durationMinutes
const interval = time.Second // refresh interval

const (
	fps              = 60 // why should stepSize depend on fps? is Update() called on every new frame?
	stepSize float64 = 1.0 / (float64(fps)) / durationMinutes
	padding          = 2
	maxWidth         = 80
)

type pomoModel struct {
	timeout  time.Time
	lastTick time.Time
	percent  float64
	progress *progress.Model
}

// transmits time from ticker to update function
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Duration(interval), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (pm pomoModel) Init() tea.Cmd {
	fmt.Printf("Timer is running! Duration is %v minutes.\n", durationMinutes)
	return tick()
}

func (pm pomoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return pm, tea.Quit
		default:
			return pm, nil
		}

	case tea.WindowSizeMsg:
		pm.progress.Width = msg.Width - padding*2 - 4
		if pm.progress.Width > maxWidth {
			pm.progress.Width = maxWidth
		}
		return pm, nil

	case tickMsg:
		pm.percent += stepSize

		t := time.Time(msg)
		if t.After(pm.timeout) {
			pm.View() // good for anything?
			return pm, tea.Quit
		}
		pm.lastTick = t

		return pm, tick()

	default:
		return pm, nil
	}

	return pm, nil
}

func (pm pomoModel) View() string {

	rem := pm.timeout.Sub(pm.lastTick).Round(time.Second)
	pad := strings.Repeat(" ", padding)

	if pm.percent >= 0.999 {
		return fmt.Sprintf("\n%s %s %s Complete! 🍅 \n\n", pad, pm.progress.View(pm.percent), pad)
	}

	return fmt.Sprintf("\n%s %s %s %v \n\n", pad, pm.progress.View(pm.percent), pad, rem)
}

func main() {

	prog, err := progress.NewModel(progress.WithDefaultScaledGradient(),
		progress.WithoutPercentage())
	if err != nil {
		fmt.Println("Could not initialize progress model:", err)
		os.Exit(1)
	}

	pm := pomoModel{
		timeout:  time.Now().Add(duration),
		lastTick: time.Now(),
		progress: prog,
	}

	if err := tea.NewProgram(pm).Start(); err != nil {
		fmt.Printf("Couldn't start program: %v\n", err)
		os.Exit(1)
	}

}
