package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"movies/backend/internal/domain"
)

var (
	ErrMissingToken = errors.New("missing_tmdb_token")
	ErrNotFound     = errors.New("not_found")
	ErrUpstream     = errors.New("upstream_error")
)

const (
	tmdbImageBaseURL = "https://image.tmdb.org/t/p/"
	tmdbPosterSize   = "w500"
)

type Client struct {
	baseURL    string
	apiToken   string
	language   string
	httpClient *http.Client
	ttl        time.Duration

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	expiresAt time.Time
	value     any
}

func NewClient(baseURL, apiToken, language string, httpClient *http.Client, ttl time.Duration) *Client {
	if baseURL == "" {
		baseURL = "https://143.204.238.90/3"
	}
	if language == "" {
		language = "ru-RU"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiToken:   apiToken,
		language:   language,
		httpClient: httpClient,
		ttl:        ttl,
		cache:      make(map[string]cacheEntry),
	}
}

func (c *Client) Search(ctx context.Context, query string, page int) (domain.SearchPage, error) {
	key := fmt.Sprintf("search|%s|%d|%s", query, page, c.language)
	if cached, ok := c.getCached(key); ok {
		return cached.(domain.SearchPage), nil
	}

	var payload searchResponse
	if err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/search/multi", url.Values{
		"query": {query},
		"page":  {strconv.Itoa(page)},
	}, &payload); err != nil {
		return domain.SearchPage{}, err
	}

	result := payload.toDomain()
	c.setCached(key, result)
	return result, nil
}

func (c *Client) Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error) {
	key := fmt.Sprintf("title|%s|%d|%s", mediaType, tmdbID, c.language)
	if cached, ok := c.getCached(key); ok {
		return cached.(domain.Title), nil
	}

	var payload titleResponse
	path := fmt.Sprintf("%s/%d", mediaType, tmdbID)
	if err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/"+path, nil, &payload); err != nil {
		return domain.Title{}, err
	}

	result := payload.toDomain(mediaType)
	c.setCached(key, result)
	return result, nil
}

func (c *Client) Discover(ctx context.Context, mediaType domain.MediaType, page int) (domain.CatalogProviderPage, error) {
	if mediaType != domain.MediaTypeMovie && mediaType != domain.MediaTypeTV {
		return domain.CatalogProviderPage{}, ErrNotFound
	}
	if page < 1 {
		page = 1
	}
	key := fmt.Sprintf("discover|%s|%d|%s", mediaType, page, c.language)
	if cached, ok := c.getCached(key); ok {
		return cached.(domain.CatalogProviderPage), nil
	}
	result, err := c.catalogPage(ctx, "/discover/"+string(mediaType), mediaType, url.Values{
		"page": {strconv.Itoa(page)}, "include_adult": {"false"}, "sort_by": {"popularity.desc"},
	})
	if err != nil {
		return domain.CatalogProviderPage{}, err
	}
	c.setCached(key, result)
	return result, nil
}

func (c *Client) Recommendations(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.CatalogProviderPage, error) {
	if (mediaType != domain.MediaTypeMovie && mediaType != domain.MediaTypeTV) || tmdbID <= 0 {
		return domain.CatalogProviderPage{}, ErrNotFound
	}
	key := fmt.Sprintf("recommendations|%s|%d|%s", mediaType, tmdbID, c.language)
	if cached, ok := c.getCached(key); ok {
		return cached.(domain.CatalogProviderPage), nil
	}
	path := fmt.Sprintf("/%s/%d/recommendations", mediaType, tmdbID)
	result, err := c.catalogPage(ctx, path, mediaType, url.Values{"page": {"1"}})
	if err != nil {
		return domain.CatalogProviderPage{}, err
	}
	c.setCached(key, result)
	return result, nil
}

func (c *Client) catalogPage(ctx context.Context, path string, mediaType domain.MediaType, query url.Values) (domain.CatalogProviderPage, error) {
	var payload catalogResponse
	if err := c.doJSON(ctx, http.MethodGet, c.baseURL+path, query, &payload); err != nil {
		return domain.CatalogProviderPage{}, err
	}
	genres, err := c.genreMap(ctx, mediaType)
	if err != nil {
		return domain.CatalogProviderPage{}, err
	}
	items := make([]domain.CatalogCandidate, 0, len(payload.Results))
	for _, item := range payload.Results {
		if item.ID <= 0 {
			continue
		}
		titleGenres := make([]string, 0, len(item.GenreIDs))
		for _, id := range item.GenreIDs {
			if name := genres[id]; name != "" {
				titleGenres = append(titleGenres, name)
			}
		}
		items = append(items, domain.CatalogCandidate{
			Title: domain.Title{
				TmdbID: item.ID, MediaType: mediaType, Title: pick(item.Title, item.Name),
				OriginalTitle: pick(item.OriginalTitle, item.OriginalName),
				ReleaseYear:   yearFromDate(pick(item.ReleaseDate, item.FirstAirDate)),
				PosterPath:    posterURL(item.PosterPath), Genres: titleGenres, Overview: item.Overview,
			},
			Popularity: item.Popularity, GenreIDs: item.GenreIDs,
		})
	}
	return domain.CatalogProviderPage{Page: payload.Page, TotalPages: payload.TotalPages, Results: items}, nil
}

