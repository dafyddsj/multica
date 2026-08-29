package budgetpolicy

import (
	"testing"
	"time"
)

func pct(n int16) *int16 { return &n }

func account(scope Scope, owner string, spent, limit int64, soften *int16, over OverLimit) Account {
	return Account{
		Budget:     BudgetRef{Scope: scope, OwnerID: owner},
		SpentTicks: spent,
		LimitTicks: limit,
		SoftenAt:   soften,
		OverLimit:  over,
	}
}

func TestDecideCompositionExample(t *testing.T) {
	agent := account(ScopeAgent, "A", 50, 100, pct(80), OverLimitAllow)
	project := account(ScopeProject, "P", 170, 200, pct(80), OverLimitPause)
	initiative := account(ScopeInitiative, "I", 1010, 1000, nil, OverLimitAllow)

	got := Decide([]Account{agent, project, initiative}, false)
	if got.Verdict != VerdictSoften || got.Holder != project.Budget {
		t.Fatalf("composed = %+v, want soften held by project", got)
	}

	initiative.OverLimit = OverLimitPause
	got = Decide([]Account{agent, project, initiative}, false)
	if got.Verdict != VerdictHold || got.Holder != initiative.Budget {
		t.Fatalf("pause flip = %+v, want hold held by initiative", got)
	}
}

func TestDecideWaiverDepth(t *testing.T) {
	agent := account(ScopeAgent, "A", 50, 100, pct(80), OverLimitAllow)
	project := account(ScopeProject, "P", 170, 200, pct(80), OverLimitPause)
	initiative := account(ScopeInitiative, "I", 1010, 1000, nil, OverLimitPause)
	project.Waived = true
	initiative.Waived = true

	got := Decide([]Account{agent, project, initiative}, false)
	if got.Verdict != VerdictProceed {
		t.Fatalf("waived resource accounts = %s, want proceed", got.Verdict)
	}

	agent.SpentTicks = 100
	agent.OverLimit = OverLimitPause
	got = Decide([]Account{agent, project, initiative}, false)
	if got.Verdict != VerdictHold || got.Holder.Scope != ScopeAgent {
		t.Fatalf("principal pause under waiver = %+v, want hold by agent", got)
	}
}

func TestDecideUnpriced(t *testing.T) {
	pause := account(ScopeProject, "P", 10, 100, pct(80), OverLimitPause)
	pause.Unpriced = 1
	if got := Decide([]Account{pause}, false); got.Verdict != VerdictHold {
		t.Fatalf("unpriced+pause = %s, want hold", got.Verdict)
	}

	allow := account(ScopeProject, "P", 10, 100, pct(80), OverLimitAllow)
	allow.Unpriced = 1
	if got := Decide([]Account{allow}, false); got.Verdict != VerdictProceed {
		t.Fatalf("unpriced+allow under threshold = %s, want proceed", got.Verdict)
	}

	allow.SpentTicks = 80
	if got := Decide([]Account{allow}, false); got.Verdict != VerdictSoften {
		t.Fatalf("unpriced+allow at threshold = %s, want soften", got.Verdict)
	}
}

func TestDecideAutopilotDowngradesSoften(t *testing.T) {
	project := account(ScopeProject, "P", 170, 200, pct(80), OverLimitPause)
	if got := Decide([]Account{project}, true); got.Verdict != VerdictProceed {
		t.Fatalf("autopilot soften = %s, want proceed", got.Verdict)
	}

	project.SpentTicks = 200
	if got := Decide([]Account{project}, true); got.Verdict != VerdictHold {
		t.Fatalf("autopilot hold still applies = %s, want hold", got.Verdict)
	}
}

func TestDecideIgnoresPrincipalWaived(t *testing.T) {
	agent := account(ScopeAgent, "A", 100, 100, nil, OverLimitPause)
	agent.Waived = true
	if got := Decide([]Account{agent}, false); got.Verdict != VerdictHold {
		t.Fatalf("waived principal = %s, want hold", got.Verdict)
	}
}

