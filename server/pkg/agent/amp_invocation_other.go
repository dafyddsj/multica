//go:build !windows

package agent

import "log/slog"

func platformAmpInvocation(_ string, _ []string, _ *slog.Logger) (string, []string, bool) {
	return "", nil, false
}
