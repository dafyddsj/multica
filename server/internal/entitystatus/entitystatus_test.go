package entitystatus

import "testing"

func TestCanonicalOrder(t *testing.T) {
	got := Canonical()
	want := []string{"planned", "in_progress", "paused", "completed", "cancelled"}
	if len(got) != len(want) {
		t.Fatalf("Canonical() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Canonical()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsClosed(t *testing.T) {
	if !IsClosed(Completed) || !IsClosed(Cancelled) {
		t.Fatal("completed and cancelled must be closed")
	}
	if IsClosed(Planned) || IsClosed(InProgress) || IsClosed(Paused) {
		t.Fatal("active categories must not be closed")
	}
}

func TestValidateKey(t *testing.T) {
	if _, err := ValidateKey("planned"); err == nil {
		t.Fatal("built-in key must be rejected")
	}
	key, err := ValidateKey(" Shipping ")
	if err != nil {
		t.Fatalf("ValidateKey(Shipping) = %v", err)
	}
	if key != "shipping" {
		t.Fatalf("ValidateKey = %q, want shipping", key)
	}
	if _, err := ValidateKey("Bad Key!"); err == nil {
		t.Fatal("spaces and punctuation must be rejected")
	}
}

func TestSlugifyKey(t *testing.T) {
	key, err := SlugifyKey("Code Review")
	if err != nil {
		t.Fatalf("SlugifyKey = %v", err)
	}
	if key != "code_review" {
		t.Fatalf("SlugifyKey = %q, want code_review", key)
	}
}

func TestIsResourceType(t *testing.T) {
	if !IsResourceType(Initiative) || !IsResourceType(Project) {
		t.Fatal("initiative and project must be valid resource types")
	}
	if IsResourceType("issue") {
		t.Fatal("issue is not an entity_status resource type")
	}
}
