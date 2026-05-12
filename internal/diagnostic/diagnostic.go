package diagnostic

import "fmt"

// Severity describes how a diagnostic should affect execution.
type Severity string

const (
	// Error blocks execution.
	Error Severity = "error"
	// Warning should be shown to the user but does not block execution.
	Warning Severity = "warning"
)

// Diagnostic is a validation or rendering message.
type Diagnostic struct {
	Severity Severity
	Message  string
}

// Errorf creates an error diagnostic.
func Errorf(format string, args ...any) Diagnostic {
	return Diagnostic{Severity: Error, Message: fmt.Sprintf(format, args...)}
}

// Warningf creates a warning diagnostic.
func Warningf(format string, args ...any) Diagnostic {
	return Diagnostic{Severity: Warning, Message: fmt.Sprintf(format, args...)}
}

// HasErrors reports whether any diagnostics are errors.
func HasErrors(diags []Diagnostic) bool {
	for _, diag := range diags {
		if diag.Severity == Error {
			return true
		}
	}
	return false
}

// FormatMany formats diagnostics for human-readable output.
func FormatMany(diags []Diagnostic) string {
	var out string
	for _, diag := range diags {
		out += fmt.Sprintf("[%s] %s\n", diag.Severity, diag.Message)
	}
	return out
}
