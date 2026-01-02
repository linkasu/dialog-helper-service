package storage

import (
	"context"
	"testing"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/models"
)

func TestMemoryStoreConfig(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	_, ok, err := store.GetConfig(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected empty config")
	}

	if err := store.SetConfig(ctx, "foo", "bar"); err != nil {
		t.Fatalf("set config: %v", err)
	}
	item, ok, err := store.GetConfig(ctx, "foo")
	if err != nil || !ok {
		t.Fatalf("expected config item")
	}
	if item.Value != "bar" {
		t.Fatalf("unexpected value: %q", item.Value)
	}
}

func TestMemoryStoreResult(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	record := ResultRecord{
		DialogHash: "d1",
		UserHash:   "u1",
		Transcript: "hello",
		Candidates: []models.AnswerCandidate{{Text: "ok", Rank: 1, Length: 2}},
		CreatedAt:  time.Now().UTC(),
	}
	if err := store.PutResult(ctx, record); err != nil {
		t.Fatalf("put result: %v", err)
	}
	got, ok, err := store.GetResult(ctx, "d1")
	if err != nil || !ok {
		t.Fatalf("expected result")
	}
	if got.Transcript != "hello" {
		t.Fatalf("unexpected transcript")
	}
}

func TestMemoryStoreAPIKey(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	record := APIKeyRecord{
		APIKeyHash: "hash",
		ClientID:   "client",
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	}
	if err := store.CreateAPIKey(ctx, record); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	got, ok, err := store.GetAPIKey(ctx, "hash")
	if err != nil || !ok {
		t.Fatalf("expected api key")
	}
	if got.ClientID != "client" {
		t.Fatalf("unexpected client id")
	}
}
