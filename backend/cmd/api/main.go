package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"movies/backend/internal/clients/tmdb"
	"movies/backend/internal/config"
	postgresrepo "movies/backend/internal/repo/postgres"
	gen "movies/backend/internal/repo/postgres/gen"
	httptransport "movies/backend/internal/transport/http"
	usecaseauth "movies/backend/internal/usecase/auth"
	usecasecomments "movies/backend/internal/usecase/comments"
	usecasecriteria "movies/backend/internal/usecase/criteria"
	usecasefeed "movies/backend/internal/usecase/feed"
	usecasefriends "movies/backend/internal/usecase/friends"
	usecaseratings "movies/backend/internal/usecase/ratings"
	usecasetitles "movies/backend/internal/usecase/titles"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := gen.New(pool)
	userRepo := postgresrepo.NewUserRepository(queries)
	criteriaRepo := postgresrepo.NewCriteriaRepository(queries)
	authSvc := usecaseauth.NewService(userRepo, cfg.BotToken, cfg.AuthTTL)
	tmdbClient := tmdb.NewClient(cfg.TMDBBaseURL, cfg.TMDBToken, cfg.TMDBLanguage, nil, 15*time.Minute)
	criteriaSvc := usecasecriteria.NewService(criteriaRepo)
	ratingRepo := postgresrepo.NewRatingRepository(pool, queries)
	titleSvc := usecasetitles.NewService(tmdbClient, ratingRepo)
	ratingSvc := usecaseratings.NewService(ratingRepo, tmdbClient)
	commentRepo := postgresrepo.NewCommentRepository(pool, queries)
	commentSvc := usecasecomments.NewService(commentRepo, tmdbClient)
	friendRepo := postgresrepo.NewFriendRepository(queries)
	friendSvc := usecasefriends.NewService(friendRepo)
	feedRepo := postgresrepo.NewFeedRepository(queries)
	feedSvc := usecasefeed.NewService(feedRepo)

	router := httptransport.NewRouter(authSvc, userRepo, titleSvc, criteriaSvc, ratingSvc, commentSvc, friendSvc, feedSvc)
	logger.Info("starting api", "addr", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
