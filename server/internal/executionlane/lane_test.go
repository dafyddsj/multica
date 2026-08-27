package executionlane

import (
	"testing"

	"github.com/multica-ai/multica/server/pkg/taskfailure"
)

func TestInitial(t *testing.T) {
	lanes := AgentLanes{
		Model:            "opus",
		LightweightModel: "haiku",
		StartLightweight: true,
	}
	got := Initial(lanes, true)
	if got.Lane != LaneLightweight || got.Model != "haiku" || !got.ForceFreshSession {
		t.Fatalf("want lightweight/haiku fresh, got %+v", got)
	}
	got = Initial(lanes, false)
	if got.Lane != LanePrimary || got.Model != "opus" {
		t.Fatalf("disabled want primary/opus, got %+v", got)
	}
	lanes.StartLightweight = false
	got = Initial(lanes, true)
	if got.Lane != LanePrimary {
		t.Fatalf("start off want primary, got %+v", got)
	}
}

func TestNextOnFailure(t *testing.T) {
	lanes := AgentLanes{
		Model:             "opus",
		LightweightModel:  "haiku",
		FailoverModel:     "sonnet",
		FailoverRuntimeID: "rt-b",
	}
	_, ok := NextOnFailure(lanes, LaneLightweight, string(taskfailure.ReasonAgentUnknown), true)
	if ok {
		t.Fatal("unknown reason must not hop")
	}
	got, ok := NextOnFailure(lanes, LaneLightweight, string(taskfailure.ReasonAgentProviderQuotaLimit), true)
	if !ok || got.Lane != LanePrimary || got.Model != "opus" || !got.ForceFreshSession {
		t.Fatalf("lightweight hop want primary/opus fresh, got %+v ok=%v", got, ok)
	}
	got, ok = NextOnFailure(lanes, LanePrimary, string(taskfailure.ReasonAgentModelNotFoundOrUnavailable), true)
	if !ok || got.Lane != LaneFailover || got.Model != "sonnet" || got.RuntimeID != "rt-b" || !got.ForceFreshSession {
		t.Fatalf("primary hop want failover/sonnet, got %+v ok=%v", got, ok)
	}
	_, ok = NextOnFailure(lanes, LaneFailover, string(taskfailure.ReasonAgentProviderCapacityOrRateLimit), true)
	if ok {
		t.Fatal("failover must not hop again")
	}
}

func TestResolveClaimOverrideWins(t *testing.T) {
	lanes := AgentLanes{Model: "opus", LightweightModel: "haiku"}
	got := ResolveClaim(lanes, LaneLightweight, "stamped-haiku")
	if got.Model != "stamped-haiku" || got.Lane != LaneLightweight {
		t.Fatalf("got %+v", got)
	}
}
