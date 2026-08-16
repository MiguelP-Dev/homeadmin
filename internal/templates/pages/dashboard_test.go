package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderDashboard(s *services.DashboardSummary, viewerRole string, lang string) string {
	buf := &bytes.Buffer{}
	err := pages.Dashboard(s, viewerRole, lang).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestDashboard_ShowsMonthYearHeading(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   0,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	currentMonth := time.Now().Format("January 2006")
	if !strings.Contains(output, currentMonth) {
		t.Errorf("expected month/year '%s' in dashboard heading", currentMonth)
	}
}

func TestDashboard_ShowsMonthlyTotal(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   1250.75,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "1,250.75") && !strings.Contains(output, "1250.75") {
		t.Error("expected monthly total 1250.75 in dashboard output")
	}
}

func TestDashboard_ZeroMonthlyTotal(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   0,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "0.00") {
		t.Error("expected 0.00 for zero monthly total")
	}
}

func TestDashboard_ShowsCategoryBreakdown(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal: 350.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "groceries", Total: 200.00},
			{Category: "rent", Total: 150.00},
		},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "Groceries") {
		t.Error("expected category 'Groceries' (translated) in breakdown table")
	}
	if !strings.Contains(output, "Rent") {
		t.Error("expected category 'Rent' (translated) in breakdown table")
	}
}

func TestDashboard_ShowsEmptyCategoryState(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   0,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "No expenses this month") {
		t.Error("expected 'No expenses this month' empty state when no categories")
	}
}

func TestDashboard_ShowsRecentExpenses(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal: 50.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "food", Total: 50.00},
		},
		RecentExpenses: []database.Expense{
			{ID: 1, Description: "Lunch", Amount: 25.00, Category: "food", Date: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)},
			{ID: 2, Description: "Dinner", Amount: 25.00, Category: "food", Date: time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)},
		},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "Lunch") {
		t.Error("expected 'Lunch' in recent expenses")
	}
	if !strings.Contains(output, "Dinner") {
		t.Error("expected 'Dinner' in recent expenses")
	}
}

func TestDashboard_ShowsEmptyRecentExpenses(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal: 0,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "food", Total: 0},
		},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "No recent expenses") {
		t.Error("expected 'No recent expenses' empty state when no recent expenses")
	}
}

func TestDashboard_HasLinkToExpenses(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   0,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "/expenses") {
		t.Error("expected link back to /expenses in dashboard")
	}
}

func TestDashboard_HasAddExpenseCTA(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   0,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, `href="/expenses/new"`) {
		t.Error("expected Add Expense CTA linking to /expenses/new in dashboard")
	}
}

func TestDashboard_ShowsCategoryTotalAmount(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal: 500.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "utilities", Total: 180.00},
		},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "180.00") {
		t.Error("expected 180.00 in category total for utilities")
	}
}

// Triangulation: different totals produce different output

func TestDashboard_Triangulation_DifferentMonthlyTotals(t *testing.T) {
	s1 := &services.DashboardSummary{MonthlyTotal: 100.00, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	s2 := &services.DashboardSummary{MonthlyTotal: 9999.99, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	out1 := mustRenderDashboard(s1, "member", "en")
	out2 := mustRenderDashboard(s2, "member", "en")
	if !strings.Contains(out1, "100.00") {
		t.Error("expected 100.00 in first dashboard")
	}
	if !strings.Contains(out2, "9,999.99") && !strings.Contains(out2, "9999.99") {
		t.Error("expected 9999.99 in second dashboard")
	}
	if strings.Contains(out1, "9999") {
		t.Error("first dashboard should not contain second dashboard's total")
	}
}

func TestDashboard_Triangulation_DifferentCategories(t *testing.T) {
	s1 := &services.DashboardSummary{
		MonthlyTotal: 100.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "groceries", Total: 100.00},
		},
		RecentExpenses: []database.Expense{},
	}
	s2 := &services.DashboardSummary{
		MonthlyTotal: 200.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "rent", Total: 200.00},
		},
		RecentExpenses: []database.Expense{},
	}
	out1 := mustRenderDashboard(s1, "member", "en")
	out2 := mustRenderDashboard(s2, "member", "en")
	if !strings.Contains(out1, "Groceries") {
		t.Error("first dashboard should show 'Groceries' category (translated)")
	}
	if !strings.Contains(out2, "Rent") {
		t.Error("second dashboard should show 'Rent' category (translated)")
	}
}

func TestDashboard_InviteCTA_VisibleForOwner(t *testing.T) {
	s := &services.DashboardSummary{MonthlyTotal: 0, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	output := mustRenderDashboard(s, database.RoleOwner, "en")
	if !strings.Contains(output, `/household`) {
		t.Error("expected Invite CTA for owner")
	}
}

func TestDashboard_InviteCTA_VisibleForAdmin(t *testing.T) {
	s := &services.DashboardSummary{MonthlyTotal: 0, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	output := mustRenderDashboard(s, database.RoleAdmin, "en")
	if !strings.Contains(output, `/household`) {
		t.Error("expected Invite CTA for admin")
	}
}

func TestDashboard_InviteCTA_HiddenForMember(t *testing.T) {
	s := &services.DashboardSummary{MonthlyTotal: 0, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	output := mustRenderDashboard(s, database.RoleMember, "en")
	if strings.Contains(output, `Invite member`) {
		t.Error("member should NOT see Invite CTA")
	}
}
