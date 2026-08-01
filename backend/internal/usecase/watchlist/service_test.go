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
	user        domain.User
	visible     bool
	titleID     int64
	titleExists bool
	addAllowed  bool
	items       []domain.WatchlistItem
	count       int64
	refs        []domain.TitleRef
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
