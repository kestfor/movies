package tmdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"movies/backend/internal/domain"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientSearchFiltersPeopleAndCaches(t *testing.T) {
	var hits int32
	client := NewClient("https://tmdb.local", "token", "ru-RU", &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/search/multi" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			if got := req.URL.Query().Get("language"); got != "ru-RU" {
				t.Fatalf("language query = %q, want ru-RU", got)
			}
			atomic.AddInt32(&hits, 1)
			body := map[string]any{
				"page":          1,
				"total_pages":   1,
				"total_results": 2,
				"results": []map[string]any{
					{"id": 1, "media_type": "person", "name": "Ignore Me"},
					{"id": 603, "media_type": "movie", "title": "The Matrix", "release_date": "1999-03-31", "poster_path": "/matrix.jpg"},
				},
			}
			return jsonResponse(body), nil
		}),
	}, time.Minute)

	page, err := client.Search(context.Background(), "matrix", 1)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(page.Results) != 1 || page.Results[0].TmdbID != 603 {
		t.Fatalf("unexpected search results: %#v", page.Results)
	}
	if got, want := page.Results[0].PosterPath, "https://image.tmdb.org/t/p/w500/matrix.jpg"; got != want {
		t.Fatalf("poster path = %q, want %q", got, want)
	}

	page, err = client.Search(context.Background(), "matrix", 1)
	if err != nil {
		t.Fatalf("second Search() error = %v", err)
	}
	if len(page.Results) != 1 || atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("cache did not work, hits=%d page=%#v", hits, page)
	}
}

func TestClientGetParsesTitleAndCaches(t *testing.T) {
	var hits int32
	client := NewClient("https://tmdb.local", "token", "ru-RU", &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/movie/603" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			if got := req.URL.Query().Get("language"); got != "ru-RU" {
				t.Fatalf("language query = %q, want ru-RU", got)
			}
			atomic.AddInt32(&hits, 1)
			body := map[string]any{
				"id":             603,
				"title":          "The Matrix",
				"original_title": "The Matrix",
				"release_date":   "1999-03-31",
				"poster_path":    "/matrix.jpg",
				"overview":       "A computer hacker learns about the true nature of reality.",
				"genres": []map[string]any{
					{"name": "Science Fiction"},
					{"name": "Action"},
				},
			}
			return jsonResponse(body), nil
		}),
	}, time.Minute)

	title, err := client.Get(context.Background(), domain.MediaTypeMovie, 603)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if title.Title != "The Matrix" || title.ReleaseYear != 1999 || len(title.Genres) != 2 {
		t.Fatalf("unexpected title: %#v", title)
	}
	if got, want := title.PosterPath, "https://image.tmdb.org/t/p/w500/matrix.jpg"; got != want {
		t.Fatalf("poster path = %q, want %q", got, want)
	}

	_, err = client.Get(context.Background(), domain.MediaTypeMovie, 603)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("cache did not work, hits=%d", hits)
	}
}

func TestClientReturnsNotFound(t *testing.T) {
	client := NewClient("https://tmdb.local", "token", "ru-RU", &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}),
	}, time.Minute)

	_, err := client.Get(context.Background(), domain.MediaTypeMovie, 404)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestClientDiscoverMapsMovieMetadataAndCachesGenres(t *testing.T) {
	var discoverHits, genreHits int32
	client := NewClient("https://tmdb.local", "token", "ru-RU", &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/discover/movie":
				atomic.AddInt32(&discoverHits, 1)
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("sort_by") != "popularity.desc" {
					t.Fatalf("unexpected discover query: %s", req.URL.RawQuery)
				}
				return jsonResponse(map[string]any{
					"page": 1, "total_pages": 2,
					"results": []map[string]any{{
						"id": 603, "title": "Матрица", "release_date": "1999-03-31",
						"genre_ids": []int{28}, "popularity": 99.5, "poster_path": "/matrix.jpg",
					}},
				}), nil
			case "/genre/movie/list":
				atomic.AddInt32(&genreHits, 1)
				return jsonResponse(map[string]any{"genres": []map[string]any{{"id": 28, "name": "Боевик"}}}), nil
			default:
				t.Fatalf("unexpected path: %s", req.URL.Path)
				return nil, nil
			}
		}),
	}, time.Minute)

	page, err := client.Discover(context.Background(), domain.MediaTypeMovie, 1)
	if err != nil || len(page.Results) != 1 {
		t.Fatalf("Discover() page=%#v err=%v", page, err)
	}
	item := page.Results[0]
	if item.Title.Title != "Матрица" || item.Popularity != 99.5 || len(item.Title.Genres) != 1 || item.Title.Genres[0] != "Боевик" {
		t.Fatalf("unexpected candidate: %#v", item)
	}
	if _, err := client.Discover(context.Background(), domain.MediaTypeMovie, 1); err != nil {
		t.Fatalf("second Discover() error=%v", err)
	}
	if discoverHits != 1 || genreHits != 1 {
		t.Fatalf("cache misses: discover=%d genres=%d", discoverHits, genreHits)
	}
}

func TestClientTVRecommendationsUseV3Endpoint(t *testing.T) {
	client := NewClient("https://tmdb.local", "token", "ru-RU", &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/tv/1396/recommendations":
				return jsonResponse(map[string]any{"page": 1, "total_pages": 1, "results": []map[string]any{{"id": 1399, "name": "Игра престолов"}}}), nil
			case "/genre/tv/list":
				return jsonResponse(map[string]any{"genres": []map[string]any{}}), nil
			default:
				t.Fatalf("unexpected path: %s", req.URL.Path)
				return nil, nil
			}
		}),
	}, time.Minute)
	page, err := client.Recommendations(context.Background(), domain.MediaTypeTV, 1396)
	if err != nil || len(page.Results) != 1 || page.Results[0].Title.MediaType != domain.MediaTypeTV {
		t.Fatalf("Recommendations() page=%#v err=%v", page, err)
	}
}

func jsonResponse(body any) *http.Response {
	data, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}
}
