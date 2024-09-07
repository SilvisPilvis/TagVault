package main

import (
	"fmt"
	"time"
)

type Spinner struct {
	sybmols   []string
	frame     int
	maxFrames int
}

func (s Spinner) NewSpinner() *Spinner {
	// sybmols := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	// sybmols := []string{"⠇", "⠏", "⠹", "⠼"}
	// sybmols := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	sybmols := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return &Spinner{
		sybmols:   sybmols,
		frame:     0,
		maxFrames: len(sybmols) - 1,
	}
}

func (s Spinner) GetMaxFrames() int {
	return s.maxFrames
}

func (s *Spinner) Tick() {
	fmt.Printf("\r%s", s.sybmols[s.frame])
	if s.frame+1 <= s.maxFrames {
		s.frame++
	} else {
		s.frame = 0
	}
}

func (s Spinner) Clear() {
	fmt.Print("\b")
}

func main() {
	spinner := new(Spinner).NewSpinner()
	fmt.Print("  Loading")
	for {
		spinner.Tick()
		time.Sleep(250 * time.Millisecond)
		spinner.Clear()
	}
}
