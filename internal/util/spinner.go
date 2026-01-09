package util

import (
	"fmt"
	"time"
)

// Spinner is a simple CLI spinner
type Spinner struct {
	message string
	stop    chan struct{}
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		stop:    make(chan struct{}),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				fmt.Printf("\r\033[36m%s\033[0m %s", frames[i%len(frames)], s.message)
				i++
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	close(s.stop)
	fmt.Print("\r\033[K") // Clear the line
}
