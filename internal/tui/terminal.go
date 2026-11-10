package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alexandrmotologa/hookcraft/internal/runner"
	"github.com/alexandrmotologa/hookcraft/internal/secrets"
	"golang.org/x/term"
)

// ANSI color codes.
const (
	Reset       = "\033[0m"
	Bold        = "\033[1m"
	Dim         = "\033[2m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	Gray        = "\033[90m"
	BgRed       = "\033[41m"
	BgGreen     = "\033[42m"
	WhiteOnRed  = "\033[97;41m"
	CheckMark   = "✓"
	CrossMark   = "✗"
	WarningMark = "!"
	SkipMark    = "↷"
)

// Output manages colored console output respecting terminal capabilities.
type Output struct {
	writer  io.Writer
	noColor bool
	isTTY   bool
}

// NewOutput creates an Output manager.
func NewOutput(w io.Writer) *Output {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	var isTTY bool
	if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}
	return &Output{
		writer:  w,
		noColor: noColor,
		isTTY:   isTTY,
	}
}

// Colorize wraps text with ANSI codes if colors are enabled.
func (o *Output) Colorize(code, text string) string {
	if o.noColor {
		return text
	}
	return code + text + Reset
}

// PrintBanner prints the HookCraft startup banner.
func (o *Output) PrintBanner(hookName string, stagedCount int) {
	fmt.Fprintf(o.writer, "%s %s %s [%s staged files]\n",
		o.Colorize(Bold+Cyan, "hookcraft"),
		o.Colorize(Dim+Gray, "::"),
		o.Colorize(Bold, hookName),
		o.Colorize(Yellow, fmt.Sprintf("%d", stagedCount)),
	)
}

// PrintStep prints the outcome of a single step.
func (o *Output) PrintStep(res runner.StepResult) {
	var statusMark string
	var timing string

	timing = o.Colorize(Dim+Gray, fmt.Sprintf("(%.1fms)", res.DurationMs))

	if res.Skipped {
		statusMark = o.Colorize(Dim+Yellow, SkipMark+" SKIPPED")
		fmt.Fprintf(o.writer, "  %s  %-28s %s %s\n",
			statusMark,
			res.StepName,
			timing,
			o.Colorize(Dim+Gray, res.SkipReason),
		)
		return
	}

	if res.Success {
		statusMark = o.Colorize(Bold+Green, CheckMark+" PASS")
		fmt.Fprintf(o.writer, "  %s  %-28s %s\n", statusMark, res.StepName, timing)
		if len(res.ModifiedFiles) > 0 {
			fmt.Fprintf(o.writer, "        %s %s\n",
				o.Colorize(Dim+Cyan, "auto-formatted:"),
				strings.Join(res.ModifiedFiles, ", "),
			)
		}
	} else {
		statusMark = o.Colorize(Bold+Red, CrossMark+" FAIL")
		fmt.Fprintf(o.writer, "  %s  %-28s %s\n", statusMark, res.StepName, timing)
		if res.Output != "" {
			lines := strings.Split(res.Output, "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					fmt.Fprintf(o.writer, "        %s\n", o.Colorize(Red, line))
				}
			}
		}
	}
}

// PrintSecretShieldAlert prints a prominent alert box for detected secret leaks.
func (o *Output) PrintSecretShieldAlert(findings []secrets.Finding) {
	border := strings.Repeat("═", 72)
	fmt.Fprintln(o.writer)
	fmt.Fprintf(o.writer, " %s\n", o.Colorize(Bold+WhiteOnRed, " [SECURITY ALERT] STAGED SECRET LEAK DETECTED "))
	fmt.Fprintf(o.writer, "%s\n", o.Colorize(Red, border))
	fmt.Fprintf(o.writer, " HookCraft blocked commit to protect repository credentials:\n\n")

	for i, f := range findings {
		fmt.Fprintf(o.writer, "  %d. %s\n", i+1, o.Colorize(Bold+Yellow, f.RuleName))
		fmt.Fprintf(o.writer, "     Location : %s:%d\n", o.Colorize(Cyan, f.FilePath), f.LineNumber)
		fmt.Fprintf(o.writer, "     Secret   : %s\n", o.Colorize(Bold+Red, f.RedactedValue))
		if f.Entropy > 0 {
			fmt.Fprintf(o.writer, "     Entropy  : %.2f bits/byte\n", f.Entropy)
		}
		fmt.Fprintf(o.writer, "     Diff Line: %s\n", o.Colorize(Dim+Gray, f.RawLine))
		fmt.Fprintln(o.writer)
	}

	fmt.Fprintf(o.writer, "%s\n", o.Colorize(Red, border))
	fmt.Fprintf(o.writer, " %s To bypass intentional test keys, add '%s' comment to the line.\n\n",
		o.Colorize(Yellow, WarningMark),
		o.Colorize(Bold, "// hookcraft:ignore"),
	)
}

// PrintSummary prints the final timing and status.
func (o *Output) PrintSummary(summary *runner.RunSummary) {
	fmt.Fprintln(o.writer)
	if summary.Success {
		fmt.Fprintf(o.writer, "%s %s %s %.1fms\n",
			o.Colorize(Bold+Green, CheckMark),
			o.Colorize(Bold, "All hooks passed cleanly"),
			o.Colorize(Dim+Gray, "in"),
			summary.DurationMs,
		)
	} else {
		fmt.Fprintf(o.writer, "%s %s %s %.1fms\n",
			o.Colorize(Bold+Red, CrossMark),
			o.Colorize(Bold, "Hook execution failed"),
			o.Colorize(Dim+Gray, "in"),
			summary.DurationMs,
		)
	}
}
