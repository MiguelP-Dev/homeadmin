package repositories

import (
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
)

func setupHouseholdTestDB(t *testing.T) (*HouseholdRepositoryImpl, *UserRepositoryImpl) {
	t.Helper()
	db := setupTestDBRaw(t)
	return NewHouseholdRepository(db), NewUserRepository(db)
}

func TestHouseholdRepo_Create(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Test Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if hh.ID == 0 {
		t.Fatal("expected household ID to be set after Create")
	}
}

func TestHouseholdRepo_FindByID_Found(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Find Me"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := houseRepo.FindByID(hh.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected household to be found, got nil")
	}
	if found.Name != "Find Me" {
		t.Errorf("expected Name='Find Me', got %s", found.Name)
	}
}

func TestHouseholdRepo_FindByID_NotFound(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	found, err := houseRepo.FindByID(999)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got household with ID %d", found.ID)
	}
}

func TestHouseholdRepo_FindByUserID_Found(t *testing.T) {
	houseRepo, userRepo := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "User's Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	user := &database.User{
		Email:        "member@example.com",
		PasswordHash: "hash",
		Role:         "member",
		HouseholdID:  &hh.ID,
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	found, err := houseRepo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindByUserID returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected household to be found, got nil")
	}
	if found.Name != "User's Household" {
		t.Errorf("expected Name='User's Household', got %s", found.Name)
	}
}

func TestHouseholdRepo_FindByUserID_NoHousehold(t *testing.T) {
	houseRepo, userRepo := setupHouseholdTestDB(t)

	user := &database.User{
		Email:        "nohouse@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	found, err := houseRepo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindByUserID returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for user with no household, got %v", found.ID)
	}
}

func TestHouseholdRepo_Update(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Original Name"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	hh.Name = "Updated Name"
	if err := houseRepo.Update(hh); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	found, _ := houseRepo.FindByID(hh.ID)
	if found == nil {
		t.Fatal("expected household to be found after Update")
	}
	if found.Name != "Updated Name" {
		t.Errorf("expected Name='Updated Name', got %s", found.Name)
	}
}

func TestHouseholdRepo_Delete(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "To Delete"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := houseRepo.Delete(hh.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Soft-deleted household should not be found by default GORM scope
	found, err := houseRepo.FindByID(hh.ID)
	if err != nil {
		t.Fatalf("FindByID after Delete failed: %v", err)
	}
	if found != nil {
		t.Error("expected nil for soft-deleted household")
	}
}

func TestHouseholdRepo_RemoveMember(t *testing.T) {
	houseRepo, userRepo := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "House for Remove"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	user := &database.User{
		Email:        "removeme@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Add member first
	if err := houseRepo.AddMember(hh.ID, user.ID, "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Then remove
	if err := houseRepo.RemoveMember(hh.ID, user.ID); err != nil {
		t.Fatalf("RemoveMember returned unexpected error: %v", err)
	}

	// Verify user's HouseholdID is nil
	updated, err := userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after RemoveMember failed: %v", err)
	}
	if updated.HouseholdID != nil {
		t.Error("expected HouseholdID to be nil after RemoveMember")
	}
}

func TestHouseholdRepo_FindByName_Found(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Alpha Family"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := houseRepo.FindByName("Alpha Family")
	if err != nil {
		t.Fatalf("FindByName returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected household to be found, got nil")
	}
	if found.ID != hh.ID {
		t.Errorf("expected ID %d, got %d", hh.ID, found.ID)
	}
	if found.Name != "Alpha Family" {
		t.Errorf("expected Name='Alpha Family', got %s", found.Name)
	}
}

func TestHouseholdRepo_FindByName_NotFound(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	found, err := houseRepo.FindByName("Missing Family")
	if err != nil {
		t.Fatalf("FindByName returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got household with ID %d", found.ID)
	}
}

func TestHouseholdRepo_FindByInviteCode_Found(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)

	hh := &database.Household{Name: "Invite Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	invite := &database.InviteCode{
		Code:        "ABC12345",
		HouseholdID: hh.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(invite).Error; err != nil {
		t.Fatalf("seed InviteCode failed: %v", err)
	}

	found, err := houseRepo.FindByInviteCode("ABC12345")
	if err != nil {
		t.Fatalf("FindByInviteCode returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected invite code to be found, got nil")
	}
	if found.Code != "ABC12345" {
		t.Errorf("expected Code='ABC12345', got %s", found.Code)
	}
	if found.HouseholdID != hh.ID {
		t.Errorf("expected HouseholdID %d, got %d", hh.ID, found.HouseholdID)
	}
}

func TestHouseholdRepo_FindByInviteCode_NotFound(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	found, err := houseRepo.FindByInviteCode("ZZZZZZZZ")
	if err != nil {
		t.Fatalf("FindByInviteCode returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got invite with code %s", found.Code)
	}
}

func TestHouseholdRepo_CreateInviteCode_Persists(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)

	hh := &database.Household{Name: "Create Invite Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	invite := &database.InviteCode{
		Code:        "CODE0001",
		HouseholdID: hh.ID,
		ExpiresAt:   expiresAt,
	}
	if err := houseRepo.CreateInviteCode(invite); err != nil {
		t.Fatalf("CreateInviteCode returned unexpected error: %v", err)
	}
	if invite.ID == 0 {
		t.Fatal("expected invite code ID to be set after CreateInviteCode")
	}

	var persisted database.InviteCode
	if err := db.First(&persisted, invite.ID).Error; err != nil {
		t.Fatalf("failed to reload invite code: %v", err)
	}
	if persisted.Code != "CODE0001" {
		t.Errorf("expected persisted Code='CODE0001', got %s", persisted.Code)
	}
	if persisted.HouseholdID != hh.ID {
		t.Errorf("expected persisted HouseholdID %d, got %d", hh.ID, persisted.HouseholdID)
	}
	if !persisted.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected persisted ExpiresAt %v, got %v", expiresAt, persisted.ExpiresAt)
	}
}

