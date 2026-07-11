# API Reference

Backend base URL locally: `http://localhost:8080`.

All endpoints except `GET /health` require:

```http
Authorization: tma <Telegram WebApp initData>
```

For local smoke requests use:

```bash
./scripts/mock_tma_auth.sh
```

Errors use one shape:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "invalid request"
  }
}
```

Common error codes: `unauthorized`, `forbidden`, `not_found`, `conflict`, `validation_failed`, `upstream_error`, `internal`.

## Health

### `GET /health`

Returns:

```json
{ "status": "ok" }
```

## Auth / Current User

### `GET /me`

Returns the current Telegram user, creating/updating it from initData.

```json
{
  "id": 1,
  "tg_id": 111,
  "username": "ivan",
  "first_name": "Иван",
  "photo_url": "https://example.com/photo.jpg",
  "created_at": "2026-07-11T12:22:20Z"
}
```

## Criteria

### `GET /criteria`

Returns active rating criteria.

```json
{
  "criteria": [
    { "id": 1, "code": "plot", "name": "Сюжет", "sort_order": 1 }
  ]
}
```

## Titles

Title routes use external TMDB identity: `:type` is `movie` or `tv`, `:tmdb_id` is TMDB id.

### `GET /search?q=&page=`

Searches TMDB `/search/multi`. Person results are filtered out. Search remains lightweight and does not include social data.

```json
{
  "page": 1,
  "total_pages": 7,
  "total_results": 126,
  "results": [
    {
      "tmdb_id": 603,
      "media_type": "movie",
      "title": "Матрица",
      "original_title": "The Matrix",
      "release_year": 1999,
      "poster_path": "/dXNAPwY7VrqMAo51EKhhCJfaGb5.jpg",
      "overview": "..."
    }
  ]
}
```

### `GET /titles/:type/:tmdb_id`

Returns a title card. Social fields are populated only if the title snapshot exists in DB.

```json
{
  "title": {
    "id": 1,
    "tmdb_id": 603,
    "media_type": "movie",
    "title": "Матрица",
    "original_title": "The Matrix",
    "release_year": 1999,
    "poster_path": "/dXNAPwY7VrqMAo51EKhhCJfaGb5.jpg",
    "genres": ["боевик", "фантастика"],
    "overview": "..."
  },
  "my_rating": {
    "user": { "id": 1, "tg_id": 111, "first_name": "Иван" },
    "avg_score": 8.5,
    "scores": { "plot": 8, "music": 9 },
    "created_at": "2026-07-11T14:00:00Z",
    "updated_at": "2026-07-11T14:00:00Z"
  },
  "friends_ratings": [
    {
      "user": { "id": 2, "tg_id": 222, "first_name": "Анна" },
      "avg_score": 9,
      "scores": { "plot": 10, "music": 8 },
      "created_at": "2026-07-11T14:01:00Z",
      "updated_at": "2026-07-11T14:01:00Z"
    }
  ],
  "friends_avg": {
    "overall": 8.8,
    "by_criteria": { "plot": 9, "music": 8.5 }
  },
  "comments_count": 3
}
```

## Ratings

### `PUT /titles/:type/:tmdb_id/rating`

Creates or replaces current user's rating. At least one score is required. Scores must be integers `1..10`. Repeated `PUT` fully replaces previous scores.

Request:

```json
{
  "scores": {
    "plot": 8,
    "music": 9
  }
}
```

Returns:

```json
{
  "rating": {
    "id": 1,
    "user_id": 1,
    "title_id": 1,
    "avg_score": 8.5,
    "scores": { "plot": 8, "music": 9 },
    "created_at": "2026-07-11T14:00:00Z",
    "updated_at": "2026-07-11T14:00:00Z"
  }
}
```

### `DELETE /titles/:type/:tmdb_id/rating`

Deletes current user's rating. Returns `204 No Content`.

## Comments

### `GET /titles/:type/:tmdb_id/comments`

Returns comment tree.

```json
{
  "comments": [
    {
      "id": 1,
      "title_id": 1,
      "user": { "id": 1, "tg_id": 111, "first_name": "Иван" },
      "body": "Комментарий",
      "is_deleted": false,
      "created_at": "2026-07-11T14:00:00Z",
      "updated_at": "2026-07-11T14:00:00Z",
      "replies": []
    }
  ]
}
```

### `POST /titles/:type/:tmdb_id/comments`

Creates a comment. `body` is trimmed and must be `1..4000` chars. `parent_id` is optional. Replying to a deleted comment returns `422`.

Request:

```json
{
  "body": "Комментарий",
  "parent_id": 1
}
```

Returns `201`:

```json
{ "comment": { "id": 2, "body": "Комментарий", "is_deleted": false } }
```

### `PATCH /comments/:id`

Edits own non-deleted comment.

Request:

```json
{ "body": "Новый текст" }
```

Returns:

```json
{ "comment": { "id": 1, "body": "Новый текст", "is_deleted": false } }
```

### `DELETE /comments/:id`

Soft-deletes own comment. Body becomes empty, replies remain.

Returns:

```json
{ "comment": { "id": 1, "body": "", "is_deleted": true } }
```

## Friends

### `GET /friends`

Returns accepted friends.

```json
{
  "friends": [
    { "id": 2, "tg_id": 222, "username": "anna", "first_name": "Анна" }
  ]
}
```

### `GET /friends/requests`

Returns incoming pending requests.

```json
{
  "requests": [
    {
      "user": { "id": 1, "tg_id": 111, "first_name": "Иван" },
      "requested_at": "2026-07-11T14:00:00Z"
    }
  ]
}
```

### `POST /friends/requests`

Creates a friend request. If a reverse pending request exists, it is auto-accepted.

Request:

```json
{ "user_id": 2 }
```

Returns `201`:

```json
{
  "friendship": {
    "requester_id": 1,
    "addressee_id": 2,
    "status": "pending",
    "created_at": "2026-07-11T14:00:00Z"
  }
}
```

Rules:

- self request -> `422`
- duplicate outgoing pending -> `409`
- already friends -> `409`
- reverse pending -> returns `status: "accepted"`

### `POST /friends/requests/:user_id/accept`

Accepts incoming request from `:user_id`.

Returns:

```json
{ "friendship": { "requester_id": 1, "addressee_id": 2, "status": "accepted" } }
```

### `DELETE /friends/requests/:user_id`

Deletes a pending request between current user and `:user_id` in either direction. Returns `204 No Content`.

### `DELETE /friends/:user_id`

Deletes accepted friendship. Returns `204 No Content`.

## Feed

### `GET /feed?cursor=&limit=`

Returns ratings from accepted friends only. Sorted by `(created_at, id) DESC`.

`limit` defaults to `20`, max `50`. `next_cursor` is opaque.

```json
{
  "items": [
    {
      "id": 10,
      "user": { "id": 2, "tg_id": 222, "first_name": "Анна" },
      "title": {
        "id": 1,
        "tmdb_id": 603,
        "media_type": "movie",
        "title": "Матрица"
      },
      "avg_score": 8.5,
      "scores": { "plot": 8, "music": 9 },
      "created_at": "2026-07-11T14:00:00Z",
      "updated_at": "2026-07-11T14:00:00Z"
    }
  ],
  "next_cursor": "opaque"
}
```

## Profile Ratings

### `GET /users/:id/ratings`

Returns ratings and stats for current user or accepted friend. For non-friends returns empty ratings and zero stats.

```json
{
  "ratings": [
    {
      "title": {
        "id": 1,
        "tmdb_id": 603,
        "media_type": "movie",
        "title": "Матрица"
      },
      "avg_score": 8.5,
      "scores": { "plot": 8, "music": 9 },
      "created_at": "2026-07-11T14:00:00Z",
      "updated_at": "2026-07-11T14:00:00Z"
    }
  ],
  "stats": {
    "count": 1,
    "avg_score": 8.5
  }
}
```

## Local Smoke Scripts

```bash
./requests/tmdb-curl.sh
./requests/stage3-curl.sh
./requests/friends-curl.sh
./requests/feed-curl.sh
```
