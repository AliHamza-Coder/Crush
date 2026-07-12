package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
)

// SelectFromList shows an interactive select menu using huh (arrow keys).
// On terminals that don't support it, falls back to a numbered list.
func SelectFromList(items []string, prompt string) string {
	if len(items) == 0 {
		return ""
	}

	options := make([]huh.Option[string], len(items))
	for i, item := range items {
		options[i] = huh.NewOption(item, item)
	}

	var selected string

	err := huh.NewSelect[string]().
		Title(prompt).
		Options(options...).
		Value(&selected).
		Run()

	if err == nil {
		return selected
	}

	// Ctrl+C / interrupt → exit cleanly
	if errors.Is(err, context.Canceled) {
		fmt.Println()
		os.Exit(0)
	}

	// Fallback for non-interactive / unsupported terminals
	return selectFromListFallback(items, prompt)
}

// selectFromListFallback shows a simple numbered list when the interactive
// menu isn't available (e.g., non-interactive terminal, CI, etc.).
func selectFromListFallback(items []string, prompt string) string {
	fmt.Printf("\n  %s\n\n", prompt)
	for i, item := range items {
		fmt.Printf("    %d) %s\n", i+1, item)
	}
	fmt.Println()

	for {
		input := ReadInput("  Enter number (1-" + strconv.Itoa(len(items)) + "): ")

		if input == "" {
			return items[0]
		}

		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(items) {
			fmt.Printf("  Invalid choice. Enter 1-%d.\n", len(items))
			continue
		}

		fmt.Println()
		return items[num-1]
	}
}
