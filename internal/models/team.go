package models

type Team struct {
	TeamName string `json:"team_name"`
	Members  []User `json:"members"` // Теперь просто []User вместо []TeamMember
}
