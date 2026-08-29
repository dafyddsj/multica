package agentmail

import "encoding/json"

func claimOverlayJSON(mcpURL, inboxKey string) (json.RawMessage, error) {
	payload := map[string]any{
		"mcpServers": map[string]any{
			"agentmail": map[string]any{
				"type": "http",
				"url":  mcpURL,
				"headers": map[string]string{
					"x-api-key": inboxKey,
				},
			},
		},
	}
	return json.Marshal(payload)
}
