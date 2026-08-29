package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/budgetpolicy"
	"github.com/multica-ai/multica/server/internal/costrate"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

var (
	ErrBudgetNotFound   = errors.New("budget not found")
	ErrWaiverNotFound   = errors.New("waiver not found")
	ErrInvalidScope     = errors.New("scope must be agent, squad, project, or initiative")
	ErrInvalidOverLimit = errors.New("over_limit must be pause or allow")
	ErrInvalidLimit     = errors.New("limit_usd_ticks must be greater than 0")
	ErrInvalidSoften    = errors.New("soften_at_percent must be between 1 and 100")
	ErrWaiverScope      = errors.New("waiver scope must be project or initiative")
	ErrWaiverWindow     = errors.New("waiver ends_at must be after starts_at")
	ErrWaiverOverlap    = errors.New("an overlapping waiver already exists for this owner")
)

const defaultSoftenAtPercent int16 = 80

// BudgetService owns budget tables and waivers. Admit, PostUsage, and
// Reconcile land in a later phase.
type BudgetService struct {
	Queries   *db.Queries
	TxStarter TxStarter
}

func NewBudgetService(q *db.Queries, tx TxStarter) *BudgetService {
	return &BudgetService{Queries: q, TxStarter: tx}
}

// SoftenPatch is the three-state soften field: omit, clear, or set.
type SoftenPatch struct {
	Set   bool
	Value *int16
}

// BudgetWrite is a full create/upsert body.
type BudgetWrite struct {
	WorkspaceID pgtype.UUID
	Scope       string
	OwnerID     pgtype.UUID
	LimitTicks  int64
	Soften      SoftenPatch
	OverLimit   string
	CreatedBy   pgtype.UUID
}

// BudgetPatch is a partial update. Soften.Set false leaves the column alone.
type BudgetPatch struct {
	LimitTicks *int64
	Soften     SoftenPatch
	OverLimit  *string
}

// WaiverWrite is a create body. The handler fills default window times.
type WaiverWrite struct {
	WorkspaceID pgtype.UUID
	Scope       string
	OwnerID     pgtype.UUID
	StartsAt    time.Time
	EndsAt      time.Time
	CreatedBy   pgtype.UUID
	Reason      *string
}

// BudgetView is one budget plus its current UTC-month period.
type BudgetView struct {
	Budget db.Budget
	Period *db.BudgetPeriod
	State  budgetpolicy.AccountState
}

// Create upserts by (workspace, scope, owner). A second write updates.
func (s *BudgetService) Create(ctx context.Context, in BudgetWrite) (BudgetView, bool, error) {
	if err := normalizeCreate(&in); err != nil {
		return BudgetView{}, false, err
	}
	existing, err := s.Queries.GetBudgetByScopeOwner(ctx, db.GetBudgetByScopeOwnerParams{
		WorkspaceID: in.WorkspaceID,
		Scope:       in.Scope,
		OwnerID:     in.OwnerID,
	})
	if err == nil {
		view, err := s.applyWrite(ctx, existing, in)
		return view, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return BudgetView{}, false, fmt.Errorf("get budget by owner: %w", err)
	}

	row, err := s.Queries.CreateBudget(ctx, db.CreateBudgetParams{
		WorkspaceID:     in.WorkspaceID,
		Scope:           in.Scope,
		OwnerID:         in.OwnerID,
		LimitUsdTicks:   in.LimitTicks,
		OverLimit:       in.OverLimit,
		CreatedBy:       in.CreatedBy,
		SoftenAtPercent: softenToInt2(in.Soften),
	})
	if err != nil {
		if isUniqueViolation(err) {
			existing, getErr := s.Queries.GetBudgetByScopeOwner(ctx, db.GetBudgetByScopeOwnerParams{
				WorkspaceID: in.WorkspaceID,
				Scope:       in.Scope,
				OwnerID:     in.OwnerID,
			})
			if getErr != nil {
				return BudgetView{}, false, fmt.Errorf("create budget: %w", err)
			}
			view, err := s.applyWrite(ctx, existing, in)
			return view, false, err
		}
		return BudgetView{}, false, fmt.Errorf("create budget: %w", err)
	}
	view, err := s.ensureCurrentPeriod(ctx, row, true)
	return view, true, err
}

