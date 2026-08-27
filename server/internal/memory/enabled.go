package memory

import (
	"context"
	"encoding/json"

	"github.com/multica-ai/multica/server/internal/featureflags"
	"github.com/multica-ai/multica/server/pkg/featureflag"
)

// WorkspaceEnabled reports the Labs toggle stored in workspace.settings.
// Missing or non-bool values are off.
func WorkspaceEnabled(settings []byte) bool {
	if len(settings) == 0 {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal(settings, &parsed); err != nil {
		return false
	}
	v, ok := parsed[WorkspaceSettingsKey]
	if !ok {
		return false
	}
	enabled, ok := v.(bool)
	return ok && enabled
}

// Available is the product gate: deployment flag and workspace Labs toggle.
func Available(ctx context.Context, flags *featureflag.Service, settings []byte) bool {
	if !featureflags.MemoryV1Enabled(ctx, flags) {
		return false
	}
	return WorkspaceEnabled(settings)
}
