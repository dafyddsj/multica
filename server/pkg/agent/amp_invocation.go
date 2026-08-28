package agent

import "log/slog"

// chooseAmpInvocation selects the program and argv for an Amp run.
// On Windows npm ships amp.cmd; cmd.exe %* re-tokenisation would mangle
// the managed flags, so we route to amp.ps1. Elsewhere this is a passthrough.
func chooseAmpInvocation(execName, lookedUp string, args []string, logger *slog.Logger) (string, []string) {
	if argv0, full, ok := platformAmpInvocation(lookedUp, args, logger); ok {
		return argv0, full
	}
	return execName, args
}
