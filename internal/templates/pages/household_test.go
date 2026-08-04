package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderHouseholdShow(hh *database.Household, members []database.User, isAdmin bool, csrfToken, inviteCode string) string {
	buf := &bytes.Buffer{}
	err := pages.HouseholdShow(hh, members, isAdmin, csrfToken, inviteCode).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func mustRenderHouseholdSetup(csrfToken, errorMsg string) string {
	buf := &bytes.Buffer{}
	err := pages.HouseholdSetup(csrfToken, errorMsg).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// --- HouseholdShow: household name ---

func TestHouseholdShow_RendersHouseholdName(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, true, "csrf-token", "")
	if !strings.Contains(output, "My Family") {
		t.Error("expected household name 'My Family' in output")
	}
}

// --- HouseholdShow: member rows with roles ---

func TestHouseholdShow_RendersMemberRowsWithRoles(t *testing.T) {
	hh := &database.Household{Name: "Roommates"}
	members := []database.User{
		{Email: "alice@example.com", Role: "admin"},
		{Email: "bob@example.com", Role: "member"},
	}
	output := mustRenderHouseholdShow(hh, members, true, "csrf-token", "")
	for _, want := range []string{"alice@example.com", "admin", "bob@example.com", "member"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in member rows", want)
		}
	}
}

// --- HouseholdShow: invite button (admin only) ---

func TestHouseholdShow_AdminSeesInviteButton(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, true, "csrf-token", "")
	if !strings.Contains(output, "/household/invite") {
		t.Error("expected invite form posting to /household/invite for admin")
	}
	if !strings.Contains(output, "Invite member") {
		t.Error("expected 'Invite member' button for admin")
	}
}

func TestHouseholdShow_MemberDoesNotSeeInviteButton(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "b@example.com", Role: "member"}}, false, "csrf-token", "")
	if strings.Contains(output, "/household/invite") {
		t.Error("member should not see the invite form")
	}
	if strings.Contains(output, "Invite member") {
		t.Error("member should not see the invite button")
	}
}

// --- HouseholdShow: optional invite code (refinement for PR4c) ---

func TestHouseholdShow_RendersInviteCodeWhenProvided(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, true, "csrf-token", "ABC12345")
	if !strings.Contains(output, "ABC12345") {
		t.Error("expected generated invite code 'ABC12345' to be visible")
	}
}

func TestHouseholdShow_OmitsInviteCodeWhenEmpty(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, true, "csrf-token", "")
	if strings.Contains(output, "ABC12345") {
		t.Error("invite code must not be rendered when empty")
	}
}

// --- HouseholdSetup: create form, join form, csrf ---

func TestHouseholdSetup_RendersCreateHouseholdForm(t *testing.T) {
	output := mustRenderHouseholdSetup("csrf-token", "")
	if !strings.Contains(output, "Create a household") {
		t.Error("expected 'Create a household' heading")
	}
	if !strings.Contains(output, `action="/household"`) {
		t.Error("expected create form posting to /household")
	}
	if !strings.Contains(output, `name="name"`) {
		t.Error("expected household name input in create form")
	}
}

func TestHouseholdSetup_RendersJoinForm(t *testing.T) {
	output := mustRenderHouseholdSetup("csrf-token", "")
	if !strings.Contains(output, `action="/household/join"`) {
		t.Error("expected join form posting to /household/join")
	}
	if !strings.Contains(output, `name="code"`) {
		t.Error("expected invite code input in join form")
	}
}

func TestHouseholdSetup_RendersCsrfHiddenInput(t *testing.T) {
	output := mustRenderHouseholdSetup("csrf-token-value", "")
	if !strings.Contains(output, `name="csrf"`) {
		t.Error("expected hidden csrf inputs in setup forms")
	}
	if !strings.Contains(output, "csrf-token-value") {
		t.Error("expected the csrf token value to be rendered in the hidden inputs")
	}
}

// --- HouseholdSetup: error message (triangulation) ---

func TestHouseholdSetup_RendersErrorMessageWhenPresent(t *testing.T) {
	output := mustRenderHouseholdSetup("csrf-token", "Household name is required")
	if !strings.Contains(output, "Household name is required") {
		t.Error("expected error message to be rendered when present")
	}
}

func TestHouseholdSetup_OmitsErrorMessageWhenEmpty(t *testing.T) {
	output := mustRenderHouseholdSetup("csrf-token", "")
	if strings.Contains(output, "Household name is required") {
		t.Error("error message must not be rendered when empty")
	}
}
