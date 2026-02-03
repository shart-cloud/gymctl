package cli

import (
	"time"

	"github.com/briandowns/spinner"
)

// NewSpinner creates a new spinner with default styling
func NewSpinner(suffix string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + suffix
	s.Color("cyan", "bold")
	return s
}

// WithSpinner runs a function with a spinner
func WithSpinner(message string, fn func() error) error {
	s := NewSpinner(message)
	s.Start()
	defer s.Stop()

	err := fn()
	s.Stop()

	if err != nil {
		ColorError.Printf("%s %s\n", IconFail, message)
		return err
	}

	ColorSuccess.Printf("%s %s\n", IconSuccess, message)
	return nil
}

// SpinnerManager manages long running operations with status updates
type SpinnerManager struct {
	spinner *spinner.Spinner
	message string
}

func NewSpinnerManager() *SpinnerManager {
	return &SpinnerManager{}
}

func (sm *SpinnerManager) Start(message string) {
	if sm.spinner != nil {
		sm.spinner.Stop()
	}
	sm.message = message
	sm.spinner = NewSpinner(message)
	sm.spinner.Start()
}

func (sm *SpinnerManager) Update(message string) {
	if sm.spinner != nil {
		sm.spinner.Suffix = " " + message
	}
	sm.message = message
}

func (sm *SpinnerManager) Success(message string) {
	if sm.spinner != nil {
		sm.spinner.Stop()
	}
	if message == "" {
		message = sm.message
	}
	ColorSuccess.Printf("%s %s\n", IconSuccess, message)
	sm.spinner = nil
}

func (sm *SpinnerManager) Fail(message string) {
	if sm.spinner != nil {
		sm.spinner.Stop()
	}
	if message == "" {
		message = sm.message
	}
	ColorError.Printf("%s %s\n", IconFail, message)
	sm.spinner = nil
}

func (sm *SpinnerManager) Stop() {
	if sm.spinner != nil {
		sm.spinner.Stop()
		sm.spinner = nil
	}
}