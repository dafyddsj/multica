//go:build windows

package agent

import "log/slog"

func platformAmpInvocation(lookedUp string, args []string, logger *slog.Logger) (string, []string, bool) {
	return rewriteCmdToPS1("amp", lookedUp, args, logger)
}
