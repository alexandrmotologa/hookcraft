package tui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// FixApprovalResult is the outcome of the interactive prompt.
type FixApprovalResult int

const (
	ApprovalYes FixApprovalResult = iota
	ApprovalNo
	ApprovalAbort
)

// AskApprovalForFixes prompts the user interactively with a 1-key press.
func AskApprovalForFixes(out *Output, modifiedFiles []string, diffFetcher func() (string, error)) (FixApprovalResult, error) {
	if len(modifiedFiles) == 0 {
		return ApprovalNo, nil
	}

	// Check if Stdin is a terminal
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// Non-interactive environment, skip prompt
		return ApprovalYes, nil
	}

	for {
		fmt.Fprintln(out.writer)
		fmt.Fprintf(out.writer, "%s %s\n",
			out.Colorize(Bold+Cyan, "[hookcraft]"),
			out.Colorize(Bold, "Formatters modified staged files:"),
		)
		for _, f := range modifiedFiles {
			fmt.Fprintf(out.writer, "  - %s\n", out.Colorize(Yellow, f))
		}

		fmt.Fprintf(out.writer, "\nAuto-stage formatted files now? [%s]es, [%s]o, [%s]iff: ",
			out.Colorize(Bold+Green, "y"),
			out.Colorize(Bold+Red, "n"),
			out.Colorize(Bold+Cyan, "d"),
		)

		// Set raw mode to read single keypress
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			// Fallback to regular line reading if raw mode fails
			return promptFallback(modifiedFiles)
		}

		var b [1]byte
		_, readErr := os.Stdin.Read(b[:])
		_ = term.Restore(fd, oldState)

		if readErr != nil {
			fmt.Fprintln(out.writer)
			return ApprovalNo, readErr
		}

		char := strings.ToLower(string(b[0]))
		fmt.Fprintf(out.writer, "%s\n", char)

		switch char {
		case "y", "\r", "\n":
			return ApprovalYes, nil
		case "n", "q":
			return ApprovalNo, nil
		case "\x03": // Ctrl+C
			return ApprovalAbort, nil
		case "d":
			// Show diff
			if diffFetcher != nil {
				diff, err := diffFetcher()
				if err != nil {
					fmt.Fprintf(out.writer, "Error getting diff: %v\n", err)
				} else {
					fmt.Fprintf(out.writer, "\n%s\n", out.Colorize(Dim+Gray, diff))
				}
			}
			continue
		default:
			fmt.Fprintf(out.writer, "Invalid choice '%s'. Please press 'y', 'n', or 'd'.\n", char)
			continue
		}
	}
}

func promptFallback(modifiedFiles []string) (FixApprovalResult, error) {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return ApprovalNo, nil
	}
	r := strings.ToLower(strings.TrimSpace(response))
	if r == "y" || r == "yes" {
		return ApprovalYes, nil
	}
	return ApprovalNo, nil
}
