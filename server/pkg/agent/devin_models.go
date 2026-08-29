package agent

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var devinModelIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_./:-]{0,127}$`)

var devinModelContainers = map[string]struct{}{
	"models":           {},
	"data":             {},
	"items":            {},
	"families":         {},
	"available_models": {},
	"availableModels":  {},
}

// discoverDevinModels runs `devin models list --format json` and parses the
// account catalog. Devin's list is account-specific, so a failed or empty
// discovery stays Fallback with no models rather than a static stand-in that
// the CLI never advertised.
func discoverDevinModels(ctx context.Context, runtimeCmd Command) (Catalog, error) {
	if runtimeCmd.Path == "" {
		runtimeCmd.Path = "devin"
	}
	if _, err := exec.LookPath(runtimeCmd.Path); err != nil {
		return Catalog{Models: []Model{}, Fallback: true}, nil
	}
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := runtimeCmd.exec(runCtx, "models", "list", "--format", "json")
	hideAgentWindow(cmd)
	out, err := outputOwned(cmd, runtimeCmd.logger)
	if err != nil && len(out) == 0 {
		return Catalog{Models: []Model{}, Fallback: true}, nil
	}
	models := parseDevinModelsJSON(out)
	if len(models) == 0 {
		return Catalog{Models: []Model{}, Fallback: true}, nil
	}
	return Catalog{Models: models}, nil
}

func parseDevinModelsJSON(raw []byte) []Model {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return []Model{}
	}
	models := []Model{}
	walkDevinModels(root, "", map[string]struct{}{}, &models)
	return models
}

func walkDevinModels(node any, inheritedProvider string, seen map[string]struct{}, models *[]Model) {
	switch v := node.(type) {
	case []any:
		for _, item := range v {
			walkDevinModels(item, inheritedProvider, seen, models)
		}
	case map[string]any:
		if model, ok := modelFromDevinObject(v, inheritedProvider); ok {
			if _, dup := seen[model.ID]; !dup {
				seen[model.ID] = struct{}{}
				*models = append(*models, model)
			}
		}
		childProvider := objectProvider(v)
		if childProvider == "" {
			childProvider = inheritedProvider
		}
		for key, child := range v {
			if _, ok := devinModelContainers[key]; !ok {
				continue
			}
			walkDevinModels(child, childProvider, seen, models)
		}
	}
}

func modelFromDevinObject(obj map[string]any, inheritedProvider string) (Model, bool) {
	id := firstDevinModelID(obj)
	if id == "" {
		return Model{}, false
	}
	provider := objectProvider(obj)
	if provider == "" {
		provider = inheritedProvider
	}
	if provider == "" {
		provider = "devin"
	}
	return Model{
		ID:       id,
		Label:    firstDevinLabel(obj, id),
		Provider: provider,
		Default:  objectDefault(obj),
	}, true
}

func firstDevinModelID(obj map[string]any) string {
	for _, key := range []string{"id", "slug", "model", "model_id", "modelId"} {
		s, ok := devinJSONString(obj[key])
		if !ok || !isDevinModelID(s) {
			continue
		}
		return s
	}
	return ""
}

func firstDevinLabel(obj map[string]any, fallback string) string {
	for _, key := range []string{"name", "display_name", "displayName", "title", "label"} {
		s, ok := devinJSONString(obj[key])
		if ok {
			return s
		}
	}
	return fallback
}

func objectProvider(obj map[string]any) string {
	for _, key := range []string{"provider", "family", "vendor"} {
		s, ok := devinJSONString(obj[key])
		if ok {
			return s
		}
	}
	return ""
}

func objectDefault(obj map[string]any) bool {
	for _, key := range []string{"default", "is_default", "isDefault"} {
		if devinJSONTrue(obj[key]) {
			return true
		}
	}
	return false
}

func isDevinModelID(s string) bool {
	return devinModelIDPattern.MatchString(s) && !looksLikePathOrFlag(s)
}

func looksLikePathOrFlag(s string) bool {
	if strings.HasPrefix(s, "-") {
		return true
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") {
		return true
	}
	if strings.Contains(s, "://") || strings.ContainsAny(s, `\`) {
		return true
	}
	if len(s) >= 3 {
		drive := s[0]
		if (drive >= 'A' && drive <= 'Z' || drive >= 'a' && drive <= 'z') && s[1] == ':' && (s[2] == '/' || s[2] == '\\') {
			return true
		}
	}
	return false
}

func devinJSONString(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	return s, s != ""
}

func devinJSONTrue(v any) bool {
	b, ok := v.(bool)
	return ok && b
}
