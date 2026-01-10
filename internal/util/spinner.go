package util

import (
	"fmt"
	"sync"
	"time"
)

// Spinner is a simple CLI spinner
type Spinner struct {
	message string
	stop    chan struct{}
	mu      sync.Mutex
	stopped bool
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
	
	// Print first frame immediately
	s.mu.Lock()
	msg := s.message
	s.mu.Unlock()
	fmt.Printf("\r\033[36m%s\033[0m %s", frames[0], msg)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 1
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				fmt.Printf("\r\033[36m%s\033[0m %s", frames[i%len(frames)], msg)
				i++
			}
		}
	}()
}

// Update updates the spinner message
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	close(s.stop)
	s.stopped = true
	fmt.Print("\r\033[K") // Clear the line
}
