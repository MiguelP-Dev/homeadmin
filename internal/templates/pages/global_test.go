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

func mustRenderGlobal(fn func(context.Context, *bytes.Buffer) error) string {
	buf := &bytes.Buffer{}
	if err := fn(context.Background(), buf); err != nil {
		panic(err)
	}
	return buf.String()
}

func globalFixture() []services.HouseholdBlock {
	return []services.HouseholdBlock{
		{
			Household: database.Household{ID: 1, Name: "Alpha Home"},
			Members: []services.HouseholdMember{
				{Email: "alpha-owner@test.com", Role: "owner"},
				{Email: "alpha-member@test.com", Role: "member"},
			},
			Expenses: []repositories.ExpenseWithUser{{
				Expense: database.Expense{
					ID: 1, Amount: 1200, Description: "Alpha Rent", Category: "rent",
					Type: database.TransactionTypeExpense,
					Date: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
				},
				OwnerEmail: "alpha-owner@test.com",
			}},
			Savings: []repositories.SavingsWithUser{{
				Savings:    database.Savings{ID: 1, Description: "Alpha Fund", Amount: 300},
				OwnerEmail: "beta-owner@test.com",
			}},
			MonthlyIncome: 500, MonthlyExpense: 1200, AllTimeNet: -700, SavingsTotal: 300,
		},
	}
}

func TestDashboardGlobal_ShowsHouseholdsAndOwners(t *testing.T) {
	output := mustRenderGlobal(func(ctx context.Context, buf *bytes.Buffer) error {
		return pages.DashboardGlobal(globalFixture(), "en").Render(ctx, buf)
	})
	for _, want := range []string{"All Households", "Alpha Home", "alpha-owner@test.com", "Alpha Rent"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in dashboard global output", want)
		}
	}
}

func TestExpensesGlobal_ShowsOwnerColumnAndSavings(t *testing.T) {
	output := mustRenderGlobal(func(ctx context.Context, buf *bytes.Buffer) error {
		return pages.ExpensesGlobal(globalFixture(), "en").Render(ctx, buf)
	})
	for _, want := range []string{"Owner", "alpha-owner@test.com", "beta-owner@test.com", "Alpha Fund"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in expenses global output", want)
		}
	}
}

func TestSavingsGlobal_ShowsHouseholdBlocks(t *testing.T) {
	output := mustRenderGlobal(func(ctx context.Context, buf *bytes.Buffer) error {
		return pages.SavingsGlobal(globalFixture(), "en").Render(ctx, buf)
	})
	if !strings.Contains(output, "Alpha Home") || !strings.Contains(output, "Alpha Fund") {
		t.Error("expected household name and savings entry in savings global output")
	}
}

func TestGlobalViews_EmptyState(t *testing.T) {
	output := mustRenderGlobal(func(ctx context.Context, buf *bytes.Buffer) error {
		return pages.DashboardGlobal(nil, "en").Render(ctx, buf)
	})
	if !strings.Contains(output, "No households yet") {
		t.Error("expected empty state in global view with no blocks")
	}
}

func TestGlobalViews_SpanishLabels(t *testing.T) {
	output := mustRenderGlobal(func(ctx context.Context, buf *bytes.Buffer) error {
		return pages.ExpensesGlobal(globalFixture(), "es").Render(ctx, buf)
	})
	for _, want := range []string{"Todos los hogares", "Propietario"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected Spanish label %q in expenses global output", want)
		}
	}
}

func mustRenderAdmin(summary *services.AdminSummary, lang string) string {
	users := []database.User{{ID: 1, Email: "owner@x.com", Role: "owner", IsAdmin: true}}
	buf := &bytes.Buffer{}
	if err := pages.Admin(users, summary, lang).Render(context.Background(), buf); err != nil {
		panic(err)
	}
	return buf.String()
}

func TestAdmin_ShowsSummaryCards(t *testing.T) {
	output := mustRenderAdmin(&services.AdminSummary{
		Households: 2, Users: 3, Transactions: 7,
		TotalIncome: 1400, TotalSavings: 300,
		Rows: []services.AdminHouseholdRow{
			{ID: 1, Name: "Alpha Home", Members: 2, MonthlyNet: -700},
			{ID: 2, Name: "Beta Home", Members: 1, MonthlyNet: 900},
		},
	}, "en")
	for _, want := range []string{"Site Summary", "Users", "$1,400.00", "$300.00", "Alpha Home", "Beta Home"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in admin output", want)
		}
	}
}

func TestAdmin_HouseholdRowsHaveOpenLinks(t *testing.T) {
	output := mustRenderAdmin(&services.AdminSummary{
		Rows: []services.AdminHouseholdRow{{ID: 1, Name: "Alpha Home", Members: 2, MonthlyNet: 100}},
	}, "en")
	for _, link := range []string{`href="/dashboard"`, `href="/expenses"`, `href="/savings"`} {
		if !strings.Contains(output, link) {
			t.Errorf("expected %q link in admin households table", link)
		}
	}
}

func TestAdmin_EmptySiteShowsEmptyState(t *testing.T) {
	output := mustRenderAdmin(&services.AdminSummary{}, "en")
	if !strings.Contains(output, "No households yet") {
		t.Error("expected households empty state in admin output")
	}
}

func TestAdmin_TranslatedStrings(t *testing.T) {
	summary := &services.AdminSummary{}
	render := func(users []database.User, lang string) string {
		buf := &bytes.Buffer{}
		if err := pages.Admin(users, summary, lang).Render(context.Background(), buf); err != nil {
			panic(err)
		}
		return buf.String()
	}
	tests := []struct {
		name string
		lang string
		want []string
	}{
		{"english", "en", []string{"Email", "Is Admin", "No users"}},
		{"spanish", "es", []string{"Correo", "¿Es administrador?", "No hay usuarios"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := render(nil, tt.lang)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected %q in admin output", tt.lang, want)
				}
			}
		})
	}

	withUser := render([]database.User{{ID: 1, Email: "u@x.com", Role: "owner", IsAdmin: true}}, "es")
	for _, want := range []string{"Propietario", "Sí"} {
		if !strings.Contains(withUser, want) {
			t.Errorf("expected %q in spanish admin output", want)
		}
	}
}
