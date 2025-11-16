package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тестовые структуры данных соответствующие API
type Team struct {
	TeamName string   `json:"team_name"`
	Members  []Member `json:"members"`
}

type Member struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type LoginRequest struct {
	UserID string `json:"user_id"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type PullRequestResponse struct {
	PullRequestID   string   `json:"pull_request_id"`
	PullRequestName string   `json:"pull_request_name"`
	AuthorID        string   `json:"author_id"`
	Status          string   `json:"status"`
	Reviewers       []string `json:"assigned_reviewers"`
}

type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_user_id"`
}

type SetIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

// TestE2E_CompleteWorkflow тестирует полный workflow сервиса PR
func TestE2E_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Настраиваем тестовое окружение
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	// Ждем пока приложение будет готово
	env.WaitForHealthCheck(t)

	t.Run("Create Team with Members", func(t *testing.T) {
		team := Team{
			TeamName: "backend-team",
			Members: []Member{
				{UserID: "user1", Username: "Alice", IsActive: true},
				{UserID: "user2", Username: "Bob", IsActive: true},
				{UserID: "user3", Username: "Charlie", IsActive: true},
				{UserID: "user4", Username: "David", IsActive: true},
			},
		}

		body, _ := json.Marshal(team)
		resp := env.MakeRequest(t, http.MethodPost, "/team/add", bytes.NewReader(body), "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Team creation should succeed")
	})

	// Логин как user1 для получения токена
	var token string
	t.Run("Login as User", func(t *testing.T) {
		loginReq := LoginRequest{UserID: "user1"}
		body, _ := json.Marshal(loginReq)

		resp := env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Login should succeed")

		var loginResp LoginResponse
		err := json.NewDecoder(resp.Body).Decode(&loginResp)
		require.NoError(t, err)
		require.NotEmpty(t, loginResp.Token)

		token = loginResp.Token
	})

	// Создание Pull Request
	var pr PullRequestResponse
	t.Run("Create Pull Request", func(t *testing.T) {
		createPR := CreatePRRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add new feature",
			AuthorID:        "user1",
		}

		body, _ := json.Marshal(createPR)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/create", bytes.NewReader(body), token)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode, "PR creation should succeed")

		var createResp struct {
			PR PullRequestResponse `json:"pr"`
		}
		err := json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		pr = createResp.PR

		assert.Equal(t, "pr-1", pr.PullRequestID)
		assert.Equal(t, "Add new feature", pr.PullRequestName)
		assert.Equal(t, "user1", pr.AuthorID)
		assert.Equal(t, "OPEN", pr.Status)
		assert.LessOrEqual(t, len(pr.Reviewers), 2, "Should have at most 2 reviewers")
		assert.Greater(t, len(pr.Reviewers), 0, "Should have at least 1 reviewer")

		// Проверяем, что автор не является ревьюером
		for _, reviewer := range pr.Reviewers {
			assert.NotEqual(t, "user1", reviewer, "Author should not be a reviewer")
		}
	})

	// Получение ревью для пользователя
	t.Run("Get User Reviews", func(t *testing.T) {
		// Проверяем, что есть хотя бы один ревьюер
		require.Greater(t, len(pr.Reviewers), 0, "PR должен иметь хотя бы одного ревьюера")

		// Логин как один из ревьюеров
		reviewerID := pr.Reviewers[0]

		loginReq := LoginRequest{UserID: reviewerID}
		body, _ := json.Marshal(loginReq)

		resp := env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
		defer resp.Body.Close()

		var loginResp LoginResponse
		_ = json.NewDecoder(resp.Body).Decode(&loginResp)
		reviewerToken := loginResp.Token

		// Получение ревью
		resp = env.MakeRequest(t, http.MethodGet, "/users/getReview?user_id="+reviewerID, nil, reviewerToken)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var reviewResp struct {
			UserID       string                `json:"user_id"`
			PullRequests []PullRequestResponse `json:"pull_requests"`
		}
		err := json.NewDecoder(resp.Body).Decode(&reviewResp)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(reviewResp.PullRequests), 1, "Should have at least 1 review assigned")
	})

	// Переназначение ревьюера
	t.Run("Reassign Reviewer", func(t *testing.T) {
		if len(pr.Reviewers) == 0 {
			t.Skip("No reviewers to reassign")
		}

		oldReviewer := pr.Reviewers[0]
		reassignReq := ReassignRequest{
			PullRequestID: "pr-1",
			OldReviewerID: oldReviewer,
		}

		body, _ := json.Marshal(reassignReq)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Reassignment should succeed")

		var updatedPR PullRequestResponse
		err := json.NewDecoder(resp.Body).Decode(&updatedPR)
		require.NoError(t, err)

		// Проверяем, что старый ревьюер больше не в списке
		found := false
		for _, reviewer := range updatedPR.Reviewers {
			if reviewer == oldReviewer {
				found = true
				break
			}
		}
		assert.False(t, found, "Old reviewer should be replaced")
	})

	// Слияние PR
	t.Run("Merge Pull Request", func(t *testing.T) {
		mergeReq := map[string]string{
			"pull_request_id": "pr-1",
		}

		body, _ := json.Marshal(mergeReq)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/merge", bytes.NewReader(body), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Merge should succeed")

		var mergeResp struct {
			PR PullRequestResponse `json:"pr"`
		}
		err := json.NewDecoder(resp.Body).Decode(&mergeResp)
		require.NoError(t, err)

		assert.Equal(t, "MERGED", mergeResp.PR.Status)
	})

	// Попытка переназначить после слияния (должна завершиться неудачей)
	t.Run("Cannot Reassign After Merge", func(t *testing.T) {
		if len(pr.Reviewers) == 0 {
			t.Skip("No reviewers to test reassignment")
			return
		}

		reassignReq := ReassignRequest{
			PullRequestID: "pr-1",
			OldReviewerID: pr.Reviewers[0],
		}

		body, _ := json.Marshal(reassignReq)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode, "Should not allow reassignment after merge")
	})
}

