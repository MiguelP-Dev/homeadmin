package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderHouseholdShow(hh *database.Household, members []database.User, viewerRole, csrfToken, inviteCode string) string {
	buf := &bytes.Buffer{}
	err := pages.HouseholdShow(hh, members, viewerRole, csrfToken, inviteCode).Render(context.Background(), buf)
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
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, database.RoleAdmin, "csrf-token", "")
	if !strings.Contains(output, "My Family") {
		t.Error("expected household name 'My Family' in output")
	}
}

// --- HouseholdShow: Add Expense CTA (all members) ---

func TestHouseholdShow_HasAddExpenseCTA(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, database.RoleAdmin, "csrf-token", "")
	if !strings.Contains(output, `href="/expenses/new"`) {
		t.Error("expected Add Expense CTA linking to /expenses/new in household page")
	}
}

// --- HouseholdShow: member rows with roles ---

func TestHouseholdShow_RendersMemberRowsWithRoles(t *testing.T) {
	hh := &database.Household{Name: "Roommates"}
	members := []database.User{
		{Email: "alice@example.com", Role: "admin"},
		{Email: "bob@example.com", Role: "member"},
	}
	output := mustRenderHouseholdShow(hh, members, database.RoleOwner, "csrf-token", "")
	for _, want := range []string{"alice@example.com", "admin", "bob@example.com", "member"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in member rows", want)
		}
	}
}

// --- HouseholdShow: invite button (admin only) ---

func TestHouseholdShow_AdminSeesInviteButton(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, database.RoleAdmin, "csrf-token", "")
	if !strings.Contains(output, "/household/invite") {
		t.Error("expected invite form posting to /household/invite for admin")
	}
	if !strings.Contains(output, "Invite member") {
		t.Error("expected 'Invite member' button for admin")
	}
}

func TestHouseholdShow_MemberDoesNotSeeInviteButton(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "b@example.com", Role: "member"}}, database.RoleMember, "csrf-token", "")
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
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, database.RoleAdmin, "csrf-token", "ABC12345")
	if !strings.Contains(output, "ABC12345") {
		t.Error("expected generated invite code 'ABC12345' to be visible")
	}
}

func TestHouseholdShow_OmitsInviteCodeWhenEmpty(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	output := mustRenderHouseholdShow(hh, []database.User{{Email: "a@example.com", Role: "admin"}}, database.RoleAdmin, "csrf-token", "")
	if strings.Contains(output, "ABC12345") {
		t.Error("invite code must not be rendered when empty")
	}
}

// --- HouseholdShow: role-change forms (owner only, RF-8/RF-13) ---

func TestHouseholdShow_OwnerSeesRoleChangeForms(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	members := []database.User{
		{ID: 1, Email: "owner@example.com", Role: database.RoleOwner},
		{ID: 2, Email: "admin@example.com", Role: database.RoleAdmin},
		{ID: 3, Email: "member@example.com", Role: database.RoleMember},
	}
	output := mustRenderHouseholdShow(hh, members, database.RoleOwner, "csrf-token", "")
	if !strings.Contains(output, `action="/household/members/2/role"`) {
		t.Error("expected role-change form for admin member posting to /household/members/2/role")
	}
	if !strings.Contains(output, `action="/household/members/3/role"`) {
		t.Error("expected role-change form for member posting to /household/members/3/role")
	}
	if !strings.Contains(output, `name="role"`) {
		t.Error("expected role select with name=role in the change form")
	}
	if !strings.Contains(output, `name="csrf"`) {
		t.Error("expected csrf hidden input in the role-change form")
	}
}

func TestHouseholdShow_OwnerGetsNoRoleFormForOwnerMember(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	members := []database.User{
		{ID: 1, Email: "owner@example.com", Role: database.RoleOwner},
		{ID: 2, Email: "member@example.com", Role: database.RoleMember},
	}
	output := mustRenderHouseholdShow(hh, members, database.RoleOwner, "csrf-token", "")
	if strings.Contains(output, `action="/household/members/1/role"`) {
		t.Error("owner must not get a role-change form for another owner")
	}
}

func TestHouseholdShow_AdminSeesInviteButNoRoleControls(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	members := []database.User{
		{ID: 1, Email: "owner@example.com", Role: database.RoleOwner},
		{ID: 2, Email: "admin@example.com", Role: database.RoleAdmin},
		{ID: 3, Email: "member@example.com", Role: database.RoleMember},
	}
	output := mustRenderHouseholdShow(hh, members, database.RoleAdmin, "csrf-token", "")
	if !strings.Contains(output, "/household/invite") {
		t.Error("admin should still see the invite CTA")
	}
	if strings.Contains(output, "/household/members/") {
		t.Error("admin must not see role-change forms (owner-only)")
	}
}

func TestHouseholdShow_MemberSeesNeitherInviteNorRoleControls(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	members := []database.User{
		{ID: 1, Email: "owner@example.com", Role: database.RoleOwner},
		{ID: 3, Email: "member@example.com", Role: database.RoleMember},
	}
	output := mustRenderHouseholdShow(hh, members, database.RoleMember, "csrf-token", "")
	if strings.Contains(output, "/household/invite") {
		t.Error("member should not see the invite form")
	}
	if strings.Contains(output, "/household/members/") {
		t.Error("member should not see role-change forms")
	}
}

// --- HouseholdShow: viewer role rendered ---

func TestHouseholdShow_RendersViewerRole(t *testing.T) {
	hh := &database.Household{Name: "My Family"}
	members := []database.User{{ID: 1, Email: "owner@example.com", Role: database.RoleOwner}}
	output := mustRenderHouseholdShow(hh, members, database.RoleOwner, "csrf-token", "")
	if !strings.Contains(output, "Your role: owner") {
		t.Error("expected the viewer's role to be rendered")
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
