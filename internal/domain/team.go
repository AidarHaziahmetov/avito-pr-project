package domain

// Team представляет группу пользователей (команду)
type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}
