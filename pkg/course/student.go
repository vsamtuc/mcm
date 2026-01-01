package course

import "time"

type Student struct {
	ID           int64     `json:"id"`
	KeycloakID   string    `json:"keycloak_id,omitempty"`
	UniversityID string    `json:"university_id,omitempty"`
	FullName     string    `json:"full_name,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

