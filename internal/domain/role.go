package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID    `db:"id"          json:"id"`
	Name        string       `db:"name"        json:"name"`
	Description string       `db:"description" json:"description"`
	CreatedAt   time.Time    `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"  json:"updated_at"`
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Description string    `db:"description" json:"description"`
	Resource    string    `db:"resource"    json:"resource"`
	Action      string    `db:"action"      json:"action"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}