// TestE2E_UserActivation тестирует сценарии активации/деактивации пользователей
func TestE2E_UserActivation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.WaitForHealthCheck(t)

	// Создание команды
	team := Team{
		TeamName: "frontend-team",
		Members: []Member{
			{UserID: "fe-user1", Username: "Emma", IsActive: true},
			{UserID: "fe-user2", Username: "Frank", IsActive: true},
			{UserID: "fe-user3", Username: "Grace", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp := env.MakeRequest(t, http.MethodPost, "/team/add", bytes.NewReader(body), "")
	resp.Body.Close()

	// Логин как fe-user1
	loginReq := LoginRequest{UserID: "fe-user1"}
	body, _ = json.Marshal(loginReq)
	resp = env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()
	token := loginResp.Token

	t.Run("Deactivate User", func(t *testing.T) {
		setActiveReq := SetIsActiveRequest{
			UserID:   "fe-user2",
			IsActive: false,
		}

		body, _ := json.Marshal(setActiveReq)
		resp := env.MakeRequest(t, http.MethodPost, "/users/setIsActive", bytes.NewReader(body), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "User deactivation should succeed")
	})

	t.Run("Create PR After Deactivation", func(t *testing.T) {
		// Создание PR - fe-user2 не должен быть назначен, так как неактивен
		createPR := CreatePRRequest{
			PullRequestID:   "pr-fe-1",
			PullRequestName: "Frontend update",
			AuthorID:        "fe-user1",
		}

		body, _ := json.Marshal(createPR)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/create", bytes.NewReader(body), token)
		defer resp.Body.Close()

		var createResp struct {
			PR PullRequestResponse `json:"pr"`
		}
		err := json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		pr := createResp.PR

		// Проверяем, что fe-user2 не назначен
		for _, reviewer := range pr.Reviewers {
			assert.NotEqual(t, "fe-user2", reviewer, "Inactive user should not be assigned as reviewer")
		}
	})

	t.Run("Reactivate User", func(t *testing.T) {
		setActiveReq := SetIsActiveRequest{
			UserID:   "fe-user2",
			IsActive: true,
		}

		body, _ := json.Marshal(setActiveReq)
		resp := env.MakeRequest(t, http.MethodPost, "/users/setIsActive", bytes.NewReader(body), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "User reactivation should succeed")
	})
}

// TestE2E_MergeIdempotency тестирует идемпотентность операции merge
func TestE2E_MergeIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.WaitForHealthCheck(t)

	// Настройка
	team := Team{
		TeamName: "devops-team",
		Members: []Member{
			{UserID: "devops1", Username: "Henry", IsActive: true},
			{UserID: "devops2", Username: "Iris", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp := env.MakeRequest(t, http.MethodPost, "/team/add", bytes.NewReader(body), "")
	resp.Body.Close()

	// Логин
	loginReq := LoginRequest{UserID: "devops1"}
	body, _ = json.Marshal(loginReq)
	resp = env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()
	token := loginResp.Token

	// Создание PR
	createPR := CreatePRRequest{
		PullRequestID:   "pr-devops-1",
		PullRequestName: "Infrastructure update",
		AuthorID:        "devops1",
	}
	body, _ = json.Marshal(createPR)
	resp = env.MakeRequest(t, http.MethodPost, "/pullRequest/create", bytes.NewReader(body), token)
	resp.Body.Close()

	t.Run("First Merge", func(t *testing.T) {
		mergeReq := map[string]string{"pull_request_id": "pr-devops-1"}
		body, _ := json.Marshal(mergeReq)

		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/merge", bytes.NewReader(body), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var mergeResp struct {
			PR PullRequestResponse `json:"pr"`
		}
		json.NewDecoder(resp.Body).Decode(&mergeResp)
		assert.Equal(t, "MERGED", mergeResp.PR.Status)
	})

	t.Run("Second Merge (Idempotent)", func(t *testing.T) {
		mergeReq := map[string]string{"pull_request_id": "pr-devops-1"}
		body, _ := json.Marshal(mergeReq)

		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/merge", bytes.NewReader(body), token)
		defer resp.Body.Close()

		// Должно вернуть успех (идемпотентно)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var mergeResp struct {
			PR PullRequestResponse `json:"pr"`
		}
		json.NewDecoder(resp.Body).Decode(&mergeResp)
		assert.Equal(t, "MERGED", mergeResp.PR.Status)
	})
}

// TestE2E_GetTeam тестирует получение информации о команде
func TestE2E_GetTeam(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.WaitForHealthCheck(t)

	// Создание команды
	team := Team{
		TeamName: "qa-team",
		Members: []Member{
			{UserID: "qa1", Username: "Jack", IsActive: true},
			{UserID: "qa2", Username: "Kate", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp := env.MakeRequest(t, http.MethodPost, "/team/add", bytes.NewReader(body), "")
	resp.Body.Close()

	// Логин
	loginReq := LoginRequest{UserID: "qa1"}
	body, _ = json.Marshal(loginReq)
	resp = env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()
	token := loginResp.Token

	t.Run("Get Team", func(t *testing.T) {
		resp := env.MakeRequest(t, http.MethodGet, "/team/get?team_name=qa-team", nil, token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var retrievedTeam Team
		err := json.NewDecoder(resp.Body).Decode(&retrievedTeam)
		require.NoError(t, err)

		assert.Equal(t, "qa-team", retrievedTeam.TeamName)
		assert.Len(t, retrievedTeam.Members, 2)
	})
}

// TestE2E_Stats тестирует эндпоинты статистики
func TestE2E_Stats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.WaitForHealthCheck(t)

	// Создание команды и нескольких PR
	team := Team{
		TeamName: "data-team",
		Members: []Member{
			{UserID: "data1", Username: "Leo", IsActive: true},
			{UserID: "data2", Username: "Mia", IsActive: true},
			{UserID: "data3", Username: "Noah", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp := env.MakeRequest(t, http.MethodPost, "/team/add", bytes.NewReader(body), "")
	resp.Body.Close()

	// Логин
	loginReq := LoginRequest{UserID: "data1"}
	body, _ = json.Marshal(loginReq)
	resp = env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()
	token := loginResp.Token

	// Создание нескольких PR
	for i := 1; i <= 3; i++ {
		createPR := CreatePRRequest{
			PullRequestID:   "pr-data-" + string(rune('0'+i)),
			PullRequestName: "Data pipeline update",
			AuthorID:        "data1",
		}
		body, _ := json.Marshal(createPR)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/create", bytes.NewReader(body), token)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	t.Run("Get General Stats", func(t *testing.T) {
		resp := env.MakeRequest(t, http.MethodGet, "/stats", nil, token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var stats map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&stats)
		require.NoError(t, err)

		// Просто проверяем, что получили какую-то статистику
		assert.NotEmpty(t, stats)
	})

	t.Run("Get User Stats", func(t *testing.T) {
		resp := env.MakeRequest(t, http.MethodGet, "/stats/user?user_id=data1", nil, token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var userStats map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&userStats)
		require.NoError(t, err)

		// Проверяем, что получили статистику для конкретного пользователя
		assert.NotEmpty(t, userStats)
	})
}

// TestE2E_SmallTeam тестирует создание PR с командой меньше 2 активных участников
func TestE2E_SmallTeam(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.WaitForHealthCheck(t)

	// Создание команды только с 1 участником
	team := Team{
		TeamName: "solo-team",
		Members: []Member{
			{UserID: "solo1", Username: "Oliver", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp := env.MakeRequest(t, http.MethodPost, "/team/add", bytes.NewReader(body), "")
	resp.Body.Close()

	// Логин
	loginReq := LoginRequest{UserID: "solo1"}
	body, _ = json.Marshal(loginReq)
	resp = env.MakeRequest(t, http.MethodPost, "/auth/login", bytes.NewReader(body), "")
	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()
	token := loginResp.Token

	t.Run("Create PR with No Available Reviewers", func(t *testing.T) {
		createPR := CreatePRRequest{
			PullRequestID:   "pr-solo-1",
			PullRequestName: "Solo work",
			AuthorID:        "solo1",
		}

		body, _ := json.Marshal(createPR)
		resp := env.MakeRequest(t, http.MethodPost, "/pullRequest/create", bytes.NewReader(body), token)
		defer resp.Body.Close()

		// Должен успешно создать PR
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp struct {
			PR PullRequestResponse `json:"pr"`
		}
		err := json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		pr := createResp.PR

		// Должно быть 0 ревьюеров (никого нет, кроме автора)
		assert.Len(t, pr.Reviewers, 0, "Should have no reviewers when team has only author")
	})
}
