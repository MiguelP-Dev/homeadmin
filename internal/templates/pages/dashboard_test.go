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
	err := pages.Dashboard(s, viewerRole, lang, "test-csrf").Render(context.Background(), buf)
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
		TotalIncome:    2000.00,
		TotalExpenses:  749.25,
		Balance:        1250.75,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "Total Income") {
		t.Error("expected 'Total Income' label in dashboard")
	}
	if !strings.Contains(output, "Total Expenses") {
		t.Error("expected 'Total Expenses' label in dashboard")
	}
	if !strings.Contains(output, "Balance") {
		t.Error("expected 'Balance' label in dashboard")
	}
	if !strings.Contains(output, "2,000.00") {
		t.Error("expected income total 2000.00 in dashboard output")
	}
	if !strings.Contains(output, "749.25") {
		t.Error("expected expense total 749.25 in dashboard output")
	}
}

func TestDashboard_ZeroMonthlyTotal(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:   0,
		TotalIncome:    0,
		TotalExpenses:  0,
		Balance:        0,
		CategoryTotals: []repositories.CategoryTotal{},
		RecentExpenses: []database.Expense{},
	}
	output := mustRenderDashboard(s, "member", "en")
	if !strings.Contains(output, "0.00") {
		t.Error("expected 0.00 for zero totals")
	}
}

func TestDashboard_ShowsCategoryBreakdown(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal:  350.00,
		TotalIncome:   500.00,
		TotalExpenses: 150.00,
		Balance:       350.00,
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
		TotalIncome:    0,
		TotalExpenses:  0,
		Balance:        0,
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
		MonthlyTotal:  50.00,
		TotalIncome:   100.00,
		TotalExpenses: 50.00,
		Balance:       50.00,
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
		MonthlyTotal:  0,
		TotalIncome:   0,
		TotalExpenses: 0,
		Balance:       0,
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
		TotalIncome:    0,
		TotalExpenses:  0,
		Balance:        0,
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
		TotalIncome:    0,
		TotalExpenses:  0,
		Balance:        0,
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
		MonthlyTotal:  500.00,
		TotalIncome:   680.00,
		TotalExpenses: 180.00,
		Balance:       500.00,
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
	s1 := &services.DashboardSummary{MonthlyTotal: 100.00, TotalIncome: 500.00, TotalExpenses: 400.00, Balance: 100.00, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	s2 := &services.DashboardSummary{MonthlyTotal: 9999.99, TotalIncome: 20000.00, TotalExpenses: 10000.01, Balance: 9999.99, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
	out1 := mustRenderDashboard(s1, "member", "en")
	out2 := mustRenderDashboard(s2, "member", "en")
	if !strings.Contains(out1, "500.00") {
		t.Error("expected 500.00 income in first dashboard")
	}
	if !strings.Contains(out2, "20,000.00") && !strings.Contains(out2, "20000.00") {
		t.Error("expected 20000.00 income in second dashboard")
	}
	if strings.Contains(out1, "20,000") && strings.Contains(out1, "20000") {
		t.Error("first dashboard should not contain second dashboard's total")
	}
}

func TestDashboard_Triangulation_DifferentCategories(t *testing.T) {
	s1 := &services.DashboardSummary{
		MonthlyTotal:  100.00,
		TotalIncome:   200.00,
		TotalExpenses: 100.00,
		Balance:       100.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "groceries", Total: 100.00},
		},
		RecentExpenses: []database.Expense{},
	}
	s2 := &services.DashboardSummary{
		MonthlyTotal:  200.00,
		TotalIncome:   400.00,
		TotalExpenses: 200.00,
		Balance:       200.00,
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

func TestDashboard_TranslatedStrings(t *testing.T) {
	tests := []struct {
		lang string
		want []string
	}{
		{"en", []string{
			"Add Expense", "Invite member", "No recent expenses",
			"Category Breakdown", "No expenses this month",
			"Back to Expenses", "Dashboard",
		}},
		{"es", []string{
			"Agregar gasto", "Invitar miembro", "Sin gastos recientes",
			"Desglose por categoría", "Sin gastos este mes",
			"Volver a gastos", "Panel",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			s := &services.DashboardSummary{MonthlyTotal: 0, CategoryTotals: []repositories.CategoryTotal{}, RecentExpenses: []database.Expense{}}
			output := mustRenderDashboard(s, database.RoleOwner, tt.lang)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected %q in dashboard output", tt.lang, want)
				}
			}
		})
	}
}

func TestDashboard_TranslatedBreakdownHeaders(t *testing.T) {
	s := &services.DashboardSummary{
		MonthlyTotal: 200.00,
		CategoryTotals: []repositories.CategoryTotal{
			{Category: "groceries", Total: 200.00},
		},
		RecentExpenses: []database.Expense{},
	}
	enOutput := mustRenderDashboard(s, database.RoleOwner, "en")
	esOutput := mustRenderDashboard(s, database.RoleOwner, "es")
	if !strings.Contains(enOutput, ">Category</th>") || !strings.Contains(enOutput, ">Total</th>") {
		t.Error("expected 'Category' and 'Total' breakdown headers in en output")
	}
	if !strings.Contains(esOutput, ">Categoría</th>") || !strings.Contains(esOutput, ">Total</th>") {
		t.Error("expected 'Categoría' and 'Total' breakdown headers in es output")
	}
}