func (s *BudgetService) applyWrite(ctx context.Context, existing db.Budget, in BudgetWrite) (BudgetView, error) {
	updated, err := s.Queries.UpdateBudget(ctx, db.UpdateBudgetParams{
		ID:                 existing.ID,
		WorkspaceID:        in.WorkspaceID,
		LimitUsdTicks:      pgtype.Int8{Int64: in.LimitTicks, Valid: true},
		SetSoftenAtPercent: pgtype.Bool{Bool: true, Valid: true},
		SoftenAtPercent:    softenToInt2(in.Soften),
		OverLimit:          pgtype.Text{String: in.OverLimit, Valid: true},
	})
	if err != nil {
		return BudgetView{}, fmt.Errorf("update budget: %w", err)
	}
	return s.ensureCurrentPeriod(ctx, updated, true)
}

func (s *BudgetService) Update(ctx context.Context, workspaceID, budgetID pgtype.UUID, patch BudgetPatch) (BudgetView, error) {
	if err := validatePatch(patch); err != nil {
		return BudgetView{}, err
	}
	existing, err := s.Queries.GetBudgetInWorkspace(ctx, db.GetBudgetInWorkspaceParams{
		ID:          budgetID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BudgetView{}, ErrBudgetNotFound
		}
		return BudgetView{}, fmt.Errorf("get budget: %w", err)
	}

	params := db.UpdateBudgetParams{
		ID:          existing.ID,
		WorkspaceID: workspaceID,
	}
	if patch.LimitTicks != nil {
		params.LimitUsdTicks = pgtype.Int8{Int64: *patch.LimitTicks, Valid: true}
	}
	if patch.Soften.Set {
		params.SetSoftenAtPercent = pgtype.Bool{Bool: true, Valid: true}
		params.SoftenAtPercent = softenToInt2(patch.Soften)
	}
	if patch.OverLimit != nil {
		params.OverLimit = pgtype.Text{String: *patch.OverLimit, Valid: true}
	}

	updated, err := s.Queries.UpdateBudget(ctx, params)
	if err != nil {
		return BudgetView{}, fmt.Errorf("update budget: %w", err)
	}
	return s.ensureCurrentPeriod(ctx, updated, false)
}

func (s *BudgetService) Delete(ctx context.Context, workspaceID, budgetID pgtype.UUID) error {
	if s.TxStarter == nil {
		return errors.New("budget delete requires a transaction starter")
	}
	_, err := s.Queries.GetBudgetInWorkspace(ctx, db.GetBudgetInWorkspaceParams{
		ID:          budgetID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBudgetNotFound
		}
		return fmt.Errorf("get budget: %w", err)
	}

	tx, err := s.TxStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.Queries.WithTx(tx)
	if err := qtx.DeleteBudgetDebits(ctx, budgetID); err != nil {
		return fmt.Errorf("delete budget debits: %w", err)
	}
	if err := qtx.DeleteBudgetPeriods(ctx, budgetID); err != nil {
		return fmt.Errorf("delete budget periods: %w", err)
	}
	if err := qtx.DeleteBudget(ctx, db.DeleteBudgetParams{ID: budgetID, WorkspaceID: workspaceID}); err != nil {
		return fmt.Errorf("delete budget: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete: %w", err)
	}
	return nil
}

func (s *BudgetService) Get(ctx context.Context, workspaceID, budgetID pgtype.UUID) (BudgetView, error) {
	row, err := s.Queries.GetBudgetInWorkspace(ctx, db.GetBudgetInWorkspaceParams{
		ID:          budgetID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BudgetView{}, ErrBudgetNotFound
		}
		return BudgetView{}, fmt.Errorf("get budget: %w", err)
	}
	return s.ensureCurrentPeriod(ctx, row, false)
}

