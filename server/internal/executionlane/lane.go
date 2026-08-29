package executionlane

import "github.com/multica-ai/multica/server/pkg/taskfailure"

// Lane is the execution slot a task was stamped with.
type Lane string

const (
	LanePrimary     Lane = "primary"
	LaneLightweight Lane = "lightweight"
	LaneFailover    Lane = "failover"
)

// AgentLanes is the user-configured model set for one agent.
type AgentLanes struct {
	Model                    string
	ThinkingLevel            string
	ServiceTier              string
	LightweightModel         string
	LightweightThinkingLevel string
	StartLightweight         bool
	FailoverRuntimeID        string
	FailoverModel            string
	FailoverThinkingLevel    string
	FailoverServiceTier      string
}

// Selection is the concrete model a task should run.
type Selection struct {
	Lane              Lane
	Model             string
	ThinkingLevel     string
	ServiceTier       string
	RuntimeID         string
	ForceFreshSession bool
}

// Initial picks the first lane for a new task.
func Initial(lanes AgentLanes) Selection {
	if lanes.StartLightweight && lanes.LightweightModel != "" {
		return Selection{
			Lane:              LaneLightweight,
			Model:             lanes.LightweightModel,
			ThinkingLevel:     lanes.LightweightThinkingLevel,
			ForceFreshSession: true,
		}
	}
	return primarySelection(lanes)
}

// ResolveClaim returns the model the claim payload should send.
// modelOverride wins for the model id when the task stamped one.
func ResolveClaim(lanes AgentLanes, lane Lane, modelOverride string) Selection {
	sel := selectionForLane(lanes, lane)
	if modelOverride != "" {
		sel.Model = modelOverride
	}
	return sel
}

// NextOnFailure returns the next unused lane after a failover-class failure.
func NextOnFailure(lanes AgentLanes, current Lane, reason string) (Selection, bool) {
	if !IsFailoverReason(reason) {
		return Selection{}, false
	}
	switch current {
	case LaneLightweight:
		sel := primarySelection(lanes)
		sel.ForceFreshSession = true
		return sel, true
	case LanePrimary, "":
		if lanes.FailoverModel == "" {
			return Selection{}, false
		}
		return Selection{
			Lane:              LaneFailover,
			Model:             lanes.FailoverModel,
			ThinkingLevel:     lanes.FailoverThinkingLevel,
			ServiceTier:       lanes.FailoverServiceTier,
			RuntimeID:         lanes.FailoverRuntimeID,
			ForceFreshSession: true,
		}, true
	default:
		return Selection{}, false
	}
}

// IsFailoverReason reports failures that should hop lanes instead of
// retrying the same model.
func IsFailoverReason(reason string) bool {
	switch taskfailure.Reason(reason) {
	case taskfailure.ReasonAgentModelNotFoundOrUnavailable,
		taskfailure.ReasonAgentProviderCapacityOrRateLimit,
		taskfailure.ReasonAgentProviderQuotaLimit:
		return true
	default:
		return false
	}
}

// ParseLane maps a stored string to a Lane. Unknown values become primary.
func ParseLane(raw string) Lane {
	switch Lane(raw) {
	case LaneLightweight, LaneFailover, LanePrimary:
		return Lane(raw)
	default:
		return LanePrimary
	}
}

func primarySelection(lanes AgentLanes) Selection {
	return Selection{
		Lane:          LanePrimary,
		Model:         lanes.Model,
		ThinkingLevel: lanes.ThinkingLevel,
		ServiceTier:   lanes.ServiceTier,
	}
}

func selectionForLane(lanes AgentLanes, lane Lane) Selection {
	switch lane {
	case LaneLightweight:
		return Selection{
			Lane:              LaneLightweight,
			Model:             lanes.LightweightModel,
			ThinkingLevel:     lanes.LightweightThinkingLevel,
			ForceFreshSession: true,
		}
	case LaneFailover:
		return Selection{
			Lane:              LaneFailover,
			Model:             lanes.FailoverModel,
			ThinkingLevel:     lanes.FailoverThinkingLevel,
			ServiceTier:       lanes.FailoverServiceTier,
			RuntimeID:         lanes.FailoverRuntimeID,
			ForceFreshSession: true,
		}
	default:
		return primarySelection(lanes)
	}
}
