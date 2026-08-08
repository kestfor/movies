package achievements

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"movies/backend/internal/domain"
)

const EvaluatorVersion = 1

var (
	ErrValidation = errors.New("validation_failed")
	ErrNotFound   = errors.New("not_found")
	ErrForbidden  = errors.New("forbidden")
)

type AwardSource string

const (
	AwardSourceLive      AwardSource = "live"
	AwardSourceBackfill  AwardSource = "backfill"
	AwardSourceReconcile AwardSource = "reconcile"
)

type StoredAward struct {
	ID        string
	UserID    int64
	Code      string
	XP        int
	EarnedAt  time.Time
	AwardedAt time.Time
	Source    AwardSource
	SeenAt    *time.Time
}

type SaveEvaluationParams struct {
	UserID      int64
	Evaluation  Evaluation
	Introduced  map[string]time.Time
	Source      AwardSource
	EvaluatedAt time.Time
}

type Repository interface {
	EnsureCatalog(ctx context.Context, definitions []Definition, fingerprint string, introducedAt time.Time) (map[string]time.Time, error)
	LoadSnapshot(ctx context.Context, userID int64) (Snapshot, error)
	SaveEvaluation(ctx context.Context, params SaveEvaluationParams) ([]StoredAward, error)
	GetUserByUUID(ctx context.Context, uuid string) (domain.User, bool, error)
	GetRelationship(ctx context.Context, viewerID, userID int64) (string, error)
	ListCircleUserIDs(ctx context.Context, userID int64) ([]int64, error)
	ListAwards(ctx context.Context, userID int64) ([]StoredAward, error)
	ListMetricValues(ctx context.Context, userID int64) (map[MetricCode]int64, error)
	ListLeaderboard(ctx context.Context, viewerID int64) ([]domain.LeaderboardEntry, error)
	ListUnseen(ctx context.Context, userID int64) ([]StoredAward, error)
	MarkSeen(ctx context.Context, userID int64, awardIDs []string) error
}

type Service struct {
	repo        Repository
	evaluator   *Evaluator
	definitions []Definition
	byCode      map[string]Definition
	fingerprint string
	logger      *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) (*Service, error) {
	definitions := Definitions()
	evaluator, err := NewEvaluator(definitions)
	if err != nil {
		return nil, err
	}
	fingerprint, err := CatalogFingerprint(definitions, EvaluatorVersion)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	byCode := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		byCode[definition.Code] = definition
	}
	return &Service{
		repo: repo, evaluator: evaluator, definitions: definitions,
		byCode: byCode, fingerprint: fingerprint, logger: logger,
	}, nil
}

func (s *Service) Fingerprint() string { return s.fingerprint }

func (s *Service) EnsureCatalog(ctx context.Context, introducedAt time.Time) (map[string]time.Time, error) {
	return s.repo.EnsureCatalog(ctx, s.definitions, s.fingerprint, introducedAt)
}

func (s *Service) EvaluateUser(ctx context.Context, userID int64, source AwardSource, evaluatedAt time.Time) ([]StoredAward, error) {
	if userID <= 0 {
		return nil, ErrValidation
	}
	if evaluatedAt.IsZero() {
		evaluatedAt = time.Now()
	}
	introduced, err := s.EnsureCatalog(ctx, evaluatedAt)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.repo.LoadSnapshot(ctx, userID)
	if err != nil {
		return nil, err
	}
	evaluation := s.evaluator.Evaluate(snapshot, evaluatedAt)
	return s.repo.SaveEvaluation(ctx, SaveEvaluationParams{
		UserID: userID, Evaluation: evaluation, Introduced: introduced,
		Source: source, EvaluatedAt: evaluatedAt,
	})
}

func (s *Service) ObserveCircle(ctx context.Context, userID int64) {
	if userID <= 0 {
		return
	}
	ids, err := s.repo.ListCircleUserIDs(ctx, userID)
	if err != nil {
		s.logger.Error("list achievement circle", "user_id", userID, "error", err)
		return
	}
	seen := make(map[int64]bool, len(ids)+1)
	ids = append(ids, userID)
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		if _, err := s.EvaluateUser(ctx, id, AwardSourceLive, time.Now()); err != nil {
			s.logger.Error("evaluate live achievements", "user_id", id, "trigger_user_id", userID, "error", err)
		}
	}
}

func (s *Service) GetByUUID(ctx context.Context, viewerID int64, rawUUID string) (domain.AchievementsPage, error) {
	if viewerID <= 0 || strings.TrimSpace(rawUUID) == "" {
		return domain.AchievementsPage{}, ErrValidation
	}
	target, ok, err := s.repo.GetUserByUUID(ctx, rawUUID)
	if err != nil {
		return domain.AchievementsPage{}, err
	}
	if !ok {
		return domain.AchievementsPage{}, ErrNotFound
	}
	relationship, err := s.repo.GetRelationship(ctx, viewerID, target.ID)
	if err != nil {
		return domain.AchievementsPage{}, err
	}
	if relationship != "self" && relationship != "friend" {
		return domain.AchievementsPage{}, ErrForbidden
	}
	if relationship == "self" {
		if _, err := s.EvaluateUser(ctx, target.ID, AwardSourceReconcile, time.Now()); err != nil {
			s.logger.Error("reconcile achievements on profile read", "user_id", target.ID, "error", err)
		}
	}
	awards, err := s.repo.ListAwards(ctx, target.ID)
	if err != nil {
		return domain.AchievementsPage{}, err
	}
	metrics, err := s.repo.ListMetricValues(ctx, target.ID)
	if err != nil {
		return domain.AchievementsPage{}, err
	}
	leaderboard, err := s.repo.ListLeaderboard(ctx, viewerID)
	if err != nil {
		return domain.AchievementsPage{}, err
	}
	return domain.AchievementsPage{
		User: target, Relationship: relationship,
		Summary:      s.summary(target.ID, awards, leaderboard),
		Achievements: s.visibleAchievements(relationship, awards, metrics),
	}, nil
}

