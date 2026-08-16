package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/components"
)

func renderNav(email string, isAdmin bool) string {
	return renderNavFull(email, isAdmin, "en", "/", "csrf-token")
}

func renderNavFull(email string, isAdmin bool, lang string, activePath string, csrfToken string) string {
	buf := &bytes.Buffer{}
	err := components.Nav(email, isAdmin, lang, activePath, csrfToken).Render(context.Background(), buf)
	if err != nil {
		panic(err) // panic in helper, catch in test
	}
	return buf.String()
}

// Unauthenticated nav — no email

func TestNav_Unauthenticated_ShowsLoginLink(t *testing.T) {
	output := renderNav("", false)
	if !strings.Contains(output, "/login") {
		t.Error("expected /login link for unauthenticated nav")
	}
}

func TestNav_Unauthenticated_ShowsRegisterLink(t *testing.T) {
	output := renderNav("", false)
	if !strings.Contains(output, "/register") {
		t.Error("expected /register link for unauthenticated nav")
	}
}

func TestNav_Unauthenticated_HidesDashboard(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, "/dashboard") {
		t.Error("unauthenticated nav should NOT show /dashboard link")
	}
}

func TestNav_Unauthenticated_HidesExpenses(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, "/expenses") {
		t.Error("unauthenticated nav should NOT show /expenses link")
	}
}

func TestNav_Unauthenticated_HidesHousehold(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, "/household") {
		t.Error("unauthenticated nav should NOT show /household link")
	}
}

func TestNav_Unauthenticated_HidesLogout(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, "Logout") {
		t.Error("unauthenticated nav should NOT show Logout button")
	}
}

// Authenticated nav — with email

func TestNav_Authenticated_ShowsDashboardLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "/dashboard") {
		t.Error("expected /dashboard link for authenticated nav")
	}
}

func TestNav_Authenticated_ShowsExpensesLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "/expenses") {
		t.Error("expected /expenses link for authenticated nav")
	}
}

func TestNav_Authenticated_ShowsHouseholdLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "/household") {
		t.Error("expected /household link for authenticated nav")
	}
}

func TestNav_Authenticated_ShowsLogoutLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "Logout") {
		t.Error("expected Logout button for authenticated nav")
	}
}

func TestNav_Authenticated_HidesLoginLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if strings.Contains(output, "/login") {
		t.Error("authenticated nav should NOT show /login link")
	}
}

func TestNav_Authenticated_HidesRegisterLink(t *testing.T) {
	output := renderNav("test@example.com", false)
	if strings.Contains(output, "/register") {
		t.Error("authenticated nav should NOT show /register link")
	}
}

// Site admin nav — with email + isAdmin

func TestNav_Admin_ShowsAdminLink(t *testing.T) {
	output := renderNav("admin@example.com", true)
	if !strings.Contains(output, "/admin") {
		t.Error("expected /admin link for site admin nav")
	}
}

func TestNav_NonAdmin_HidesAdminLink(t *testing.T) {
	output := renderNav("user@example.com", false)
	if strings.Contains(output, "/admin") {
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

// --- NEW WU-4 TESTS: Drawer, Active link, CSRF, i18n ---

// Drawer: scrim element exists
func TestNav_Drawer_ScrimElementExists(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "id=\"nav-scrim\"") {
		t.Error("expected nav-scrim element in nav output")
	}
}

// Drawer: menu starts hidden
func TestNav_Drawer_MenuStartsHidden(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "id=\"nav-menu\"") {
		t.Error("expected nav-menu element")
	}
	// The menu should have the hidden class for mobile (hidden md:flex)
	if !strings.Contains(output, "hidden md:flex") {
		t.Error("expected nav-menu to start with hidden md:flex classes")
	}
}

// Drawer: JS has Escape key listener
func TestNav_Drawer_EscapeKeyListener(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "Escape") {
		t.Error("expected Escape key event listener in nav script")
	}
}

// Drawer: JS toggles scrim visibility
func TestNav_Drawer_ScrimToggle(t *testing.T) {
	output := renderNav("test@example.com", false)
	if !strings.Contains(output, "nav-scrim") {
		t.Error("expected nav-scrim references in JS toggle logic")
	}
}

// Active link: aria-current="page" present on matching link
func TestNav_ActiveLink_AriaCurrent(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/dashboard", "csrf-token")
	if !strings.Contains(output, "aria-current=\"page\"") {
		t.Error("expected aria-current=\"page\" on the active link for /dashboard")
	}
}

// Active link: non-active link does NOT get aria-current
func TestNav_ActiveLink_NonActiveNoAria(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/expenses", "csrf-token")
	// /dashboard link should NOT have aria-current="page"
	if strings.Count(output, "aria-current=\"page\"") != 1 {
		// Only /expenses should be active
		if strings.Contains(output, "href=\"/dashboard\"") && strings.Contains(output, "aria-current=\"page\"") {
			t.Error("dashboard link should NOT have aria-current when expenses is active")
		}
	}
}

