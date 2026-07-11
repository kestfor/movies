package postgres

import (
	"encoding/json"
	"math"
	"math/big"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"

	"github.com/jackc/pgx/v5/pgtype"
)

func toDomainUser(user gen.User) domain.User {
	return toDomainUserFields(user.ID, user.Uuid, user.TgID, user.Username, user.FirstName, user.PhotoUrl, user.CreatedAt)
}

func toDomainUserFields(
	id int64,
	uuid pgtype.UUID,
	tgID int64,
	username pgtype.Text,
	firstName string,
	photoURL pgtype.Text,
	createdAt pgtype.Timestamptz,
) domain.User {
	return domain.User{
		ID:        id,
		UUID:      uuidToString(uuid),
		TgID:      tgID,
		Username:  textToString(username),
		FirstName: firstName,
		PhotoURL:  textToString(photoURL),
		CreatedAt: createdAt.Time,
	}
}

func uuidToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

func uuidFromString(value string) (pgtype.UUID, bool) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil || !uuid.Valid {
		return pgtype.UUID{}, false
	}
	return uuid, true
}

func toNullText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func toNullInt4(value int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(value), Valid: value != 0}
}

func toNumericTenths(value int) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(int64(value)), Exp: -1, Valid: true}
}

func numericToFloat64(value pgtype.Numeric) float64 {
	if !value.Valid || value.Int == nil {
		return 0
	}
	result, _ := new(big.Rat).SetFrac(value.Int, big.NewInt(1)).Float64()
	if value.Exp > 0 {
		result *= math.Pow10(int(value.Exp))
	} else if value.Exp < 0 {
		result /= math.Pow10(int(-value.Exp))
	}
	return result
}

func textToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func int8ToInt64(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func toNullInt8(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: value != 0}
}

func marshalGenres(genres []string) []byte {
	if len(genres) == 0 {
		return nil
	}
	data, err := json.Marshal(genres)
	if err != nil {
		return nil
	}
	return data
}

func unmarshalGenres(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	var genres []string
	if err := json.Unmarshal(data, &genres); err != nil {
		return nil
	}
	return genres
}

func toGenMediaType(mediaType domain.MediaType) gen.MediaType {
	return gen.MediaType(mediaType)
}
