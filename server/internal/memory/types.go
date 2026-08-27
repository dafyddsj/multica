package memory

import (
	"strings"
	"unicode/utf8"
)

const (
	ScopeWorkspace  = "workspace"
	ScopeInitiative = "initiative"
	ScopeProject    = "project"
	ScopeIssue      = "issue"
	ScopeSquad      = "squad"
	ScopeAgent      = "agent"
	ScopeUser       = "user"

	KindFact        = "fact"
	KindPreference  = "preference"
	KindProcedure   = "procedure"
	KindObservation = "observation"

	MaxBodyRunes     = 4000
	MaxBankEntries   = 200
	DefaultListLimit = 50
	MaxListLimit     = 100
	MaxRecallHits    = 8
	MaxRecallRunes   = 400

	// WorkspaceSettingsKey is the Labs toggle in workspace.settings.
	WorkspaceSettingsKey = "memory_enabled"
)

var scopes = map[string]struct{}{
	ScopeWorkspace:  {},
	ScopeInitiative: {},
	ScopeProject:    {},
	ScopeIssue:      {},
	ScopeSquad:      {},
	ScopeAgent:      {},
	ScopeUser:       {},
}

var kinds = map[string]struct{}{
	KindFact:        {},
	KindPreference:  {},
	KindProcedure:   {},
	KindObservation: {},
}

func ValidScope(scope string) bool {
	_, ok := scopes[scope]
	return ok
}

func ValidKind(kind string) bool {
	_, ok := kinds[kind]
	return ok
}

func NormalizeKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return KindFact
	}
	return kind
}

func ValidateBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "body is required"
	}
	if utf8.RuneCountInString(body) > MaxBodyRunes {
		return "body must be at most 4000 characters"
	}
	return ""
}

func ClampListLimit(limit int32) int32 {
	if limit <= 0 {
		return DefaultListLimit
	}
	if limit > MaxListLimit {
		return MaxListLimit
	}
	return limit
}
