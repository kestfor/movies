package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"movies/backend/internal/domain"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrValidation   = errors.New("validation_failed")
)

type UserRepository interface {
	UpsertTelegramUser(ctx context.Context, params UpsertTelegramUserParams) (domain.User, error)
	GetByID(ctx context.Context, id int64) (domain.User, error)
}

type UpsertTelegramUserParams struct {
	TgID      int64
	Username  string
	FirstName string
	PhotoURL  string
}

type Service struct {
	repo     UserRepository
	botToken string
	authTTL  time.Duration
}

func NewService(repo UserRepository, botToken string, authTTL time.Duration) *Service {
	return &Service{
		repo:     repo,
		botToken: botToken,
		authTTL:  authTTL,
	}
}

func (s *Service) Authenticate(ctx context.Context, initData string) (domain.User, error) {
	values, err := parseQueryPairs(initData)
	if err != nil {
		return domain.User{}, ErrUnauthorized
	}
	if err := validateInitData(values, s.botToken); err != nil {
		return domain.User{}, ErrUnauthorized
	}

	payload, err := parseInitPayload(values)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			return domain.User{}, ErrUnauthorized
		}
		return domain.User{}, err
	}

	if time.Since(time.Unix(payload.AuthDate, 0).UTC()) > s.authTTL {
		return domain.User{}, ErrUnauthorized
	}

	user, err := s.repo.UpsertTelegramUser(ctx, UpsertTelegramUserParams{
		TgID:      payload.User.ID,
		Username:  payload.User.Username,
		FirstName: payload.User.FirstName,
		PhotoURL:  payload.User.PhotoURL,
	})
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func validateInitData(values url.Values, botToken string) error {
	if botToken == "" {
		return ErrUnauthorized
	}

	checkHash := values.Get("hash")
	if checkHash == "" {
		return ErrUnauthorized
	}

	dataCheck := buildDataCheckString(values, "hash")
	secretKeyMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretKeyMAC.Write([]byte(botToken))
	secretKey := secretKeyMAC.Sum(nil)

	checkMAC := hmac.New(sha256.New, secretKey)
	_, _ = checkMAC.Write([]byte(dataCheck))
	expected := fmt.Sprintf("%x", checkMAC.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(checkHash)) {
		return ErrUnauthorized
	}

	return nil
}

type initPayload struct {
	AuthDate int64 `json:"auth_date"`
	User     struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		PhotoURL  string `json:"photo_url"`
	} `json:"user"`
}

func parseInitPayload(values url.Values) (initPayload, error) {
	var payload initPayload
	if raw := values.Get("auth_date"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return initPayload{}, err
		}
		payload.AuthDate = parsed
	}
	if raw := values.Get("user"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.User); err != nil {
			return initPayload{}, err
		}
	}
	if payload.AuthDate == 0 || payload.User.ID == 0 || payload.User.FirstName == "" {
		return initPayload{}, ErrValidation
	}

	return payload, nil
}

func parseQueryPairs(raw string) (url.Values, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, ErrValidation
	}
	return values, nil
}

func buildDataCheckString(values url.Values, exclude string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == exclude {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(values.Get(key))
	}
	return b.String()
}
