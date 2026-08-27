package memory

import "context"

// SearchQuery is the engine-neutral recall request. Native ILIKE and a
// future Hindsight adapter both consume this.
type SearchQuery struct {
	WorkspaceID  string
	Query        string
	InitiativeID string
	ProjectID    string
	IssueID      string
	SquadID      string
	AgentID      string
	UserID       string
	Limit        int32
}

// Hit is one recalled entry after any truncation the caller asked for.
type Hit struct {
	ID      string `json:"id"`
	Scope   string `json:"scope"`
	OwnerID string `json:"owner_id"`
	Body    string `json:"body"`
	Kind    string `json:"kind"`
}

// MemoryEngine is the optional retrieval backend. v1 ships NativeEngine.
// A later Hindsight (or other) engine implements the same interface.
// Agents never call an engine. They call Multica.
type MemoryEngine interface {
	Search(ctx context.Context, q SearchQuery) ([]Hit, error)
}
