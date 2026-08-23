package services

import (
	"errors"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

// SiteAdminService provides site-wide administration operations (RF-11) and
// the cross-household overview behind site-admin global views.
type SiteAdminService struct {
	users     repositories.UserRepository
	household repositories.HouseholdRepository
	expenses  repositories.ExpenseRepository
	savings   SavingsRepository
}

// NewSiteAdminService creates a new SiteAdminService. The expense and savings
// repositories back SiteAdminOverview; passing nil for either is only valid
// when the overview is never called (e.g. PromoteAdmin-only usage).
func NewSiteAdminService(users repositories.UserRepository, household repositories.HouseholdRepository, expenses repositories.ExpenseRepository, savings SavingsRepository) *SiteAdminService {
	return &SiteAdminService{users: users, household: household, expenses: expenses, savings: savings}
}

// PromoteAdmin sets IsAdmin=true on the user identified by email.
// Returns ErrNotFound if no user matches the email.
func (s *SiteAdminService) PromoteAdmin(email string) error {
	user, err := s.users.FindByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	user.IsAdmin = true
	return s.users.Update(user)
}

// ListUsers returns all registered users.
func (s *SiteAdminService) ListUsers() ([]database.User, error) {
	return s.users.ListAllUsers()
}

// ListHouseholds returns all households with members eager-loaded.
func (s *SiteAdminService) ListHouseholds() ([]database.Household, error) {
	return s.household.ListAllHouseholds()
}

// HouseholdMember is the member projection shown in site-admin views.
type HouseholdMember struct {
	Email string
	Role  string
}

// HouseholdBlock groups one household with its members and every transaction
// in the site, plus precomputed aggregates for rendering.
type HouseholdBlock struct {
	Household      database.Household
	Members        []HouseholdMember
	Expenses       []repositories.ExpenseWithUser
	Savings        []repositories.SavingsWithUser
	MonthlyIncome  float64 // income transactions dated in the current month
	MonthlyExpense float64 // expense transactions dated in the current month
	SavingsTotal   float64 // sum of savings amounts (all time)
	AllTimeNet     float64 // all-time income minus expenses
}

// SiteAdminOverview returns every household as a block containing its members,
// all of its expenses and savings with owner emails, and monthly/all-time
// aggregates. Households are ordered by name; within a block, expenses are
// ordered by date descending and savings by creation date descending.
func (s *SiteAdminService) SiteAdminOverview() ([]HouseholdBlock, error) {
	if s.household == nil || s.expenses == nil || s.savings == nil {
		return nil, errors.New("site admin service missing repositories")
	}

	households, err := s.household.ListAllHouseholds()
	if err != nil {
		return nil, err
	}

	expenses, err := s.expenses.ListAllWithUsers(database.ExpenseFilters{})
	if err != nil {
		return nil, err
	}

	savings, err := s.savings.ListAllWithUsers()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	blocksByHH := make(map[uint]*HouseholdBlock, len(households))
	for i := range households {
		hh := &households[i]
		block := HouseholdBlock{Household: *hh}
		for _, m := range hh.Members {
			block.Members = append(block.Members, HouseholdMember{Email: m.Email, Role: m.Role})
		}
		blocksByHH[hh.ID] = &block
	}

	for _, e := range expenses {
		block, ok := blocksByHH[e.HouseholdID]
		if !ok {
			continue
		}
		block.Expenses = append(block.Expenses, e)
		if e.Type == database.TransactionTypeIncome {
			block.AllTimeNet += e.Amount
			if !e.Date.Before(monthStart) && e.Date.Before(monthEnd) {
				block.MonthlyIncome += e.Amount
			}
		} else {
			block.AllTimeNet -= e.Amount
			if !e.Date.Before(monthStart) && e.Date.Before(monthEnd) {
				block.MonthlyExpense += e.Amount
			}
		}
	}

	for _, sv := range savings {
		block, ok := blocksByHH[sv.HouseholdID]
		if !ok {
			continue
		}
		block.Savings = append(block.Savings, sv)
		block.SavingsTotal += sv.Amount
	}

	blocks := make([]HouseholdBlock, 0, len(households))
	for _, hh := range households {
		blocks = append(blocks, *blocksByHH[hh.ID])
	}
	return blocks, nil
}

// AdminSummary aggregates site-wide counts and totals for the /admin summary
// section, with one row per household for the households table.
type AdminSummary struct {
	Households   int
	Users        int
	Transactions int
	TotalIncome  float64 // all-time income across the site
	TotalSavings float64 // all-time savings across the site
	Rows         []AdminHouseholdRow
}

// AdminHouseholdRow is one household line in the /admin summary table.
type AdminHouseholdRow struct {
	ID         uint
	Name       string
	Members    int
	MonthlyNet float64 // MonthlyIncome - MonthlyExpense for the household
}

// BuildAdminSummary derives site totals from an overview's household blocks.
func BuildAdminSummary(blocks []HouseholdBlock) AdminSummary {
	summary := AdminSummary{Households: len(blocks)}
	for _, b := range blocks {
		summary.Users += len(b.Members)
		summary.Transactions += len(b.Expenses)
		summary.TotalSavings += b.SavingsTotal
		for _, e := range b.Expenses {
			if e.Type == database.TransactionTypeIncome {
				summary.TotalIncome += e.Amount
			}
		}
		summary.Rows = append(summary.Rows, AdminHouseholdRow{
			ID:         b.Household.ID,
			Name:       b.Household.Name,
			Members:    len(b.Members),
			MonthlyNet: b.MonthlyIncome - b.MonthlyExpense,
		})
	}
	return summary
}
