package integration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/aidar/avito-pr-project/internal/app"
	"github.com/aidar/avito-pr-project/internal/config"
)

// TestEnvironment содержит все ресурсы необходимые для интеграционных тестов
type TestEnvironment struct {
	PostgresContainer *postgres.PostgresContainer
	App               *app.App
	BaseURL           string
	DB                *pgxpool.Pool
	ctx               context.Context
}

// SetupTestEnvironment создает и инициализирует полное тестовое окружение
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()
	ctx := context.Background()

	// Запускаем PostgreSQL контейнер
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("pr_service_test"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Получаем строку подключения
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Применяем миграции
	applyMigrations(t, connStr)

	// Парсим строку подключения для получения компонентов
	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Создаем конфигурацию для приложения
	// Используем высокий порт для тестов чтобы избежать конфликтов
	testPort := "18080"
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: testPort,
			Host: "127.0.0.1",
		},
		Database: config.DatabaseConfig{
			Host:     host,
			Port:     port.Port(),
			User:     "test_user",
			Password: "test_password",
			Name:     "pr_service_test",
			SSLMode:  "disable",
			MaxConns: 25,
			MinConns: 5,
		},
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key-for-integration-tests",
			ExpirationHours: 24,
		},
	}

	// Создаем и инициализируем приложение
	application, err := app.New(cfg)
	require.NoError(t, err, "Failed to create application")

	err = application.Initialize(ctx)
	require.NoError(t, err, "Failed to initialize application")

	// Запускаем сервер в фоне
	serverStarted := make(chan bool, 1)
	go func() {
		serverStarted <- true
		if err := application.Run(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Ждем запуска сервера
	<-serverStarted
	time.Sleep(500 * time.Millisecond)

	// Создаем базовый URL с тестовым портом
	baseURL := fmt.Sprintf("http://%s:%s", cfg.Server.Host, testPort)

	// Создаем подключение к БД для прямых запросов в тестах
	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	return &TestEnvironment{
		PostgresContainer: pgContainer,
		App:               application,
		BaseURL:           baseURL,
		DB:                pool,
		ctx:               ctx,
	}
}

// Cleanup очищает все тестовые ресурсы
func (te *TestEnvironment) Cleanup(t *testing.T) {
	t.Helper()

	// Останавливаем приложение
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if te.App != nil {
		_ = te.App.Shutdown(shutdownCtx)
	}

	// Закрываем подключение к БД
	if te.DB != nil {
		te.DB.Close()
	}

	// Останавливаем PostgreSQL контейнер
	if te.PostgresContainer != nil {
		_ = te.PostgresContainer.Terminate(te.ctx)
	}
}

// applyMigrations применяет миграции БД
func applyMigrations(t *testing.T, connStr string) {
	t.Helper()

	// Открываем подключение к БД
	db, err := sql.Open("pgx/v5", connStr)
	require.NoError(t, err, "Failed to open database connection")
	defer db.Close()

	// Читаем файл миграции
	projectRoot := getProjectRoot(t)
	migrationPath := filepath.Join(projectRoot, "migrations", "000001_init_schema.up.sql")

	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err, "Failed to read migration file")

	// Выполняем миграцию
	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err, "Failed to apply migration")

	t.Log("Migrations applied successfully")
}

// getProjectRoot возвращает корневую директорию проекта
func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Поднимаемся по директориям пока не найдем go.mod
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

// MakeRequest вспомогательная функция для HTTP запросов в тестах
func (te *TestEnvironment) MakeRequest(t *testing.T, method, path string, body io.Reader, token string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, te.BaseURL+path, body)
	require.NoError(t, err, "Failed to create request")

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to make request")

	return resp
}

// WaitForHealthCheck ждет пока приложение станет доступным
func (te *TestEnvironment) WaitForHealthCheck(t *testing.T) {
	t.Helper()

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(te.BaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("Application did not become healthy in time")
}
