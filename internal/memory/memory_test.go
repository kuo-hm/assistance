package memory

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStorePersistsTurnsAndSummary(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "assistant.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if err := store.SaveTurn(ctx, "s1", "user", "en-US", "remember that I like short answers"); err != nil {
		t.Fatalf("SaveTurn user error = %v", err)
	}
	if err := store.SaveTurn(ctx, "s1", "assistant", "en-US", "I will keep answers short."); err != nil {
		t.Fatalf("SaveTurn assistant error = %v", err)
	}
	if err := store.UpdateSummary(ctx, "s1"); err != nil {
		t.Fatalf("UpdateSummary error = %v", err)
	}

	mem, err := store.LoadContext(ctx, "next")
	if err != nil {
		t.Fatalf("LoadContext error = %v", err)
	}
	if mem.Summary == "" {
		t.Fatal("expected summary")
	}
	if len(mem.Recent) != 2 {
		t.Fatalf("recent turn count = %d", len(mem.Recent))
	}
	if mem.Recent[0].Role != "user" {
		t.Fatalf("first recent role = %q", mem.Recent[0].Role)
	}
}