func TestHouseholdRepo_CreateInviteCode_DuplicateCode(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)

	hh := &database.Household{Name: "Duplicate Invite Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	first := &database.InviteCode{
		Code:        "DUP00001",
		HouseholdID: hh.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := houseRepo.CreateInviteCode(first); err != nil {
		t.Fatalf("first CreateInviteCode failed: %v", err)
	}

	second := &database.InviteCode{
		Code:        "DUP00001",
		HouseholdID: hh.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := houseRepo.CreateInviteCode(second); err == nil {
		t.Fatal("expected unique-index error for duplicate code, got nil")
	}
}

func TestHouseholdRepo_MarkUsed(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)
	userRepo := NewUserRepository(db)

	hh := &database.Household{Name: "Mark Used Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	user := &database.User{
		Email:        "joiner@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	invite := &database.InviteCode{
		Code:        "MARK0001",
		HouseholdID: hh.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(invite).Error; err != nil {
		t.Fatalf("seed InviteCode failed: %v", err)
	}

	if err := houseRepo.MarkUsed(invite.ID, user.ID); err != nil {
		t.Fatalf("MarkUsed returned unexpected error: %v", err)
	}

	var persisted database.InviteCode
	if err := db.First(&persisted, invite.ID).Error; err != nil {
		t.Fatalf("failed to reload invite code: %v", err)
	}
	if persisted.UsedBy == nil {
		t.Fatal("expected UsedBy to be set after MarkUsed")
	}
	if *persisted.UsedBy != user.ID {
		t.Errorf("expected UsedBy %d, got %d", user.ID, *persisted.UsedBy)
	}
}

func TestHouseholdRepo_GetMembers_ReturnsMembers(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)
	userRepo := NewUserRepository(db)

	hh := &database.Household{Name: "Member Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	members := []*database.User{
		{Email: "admin@example.com", PasswordHash: "hash", Role: "admin", HouseholdID: &hh.ID},
		{Email: "member@example.com", PasswordHash: "hash", Role: "member", HouseholdID: &hh.ID},
	}
	for _, m := range members {
		if err := userRepo.Create(m); err != nil {
			t.Fatalf("Create user %s failed: %v", m.Email, err)
		}
	}

	found, err := houseRepo.GetMembers(hh.ID)
	if err != nil {
		t.Fatalf("GetMembers returned unexpected error: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 members, got %d", len(found))
	}

	roles := map[string]bool{}
	for _, m := range found {
		if m.HouseholdID == nil || *m.HouseholdID != hh.ID {
			t.Errorf("member %s has wrong HouseholdID %v", m.Email, m.HouseholdID)
		}
		roles[m.Role] = true
	}
	if !roles["admin"] {
		t.Error("expected a member with role 'admin'")
	}
	if !roles["member"] {
		t.Error("expected a member with role 'member'")
	}
}

func TestHouseholdRepo_GetMembers_Empty(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)

	hh := &database.Household{Name: "Empty Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	// Precondition: household exists but has zero members.
	found, err := houseRepo.GetMembers(hh.ID)
	if err != nil {
		t.Fatalf("GetMembers returned unexpected error: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("expected 0 members for household with no members, got %d", len(found))
	}
}

// TestHouseholdRepo_ListAllHouseholds_WithMemberCounts verifies ListAllHouseholds
// (RF-11 / design D9): every household is returned with its members preloaded,
// so the site-admin page can render member counts.
func TestHouseholdRepo_ListAllHouseholds_WithMemberCounts(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)
	userRepo := NewUserRepository(db)

	full := &database.Household{Name: "Full House"}
	empty := &database.Household{Name: "Empty House"}
	if err := houseRepo.Create(full); err != nil {
		t.Fatalf("Create full household failed: %v", err)
	}
	if err := houseRepo.Create(empty); err != nil {
		t.Fatalf("Create empty household failed: %v", err)
	}

	for _, email := range []string{"m1@example.com", "m2@example.com"} {
		user := &database.User{Email: email, PasswordHash: "hash", Role: database.RoleMember, HouseholdID: &full.ID}
		if err := userRepo.Create(user); err != nil {
			t.Fatalf("Create user %s failed: %v", email, err)
		}
	}

	all, err := houseRepo.ListAllHouseholds()
	if err != nil {
		t.Fatalf("ListAllHouseholds returned unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 households, got %d", len(all))
	}

	counts := map[string]int{}
	for _, h := range all {
		counts[h.Name] = len(h.Members)
	}
	if counts["Full House"] != 2 {
		t.Errorf("Full House member count = %d, want 2", counts["Full House"])
	}
	if counts["Empty House"] != 0 {
		t.Errorf("Empty House member count = %d, want 0", counts["Empty House"])
	}
}

// TestHouseholdRepo_ListAllHouseholds_Empty verifies the empty result on a fresh
// database (triangulation: no households exist).
func TestHouseholdRepo_ListAllHouseholds_Empty(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	all, err := houseRepo.ListAllHouseholds()
	if err != nil {
		t.Fatalf("ListAllHouseholds returned unexpected error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 households on empty db, got %d", len(all))
	}
}
