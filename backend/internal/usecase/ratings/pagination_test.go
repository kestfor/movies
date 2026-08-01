package ratings

import (
	"testing"
	"time"

	"movies/backend/internal/domain"
)

func TestParseListQueryDefaultsAndRejectsInvalid(t *testing.T) {
	query, err := parseListQuery("", "", "", 0)
	if err != nil {
		t.Fatalf("parseListQuery() error = %v", err)
	}
	if query.Sort != SortRecent || query.Order != OrderDesc || query.Limit != 20 {
		t.Fatalf("unexpected defaults: %#v", query)
	}
	if _, err := parseListQuery("unknown", OrderDesc, "", 20); err != ErrValidation {
		t.Fatalf("invalid sort error = %v, want ErrValidation", err)
	}
	if _, err := parseListQuery(SortScore, "sideways", "", 20); err != ErrValidation {
		t.Fatalf("invalid order error = %v, want ErrValidation", err)
	}
}

func TestProfileCursorRoundTripAndParameterBinding(t *testing.T) {
	updatedAt := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	base := parsedListQuery{Sort: SortRecent, Order: OrderDesc, Limit: 20}
	raw := encodeListCursor(base, domain.ProfileRating{ID: 42, UpdatedAt: updatedAt})
	query, err := parseListQuery(SortRecent, OrderDesc, raw, 20)
	if err != nil {
		t.Fatalf("parseListQuery() error = %v", err)
	}
	if query.Cursor.ID != 42 || !query.Cursor.Recent.Equal(updatedAt) {
		t.Fatalf("unexpected cursor: %#v", query.Cursor)
	}
	if _, err := parseListQuery(SortRecent, OrderAsc, raw, 20); err != ErrValidation {
		t.Fatalf("cursor reused with another order error = %v, want ErrValidation", err)
	}
}

func TestProfileScoreAndTitleCursors(t *testing.T) {
	tests := []struct {
		name   string
		query  parsedListQuery
		rating domain.ProfileRating
	}{
		{"score", parsedListQuery{Sort: SortScore, Order: OrderDesc, Limit: 20}, domain.ProfileRating{ID: 7, AvgScore: 8.5}},
		{"title", parsedListQuery{Sort: SortTitle, Order: OrderAsc, Limit: 20}, domain.ProfileRating{ID: 8, Title: domain.Title{Title: "Матрица"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := encodeListCursor(tt.query, tt.rating)
			got, err := parseListQuery(tt.query.Sort, tt.query.Order, raw, 20)
			if err != nil || got.Cursor.ID != tt.rating.ID {
				t.Fatalf("round trip got=%#v err=%v", got, err)
			}
		})
	}
}
