package i18n

import "testing"

// TestT verifies T returns the exact dictionary string per language
// (design D2, REQ-I18N-2). Real dictionary lookups — not smoke tests.
func TestT(t *testing.T) {
	tests := []struct {
		name string
		lang string
		key  string
		want string
	}{
		{"en dashboard", "en", "nav.dashboard", "Dashboard"},
		{"es dashboard", "es", "nav.dashboard", "Panel"},
		{"en expenses", "en", "nav.expenses", "Expenses"},
		{"es expenses", "es", "nav.expenses", "Gastos"},
		{"en logout", "en", "nav.logout", "Logout"},
		{"es logout", "es", "nav.logout", "Cerrar sesión"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := T(tt.lang, tt.key); got != tt.want {
				t.Errorf("T(%q, %q) = %q, want %q", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}

// TestTMissingKeyReturnsKey verifies fail-loud behavior: a missing key must
// return the key itself so translation gaps are visible (design D2).
func TestTMissingKeyReturnsKey(t *testing.T) {
	if got := T("en", "missing.key"); got != "missing.key" {
		t.Errorf("T(en, missing.key) = %q, want the key itself", got)
	}
	if got := T("es", "missing.key"); got != "missing.key" {
		t.Errorf("T(es, missing.key) = %q, want the key itself", got)
	}
}

// TestTUnknownLangFallsBackToEnglish verifies the design D1/D5 default:
// any language other than "es" resolves to the English dictionary.
func TestTUnknownLangFallsBackToEnglish(t *testing.T) {
	if got := T("fr", "nav.dashboard"); got != "Dashboard" {
		t.Errorf("T(fr, nav.dashboard) = %q, want English fallback", got)
	}
	if got := T("", "nav.dashboard"); got != "Dashboard" {
		t.Errorf("T(\"\", nav.dashboard) = %q, want English fallback", got)
	}
}

// TestTfInterpolatesArgs verifies Tf substitutes fmt-style arguments into the
// translated template (REQ-I18N-2).
func TestTfInterpolatesArgs(t *testing.T) {
	got := Tf("en", "validation.required", "Name")
	if want := "The Name field is required."; got != want {
		t.Errorf("Tf(en, validation.required, Name) = %q, want %q", got, want)
	}
	got = Tf("es", "validation.required", "Nombre")
	if want := "El campo Nombre es obligatorio."; got != want {
		t.Errorf("Tf(es, validation.required, Nombre) = %q, want %q", got, want)
	}
}

// TestTfNoArgsReturnsValueUnchanged verifies Tf with no args returns the
// dictionary value verbatim (no Sprintf noise).
func TestTfNoArgsReturnsValueUnchanged(t *testing.T) {
	if got := Tf("es", "nav.logout"); got != "Cerrar sesión" {
		t.Errorf("Tf(es, nav.logout) = %q, want %q", got, "Cerrar sesión")
	}
}

// TestTfMissingKeyReturnsKey verifies Tf is also fail-loud on missing keys.
func TestTfMissingKeyReturnsKey(t *testing.T) {
	if got := Tf("en", "missing.key", "x"); got != "missing.key" {
		t.Errorf("Tf(en, missing.key, x) = %q, want the key itself", got)
	}
}

// TestDictionariesCoverAllDesignKeys verifies both dictionaries contain every
// key from the design contract (error/auth/validation/expense/household/
// admin/nav/common/category/visibility). T returns the key itself for missing
// keys, so equality with the key proves presence.
func TestDictionariesCoverAllDesignKeys(t *testing.T) {
	keys := []string{
		"error.title", "error.home", "error.internal_server",
		"auth.invalid_credentials", "auth.email_registered", "auth.site_admin_required",
		"validation.required", "validation.min_length", "validation.max_length",
		"validation.positive", "validation.email", "validation.in",
		"expense.household_required", "expense.invalid_amount", "expense.invalid_date",
		"expense.not_found", "expense.permission", "expense.validation_failed",
		"household.name_required", "household.already_has", "household.invalid_code",
		"household.expired", "household.used", "household.not_admin",
		"household.no_household", "household.member_not_found", "household.role_forbidden",
		"household.self_role", "household.self_removal",
		"household.role.owner", "household.role.admin", "household.role.member",
		"admin.load_failed",
		"nav.dashboard", "nav.expenses", "nav.household", "nav.admin", "nav.logout",
		"common.create", "common.save", "common.cancel",
		"category.dining_out", "category.education", "category.entertainment",
		"category.groceries", "category.household", "category.insurance", "category.other",
		"category.personal_care", "category.rent", "category.savings",
		"category.subscriptions", "category.transportation", "category.utilities",
		"visibility.visible_editable", "visibility.visible_only", "visibility.hidden_private",
	}
	for _, lang := range []string{"en", "es"} {
		for _, key := range keys {
			if got := T(lang, key); got == key {
				t.Errorf("dictionary %q is missing key %q", lang, key)
			}
		}
	}
}

// TestExactSpanishStrings spot-checks full Spanish sentences (task 1.4:
// exact es strings asserted).
func TestExactSpanishStrings(t *testing.T) {
	tests := []struct{ key, want string }{
		{"error.internal_server", "Se produjo un error interno. Inténtalo de nuevo."},
		{"auth.invalid_credentials", "Correo o contraseña inválidos."},
		{"expense.invalid_amount", "Monto inválido."},
		{"household.self_removal", "No puedes eliminarte del hogar."},
		{"admin.load_failed", "No se pudo cargar el panel de administración."},
	}
	for _, tt := range tests {
		if got := T("es", tt.key); got != tt.want {
			t.Errorf("T(es, %q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
