package watchlist

import (
	"context"
	"testing"
	"time"

	"movies/backend/internal/domain"
)

type fakeProvider struct{ title domain.Title }

func (f fakeProvider) Get(context.Context, domain.MediaType, int64) (domain.Title, error) {
	return f.title, nil
}

type fakeRepository struct {
	user           domain.User
	visible        bool
	titleID        int64
	titleExists    bool
	addAllowed     bool
	items          []domain.WatchlistItem
	count          int64
	refs           []domain.TitleRef
	friendIDs      []int64
	matches        []domain.WatchlistMatchItem
	resolvedUUIDs  []string
	matchFriendIDs []int64
	matchCursor    MatchesCursor
	matchLimit     int
}

func (f *fakeRepository) GetUserByUUID(context.Context, string) (domain.User, bool, error) {
	return f.user, f.user.ID != 0, nil
}
func (f *fakeRepository) UserCanSeeWatchlist(context.Context, int64, int64) (bool, error) {
	return f.visible, nil
}
func (f *fakeRepository) GetTitleID(context.Context, domain.MediaType, int64) (int64, bool, error) {
	return f.titleID, f.titleExists, nil
}
func (f *fakeRepository) Add(context.Context, int64, domain.Title) (bool, error) {
	return f.addAllowed, nil
}
func (f *fakeRepository) Remove(context.Context, int64, domain.MediaType, int64) error { return nil }
func (f *fakeRepository) List(context.Context, int64, Cursor, int) ([]domain.WatchlistItem, error) {
	return f.items, nil
}
func (f *fakeRepository) Count(context.Context, int64) (int64, error) { return f.count, nil }
func (f *fakeRepository) IsInWatchlist(context.Context, int64, domain.MediaType, int64) (bool, error) {
	return true, nil
}
func (f *fakeRepository) ListRefs(context.Context, int64) ([]domain.TitleRef, error) {
	return f.refs, nil
}
func (f *fakeRepository) ResolveAcceptedFriendIDs(_ context.Context, _ int64, friendUUIDs []string) ([]int64, error) {
	f.resolvedUUIDs = append([]string(nil), friendUUIDs...)
	return f.friendIDs, nil
}
func (f *fakeRepository) ListMatches(_ context.Context, _ int64, friendIDs []int64, cursor MatchesCursor, limit int) ([]domain.WatchlistMatchItem, error) {
	f.matchFriendIDs = append([]int64(nil), friendIDs...)
	f.matchCursor = cursor
	f.matchLimit = limit
	return f.matches, nil
}

func TestAddRejectsRatedTitle(t *testing.T) {
	repo := &fakeRepository{titleID: 10, titleExists: true, addAllowed: false}
	service := NewService(repo, fakeProvider{})
	if err := service.Add(context.Background(), 1, domain.MediaTypeMovie, 603); err != ErrConflict {
		t.Fatalf("Add() error = %v, want ErrConflict", err)
	}
}

func TestListByUUIDPaginatesAndHidesPrivateList(t *testing.T) {
	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		user: domain.User{ID: 2}, visible: true, count: 3,
		items: []domain.WatchlistItem{
			{Title: domain.Title{ID: 11}, AddedAt: now},
			{Title: domain.Title{ID: 10}, AddedAt: now.Add(-time.Hour)},
			{Title: domain.Title{ID: 9}, AddedAt: now.Add(-2 * time.Hour)},
		},
	}
	service := NewService(repo, fakeProvider{})
	page, err := service.ListByUUID(context.Background(), 1, "uuid", "", 2)
	if err != nil || len(page.Items) != 2 || page.NextCursor == "" || page.TotalCount != 3 {
		t.Fatalf("unexpected page=%#v err=%v", page, err)
	}
	repo.visible = false
	page, err = service.ListByUUID(context.Background(), 1, "uuid", "", 2)
	if err != nil || len(page.Items) != 0 || page.TotalCount != 0 {
		t.Fatalf("private page leaked data: %#v err=%v", page, err)
	}
}

