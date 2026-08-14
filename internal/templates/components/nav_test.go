package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/components"
)

func renderNav(email string, isAdmin bool) string {
	buf := &bytes.Buffer{}
	err := components.Nav(email, isAdmin).Render(context.Background(), buf)
	if err != nil {
		panic(err) // panic in helper, catch in test
	}
	return buf.String()
}

// Unauthenticated nav — no email

func TestNav_Unauthenticated_ShowsLoginLink(t *testing.T) {
	output := renderNav("", false)
	if !strings.Contains(output, `/login`) {
		t.Error("expected /login link for unauthenticated nav")
	}
}

func TestNav_Unauthenticated_ShowsRegisterLink(t *testing.T) {
	output := renderNav("", false)
	if !strings.Contains(output, `/register`) {
		t.Error("expected /register link for unauthenticated nav")
	}
}

func TestNav_Unauthenticated_HidesDashboard(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, `/dashboard`) {
		t.Error("unauthenticated nav should NOT show /dashboard link")
	}
}

func TestNav_Unauthenticated_HidesExpenses(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, `/expenses`) {
		t.Error("unauthenticated nav should NOT show /expenses link")
	}
}

func TestNav_Unauthenticated_HidesHousehold(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, `/household`) {
		t.Error("unauthenticated nav should NOT show /household link")
	}
}

func TestNav_Unauthenticated_HidesLogout(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, `Logout`) {
		t.Error("unauthenticated nav should NOT show Logout button")
	}
}

// Authenticated nav — with email

func TestNav_Authenticated_ShowsDashboardLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, `/dashboard`) {
		t.Error("expected /dashboard link for authenticated nav")
	}
}

func TestNav_Authenticated_ShowsExpensesLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, `/expenses`) {
		t.Error("expected /expenses link for authenticated nav")
	}
}

func TestNav_Authenticated_ShowsHouseholdLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, `/household`) {
		t.Error("expected /household link for authenticated nav")
	}
}

func TestNav_Authenticated_ShowsLogoutLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, `Logout`) {
		t.Error("expected Logout button for authenticated nav")
	}
}

func TestNav_Authenticated_HidesLoginLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if strings.Contains(output, `/login`) {
		t.Error("authenticated nav should NOT show /login link")
	}
}

func TestNav_Authenticated_HidesRegisterLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if strings.Contains(output, `/register`) {
		t.Error("authenticated nav should NOT show /register link")
	}
}

// Site admin nav — with email + isAdmin

func TestNav_Admin_ShowsAdminLink(t *testing.T) {
	output := renderNav("admin@example.com", true)
	if !strings.Contains(output, `/admin`) {
		t.Error("expected /admin link for site admin nav")
	}
}

func TestNav_NonAdmin_HidesAdminLink(t *testing.T) {
	output := renderNav("user@example.com", false)
	if strings.Contains(output, `/admin`) {
		t.Error("non-admin nav should NOT show /admin link")
	}
}

// Triangulation: nav renders as a <nav> element for all states

func TestNav_RendersAsNavElement(t *testing.T) {
	output := renderNav("", false)
	if !strings.Contains(output, "<nav") {
		t.Error("expected <nav> element in output")
	}
	output2 := renderNav("test@example.com", false)
	if !strings.Contains(output2, "<nav") {
		t.Error("expected <nav> element in authenticated output")
	}
}

func TestNav_Authenticated_IncludesEmail(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "test@example.com") {
		t.Error("expected email to appear in authenticated nav")
	}
}
