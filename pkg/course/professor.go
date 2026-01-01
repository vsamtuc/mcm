package course

import "time"

// Professor represents a user that can be assigned to teach courses.
type Professor struct {
	ID         int64     `json:"id"`
	KeycloakID string    `json:"keycloak_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Email      string    `json:"email,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
