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
	"github.com/gen2brain/beeep"
)

// TODO
// make breaks configurable
// default value for pomoMinutes instead of placeholder
// add more key controls (e.g. pause timer)
// altscreen?
// understand implications of WindowSizeMsg better
// show that progress had started even for long pomodoros (time until first segment of progress bar appears)
// add tests
// represent the status with an enum?

// heavily based on
// https://github.com/charmbracelet/bubbletea/blob/master/examples/countdown/main.go
// https://github.com/charmbracelet/bubbletea/blob/master/examples/progress/main.go
// https://github.com/charmbracelet/bubbletea/blob/master/examples/textinput/main.go

const (
	fps               = 60
	padding           = 2
	maxWidth          = 80
	refreshInterval   = time.Second
	shortBreakMinutes = 2 // debug
	longBreakMinutes  = 3 // debug
)

// maybe move some of these into struct
var (
	pomoMinutes       int
	inputReceived     bool
	inputComplete     bool
	timerRunning      bool
	stepSize          float64
	runningPom        bool
	runningShortBreak bool
	runningLongBreak  bool
	awaitingUserInput bool
)

type pomoModel struct {
	timeout   time.Time
	lastTick  time.Time
	percent   float64
	progress  *progress.Model
	textinput textinput.Model // value not pointer, let's see if this makes sense
	history   pomoHistory
}

// history of completed pomodoros
type pomoHistory struct {
	poms        int
	shortBreaks int
	longBreaks  int
}

// transmits time from ticker to update function
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Duration(refreshInterval), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (pm pomoModel) Init() tea.Cmd {
	fmt.Println("\n\n🍅 POMOBUBBLE 🍅")
	// this should be in View() but it always shows up several times there
	fmt.Println("How long do you want your pomodoros?\nEnter a number betweeen 1 and 99 for length in minutes, then press Enter to start.")
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
			// we are still waiting for the user to specify pomodoro duration
			if !inputComplete {
				// if we haven't received any input yet, don't react
				if !inputReceived {
					return pm, nil
				} else {
					// the user has entered the desired pomodoro duration.
					// initialize and start the timer
					min, err := strconv.Atoi(pm.textinput.Value()) // ugly workaround
					pomoMinutes = min
					if err != nil {
						fmt.Printf("Error parsing input: %v\n", err)
						return pm, tea.Quit
					}
					if pomoMinutes < 1 {
						fmt.Println("Pomodoro duration cannot be less than 1 minute")
						return pm, tea.Quit
					}
					pm.initTimer(pomoMinutes)
					inputComplete = true
					timerRunning = true
					runningPom = true
					return pm, tick()
				}
			} else {
				if awaitingUserInput {
					// find out what's the next interval to start
					awaitingUserInput = false
					if runningPom {
						pm.initTimer(pomoMinutes)
						timerRunning = true
						return pm, tick()
					} else if runningShortBreak {
						pm.initTimer(shortBreakMinutes)
						timerRunning = true
						return pm, tick()
					} else if runningLongBreak {
						pm.initTimer(longBreakMinutes)
						timerRunning = true
						return pm, tick()
					}
				}
				// we weren't expecting the user to provide input
				return pm, nil
			}

		default:
			if !inputComplete {
				pm.textinput.Focus()
				pm.textinput, cmd = pm.textinput.Update(msg)
				inputReceived = true
				return pm, cmd
			}
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
			timerRunning = false
			pm.updateStatus()
			notify()
			awaitingUserInput = true
			return pm, textinput.Blink
		}
		pm.lastTick = t

		return pm, tick()

	default:
		return pm, nil
	}

}

func (pm *pomoModel) initTimer(durationMinutes int) error {
	pm.percent = 0

	timerDuration := time.Duration(durationMinutes) * time.Minute
	stepSize = 1.0 / (float64(fps)) / float64(durationMinutes) // why should stepSize depend on fps? is Update() called on every new frame?
	pm.timeout = time.Now().Add(timerDuration)
	pm.lastTick = time.Now()

	return nil
}

// after a time interval is done, update based on what is next
func (pm *pomoModel) updateStatus() {
	if runningPom {
		pm.history.poms++
		runningPom = false
		if pm.history.poms%4 == 0 {
			runningLongBreak = true
		} else {
			runningShortBreak = true
		}
		return
	}
	if runningShortBreak {
		pm.history.shortBreaks++
		runningShortBreak = false
		runningPom = true
		return
	}

	if runningLongBreak {
		pm.history.longBreaks++
		runningLongBreak = false
		runningPom = true
		return
	}

}

func notify() {
	if runningPom {
		beeep.Alert("PomoBubble", "Time to work!", "assets/pom.png")
	} else if runningShortBreak {
		beeep.Alert("PomoBubble", "Time for a short break!", "assets/coffee.png")
	} else if runningLongBreak {
		beeep.Alert("PomoBubble", "Time for a long break!", "assets/palmtree.png")
	}
}

// View renders UI based on model and current status
func (pm pomoModel) View() string {

	// wait for user to enter valid pomodoro duration
	if !inputComplete {
		return fmt.Sprintf(
			"\n\n%s\n\n%s",
			pm.textinput.View(),
			"(esc to quit)",
		) + "\n"
	}

	// show progress bar for ongoing pomodoro or break
	rem := pm.timeout.Sub(pm.lastTick).Round(time.Second)
	pad := strings.Repeat(" ", padding)

	if timerRunning {
		if runningPom {
			return fmt.Sprintf("\nCompleted pomodoros: %v\n\nCurrent pomodoro:\n\n%s %s %s %v \n\n", pm.history.poms, pad, pm.progress.View(pm.percent), pad, rem)
		} else {
			return fmt.Sprintf("\nCompleted pomodoros: %v\n\nCurrent break:\n\n%s %s %s %v \n\n", pm.history.poms, pad, pm.progress.View(pm.percent), pad, rem)
		}
	} else {
		// status has already been updated
		if runningPom {
			return fmt.Sprintf("\n%s %s %s Complete! 🌴 \n\nPress Enter to start pomodoro.\n\n", pad, pm.progress.View(pm.percent), pad)
		} else {
			return fmt.Sprintf("\n%s %s %s Complete! 🍅 \n\nPress Enter to start break.\n\n", pad, pm.progress.View(pm.percent), pad)
		}
	}
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
		history:   pomoHistory{poms: 0, shortBreaks: 0, longBreaks: 0},
	}

	if err := tea.NewProgram(pm).Start(); err != nil {
		fmt.Printf("Couldn't start program: %v\n", err)
		os.Exit(1)
	}

}
