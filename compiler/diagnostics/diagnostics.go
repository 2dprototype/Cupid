package diagnostics

import (
	"fmt"
	"strings"
)

type Diagnostic struct {
	Code     string
	Message  string
	File     string
	Line     int
	Column   int
	SpanLen  int
	Help     string
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("error[%s]: %s\n  --> %s:%d:%d", d.Code, d.Message, d.File, d.Line, d.Column)
}

func FormatError(source string, diag Diagnostic) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\x1b[1;31merror[%s]: %s\x1b[0m\n", diag.Code, diag.Message))
	sb.WriteString(fmt.Sprintf("  \x1b[1;34m-->\x1b[0m %s:%d:%d\n", diag.File, diag.Line, diag.Column))

	lines := strings.Split(source, "\n")
	if diag.Line > 0 && diag.Line <= len(lines) {
		sb.WriteString("   \x1b[1;34m|\x1b[0m\n")
		// Print line before, target line, line after
		if diag.Line > 1 {
			sb.WriteString(fmt.Sprintf("%2d \x1b[1;34m|\x1b[0m %s\n", diag.Line-1, lines[diag.Line-2]))
		}
		sb.WriteString(fmt.Sprintf("%2d \x1b[1;34m|\x1b[0m %s\n", diag.Line, lines[diag.Line-1]))
		
		// Highlight underline
		sb.WriteString("   \x1b[1;34m|\x1b[0m ")
		for i := 1; i < diag.Column; i++ {
			sb.WriteString(" ")
		}
		span := diag.SpanLen
		if span <= 0 {
			span = 1
		}
		sb.WriteString("\x1b[1;31m")
		for i := 0; i < span; i++ {
			sb.WriteString("^")
		}
		sb.WriteString(" \x1b[1;31mhere\x1b[0m\n")

		if diag.Line < len(lines) {
			sb.WriteString(fmt.Sprintf("%2d \x1b[1;34m|\x1b[0m %s\n", diag.Line+1, lines[diag.Line]))
		}
		sb.WriteString("   \x1b[1;34m|\x1b[0m\n")
	}

	if diag.Help != "" {
		sb.WriteString(fmt.Sprintf("\x1b[1;32mhelp:\x1b[0m\n%s\n", diag.Help))
	}
	return sb.String()
}