func (s *BudgetService) List(ctx context.Context, workspaceID pgtype.UUID) ([]BudgetView, error) {
	rows, err := s.Queries.ListBudgets(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	out := make([]BudgetView, 0, len(rows))
	for _, row := range rows {
		view, err := s.ensureCurrentPeriod(ctx, row, false)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *BudgetService) CreateWaiver(ctx context.Context, in WaiverWrite) (db.BudgetWaiver, error) {
	if in.Scope != string(budgetpolicy.ScopeProject) && in.Scope != string(budgetpolicy.ScopeInitiative) {
		return db.BudgetWaiver{}, ErrWaiverScope
	}
	if !in.EndsAt.After(in.StartsAt) {
		return db.BudgetWaiver{}, ErrWaiverWindow
	}
	n, err := s.Queries.CountOverlappingBudgetWaivers(ctx, db.CountOverlappingBudgetWaiversParams{
		WorkspaceID: in.WorkspaceID,
		Scope:       in.Scope,
		OwnerID:     in.OwnerID,
		EndsAt:      timestamptz(in.EndsAt),
		StartsAt:    timestamptz(in.StartsAt),
	})
	if err != nil {
		return db.BudgetWaiver{}, fmt.Errorf("count overlapping waivers: %w", err)
	}
	if n > 0 {
		return db.BudgetWaiver{}, ErrWaiverOverlap
	}
	row, err := s.Queries.CreateBudgetWaiver(ctx, db.CreateBudgetWaiverParams{
		WorkspaceID: in.WorkspaceID,
		Scope:       in.Scope,
		OwnerID:     in.OwnerID,
		StartsAt:    timestamptz(in.StartsAt),
		EndsAt:      timestamptz(in.EndsAt),
		CreatedBy:   in.CreatedBy,
		Reason:      textPtr(in.Reason),
	})
	if err != nil {
		return db.BudgetWaiver{}, fmt.Errorf("create waiver: %w", err)
	}
	return row, nil
}

func (s *BudgetService) DeleteWaiver(ctx context.Context, workspaceID, waiverID pgtype.UUID) error {
	_, err := s.Queries.GetBudgetWaiverInWorkspace(ctx, db.GetBudgetWaiverInWorkspaceParams{
		ID:          waiverID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWaiverNotFound
		}
		return fmt.Errorf("get waiver: %w", err)
	}
	if err := s.Queries.DeleteBudgetWaiver(ctx, db.DeleteBudgetWaiverParams{
		ID:          waiverID,
		WorkspaceID: workspaceID,
	}); err != nil {
		return fmt.Errorf("delete waiver: %w", err)
	}
	return nil
}

func (s *BudgetService) ListWaivers(ctx context.Context, workspaceID pgtype.UUID) ([]db.BudgetWaiver, error) {
	rows, err := s.Queries.ListBudgetWaivers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list waivers: %w", err)
	}
	return rows, nil
}

func (s *BudgetService) ensureCurrentPeriod(ctx context.Context, budget db.Budget, backfill bool) (BudgetView, error) {
	start, end := budgetpolicy.MonthWindow(time.Now())
	period, err := s.Queries.GetBudgetPeriodByStart(ctx, db.GetBudgetPeriodByStartParams{
		BudgetID:    budget.ID,
		PeriodStart: timestamptz(start),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return BudgetView{}, fmt.Errorf("get budget period: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		period, err = s.Queries.CreateBudgetPeriod(ctx, db.CreateBudgetPeriodParams{
			BudgetID:    budget.ID,
			WorkspaceID: budget.WorkspaceID,
			PeriodStart: timestamptz(start),
			PeriodEnd:   timestamptz(end),
		})
		if err != nil {
			if !isUniqueViolation(err) {
				return BudgetView{}, fmt.Errorf("create budget period: %w", err)
			}
			period, err = s.Queries.GetBudgetPeriodByStart(ctx, db.GetBudgetPeriodByStartParams{
				BudgetID:    budget.ID,
				PeriodStart: timestamptz(start),
			})
			if err != nil {
				return BudgetView{}, fmt.Errorf("get budget period after race: %w", err)
			}
		}
		backfill = true
	}
	if backfill {
		if err := s.backfillPeriod(ctx, budget, period); err != nil {
			return BudgetView{}, err
		}
		period, err = s.Queries.RecalcBudgetPeriodTotals(ctx, period.ID)
		if err != nil {
			return BudgetView{}, fmt.Errorf("recalc period totals: %w", err)
		}
	}
	return s.view(ctx, budget, &period)
}

func (s *BudgetService) backfillPeriod(ctx context.Context, budget db.Budget, period db.BudgetPeriod) error {
	lines, err := s.Queries.ListBudgetUsageForBackfill(ctx, db.ListBudgetUsageForBackfillParams{
		WorkspaceID: budget.WorkspaceID,
		PeriodStart: period.PeriodStart,
		PeriodEnd:   period.PeriodEnd,
		Scope:       budget.Scope,
		OwnerID:     budget.OwnerID,
	})
	if err != nil {
		return fmt.Errorf("list usage for backfill: %w", err)
	}
	for _, line := range lines {
		ticks, pricedBy := costrate.PriceTicks(costrate.Line{
			Provider:         line.Provider,
			Model:            line.Model,
			CostUSDTicks:     int8Value(line.CostUsdTicks),
			InputTokens:      line.InputTokens,
			OutputTokens:     line.OutputTokens,
			CacheReadTokens:  line.CacheReadTokens,
			CacheWriteTokens: line.CacheWriteTokens,
		})
		if _, err := s.Queries.UpsertBudgetDebit(ctx, db.UpsertBudgetDebitParams{
			WorkspaceID:    budget.WorkspaceID,
			BudgetID:       budget.ID,
			BudgetPeriodID: period.ID,
			TaskID:         line.TaskID,
			Provider:       line.Provider,
			Model:          line.Model,
			AmountUsdTicks: ticks,
			PricedBy:       string(pricedBy),
		}); err != nil {
			return fmt.Errorf("upsert budget debit: %w", err)
		}
	}
	return nil
}

func (s *BudgetService) view(ctx context.Context, budget db.Budget, period *db.BudgetPeriod) (BudgetView, error) {
	account := accountOf(budget, period)
	if budget.Scope == string(budgetpolicy.ScopeSquad) {
		hasStamp, err := s.Queries.SquadHasOriginStamp(ctx, db.SquadHasOriginStampParams{
			WorkspaceID:         budget.WorkspaceID,
			BudgetOriginSquadID: budget.OwnerID,
		})
		if err != nil {
			return BudgetView{}, fmt.Errorf("squad origin stamp: %w", err)
		}
		account.Unattributed = !hasStamp
	}
	waived, err := s.resourceWaived(ctx, budget)
	if err != nil {
		return BudgetView{}, err
	}
	account.Waived = waived
	return BudgetView{
		Budget: budget,
		Period: period,
		State:  budgetpolicy.StateOf(account),
	}, nil
}

func (s *BudgetService) resourceWaived(ctx context.Context, budget db.Budget) (bool, error) {
	if budget.Scope != string(budgetpolicy.ScopeProject) && budget.Scope != string(budgetpolicy.ScopeInitiative) {
		return false, nil
	}
	now := time.Now()
	waivers, err := s.Queries.ListActiveBudgetWaivers(ctx, db.ListActiveBudgetWaiversParams{
		WorkspaceID: budget.WorkspaceID,
		StartsAt:    timestamptz(now),
	})
	if err != nil {
		return false, fmt.Errorf("list active waivers: %w", err)
	}
	account := budgetpolicy.BudgetRef{
		Scope:   budgetpolicy.Scope(budget.Scope),
		OwnerID: util.UUIDToString(budget.OwnerID),
	}
	task, err := s.syntheticTask(ctx, budget)
	if err != nil {
		return false, err
	}
	for _, row := range waivers {
		w := budgetpolicy.Waiver{
			Scope:   budgetpolicy.Scope(row.Scope),
			OwnerID: util.UUIDToString(row.OwnerID),
		}
		if budgetpolicy.WaiverCovers(w, account, task) {
			return true, nil
		}
	}
	return false, nil
}

func (s *BudgetService) syntheticTask(ctx context.Context, budget db.Budget) (budgetpolicy.TaskRef, error) {
	switch budget.Scope {
	case string(budgetpolicy.ScopeProject):
		task := budgetpolicy.TaskRef{ProjectID: util.UUIDToString(budget.OwnerID)}
		project, err := s.Queries.GetProjectInWorkspace(ctx, db.GetProjectInWorkspaceParams{
			ID:          budget.OwnerID,
			WorkspaceID: budget.WorkspaceID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return task, nil
			}
			return budgetpolicy.TaskRef{}, fmt.Errorf("load project for waiver: %w", err)
		}
		if project.InitiativeID.Valid {
			task.InitiativeID = util.UUIDToString(project.InitiativeID)
		}
		return task, nil
	case string(budgetpolicy.ScopeInitiative):
		return budgetpolicy.TaskRef{InitiativeID: util.UUIDToString(budget.OwnerID)}, nil
	default:
		return budgetpolicy.TaskRef{}, nil
	}
}

func accountOf(budget db.Budget, period *db.BudgetPeriod) budgetpolicy.Account {
	a := budgetpolicy.Account{
		Budget: budgetpolicy.BudgetRef{
			Scope:   budgetpolicy.Scope(budget.Scope),
			OwnerID: util.UUIDToString(budget.OwnerID),
		},
		LimitTicks: budget.LimitUsdTicks,
		OverLimit:  budgetpolicy.OverLimit(budget.OverLimit),
	}
	if budget.SoftenAtPercent.Valid {
		v := budget.SoftenAtPercent.Int16
		a.SoftenAt = &v
	}
	if period != nil {
		a.SpentTicks = period.SpentUsdTicks
		a.Unpriced = int(period.UnpricedLineCount)
	}
	return a
}

func normalizeCreate(in *BudgetWrite) error {
	if !in.Soften.Set {
		v := defaultSoftenAtPercent
		in.Soften = SoftenPatch{Set: true, Value: &v}
	}
	if !validScope(in.Scope) {
		return ErrInvalidScope
	}
	if in.OverLimit != string(budgetpolicy.OverLimitPause) && in.OverLimit != string(budgetpolicy.OverLimitAllow) {
		return ErrInvalidOverLimit
	}
	if in.LimitTicks <= 0 {
		return ErrInvalidLimit
	}
	return validateSoften(in.Soften)
}

func validatePatch(patch BudgetPatch) error {
	if patch.LimitTicks != nil && *patch.LimitTicks <= 0 {
		return ErrInvalidLimit
	}
	if patch.OverLimit != nil && *patch.OverLimit != string(budgetpolicy.OverLimitPause) && *patch.OverLimit != string(budgetpolicy.OverLimitAllow) {
		return ErrInvalidOverLimit
	}
	return validateSoften(patch.Soften)
}

func validateSoften(s SoftenPatch) error {
	if !s.Set || s.Value == nil {
		return nil
	}
	if *s.Value < 1 || *s.Value > 100 {
		return ErrInvalidSoften
	}
	return nil
}

func validScope(scope string) bool {
	switch budgetpolicy.Scope(scope) {
	case budgetpolicy.ScopeAgent, budgetpolicy.ScopeSquad, budgetpolicy.ScopeProject, budgetpolicy.ScopeInitiative:
		return true
	default:
		return false
	}
}

func softenToInt2(s SoftenPatch) pgtype.Int2 {
	if !s.Set || s.Value == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: *s.Value, Valid: true}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func textPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	trimmed := *s
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

func int8Value(v pgtype.Int8) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}
