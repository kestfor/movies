package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"movies/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeAuthenticator struct {
	user domain.User
	err  error
}

func (f fakeAuthenticator) Authenticate(context.Context, string) (domain.User, error) {
	return f.user, f.err
}

type fakeUserGetter struct {
	user          domain.User
	searchResults []domain.UserSearchResult
	err           error
}

func (f fakeUserGetter) GetByID(context.Context, int64) (domain.User, error) {
	return f.user, f.err
}

func (f fakeUserGetter) GetByUUID(context.Context, string) (domain.User, bool, error) {
	return f.user, f.user.ID != 0, f.err
}

func (f fakeUserGetter) SearchByUsernamePrefix(context.Context, int64, string, int32) ([]domain.UserSearchResult, error) {
	return f.searchResults, f.err
}

type fakeTitleSearcher struct {
	searchPage domain.SearchPage
	title      domain.Title
	err        error
}

func (f fakeTitleSearcher) Search(context.Context, string, int) (domain.SearchPage, error) {
	return f.searchPage, f.err
}

func (f fakeTitleSearcher) Get(context.Context, domain.MediaType, int64) (domain.Title, error) {
	return f.title, f.err
}

func (f fakeTitleSearcher) GetCard(context.Context, int64, domain.MediaType, int64) (domain.TitleCard, error) {
	return domain.TitleCard{Title: f.title}, f.err
}

type fakeCriteriaLister struct {
	criteria []domain.Criterion
	err      error
}

func (f fakeCriteriaLister) ListActive(context.Context) ([]domain.Criterion, error) {
	return f.criteria, f.err
}

type fakeRatingManager struct {
	rating domain.Rating
	err    error
}

type fakeWatchlistManager struct {
	page             domain.WatchlistPage
	matchesPage      domain.WatchlistMatchesPage
	matchFriendUUIDs *[]string
	matchCursor      *string
	matchLimit       *int
	err              error
}

func (f fakeWatchlistManager) Add(context.Context, int64, domain.MediaType, int64) error {
	return f.err
}
func (f fakeWatchlistManager) Remove(context.Context, int64, domain.MediaType, int64) error {
	return f.err
}
func (f fakeWatchlistManager) ListByUUID(context.Context, int64, string, string, int) (domain.WatchlistPage, error) {
	return f.page, f.err
}
func (f fakeWatchlistManager) ListMatches(_ context.Context, _ int64, friendUUIDs []string, cursor string, limit int) (domain.WatchlistMatchesPage, error) {
	if f.matchFriendUUIDs != nil {
		*f.matchFriendUUIDs = append([]string(nil), friendUUIDs...)
	}
	if f.matchCursor != nil {
		*f.matchCursor = cursor
	}
	if f.matchLimit != nil {
		*f.matchLimit = limit
	}
	return f.matchesPage, f.err
}
func (f fakeWatchlistManager) IsInWatchlist(context.Context, int64, domain.MediaType, int64) (bool, error) {
	return false, f.err
}
func (f fakeWatchlistManager) Statuses(context.Context, int64, []domain.TitleRef) (map[domain.TitleRef]bool, error) {
	return map[domain.TitleRef]bool{}, f.err
}

func (f fakeRatingManager) Upsert(context.Context, int64, domain.MediaType, int64, map[string]int) (domain.Rating, error) {
	return f.rating, f.err
}

func (f fakeRatingManager) Delete(context.Context, int64, domain.MediaType, int64) error {
	return f.err
}

func (f fakeRatingManager) ListUserRatingsByUUID(context.Context, int64, string, string, string, string, int) (domain.ProfileRatingsPage, error) {
	return domain.ProfileRatingsPage{}, f.err
}

type fakeCommentManager struct {
	comments []domain.Comment
	comment  domain.Comment
	err      error
}

func (f fakeCommentManager) List(context.Context, domain.MediaType, int64) ([]domain.Comment, error) {
	return f.comments, f.err
}

func (f fakeCommentManager) Create(context.Context, int64, domain.MediaType, int64, int64, string) (domain.Comment, error) {
	return f.comment, f.err
}

func (f fakeCommentManager) Update(context.Context, int64, int64, string) (domain.Comment, error) {
	return f.comment, f.err
}

func (f fakeCommentManager) Delete(context.Context, int64, int64) (domain.Comment, error) {
	return f.comment, f.err
}

type fakeFriendManager struct {
	friends    []domain.User
	requests   []domain.FriendRequest
	friendship domain.Friendship
	err        error
}

func (f fakeFriendManager) ListFriends(context.Context, int64) ([]domain.User, error) {
	return f.friends, f.err
}

func (f fakeFriendManager) ListIncomingRequests(context.Context, int64) ([]domain.FriendRequest, error) {
	return f.requests, f.err
}

func (f fakeFriendManager) CreateRequest(context.Context, int64, int64) (domain.Friendship, error) {
	return f.friendship, f.err
}

