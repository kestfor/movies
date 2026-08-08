package comments

import (
	"context"
	"errors"
	"strings"

	"movies/backend/internal/domain"
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrNotFound   = errors.New("not_found")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
)

type Provider interface {
	Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error)
}

type AchievementObserver interface {
	ObserveCircle(ctx context.Context, userID int64)
}

type Repository interface {
	GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error)
	Get(ctx context.Context, id int64) (domain.Comment, bool, error)
	Create(ctx context.Context, params CreateCommentParams) (domain.Comment, error)
	ListByTitle(ctx context.Context, titleID int64) ([]domain.Comment, error)
	UpdateBody(ctx context.Context, id, userID int64, body string) (domain.Comment, error)
	SoftDelete(ctx context.Context, id, userID int64) (domain.Comment, error)
}

type CreateCommentParams struct {
	UserID   int64
	Title    domain.Title
	ParentID int64
	Body     string
}

type Service struct {
	repo         Repository
	provider     Provider
	achievements AchievementObserver
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) SetAchievementObserver(observer AchievementObserver) {
	s.achievements = observer
}

func (s *Service) List(ctx context.Context, mediaType domain.MediaType, tmdbID int64) ([]domain.Comment, error) {
	if !validTitleRef(mediaType, tmdbID) {
		return nil, ErrValidation
	}

	titleID, ok, err := s.repo.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []domain.Comment{}, nil
	}

	flat, err := s.repo.ListByTitle(ctx, titleID)
	if err != nil {
		return nil, err
	}
	return buildTree(flat), nil
}

func (s *Service) Create(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64, parentID int64, body string) (domain.Comment, error) {
	body, err := normalizeBody(body)
	if err != nil {
		return domain.Comment{}, err
	}
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return domain.Comment{}, ErrValidation
	}

	titleID, titleExists, err := s.repo.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil {
		return domain.Comment{}, err
	}

	if parentID != 0 {
		if !titleExists {
			return domain.Comment{}, ErrValidation
		}
		parent, ok, err := s.repo.Get(ctx, parentID)
		if err != nil {
			return domain.Comment{}, err
		}
		if !ok || parent.TitleID != titleID || parent.IsDeleted {
			return domain.Comment{}, ErrValidation
		}
	}

	title := domain.Title{TmdbID: tmdbID, MediaType: mediaType}
	if titleExists {
		title.ID = titleID
	} else {
		title, err = s.provider.Get(ctx, mediaType, tmdbID)
		if err != nil {
			return domain.Comment{}, err
		}
	}

	comment, err := s.repo.Create(ctx, CreateCommentParams{
		UserID:   userID,
		Title:    title,
		ParentID: parentID,
		Body:     body,
	})
	if err != nil {
		return domain.Comment{}, err
	}
	if s.achievements != nil {
		s.achievements.ObserveCircle(ctx, userID)
	}
	return comment, nil
}

func (s *Service) Update(ctx context.Context, userID, commentID int64, body string) (domain.Comment, error) {
	body, err := normalizeBody(body)
	if err != nil {
		return domain.Comment{}, err
	}
	if userID == 0 || commentID <= 0 {
		return domain.Comment{}, ErrValidation
	}

	comment, ok, err := s.repo.Get(ctx, commentID)
	if err != nil {
		return domain.Comment{}, err
	}
	if !ok {
		return domain.Comment{}, ErrNotFound
	}
	if comment.IsDeleted {
		return domain.Comment{}, ErrConflict
	}
	if commentUserID(comment) != userID {
		return domain.Comment{}, ErrForbidden
	}

	return s.repo.UpdateBody(ctx, commentID, userID, body)
}

func (s *Service) Delete(ctx context.Context, userID, commentID int64) (domain.Comment, error) {
	if userID == 0 || commentID <= 0 {
		return domain.Comment{}, ErrValidation
	}

	comment, ok, err := s.repo.Get(ctx, commentID)
	if err != nil {
		return domain.Comment{}, err
	}
	if !ok {
		return domain.Comment{}, ErrNotFound
	}
	if commentUserID(comment) != userID {
		return domain.Comment{}, ErrForbidden
	}

	return s.repo.SoftDelete(ctx, commentID, userID)
}

func normalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > 4000 {
		return "", ErrValidation
	}
	return body, nil
}

func validTitleRef(mediaType domain.MediaType, tmdbID int64) bool {
	return (mediaType == domain.MediaTypeMovie || mediaType == domain.MediaTypeTV) && tmdbID > 0
}

func buildTree(flat []domain.Comment) []domain.Comment {
	byID := make(map[int64]*domain.Comment, len(flat))
	nodes := make([]domain.Comment, len(flat))
	for i := range flat {
		nodes[i] = flat[i]
		nodes[i].Replies = nil
		byID[nodes[i].ID] = &nodes[i]
	}

	rootIDs := make([]int64, 0)
	for i := range nodes {
		comment := &nodes[i]
		if comment.ParentID == 0 {
			rootIDs = append(rootIDs, comment.ID)
			continue
		}
		parent, ok := byID[comment.ParentID]
		if !ok {
			rootIDs = append(rootIDs, comment.ID)
			continue
		}
		parent.Replies = append(parent.Replies, *comment)
	}

	roots := make([]domain.Comment, 0, len(rootIDs))
	for _, id := range rootIDs {
		roots = append(roots, *byID[id])
	}
	return roots
}

func commentUserID(comment domain.Comment) int64 {
	return comment.User.ID
}
