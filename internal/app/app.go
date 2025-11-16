package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aidar/avito-pr-project/internal/config"
	"github.com/aidar/avito-pr-project/internal/handler"
	"github.com/aidar/avito-pr-project/internal/middleware"
	"github.com/aidar/avito-pr-project/internal/repository/postgres"
	"github.com/aidar/avito-pr-project/internal/service"
)

// App представляет приложение со всеми зависимостями
type App struct {
	config *config.Config
	db     *pgxpool.Pool
	server *http.Server
	logger *slog.Logger
}

// New создает новый экземпляр приложения
func New(cfg *config.Config) (*App, error) {
	// Инициализируем структурированный логгер (JSON формат)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app := &App{
		config: cfg,
		logger: logger,
	}

	return app, nil
}

// Initialize инициализирует все компоненты приложения
func (a *App) Initialize(ctx context.Context) error {
	// Подключаемся к базе данных
	if err := a.connectDB(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Настраиваем HTTP сервер и роутинг
	a.setupServer()

	a.logger.Info("Application initialized successfully")
	return nil
}

// connectDB устанавливает подключение к PostgreSQL с connection pool
func (a *App) connectDB(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(a.config.Database.DSN())
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	// Настраиваем размеры connection pool
	poolConfig.MaxConns = a.config.Database.MaxConns
	poolConfig.MinConns = a.config.Database.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем подключение к БД
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = pool
	a.logger.Info("Connected to database")
	return nil
}

// setupServer инициализирует HTTP роутер и обработчики
func (a *App) setupServer() {
	// Инициализируем слой репозиториев (работа с БД)
	userRepo := postgres.NewUserRepository(a.db)
	teamRepo := postgres.NewTeamRepository(a.db)
	prRepo := postgres.NewPullRequestRepository(a.db)

	// Инициализируем слой сервисов (бизнес-логика)
	reviewerSelector := service.NewReviewerSelector()
	userService := service.NewUserService(userRepo)
	teamService := service.NewTeamService(teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, reviewerSelector)
	authService := service.NewAuthService(
		userRepo,
		a.config.JWT.Secret,
		a.config.JWT.GetExpiration(),
	)
	statsService := service.NewStatsService(a.db)

	// Инициализируем HTTP обработчики
	authHandler := handler.NewAuthHandler(authService)
	teamHandler := handler.NewTeamHandler(teamService)
	userHandler := handler.NewUserHandler(userService, prService)
	prHandler := handler.NewPullRequestHandler(prService)
	statsHandler := handler.NewStatsHandler(statsService)

	// Инициализируем middleware для JWT авторизации
	authMiddleware := middleware.AuthMiddleware(authService)

	// Настраиваем роутер
	r := chi.NewRouter()

	// Глобальные middleware (применяются ко всем запросам)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Публичные эндпоинты (без авторизации)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
	})

	// Health check для мониторинга
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			a.logger.Error("Failed to write health check response", "error", err)
		}
	})

	// Создание команды доступно без токена (для начальной настройки)
	// В production рекомендуется защитить или использовать seed-скрипт
	r.Post("/team/add", teamHandler.AddTeam)

	// Защищенные эндпоинты (требуют JWT токен в заголовке Authorization)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		// Эндпоинты команд
		r.Get("/team/get", teamHandler.GetTeam)

		// Эндпоинты пользователей
		r.Post("/users/setIsActive", userHandler.SetIsActive)
		r.Get("/users/getReview", userHandler.GetReview)

		// Эндпоинты Pull Request'ов
		r.Post("/pullRequest/create", prHandler.CreatePR)
		r.Post("/pullRequest/merge", prHandler.MergePR)
		r.Post("/pullRequest/reassign", prHandler.Reassign)

		// Эндпоинты статистики (дополнительное задание)
		r.Get("/stats", statsHandler.GetStats)
		r.Get("/stats/user", statsHandler.GetUserStats)
	})

	// Создаем HTTP сервер с настройками таймаутов
	addr := fmt.Sprintf("%s:%s", a.config.Server.Host, a.config.Server.Port)
	a.server = &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	a.logger.Info("HTTP server configured", "addr", addr)
}

// Run запускает HTTP сервер
func (a *App) Run() error {
	a.logger.Info("Starting HTTP server", "addr", a.server.Addr)
	return a.server.ListenAndServe()
}

// Shutdown корректно останавливает приложение
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("Shutting down application")

	// Останавливаем HTTP сервер (ждем завершения текущих запросов)
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	// Закрываем подключения к базе данных
	if a.db != nil {
		a.db.Close()
	}

	a.logger.Info("Application stopped gracefully")
	return nil
}
