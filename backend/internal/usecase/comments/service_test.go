package comments

import (
	"testing"

	"movies/backend/internal/domain"
)

func TestBuildTreePreservesArbitraryDepth(t *testing.T) {
	flat := []domain.Comment{
		{ID: 1, Body: "root"},
		{ID: 2, ParentID: 1, Body: "reply"},
		{ID: 3, ParentID: 2, Body: "reply to reply"},
		{ID: 4, ParentID: 3, Body: "fourth level"},
		{ID: 5, ParentID: 1, Body: "sibling reply"},
	}

	tree := buildTree(flat)
	if len(tree) != 1 {
		t.Fatalf("got %d roots, want 1", len(tree))
	}
	if len(tree[0].Replies) != 2 {
		t.Fatalf("got %d first-level replies, want 2", len(tree[0].Replies))
	}
	if got := tree[0].Replies[0].Replies; len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("second-level replies = %#v, want comment 3", got)
	}
	if got := tree[0].Replies[0].Replies[0].Replies; len(got) != 1 || got[0].ID != 4 {
		t.Fatalf("third-level replies = %#v, want comment 4", got)
	}
}

func TestBuildTreeKeepsOrphanAsRoot(t *testing.T) {
	tree := buildTree([]domain.Comment{{ID: 2, ParentID: 99, Body: "orphan"}})

	if len(tree) != 1 || tree[0].ID != 2 {
		t.Fatalf("tree = %#v, want orphan comment as root", tree)
	}
}