func (f fakeFriendManager) CreateRequestByUUID(context.Context, int64, string) (domain.Friendship, error) {
	return f.friendship, f.err
}

func (f fakeFriendManager) AcceptRequest(context.Context, int64, int64) (domain.Friendship, error) {
	return f.friendship, f.err
}

func (f fakeFriendManager) AcceptRequestByUUID(context.Context, int64, string) (domain.Friendship, error) {
	return f.friendship, f.err
}

func (f fakeFriendManager) DeleteRequest(context.Context, int64, int64) error {
	return f.err
}

func (f fakeFriendManager) DeleteRequestByUUID(context.Context, int64, string) error {
	return f.err
}

func (f fakeFriendManager) DeleteFriend(context.Context, int64, int64) error {
	return f.err
}

func (f fakeFriendManager) DeleteFriendByUUID(context.Context, int64, string) error {
	return f.err
}

type fakeFeedLister struct {
	page domain.FeedPage
	err  error
}

type fakeCatalogManager struct {
	page domain.CatalogPage
	err  error
}

func (f fakeCatalogManager) Discover(context.Context, int64, string, string, int) (domain.CatalogPage, error) {
	return f.page, f.err
}
func (f fakeCatalogManager) Recommendations(context.Context, int64, string, int) (domain.CatalogPage, error) {
	return f.page, f.err
}

func (f fakeFeedLister) List(context.Context, int64, string, int) (domain.FeedPage, error) {
	return f.page, f.err
}

type fakeNotificationManager struct {
	page  domain.NotificationsPage
	count int64
	err   error
}

func (f fakeNotificationManager) List(context.Context, int64, string, int) (domain.NotificationsPage, error) {
	return f.page, f.err
}

func (f fakeNotificationManager) CountUnread(context.Context, int64) (int64, error) {
	return f.count, f.err
}

func (f fakeNotificationManager) MarkRead(context.Context, int64, int64) error {
	return f.err
}

func (f fakeNotificationManager) MarkAllRead(context.Context, int64) error {
	return f.err
}

func TestMeReturnsCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	want := domain.User{
		ID:        1,
		UUID:      "11111111-1111-1111-1111-111111111111",
		TgID:      111,
		Username:  "ivan",
		FirstName: "Иван",
		PhotoURL:  "https://photo",
		CreatedAt: time.Unix(1, 0).UTC(),
	}
	router := NewRouter(fakeAuthenticator{user: want}, fakeUserGetter{user: want}, fakeTitleSearcher{}, fakeCriteriaLister{}, fakeRatingManager{}, fakeWatchlistManager{}, fakeCommentManager{}, fakeFriendManager{}, fakeFeedLister{}, fakeCatalogManager{}, fakeNotificationManager{})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["first_name"] != "Иван" || got["username"] != "ivan" {
		t.Fatalf("unexpected response: %v", got)
	}
	if got["uuid"] != want.UUID {
		t.Fatalf("uuid = %v, want %s", got["uuid"], want.UUID)
	}
	if _, ok := got["id"]; ok {
		t.Fatalf("response leaks internal id: %v", got)
	}
}

func TestMeRejectsMissingAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(fakeAuthenticator{}, fakeUserGetter{}, fakeTitleSearcher{}, fakeCriteriaLister{}, fakeRatingManager{}, fakeWatchlistManager{}, fakeCommentManager{}, fakeFriendManager{}, fakeFeedLister{}, fakeCatalogManager{}, fakeNotificationManager{})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestSearchReturnsResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}},
		fakeUserGetter{user: domain.User{ID: 1}},
		fakeTitleSearcher{searchPage: domain.SearchPage{
			Page:         1,
			TotalPages:   2,
			TotalResults: 1,
			Results: []domain.SearchResult{{
				TmdbID:    603,
				MediaType: domain.MediaTypeMovie,
				Title:     "Матрица",
			}},
		}},
		fakeCriteriaLister{},
		fakeRatingManager{},
		fakeWatchlistManager{},
		fakeCommentManager{},
		fakeFriendManager{},
		fakeFeedLister{},
		fakeCatalogManager{},
		fakeNotificationManager{},
	)

	req := httptest.NewRequest(http.MethodGet, "/search?q=matrix&page=1", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "\"tmdb_id\":603") {
		t.Fatalf("response does not contain search result: %s", rec.Body.String())
	}
}

func TestUserSearchReturnsRelationshipResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}},
		fakeUserGetter{
			user: domain.User{ID: 1},
			searchResults: []domain.UserSearchResult{{
				User:           domain.User{ID: 2, UUID: "22222222-2222-2222-2222-222222222222", Username: "ivan", FirstName: "Иван"},
				Relationship:   "none",
				CanSendRequest: true,
			}},
		},
		fakeTitleSearcher{},
		fakeCriteriaLister{},
		fakeRatingManager{},
		fakeWatchlistManager{},
		fakeCommentManager{},
		fakeFriendManager{},
		fakeFeedLister{},
		fakeCatalogManager{},
		fakeNotificationManager{},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/search?q=iv", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "\"relationship\":\"none\"") {
		t.Fatalf("response does not contain relationship: %s", rec.Body.String())
	}
}

func TestGetTitleReturnsTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	want := domain.Title{
		TmdbID:    603,
		MediaType: domain.MediaTypeMovie,
		Title:     "Матрица",
	}
	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}},
		fakeUserGetter{user: domain.User{ID: 1}},
		fakeTitleSearcher{title: want},
		fakeCriteriaLister{},
		fakeRatingManager{},
		fakeWatchlistManager{},
		fakeCommentManager{},
		fakeFriendManager{},
		fakeFeedLister{},
		fakeCatalogManager{},
		fakeNotificationManager{},
	)

	req := httptest.NewRequest(http.MethodGet, "/titles/movie/603", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "\"title\":{\"tmdb_id\":603") {
		t.Fatalf("response does not contain title: %s", rec.Body.String())
	}
}

func TestDiscoverReturnsCatalogItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}}, fakeUserGetter{}, fakeTitleSearcher{}, fakeCriteriaLister{},
		fakeRatingManager{}, fakeWatchlistManager{}, fakeCommentManager{}, fakeFriendManager{}, fakeFeedLister{},
		fakeCatalogManager{page: domain.CatalogPage{Items: []domain.CatalogItem{{
			Title: domain.Title{TmdbID: 603, MediaType: domain.MediaTypeMovie, Title: "Матрица"}, InWatchlist: true,
		}}}}, fakeNotificationManager{},
	)
	req := httptest.NewRequest(http.MethodGet, "/discover?type=movie&limit=20", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "\"in_watchlist\":true") {
		t.Fatalf("unexpected response status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWatchlistRoutesAreIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}}, fakeUserGetter{}, fakeTitleSearcher{}, fakeCriteriaLister{},
		fakeRatingManager{}, fakeWatchlistManager{}, fakeCommentManager{}, fakeFriendManager{}, fakeFeedLister{},
		fakeCatalogManager{}, fakeNotificationManager{},
	)
	for _, test := range []struct {
		method string
		want   int
	}{
		{http.MethodPut, http.StatusOK},
		{http.MethodDelete, http.StatusNoContent},
	} {
		req := httptest.NewRequest(test.method, "/titles/movie/603/watchlist", nil)
		req.Header.Set("Authorization", "tma init-data")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != test.want {
			t.Fatalf("%s status=%d want=%d body=%s", test.method, rec.Code, test.want, rec.Body.String())
		}
	}
}

func TestWatchlistMatchesRoutePassesFiltersAndReturnsPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var friendUUIDs []string
	var cursor string
	var limit int
	watchlist := fakeWatchlistManager{
		matchesPage: domain.WatchlistMatchesPage{Items: []domain.WatchlistMatchItem{{
			Title:        domain.Title{TmdbID: 603, MediaType: domain.MediaTypeMovie, Title: "Матрица"},
			Users:        []domain.User{{UUID: "me", FirstName: "Илья"}, {UUID: "friend", FirstName: "Аня"}},
			MatchesCount: 2,
		}}},
		matchFriendUUIDs: &friendUUIDs, matchCursor: &cursor, matchLimit: &limit,
	}
	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}}, fakeUserGetter{}, fakeTitleSearcher{}, fakeCriteriaLister{},
		fakeRatingManager{}, watchlist, fakeCommentManager{}, fakeFriendManager{}, fakeFeedLister{},
		fakeCatalogManager{}, fakeNotificationManager{},
	)
	req := httptest.NewRequest(http.MethodGet, "/watchlist/matches?friend_id=b&friend_id=a&cursor=opaque&limit=7", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"matches_count":2`) || !strings.Contains(rec.Body.String(), `"users":[`) {
		t.Fatalf("unexpected response status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(friendUUIDs) != 2 || friendUUIDs[0] != "b" || friendUUIDs[1] != "a" || cursor != "opaque" || limit != 7 {
		t.Fatalf("handler args friends=%v cursor=%q limit=%d", friendUUIDs, cursor, limit)
	}
}

func TestWatchlistMatchesRouteRejectsInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(
		fakeAuthenticator{user: domain.User{ID: 1}}, fakeUserGetter{}, fakeTitleSearcher{}, fakeCriteriaLister{},
		fakeRatingManager{}, fakeWatchlistManager{}, fakeCommentManager{}, fakeFriendManager{}, fakeFeedLister{},
		fakeCatalogManager{}, fakeNotificationManager{},
	)
	req := httptest.NewRequest(http.MethodGet, "/watchlist/matches?limit=zero", nil)
	req.Header.Set("Authorization", "tma init-data")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), `"validation_failed"`) {
		t.Fatalf("unexpected response status=%d body=%s", rec.Code, rec.Body.String())
	}
}
