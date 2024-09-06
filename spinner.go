package main

import (
	"fmt"
	"time"
)

type Spinner struct {
	sybmols  []string
	frame    int
	maxFrame int
}

func Tick(s Spinner) int {
	fmt.Print(s.sybmols[s.frame])
	if s.frame+1 <= s.maxFrame {
		s.frame += 1
	} else {
		s.frame = 0
	}
	return s.frame
}

func (s Spinner) Clear() {
	fmt.Print("\b")
}

func main() {
	spinner := Spinner{
		sybmols:  []string{"⠇", "⠏", "⠹", "⠼"},
		frame:    0,
		maxFrame: 3,
	}
	fmt.Print("Loading ")
	for {
		spinner.frame = Tick(spinner)
		time.Sleep(250 * time.Millisecond)
		spinner.Clear()
	}
}