// CSRF: logout form contains hidden csrf input
func TestNav_CSRF_LogoutFormHasCsrfInput(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "my-csrf-token")
	if !strings.Contains(output, "name=\"csrf\"") {
		t.Error("expected hidden csrf input in logout form")
	}
	if !strings.Contains(output, "value=\"my-csrf-token\"") {
		t.Error("expected csrf token value in logout form hidden input")
	}
}

// i18n: English nav labels
func TestNav_I18N_EnglishLabels(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "tok")
	if !strings.Contains(output, "Dashboard") {
		t.Error("expected 'Dashboard' label in English nav")
	}
	if !strings.Contains(output, "Expenses") {
		t.Error("expected 'Expenses' label in English nav")
	}
	if !strings.Contains(output, "Household") {
		t.Error("expected 'Household' label in English nav")
	}
	if !strings.Contains(output, "Logout") {
		t.Error("expected 'Logout' label in English nav")
	}
}

// i18n: Spanish nav labels
func TestNav_I18N_SpanishLabels(t *testing.T) {
	output := renderNavFull("test@example.com", false, "es", "/", "tok")
	// Spanish translations from es.go
	if !strings.Contains(output, "Panel") {
		t.Error("expected 'Panel' (Spanish for Dashboard) in nav")
	}
	if !strings.Contains(output, "Gastos") {
		t.Error("expected 'Gastos' (Spanish for Expenses) in nav")
	}
	if !strings.Contains(output, "Hogar") {
		t.Error("expected 'Hogar' (Spanish for Household) in nav")
	}
	if !strings.Contains(output, "Cerrar sesión") {
		t.Error("expected 'Cerrar sesión' (Spanish for Logout) in nav")
	}
}

// i18n: Admin link uses i18n for admin users
func TestNav_I18N_AdminLabel_Spanish(t *testing.T) {
	output := renderNavFull("admin@example.com", true, "es", "/", "tok")
	if !strings.Contains(output, "Administración") {
		t.Error("expected 'Administración' (Spanish for Admin) in nav")
	}
}

// --- WU-5: Language switcher forms ---

// Lang switcher: EN form present for authenticated users
func TestNav_LangSwitcher_EN_FormPresent(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "tok")
	if !strings.Contains(output, "action=\"/settings/lang\"") {
		t.Error("expected /settings/lang form action in authenticated nav")
	}
	if !strings.Contains(output, "name=\"lang\" value=\"en\"") {
		t.Error("expected hidden input with lang=en")
	}
	if !strings.Contains(output, ">EN</button>") {
		t.Error("expected EN button text")
	}
}

// Lang switcher: ES form present for authenticated users
func TestNav_LangSwitcher_ES_FormPresent(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "tok")
	if !strings.Contains(output, "name=\"lang\" value=\"es\"") {
		t.Error("expected hidden input with lang=es")
	}
	if !strings.Contains(output, ">ES</button>") {
		t.Error("expected ES button text")
	}
}

// Lang switcher: current lang is bold
func TestNav_LangSwitcher_CurrentLangBold(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "tok")
	if !strings.Contains(output, "font-bold") {
		t.Error("expected font-bold on current language button")
	}
}

// Lang switcher: non-current lang is grayed out
func TestNav_LangSwitcher_NonCurrentLangIsGrayed(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "tok")
	// The non-active language (ES) should have text-gray-400 class
	if !strings.Contains(output, "text-gray-400") {
		t.Error("expected text-gray-400 on non-active language button")
	}
}

// Lang switcher: CSRF token included in forms
func TestNav_LangSwitcher_CSRFTokenIncluded(t *testing.T) {
	output := renderNavFull("test@example.com", false, "en", "/", "my-csrf-token-123")
	if !strings.Contains(output, "value=\"my-csrf-token-123\"") {
		t.Error("expected CSRF token in lang switcher form")
	}
}

// Lang switcher: not shown for unauthenticated users
func TestNav_LangSwitcher_HiddenForUnauth(t *testing.T) {
	output := renderNav("", false)
	if strings.Contains(output, "/settings/lang") {
		t.Error("lang switcher should NOT be shown for unauthenticated users")
	}
}

// Lang switcher: Spanish active highlights ES
func TestNav_LangSwitcher_SpanishActiveHighlightsES(t *testing.T) {
	output := renderNavFull("test@example.com", false, "es", "/", "tok")
	if !strings.Contains(output, "font-bold") {
		t.Error("expected font-bold on ES button when lang=es")
	}
}
