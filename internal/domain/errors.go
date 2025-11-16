package domain

import "errors"

// Доменные ошибки согласно OpenAPI спецификации
var (
	// ErrTeamExists возвращается при попытке создать уже существующую команду
	ErrTeamExists = errors.New("team already exists")

	// ErrPRExists возвращается при попытке создать уже существующий PR
	ErrPRExists = errors.New("pull request already exists")

	// ErrPRMerged возвращается при попытке изменить смерженный PR
	ErrPRMerged = errors.New("cannot modify merged pull request")

	// ErrNotAssigned возвращается при попытке переназначить неназначенного ревьювера
	ErrNotAssigned = errors.New("reviewer is not assigned to this PR")

	// ErrNoCandidate возвращается когда нет доступных ревьюверов для назначения
	ErrNoCandidate = errors.New("no active replacement candidate in team")

	// ErrNotFound возвращается когда ресурс не найден
	ErrNotFound = errors.New("resource not found")

	// ErrUserNotFound возвращается когда пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrTeamNotFound возвращается когда команда не найдена
	ErrTeamNotFound = errors.New("team not found")

	// ErrPRNotFound возвращается когда PR не найден
	ErrPRNotFound = errors.New("pull request not found")

	// ErrUnauthorized возвращается при неудачной аутентификации
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidToken возвращается когда JWT токен невалиден
	ErrInvalidToken = errors.New("invalid token")
)

// ErrorCode представляет коды ошибок API из OpenAPI спецификации
type ErrorCode string

// Коды ошибок согласно OpenAPI спецификации
const (
	CodeTeamExists  ErrorCode = "TEAM_EXISTS"  // Команда уже существует
	CodePRExists    ErrorCode = "PR_EXISTS"    // Pull request уже существует
	CodePRMerged    ErrorCode = "PR_MERGED"    // Нельзя изменить смерженный PR
	CodeNotAssigned ErrorCode = "NOT_ASSIGNED" // Ревьювер не назначен
	CodeNoCandidate ErrorCode = "NO_CANDIDATE" // Нет активных кандидатов для замены
	CodeNotFound    ErrorCode = "NOT_FOUND"    // Ресурс не найден
)

// MapErrorToCode преобразует доменные ошибки в коды ошибок API
func MapErrorToCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrTeamExists):
		return CodeTeamExists
	case errors.Is(err, ErrPRExists):
		return CodePRExists
	case errors.Is(err, ErrPRMerged):
		return CodePRMerged
	case errors.Is(err, ErrNotAssigned):
		return CodeNotAssigned
	case errors.Is(err, ErrNoCandidate):
		return CodeNoCandidate
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrTeamNotFound), errors.Is(err, ErrPRNotFound):
		return CodeNotFound
	default:
		return CodeNotFound
	}
}