func TestDecideLatticeAssociativity(t *testing.T) {
	cases := []Account{
		account(ScopeAgent, "ok", 10, 100, pct(80), OverLimitAllow),
		account(ScopeProject, "soft", 80, 100, pct(80), OverLimitAllow),
		account(ScopeInitiative, "hold", 100, 100, nil, OverLimitPause),
	}
	direct := Decide(cases, false)
	left := Decide([]Account{
		{Budget: BudgetRef{Scope: ScopeAgent, OwnerID: "fold"}, OverLimit: OverLimitAllow},
	}, false)
	// Fold left then right: Decide is a max, so any permutation must match.
	perms := [][]Account{
		{cases[0], cases[1], cases[2]},
		{cases[2], cases[0], cases[1]},
		{cases[1], cases[2], cases[0]},
	}
	for _, p := range perms {
		got := Decide(p, false)
		if got.Verdict != direct.Verdict {
			t.Fatalf("permutation %v = %s, want %s", p, got.Verdict, direct.Verdict)
		}
	}
	if left.Verdict != VerdictProceed {
		t.Fatalf("empty-ish fold seed = %s, want proceed", left.Verdict)
	}
}

func TestStateOf(t *testing.T) {
	tests := []struct {
		name string
		a    Account
		want AccountState
	}{
		{"ok", account(ScopeAgent, "A", 10, 100, pct(80), OverLimitAllow), StateOK},
		{"softened", account(ScopeAgent, "A", 80, 100, pct(80), OverLimitAllow), StateSoftened},
		{"exhausted", account(ScopeAgent, "A", 100, 100, pct(80), OverLimitPause), StateExhausted},
		{"pricing_incomplete", func() Account {
			a := account(ScopeProject, "P", 10, 100, pct(80), OverLimitPause)
			a.Unpriced = 2
			return a
		}(), StatePricingIncomplete},
		{"unattributed", Account{Budget: BudgetRef{Scope: ScopeSquad}, Unattributed: true}, StateUnattributed},
		{"waived", func() Account {
			a := account(ScopeProject, "P", 200, 100, nil, OverLimitPause)
			a.Waived = true
			return a
		}(), StateWaived},
		{"principal waived stays exhausted", func() Account {
			a := account(ScopeAgent, "A", 200, 100, nil, OverLimitPause)
			a.Waived = true
			return a
		}(), StateExhausted},
	}
	for _, tc := range tests {
		if got := StateOf(tc.a); got != tc.want {
			t.Fatalf("%s: StateOf = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestWaiverCovers(t *testing.T) {
	projectW := Waiver{Scope: ScopeProject, OwnerID: "P"}
	initiativeW := Waiver{Scope: ScopeInitiative, OwnerID: "I"}
	taskOnP := TaskRef{ProjectID: "P", InitiativeID: "I"}
	taskOnOther := TaskRef{ProjectID: "Q", InitiativeID: "J"}

	tests := []struct {
		name    string
		w       Waiver
		account BudgetRef
		task    TaskRef
		want    bool
	}{
		{"project matches project", projectW, BudgetRef{Scope: ScopeProject, OwnerID: "P"}, taskOnP, true},
		{"project covers parent initiative", projectW, BudgetRef{Scope: ScopeInitiative, OwnerID: "I"}, taskOnP, true},
		{"project does not cover other initiative", projectW, BudgetRef{Scope: ScopeInitiative, OwnerID: "I"}, taskOnOther, false},
		{"project does not cover agent", projectW, BudgetRef{Scope: ScopeAgent, OwnerID: "A"}, taskOnP, false},
		{"project does not cover squad", projectW, BudgetRef{Scope: ScopeSquad, OwnerID: "S"}, taskOnP, false},
		{"initiative matches initiative", initiativeW, BudgetRef{Scope: ScopeInitiative, OwnerID: "I"}, taskOnP, true},
		{"initiative covers child project", initiativeW, BudgetRef{Scope: ScopeProject, OwnerID: "P"}, taskOnP, true},
		{"initiative does not cover other project", initiativeW, BudgetRef{Scope: ScopeProject, OwnerID: "Q"}, taskOnOther, false},
		{"initiative does not cover agent", initiativeW, BudgetRef{Scope: ScopeAgent, OwnerID: "A"}, taskOnP, false},
	}
	for _, tc := range tests {
		if got := WaiverCovers(tc.w, tc.account, tc.task); got != tc.want {
			t.Fatalf("%s: WaiverCovers = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMonthWindowUTC(t *testing.T) {
	start, end := MonthWindow(time.Date(2026, 8, 27, 23, 30, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("August start = %s", start)
	}
	if !end.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("August end = %s", end)
	}

	// Local September morning east of UTC can still be August UTC.
	loc := time.FixedZone("UTC+12", 12*60*60)
	start, end = MonthWindow(time.Date(2026, 9, 1, 8, 0, 0, 0, loc))
	if !start.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) || !end.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("offset morning window = %s .. %s, want August UTC", start, end)
	}

	start, end = MonthWindow(time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) || !end.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("December window = %s .. %s", start, end)
	}
}
