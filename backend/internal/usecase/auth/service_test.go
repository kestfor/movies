package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"movies/backend/internal/domain"
)

type fakeUserRepo struct {
	users map[int64]domain.User
	next  int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users: make(map[int64]domain.User),
		next:  1,
	}
}

func (r *fakeUserRepo) UpsertTelegramUser(_ context.Context, params UpsertTelegramUserParams) (domain.User, error) {
	user, ok := r.users[params.TgID]
	if !ok {
		user = domain.User{ID: r.next, TgID: params.TgID, CreatedAt: time.Unix(1, 0).UTC()}
		r.next++
	}
	user.Username = params.Username
	user.FirstName = params.FirstName
	user.PhotoURL = params.PhotoURL
	r.users[params.TgID] = user
	return user, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id int64) (domain.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return domain.User{}, context.Canceled
}

func TestServiceAuthenticate(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, "bot-token", 24*time.Hour)

	initData := signedInitData(t, "bot-token", 111, "ivan", "Иван", "https://photo")
	user, err := svc.Authenticate(context.Background(), initData)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if user.ID != 1 || user.TgID != 111 || user.Username != "ivan" || user.FirstName != "Иван" || user.PhotoURL != "https://photo" {
		t.Fatalf("unexpected user: %#v", user)
	}

	updated := signedInitData(t, "bot-token", 111, "ivan2", "Иван", "https://photo2")
	user, err = svc.Authenticate(context.Background(), updated)
	if err != nil {
		t.Fatalf("second Authenticate() error = %v", err)
	}
	if user.ID != 1 || user.Username != "ivan2" || user.PhotoURL != "https://photo2" {
		t.Fatalf("unexpected updated user: %#v", user)
	}
}

func TestServiceAuthenticateRejectsExpiredAuthDate(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, "bot-token", time.Minute)

	initData := signedInitDataAt(t, "bot-token", 111, "ivan", "Иван", "https://photo", time.Now().Add(-2*time.Minute))
	if _, err := svc.Authenticate(context.Background(), initData); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() error = %v, want unauthorized", err)
	}
}

func TestServiceAuthenticateRejectsBadSignature(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, "bot-token", 24*time.Hour)

	initData := signedInitData(t, "bot-token", 111, "ivan", "Иван", "https://photo")
	initData = strings.Replace(initData, "hash=", "hash=deadbeef", 1)
	if _, err := svc.Authenticate(context.Background(), initData); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() error = %v, want unauthorized", err)
	}
}

func signedInitData(t *testing.T, botToken string, tgID int64, username, firstName, photoURL string) string {
	t.Helper()
	return signedInitDataAt(t, botToken, tgID, username, firstName, photoURL, time.Now().UTC())
}

func signedInitDataAt(t *testing.T, botToken string, tgID int64, username, firstName, photoURL string, authTime time.Time) string {
	t.Helper()
	values := url.Values{}
	userJSON := fmt.Sprintf(`{"id":%d,"username":%q,"first_name":%q,"photo_url":%q}`, tgID, username, firstName, photoURL)
	values.Set("auth_date", strconv.FormatInt(authTime.Unix(), 10))
	values.Set("query_id", "AAEAAAE")
	values.Set("user", userJSON)

	dataCheck := buildDataCheckString(values, "hash")
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	checkMAC := hmac.New(sha256.New, secretKey)
	_, _ = checkMAC.Write([]byte(dataCheck))
	values.Set("hash", fmt.Sprintf("%x", checkMAC.Sum(nil)))
	return values.Encode()
}
