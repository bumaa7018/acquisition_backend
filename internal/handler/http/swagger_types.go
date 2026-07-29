// swagger:model
// Request / Response DTO types used only for Swagger documentation.
//
//nolint:unused
package http

type registerBody struct {
	Email     string `json:"email" example:"user@example.com"`
	Password  string `json:"password" example:"Password123!"`
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
}

type loginBody struct {
	Email    string `json:"email" example:"admin@example.com"`
	Password string `json:"password" example:"Admin123!"`
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type forgotPasswordBody struct {
	Email string `json:"email" example:"user@example.com"`
}

type resetPasswordBody struct {
	Token       string `json:"token" example:"abc123hex"`
	NewPassword string `json:"new_password" example:"NewPassword123!"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGci..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type messageResponse struct {
	Message string `json:"message" example:"operation successful"`
}

type createUserBody struct {
	Username  string   `json:"username" example:"jdoe"`
	Email     string   `json:"email" example:"user@example.com"`
	Password  string   `json:"password" example:"Password123!"`
	FirstName string   `json:"first_name" example:"John"`
	LastName  string   `json:"last_name" example:"Doe"`
	Position  string   `json:"position" example:"Ахлах мэргэжилтэн"`
	IsActive  bool     `json:"is_active" example:"true"`
	RoleIDs   []string `json:"role_ids" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type changePasswordBody struct {
	Password string `json:"password" example:"Password123!"`
}

type assignRoleBody struct {
	RoleID string `json:"role_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type updateUserBody struct {
	Email     string `json:"email" example:"user@example.com"`
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
	IsActive  *bool  `json:"is_active" example:"true"`
}

type createRoleBody struct {
	Name        string `json:"name" example:"editor"`
	Description string `json:"description" example:"Content editor"`
}

type updateRoleBody struct {
	Name        string `json:"name" example:"editor"`
	Description string `json:"description" example:"Updated description"`
}

type assignPermissionBody struct {
	PermissionID string `json:"permission_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type createPermissionBody struct {
	Name        string `json:"name" example:"reports:read"`
	Description string `json:"description" example:"Read reports"`
	Resource    string `json:"resource" example:"reports"`
	Action      string `json:"action" example:"read"`
}

// Generic API response wrappers

type dataResponse struct {
	Data any `json:"data"`
}

type errorResponse struct {
	Error string `json:"error" example:"not found"`
}

type paginatedUsersResponse struct {
	Data       []userResponse `json:"data"`
	Total      int64          `json:"total" example:"100"`
	Page       int            `json:"page" example:"1"`
	PageSize   int            `json:"page_size" example:"20"`
	TotalPages int64          `json:"total_pages" example:"5"`
}

type userResponse struct {
	ID        string  `json:"ID" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string  `json:"Email" example:"user@example.com"`
	FirstName string  `json:"FirstName" example:"John"`
	LastName  string  `json:"LastName" example:"Doe"`
	IsActive  bool    `json:"IsActive" example:"true"`
	CreatedAt string  `json:"CreatedAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt string  `json:"UpdatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt *string `json:"DeletedAt" example:"null"`
}

type roleResponse struct {
	ID          string               `json:"ID" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string               `json:"Name" example:"admin"`
	Description string               `json:"Description" example:"System administrator"`
	CreatedAt   string               `json:"CreatedAt" example:"2026-01-01T00:00:00Z"`
	Permissions []permissionResponse `json:"Permissions"`
}

type permissionResponse struct {
	ID          string `json:"ID" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"Name" example:"users:read"`
	Description string `json:"Description" example:"Read users"`
	Resource    string `json:"Resource" example:"users"`
	Action      string `json:"Action" example:"read"`
	CreatedAt   string `json:"CreatedAt" example:"2026-01-01T00:00:00Z"`
}