func TestStatusesUsesSingleStoredRefSet(t *testing.T) {
	wanted := domain.TitleRef{TmdbID: 603, MediaType: domain.MediaTypeMovie}
	other := domain.TitleRef{TmdbID: 1396, MediaType: domain.MediaTypeTV}
	repo := &fakeRepository{refs: []domain.TitleRef{wanted}}
	service := NewService(repo, fakeProvider{})
	statuses, err := service.Statuses(context.Background(), 1, []domain.TitleRef{wanted, other})
	if err != nil || !statuses[wanted] || statuses[other] {
		t.Fatalf("unexpected statuses=%v err=%v", statuses, err)
	}
}

func TestListMatchesNormalizesFriendsAndPaginates(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		friendIDs: []int64{9, 3},
		matches: []domain.WatchlistMatchItem{
			{Title: domain.Title{ID: 15}, MatchesCount: 4, LatestAddedAt: now},
			{Title: domain.Title{ID: 14}, MatchesCount: 3, LatestAddedAt: now.Add(-time.Hour)},
			{Title: domain.Title{ID: 13}, MatchesCount: 2, LatestAddedAt: now.Add(-2 * time.Hour)},
		},
	}
	service := NewService(repo, fakeProvider{})
	page, err := service.ListMatches(context.Background(), 1, []string{"friend-b", "friend-a", "friend-b"}, "", 2)
	if err != nil {
		t.Fatalf("ListMatches() error = %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("unexpected page = %#v", page)
	}
	if got, want := repo.resolvedUUIDs, []string{"friend-a", "friend-b"}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("resolved UUIDs = %v, want %v", got, want)
	}
	if got := repo.matchFriendIDs; len(got) != 2 || got[0] != 3 || got[1] != 9 {
		t.Fatalf("friend IDs = %v, want [3 9]", got)
	}
	if repo.matchLimit != 3 {
		t.Fatalf("repository limit = %d, want 3", repo.matchLimit)
	}

	repo.matches = nil
	page, err = service.ListMatches(context.Background(), 1, []string{"friend-b", "friend-a"}, page.NextCursor, 2)
	if err != nil || page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("next page = %#v, error = %v", page, err)
	}
	if repo.matchCursor.MatchesCount != 3 || repo.matchCursor.TitleID != 14 || !repo.matchCursor.LatestAddedAt.Equal(now.Add(-time.Hour)) {
		t.Fatalf("decoded cursor = %#v", repo.matchCursor)
	}
}

func TestListMatchesRejectsInvalidFriendSelection(t *testing.T) {
	repo := &fakeRepository{friendIDs: []int64{2}}
	service := NewService(repo, fakeProvider{})
	for _, friendUUIDs := range [][]string{{""}, {"friend-a", "friend-b"}} {
		if _, err := service.ListMatches(context.Background(), 1, friendUUIDs, "", 20); err != ErrValidation {
			t.Fatalf("ListMatches(%v) error = %v, want ErrValidation", friendUUIDs, err)
		}
	}
}

func TestListMatchesCursorIsBoundToFriendFilter(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		friendIDs: []int64{2},
		matches: []domain.WatchlistMatchItem{
			{Title: domain.Title{ID: 15}, MatchesCount: 2, LatestAddedAt: now},
			{Title: domain.Title{ID: 14}, MatchesCount: 2, LatestAddedAt: now.Add(-time.Hour)},
		},
	}
	service := NewService(repo, fakeProvider{})
	page, err := service.ListMatches(context.Background(), 1, []string{"friend-a"}, "", 1)
	if err != nil || page.NextCursor == "" {
		t.Fatalf("first page = %#v, error = %v", page, err)
	}
	if _, err := service.ListMatches(context.Background(), 1, []string{"friend-b"}, page.NextCursor, 1); err != ErrValidation {
		t.Fatalf("cursor mismatch error = %v, want ErrValidation", err)
	}
}
