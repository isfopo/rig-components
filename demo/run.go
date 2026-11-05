// Package example provides a tiny, reusable harness for running a single
// component in isolation during development — the closest thing to a
// Storybook "story" for a terminal UI component.
//
// Usage:
//
//	func main() {
//	    m := myComponentModel{...}
//	    if err := demo.Run(m); err != nil {
//	        log.Fatal(err)
//	    }
//	}
package demo

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts a Bubble Tea program for m in the alternate screen with a
// sensible default configuration. It blocks until the program exits.
func Run(m tea.Model) error {
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run component: %w", err)
	}
	return nil
}

// Fatal runs m and exits the process on error. Convenience for example mains.
func Fatal(m tea.Model) {
	if err := Run(m); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
