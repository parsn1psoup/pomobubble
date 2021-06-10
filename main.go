package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"

	tea "github.com/charmbracelet/bubbletea"
)

// TODO
// add break duration
// add more key controls (e.g. pause timer)
// track across multiple pomodoros
// altscreen?
// how to notify (sound)?
// understand implications of WindowSizeMsg better
// show that progress had started even for long pomodoros (time until first segment of progress bar appears)
// add tests

// heavily based on
// https://github.com/charmbracelet/bubbletea/blob/master/examples/countdown/main.go
// https://github.com/charmbracelet/bubbletea/blob/master/examples/progress/main.go
// https://github.com/charmbracelet/bubbletea/blob/master/examples/textinput/main.go

const (
	fps             = 60
	padding         = 2
	maxWidth        = 80
	refreshInterval = time.Second
)

var (
	pomoMinutes   int           = 1
	pomoDuration  time.Duration = time.Duration(pomoMinutes) * time.Minute
	inputReceived bool
	inputComplete bool
	stepSize      float64
)

type pomoModel struct {
	timeout   time.Time
	lastTick  time.Time
	percent   float64
	progress  *progress.Model
	textinput textinput.Model // value not pointer, let's see if this makes sense
}

// transmits time from ticker to update function
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Duration(refreshInterval), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (pm pomoModel) Init() tea.Cmd {
	fmt.Println("🍅 How long do you want your pomodoros? 🍅\nEnter a number betweeen 1 and 99 for length in minutes, then press Enter to start.")
	return textinput.Blink
}

// Update handles messages (user input, window resizing, progress updates)
func (pm pomoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return pm, tea.Quit
		case "enter":
			if !inputReceived {
				return pm, nil
			} else {
				pomoMinutes, err := strconv.Atoi(pm.textinput.Value())
				if err != nil {
					fmt.Printf("Error parsing input: %v\n", err)
					return pm, tea.Quit
				}
				if pomoMinutes < 1 {
					fmt.Println("Pomodoro duration cannot be less than 1 minute")
					return pm, tea.Quit
				}
				pomoDuration = time.Duration(pomoMinutes) * time.Minute
				stepSize = 1.0 / (float64(fps)) / float64(pomoMinutes) // why should stepSize depend on fps? is Update() called on every new frame?
				pm.timeout = time.Now().Add(pomoDuration)
				pm.lastTick = time.Now()
				inputComplete = true
				return pm, tick()
			}

		default:
			pm.textinput.Focus()
			pm.textinput, cmd = pm.textinput.Update(msg)
			inputReceived = true
			return pm, cmd

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

}

// View calculates the remaining time (rem) and displays an updated status on progress
func (pm pomoModel) View() string {

	if !inputComplete {
		return fmt.Sprintf(
			"\n\n%s\n\n%s",
			pm.textinput.View(),
			"(esc to quit)",
		) + "\n"
	}

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

	// error could be added by wrapping the textinput.Model but... too much wrapping?
	text := textinput.NewModel()
	if err != nil {
		fmt.Println("Could not initialize textinput model:", err)
		os.Exit(1)
	}
	text.Placeholder = "25"
	text.CharLimit = 2 // if your pomodoro is longer than 3 hours then you don't need pomodoros

	pm := pomoModel{
		progress:  prog,
		textinput: text,
	}

	if err := tea.NewProgram(pm).Start(); err != nil {
		fmt.Printf("Couldn't start program: %v\n", err)
		os.Exit(1)
	}

}
