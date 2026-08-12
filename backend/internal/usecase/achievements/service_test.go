package achievements

import (
	"testing"
	"time"
)

func TestVisibleFriendAchievementsKeepsUnknownSecretOpaque(t *testing.T) {
	secret := testAchievementDefinition("secret", true, 1)
	public := testAchievementDefinition("public", false, 2)
	service := &Service{definitions: []Definition{secret, public}}
	earnedAt := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	targetAwards := []StoredAward{
		{ID: "target-secret-award", Code: secret.Code, EarnedAt: earnedAt},
		{ID: "target-public-award", Code: public.Code, EarnedAt: earnedAt},
	}

	items := service.visibleAchievements("friend", targetAwards, nil, nil)

	if len(items) != 2 {
		t.Fatalf("achievements = %d, want 2", len(items))
	}
	hidden := items[0]
	if hidden.AwardID != "target-secret-award" || !hidden.Secret || !hidden.Unlocked {
		t.Fatalf("hidden secret metadata = %+v", hidden)
	}
	if hidden.Code != "" || hidden.Title != "" || hidden.Description != "" || hidden.Icon != "" || hidden.XP != 0 || hidden.Progress != nil {
		t.Fatalf("unknown secret leaked details: %+v", hidden)
	}
	visible := items[1]
	if visible.Code != public.Code || visible.Title != public.Title || visible.AwardID != "target-public-award" {
		t.Fatalf("public achievement = %+v", visible)
	}
}

func TestVisibleFriendAchievementsRevealsSharedSecret(t *testing.T) {
	secret := testAchievementDefinition("shared-secret", true, 1)
	service := &Service{definitions: []Definition{secret}}
	targetEarnedAt := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	viewerEarnedAt := targetEarnedAt.Add(-24 * time.Hour)
	targetAwards := []StoredAward{{ID: "target-award", UserID: 2, Code: secret.Code, EarnedAt: targetEarnedAt}}
	viewerAwards := []StoredAward{{ID: "viewer-award", UserID: 1, Code: secret.Code, EarnedAt: viewerEarnedAt}}

	items := service.visibleAchievements("friend", targetAwards, viewerAwards, nil)

	if len(items) != 1 {
		t.Fatalf("achievements = %d, want 1", len(items))
	}
	item := items[0]
	if item.Code != secret.Code || item.Title != secret.Title || item.Description != secret.Description || item.Icon != secret.Icon || item.XP != secret.XP {
		t.Fatalf("shared secret details = %+v", item)
	}
	if item.AwardID != "target-award" {
		t.Fatalf("award_id = %q, want target award", item.AwardID)
	}
	if item.EarnedAt == nil || !item.EarnedAt.Equal(targetEarnedAt) {
		t.Fatalf("earned_at = %v, want target earned_at %v", item.EarnedAt, targetEarnedAt)
	}
	if !item.Secret || !item.Unlocked || item.Progress == nil || item.Progress.Value != secret.Target || item.Progress.Target != secret.Target {
		t.Fatalf("shared secret state = %+v", item)
	}
}

func testAchievementDefinition(code string, secret bool, sortOrder int) Definition {
	return Definition{
		Code: code, Metric: MetricRatingsTotal, Target: 3, XP: 350,
		Category: CategoryRatings, Secret: secret, Icon: "🎬",
		Title: "Тестовая ачивка", Description: "Выполнить тестовое условие",
		SortOrder: sortOrder, AwardPolicy: AwardPolicyLifetime,
	}
}