func (s *Service) Leaderboard(ctx context.Context, userID int64) (domain.Leaderboard, error) {
	if userID <= 0 {
		return domain.Leaderboard{}, ErrValidation
	}
	items, err := s.repo.ListLeaderboard(ctx, userID)
	if err != nil {
		return domain.Leaderboard{}, err
	}
	for index := range items {
		items[index].Level = Level(items[index].TotalXP)
	}
	return domain.Leaderboard{Items: items}, nil
}

func (s *Service) Unseen(ctx context.Context, userID int64) (domain.UnseenAchievements, error) {
	if userID <= 0 {
		return domain.UnseenAchievements{}, ErrValidation
	}
	awards, err := s.repo.ListUnseen(ctx, userID)
	if err != nil {
		return domain.UnseenAchievements{}, err
	}
	result := domain.UnseenAchievements{Items: []domain.Achievement{}}
	for _, award := range awards {
		if award.Source == AwardSourceBackfill {
			result.BackfillCount++
			result.BackfillAwardIDs = append(result.BackfillAwardIDs, award.ID)
			continue
		}
		definition, ok := s.byCode[award.Code]
		if !ok {
			continue
		}
		result.Items = append(result.Items, toDomainAchievement(definition, &award, definition.Target))
	}
	return result, nil
}

func (s *Service) MarkSeen(ctx context.Context, userID int64, awardIDs []string) error {
	if userID <= 0 || len(awardIDs) == 0 || len(awardIDs) > 100 {
		return ErrValidation
	}
	unique := make([]string, 0, len(awardIDs))
	seen := make(map[string]bool)
	for _, id := range awardIDs {
		if !validUUID(id) {
			return ErrValidation
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return s.repo.MarkSeen(ctx, userID, unique)
}

func (s *Service) summary(userID int64, awards []StoredAward, leaderboard []domain.LeaderboardEntry) domain.GamificationSummary {
	totalXP := 0
	for _, award := range awards {
		totalXP += award.XP
	}
	level := Level(totalXP)
	summary := domain.GamificationSummary{
		TotalXP: totalXP, Level: level, RankTitle: RankTitle(level),
		CurrentLevelXP: RequiredXP(level), NextLevelXP: RequiredXP(level + 1),
		UnlockedCount: len(awards), TotalCount: len(s.definitions),
	}
	for _, item := range leaderboard {
		if item.User.ID == userID {
			summary.LeaderboardRank = item.Rank
			break
		}
	}
	return summary
}

func (s *Service) visibleAchievements(relationship string, awards []StoredAward, metrics map[MetricCode]int64) []domain.Achievement {
	byCode := make(map[string]StoredAward, len(awards))
	for _, award := range awards {
		byCode[award.Code] = award
	}
	if relationship == "friend" {
		result := make([]domain.Achievement, 0, len(awards))
		for _, definition := range s.definitions {
			award, ok := byCode[definition.Code]
			if !ok {
				continue
			}
			if definition.Secret {
				earnedAt := award.EarnedAt
				result = append(result, domain.Achievement{
					AwardID: award.ID, Secret: true, Unlocked: true,
					SortOrder: definition.SortOrder, EarnedAt: &earnedAt,
				})
				continue
			}
			result = append(result, toDomainAchievement(definition, &award, definition.Target))
		}
		return result
	}
	result := make([]domain.Achievement, 0, len(s.definitions))
	for _, definition := range s.definitions {
		award, unlocked := byCode[definition.Code]
		if definition.Secret && !unlocked {
			result = append(result, domain.Achievement{Secret: true, SortOrder: definition.SortOrder})
			continue
		}
		value := metrics[definition.Metric]
		if value > definition.Target {
			value = definition.Target
		}
		if unlocked {
			result = append(result, toDomainAchievement(definition, &award, value))
		} else {
			result = append(result, toDomainAchievement(definition, nil, value))
		}
	}
	return result
}

func toDomainAchievement(definition Definition, award *StoredAward, value int64) domain.Achievement {
	item := domain.Achievement{
		Code: definition.Code, Title: definition.Title, Description: definition.Description,
		Category: string(definition.Category), Icon: definition.Icon, XP: definition.XP,
		Secret: definition.Secret, SortOrder: definition.SortOrder,
		Progress: &domain.AchievementProgress{Value: value, Target: definition.Target},
	}
	if award != nil {
		earnedAt := award.EarnedAt
		item.AwardID = award.ID
		item.Unlocked = true
		item.EarnedAt = &earnedAt
		item.Progress.Value = definition.Target
	}
	return item
}

func validUUID(value string) bool {
	compact := strings.ReplaceAll(strings.TrimSpace(value), "-", "")
	if len(compact) != 32 {
		return false
	}
	_, err := hex.DecodeString(compact)
	return err == nil
}

func sortedAwardIDs(awards []StoredAward) []string {
	ids := make([]string, 0, len(awards))
	for _, award := range awards {
		ids = append(ids, award.ID)
	}
	sort.Strings(ids)
	return ids
}
