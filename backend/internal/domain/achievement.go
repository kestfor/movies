package domain

import "time"

type AchievementProgress struct {
	Value  int64 `json:"value"`
	Target int64 `json:"target"`
}

type Achievement struct {
	AwardID     string               `json:"award_id,omitempty"`
	Code        string               `json:"code,omitempty"`
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	Category    string               `json:"category,omitempty"`
	Icon        string               `json:"icon,omitempty"`
	XP          int                  `json:"xp,omitempty"`
	Secret      bool                 `json:"secret"`
	Unlocked    bool                 `json:"unlocked"`
	SortOrder   int                  `json:"sort_order"`
	EarnedAt    *time.Time           `json:"earned_at,omitempty"`
	Progress    *AchievementProgress `json:"progress,omitempty"`
}

type GamificationSummary struct {
	TotalXP         int    `json:"total_xp"`
	Level           int    `json:"level"`
	RankTitle       string `json:"rank_title"`
	CurrentLevelXP  int    `json:"current_level_xp"`
	NextLevelXP     int    `json:"next_level_xp"`
	UnlockedCount   int    `json:"unlocked_count"`
	TotalCount      int    `json:"total_count"`
	LeaderboardRank int    `json:"leaderboard_rank,omitempty"`
}

type AchievementsPage struct {
	User         User                `json:"user"`
	Relationship string              `json:"relationship"`
	Summary      GamificationSummary `json:"summary"`
	Achievements []Achievement       `json:"achievements"`
}

type LeaderboardEntry struct {
	Rank          int  `json:"rank"`
	User          User `json:"user"`
	TotalXP       int  `json:"total_xp"`
	Level         int  `json:"level"`
	UnlockedCount int  `json:"unlocked_count"`
}

type Leaderboard struct {
	Items []LeaderboardEntry `json:"items"`
}

type UnseenAchievements struct {
	Items            []Achievement `json:"items"`
	BackfillCount    int           `json:"backfill_count"`
	BackfillAwardIDs []string      `json:"backfill_award_ids,omitempty"`
}

type NotificationAchievement struct {
	AwardID string `json:"award_id"`
	Title   string `json:"title,omitempty"`
	Secret  bool   `json:"secret"`
}
