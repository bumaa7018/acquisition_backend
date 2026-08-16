package domain

import (
	"time"

	"github.com/google/uuid"
)

// User — sdplatform.sd_user. ID нь user_id (serial4).
type User struct {
	ID         ID         `db:"user_id" json:"id"`
	Username   string     `db:"username" json:"username"`
	Email      string     `db:"email" json:"email"`
	Password   string     `db:"password" json:"-"`
	FirstName  string     `db:"firstname" json:"first_name"`
	LastName   string     `db:"lastname" json:"last_name"`
	Position   string     `json:"position"`
	EmployeeID *ID        `json:"employee_id,omitempty"`
	Employee   *Employee  `json:"employee,omitempty"`
	IsActive   bool       `db:"is_active" json:"is_active"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
	Roles      []Role     `json:"roles,omitempty"`
}

// PasswordResetToken — энэ системийн өөрийн хүснэгт (sdplatform.password_reset_tokens).
// Токены өөрийн ID нь uuid хэвээр; user_id нь sd_user.user_id-г заана.
type PasswordResetToken struct {
	ID        uuid.UUID `db:"id"`
	UserID    ID        `db:"user_id"`
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
	Used      bool      `db:"used"`
	CreatedAt time.Time `db:"created_at"`
}

type RevokedToken struct {
	JTI       string    `db:"jti"`
	UserID    ID        `db:"user_id"`
	TokenType string    `db:"token_type"`
	ExpiresAt time.Time `db:"expires_at"`
	RevokedAt time.Time `db:"revoked_at"`
}

type UserFilter struct {
	Search   string
	Role     string
	IsActive *bool
	Page     int
	PageSize int
}
