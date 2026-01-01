package course

import "time"

type Team struct {
	ID        int64     `json:"id"`
	CourseID  int64     `json:"course_id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type CreateTeamInput struct {
	CourseID int64  	`json:"course_id,omitempty"`
	Name     string 	`json:"name,omitempty"`
	Status   string 	`json:"status,omitempty"`
}

type UpdateTeamInput struct {
	Name   *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