func (c *Client) genreMap(ctx context.Context, mediaType domain.MediaType) (map[int64]string, error) {
	key := "genres|" + string(mediaType) + "|" + c.language
	if cached, ok := c.getCached(key); ok {
		return cached.(map[int64]string), nil
	}
	var payload struct {
		Genres []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}
	if err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/genre/"+string(mediaType)+"/list", nil, &payload); err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(payload.Genres))
	for _, genre := range payload.Genres {
		result[genre.ID] = genre.Name
	}
	c.setCached(key, result)
	return result, nil
}

func (c *Client) doJSON(ctx context.Context, method, rawURL string, query url.Values, out any) error {
	if c.apiToken == "" {
		return ErrMissingToken
	}

	reqURL, err := url.Parse(rawURL)
	if err != nil {
		return ErrUpstream
	}
	query = cloneValues(query)
	if c.language != "" {
		query.Set("language", c.language)
	}
	if len(query) > 0 {
		reqURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), nil)
	if err != nil {
		return ErrUpstream
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrUpstream
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return ErrUpstream
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return ErrUpstream
	}
	return nil
}

func cloneValues(values url.Values) url.Values {
	clone := make(url.Values, len(values))
	for key, value := range values {
		clone[key] = append([]string(nil), value...)
	}
	return clone
}

func (c *Client) getCached(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.cache, key)
		return nil, false
	}
	return entry.value, true
}

func (c *Client) setCached(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{expiresAt: time.Now().Add(c.ttl), value: value}
}

type searchResponse struct {
	Page         int          `json:"page"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
	Results      []searchItem `json:"results"`
}

type catalogResponse struct {
	Page       int           `json:"page"`
	TotalPages int           `json:"total_pages"`
	Results    []catalogItem `json:"results"`
}

type catalogItem struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	OriginalTitle string  `json:"original_title"`
	OriginalName  string  `json:"original_name"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	PosterPath    string  `json:"poster_path"`
	Overview      string  `json:"overview"`
	Popularity    float64 `json:"popularity"`
	GenreIDs      []int64 `json:"genre_ids"`
}

type searchItem struct {
	ID            int64  `json:"id"`
	MediaType     string `json:"media_type"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
	PosterPath    string `json:"poster_path"`
	Overview      string `json:"overview"`
}

func (r searchResponse) toDomain() domain.SearchPage {
	items := make([]domain.SearchResult, 0, len(r.Results))
	for _, item := range r.Results {
		if item.MediaType == "person" || item.ID == 0 {
			continue
		}
		mediaType := domain.MediaType(item.MediaType)
		if mediaType != domain.MediaTypeMovie && mediaType != domain.MediaTypeTV {
			continue
		}
		items = append(items, domain.SearchResult{
			TmdbID:        item.ID,
			MediaType:     mediaType,
			Title:         pick(item.Title, item.Name),
			OriginalTitle: pick(item.OriginalTitle, item.OriginalName),
			ReleaseYear:   yearFromDate(pick(item.ReleaseDate, item.FirstAirDate)),
			PosterPath:    posterURL(item.PosterPath),
			Overview:      item.Overview,
		})
	}

	return domain.SearchPage{
		Page:         r.Page,
		TotalPages:   r.TotalPages,
		TotalResults: r.TotalResults,
		Results:      items,
	}
}

type titleResponse struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Name          string     `json:"name"`
	OriginalTitle string     `json:"original_title"`
	OriginalName  string     `json:"original_name"`
	ReleaseDate   string     `json:"release_date"`
	FirstAirDate  string     `json:"first_air_date"`
	PosterPath    string     `json:"poster_path"`
	Genres        []genreDTO `json:"genres"`
	Overview      string     `json:"overview"`
}

type genreDTO struct {
	Name string `json:"name"`
}

func (r titleResponse) toDomain(mediaType domain.MediaType) domain.Title {
	genres := make([]string, 0, len(r.Genres))
	for _, genre := range r.Genres {
		if genre.Name != "" {
			genres = append(genres, genre.Name)
		}
	}

	return domain.Title{
		TmdbID:        r.ID,
		MediaType:     mediaType,
		Title:         pick(r.Title, r.Name),
		OriginalTitle: pick(r.OriginalTitle, r.OriginalName),
		ReleaseYear:   yearFromDate(pick(r.ReleaseDate, r.FirstAirDate)),
		PosterPath:    posterURL(r.PosterPath),
		Genres:        genres,
		Overview:      r.Overview,
	}
}

func posterURL(path string) string {
	if path == "" || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return tmdbImageBaseURL + tmdbPosterSize + path
}

func pick(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func yearFromDate(raw string) int {
	if len(raw) < 4 {
		return 0
	}
	year, err := strconv.Atoi(raw[:4])
	if err != nil {
		return 0
	}
	return year
}
