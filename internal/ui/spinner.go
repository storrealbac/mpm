package ui

import (
	"fmt"
	"time"
)

// Spinner represents a simple console spinner
type Spinner struct {
	frames []string
	index  int
	active bool
	msg    string
}

// NewSpinner creates a new spinner instance
func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		index:  0,
		active: false,
		msg:    message,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.active = true
	go func() {
		for s.active {
			frame := InfoStyle.Render(s.frames[s.index])
			fmt.Printf("\r%s %s", frame, s.msg)
			s.index = (s.index + 1) % len(s.frames)
			time.Sleep(80 * time.Millisecond)
		}
	}()
}

// Stop halts the spinner and clears the line
func (s *Spinner) Stop() {
	s.active = false
	time.Sleep(100 * time.Millisecond) // Give goroutine time to finish
	fmt.Print("\r\033[K")              // Clear the line
}

// Success stops the spinner and shows a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	PrintSuccess(message)
}

// Error stops the spinner and shows an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	PrintError(message)
}
