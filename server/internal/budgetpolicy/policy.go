package budgetpolicy

import "time"

// Verdict is the composed claim decision. Order is proceed < soften < hold.
type Verdict int

const (
	VerdictProceed Verdict = iota
	VerdictSoften
	VerdictHold
)

func (v Verdict) String() string {
	switch v {
	case VerdictProceed:
		return "proceed"
	case VerdictSoften:
		return "soften"
	case VerdictHold:
		return "hold"
	default:
		return "proceed"
	}
}

// OverLimit is the configured action once spent meets the limit.
type OverLimit string

const (
	OverLimitPause OverLimit = "pause"
	OverLimitAllow OverLimit = "allow"
)

// Scope is one budget owner kind.
type Scope string

const (
	ScopeAgent      Scope = "agent"
	ScopeSquad      Scope = "squad"
	ScopeProject    Scope = "project"
	ScopeInitiative Scope = "initiative"
)

func (s Scope) isResource() bool {
	return s == ScopeProject || s == ScopeInitiative
}

func (s Scope) isPrincipal() bool {
	return s == ScopeAgent || s == ScopeSquad
}

// BudgetRef names one covering account.
type BudgetRef struct {
	Scope   Scope
	OwnerID string
}

// TaskRef is the frozen coverage snapshot on a task.
type TaskRef struct {
	ProjectID    string
	InitiativeID string
}

// AccountState is the bar label for one account.
type AccountState string

const (
	StateOK                 AccountState = "ok"
	StateSoftened           AccountState = "softened"
	StateExhausted          AccountState = "exhausted"
	StatePricingIncomplete  AccountState = "pricing_incomplete"
	StateUnattributed       AccountState = "unattributed"
	StateWaived             AccountState = "waived"
)

// Account is one covering budget at Decide time.
type Account struct {
	Budget     BudgetRef
	LimitTicks int64
	SpentTicks int64
	Unpriced   int
	SoftenAt   *int16
	OverLimit  OverLimit
	Waived     bool
	// Unattributed is a squad (or other) bar with no origin stamp.
	// Spent stays 0. Decide treats it as proceed.
	Unattributed bool
}

// Admission is the composed claim verdict.
type Admission struct {
	Verdict Verdict
	Holder  BudgetRef
}

// Waiver is a live resource-scope carve-out.
type Waiver struct {
	Scope   Scope
	OwnerID string
}

// Decide returns the strictest local verdict. Autopilot downgrades soften
// to proceed. A waived resource account contributes proceed. Principal
// Waived is ignored so a waiver cannot clear an agent or squad pause.
func Decide(accounts []Account, forAutopilot bool) Admission {
	best := Admission{Verdict: VerdictProceed}
	for _, a := range accounts {
		v := localVerdict(a)
		if forAutopilot && v == VerdictSoften {
			v = VerdictProceed
		}
		if v > best.Verdict {
			best = Admission{Verdict: v, Holder: a.Budget}
		}
	}
	return best
}

func localVerdict(a Account) Verdict {
	if a.Unattributed {
		return VerdictProceed
	}
	if a.Waived && a.Budget.Scope.isResource() {
		return VerdictProceed
	}
	if a.Unpriced > 0 && a.OverLimit == OverLimitPause {
		return VerdictHold
	}
	if a.LimitTicks > 0 && a.SpentTicks >= a.LimitTicks && a.OverLimit == OverLimitPause {
		return VerdictHold
	}
	if crossedSoften(a) {
		return VerdictSoften
	}
	return VerdictProceed
}

func crossedSoften(a Account) bool {
	if a.SoftenAt == nil || a.LimitTicks <= 0 {
		return false
	}
	percent := *a.SoftenAt
	if percent < 1 {
		return false
	}
	return a.SpentTicks*100 >= a.LimitTicks*int64(percent)
}

// StateOf is the bar label for one account. It does not compose.
func StateOf(a Account) AccountState {
	if a.Unattributed {
		return StateUnattributed
	}
	if a.Waived && a.Budget.Scope.isResource() {
		return StateWaived
	}
	if a.Unpriced > 0 {
		return StatePricingIncomplete
	}
	if a.LimitTicks > 0 && a.SpentTicks >= a.LimitTicks {
		return StateExhausted
	}
	if crossedSoften(a) {
		return StateSoftened
	}
	return StateOK
}

// WaiverCovers is the depth rule. Agent and squad accounts never match.
// A project waiver covers that project and the parent initiative when the
// task is stamped with the waived project. An initiative waiver covers that
// initiative and child project accounts whose task initiative stamp matches.
func WaiverCovers(w Waiver, account BudgetRef, task TaskRef) bool {
	if account.Scope.isPrincipal() || w.OwnerID == "" {
		return false
	}
	switch w.Scope {
	case ScopeProject:
		if account.Scope == ScopeProject && account.OwnerID == w.OwnerID {
			return true
		}
		return account.Scope == ScopeInitiative && task.ProjectID == w.OwnerID
	case ScopeInitiative:
		if account.Scope == ScopeInitiative && account.OwnerID == w.OwnerID {
			return true
		}
		return account.Scope == ScopeProject && task.InitiativeID == w.OwnerID
	default:
		return false
	}
}

// MonthWindow is the UTC calendar month containing t. End is the exclusive
// start of the next month.
func MonthWindow(t time.Time) (start, end time.Time) {
	utc := t.UTC()
	start = time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0)
	return start, end
}
