package domain

import "time"

type DocumentType struct {
	ID          int        `json:"id"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	UpdatedBy   *string    `json:"updated_by,omitempty"`
}
