package codex

import "testing"

func TestValidateSafety(t *testing.T) {
	if err := ValidateSafety("normal task", "go test ./..."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateSafety("implement x", "go test ./... && rm -rf /tmp/1"); err == nil {
		t.Fatal("expected danger detection on test command")
	}
}
