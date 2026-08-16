package http

import (
	"net/http"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	userSvc  service.UserService
	roleSvc  service.RoleService
	validate *validator.Validate
}

func NewUserHandler(userSvc service.UserService, roleSvc service.RoleService) *UserHandler {
	return &UserHandler{userSvc: userSvc, roleSvc: roleSvc, validate: validator.New()}
}

// callerID — JWT context-оос дуудагчийн ID. Байхгүй бол 0.
func callerID(c *gin.Context) domain.ID {
	raw, ok := c.Get(middleware.CtxUserID)
	if !ok {
		return 0
	}
	id, _ := raw.(domain.ID)
	return id
}

// isSelf — үйлдлийн бай нь дуудагч өөрөө эсэх.
func isSelf(c *gin.Context, target domain.ID) bool {
	caller := callerID(c)
	return caller != 0 && caller == target
}

// callerCanGrantRole — эрх нэмэгдүүлэлтээс сэргийлнэ: дуудагч зөвхөн өөрт нь
// бүрэн байгаа эрхийн багцтай роль-ыг л бусдад олгож чадна. (Админ бүх эрхтэй
// тул ямар ч роль олгоно.) roles:AssignPermission-ийн адил зарчим.
func (h *UserHandler) callerCanGrantRole(c *gin.Context, role *domain.Role) bool {
	if callerHasRole(c, "admin") && roleIsSystem10Only(role) {
		return true
	}
	for _, perm := range role.Permissions {
		if !callerHasPermission(c, perm.Name) {
			return false
		}
	}
	return true
}

func callerHasRole(c *gin.Context, roleName string) bool {
	rolesAny, ok := c.Get(middleware.CtxRoles)
	if !ok {
		return false
	}
	roles, ok := rolesAny.([]string)
	if !ok {
		return false
	}
	for _, role := range roles {
		if role == roleName {
			return true
		}
	}
	return false
}

func roleIsSystem10Only(role *domain.Role) bool {
	if role == nil || len(role.Permissions) == 0 {
		return false
	}
	for _, perm := range role.Permissions {
		switch perm.Resource {
		case "land", "compensation", "decision":
			continue
		default:
			return false
		}
	}
	return true
}

// callerOutranksTarget — дуудагч бүрэн эрхийн хувьд байг "хамардаг" эсэх, буюу
// байд байгаа эрх бүр дуудагчид ч байгаа эсэх.
//
// Яагаад: users:update эрх нь нууц үг сэргээх болон имэйл солих боломж өгдөг.
// Хоёулаа хэрэглэгчийн эзэмшлийг гартаа авах (account takeover) зам —
// нууц үгийг шууд тохируулах, эсвэл имэйлээ өөрийнхөөр сольж
// /auth/forgot-password-оор reset token хүлээн авах. Иймд зөвхөн users:update
// эрхтэй хэрэглэгч админыг гартаа авч эрхээ нэмэгдүүлэхээс сэргийлж, өөрөөсөө
// өндөр эрхтэй бүртгэлийг өөрчлөхийг хориглоно.
func (h *UserHandler) callerOutranksTarget(c *gin.Context, targetID domain.ID) (bool, error) {
	targetPerms, err := h.userSvc.GetPermissions(c.Request.Context(), targetID)
	if err != nil {
		return false, err
	}
	for _, perm := range targetPerms {
		if !callerHasPermission(c, perm) {
			return false, nil
		}
	}
	return true, nil
}

// GetMe godoc
// @Summary      Get own profile
// @Description  Returns the authenticated user's profile
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dataResponse{data=userResponse}
// @Failure      401  {object}  errorResponse
// @Router       /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get(middleware.CtxUserID)
	id, _ := userID.(domain.ID)

	user, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// List godoc
// @Summary      List users
// @Description  Returns a paginated list of users. Requires users:read permission.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        page      query  int     false  "Page number"       default(1)
// @Param        page_size query  int     false  "Items per page"    default(20)
// @Param        search    query  string  false  "Search by email/name"
// @Param        role      query  string  false  "Filter by role name"
// @Param        is_active query  bool    false  "Filter by active status"
// @Success      200  {object}  paginatedUsersResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Router       /users [get]
func (h *UserHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	filter := domain.UserFilter{
		Search:   c.Query("search"),
		Role:     c.Query("role"),
		Page:     page,
		PageSize: pageSize,
	}
	if activeStr := c.Query("is_active"); activeStr != "" {
		active := activeStr == "true"
		filter.IsActive = &active
	}

	users, total, err := h.userSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, users, total, page, pageSize)
}

// GetByID godoc
// @Summary      Get user by ID
// @Description  Returns a single user. Requires users:read permission.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse "Invalid UUID"
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, okID := parseID(c.Param("id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	user, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// Create godoc
// @Summary      Create user
// @Description  Create a new user. Requires users:create permission.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body createUserBody true "User payload"
// @Success      201  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      409  {object}  errorResponse "Email already exists"
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username   string        `json:"username" validate:"required"`
		Email      string        `json:"email"`
		Password   string        `json:"password" validate:"required,min=8"`
		FirstName  string        `json:"first_name"`
		LastName   string        `json:"last_name"`
		Position   string        `json:"position"`
		EmployeeID *domain.ID    `json:"employee_id"`
		Employee   *employeeBody `json:"employee"`
		IsActive   *bool         `json:"is_active"`
		RoleIDs    []string      `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}

	// Ролиудыг хэрэглэгч үүсгэхээс ӨМНӨ шалгана — эрхгүй роль дурдсан хүсэлт
	// хагас биелж (хэрэглэгч роль-гүй үүсээд) үлдэхээс сэргийлнэ.
	roles := make([]*domain.Role, 0, len(req.RoleIDs))
	for _, raw := range req.RoleIDs {
		roleID, okID := parseID(raw)
		if !okID {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return
		}
		role, err := h.roleSvc.GetByID(c.Request.Context(), roleID)
		if err != nil {
			response.Error(c, err)
			return
		}
		if !h.callerCanGrantRole(c, role) {
			response.Error(c, domain.ErrForbidden)
			return
		}
		roles = append(roles, role)
	}

	var employeeReq *service.CreateEmployeeRequest
	if req.Employee != nil {
		converted, err := employeeReqFromBody(*req.Employee)
		if err != nil {
			response.AbortValidation(c, err)
			return
		}
		employeeReq = &converted
	}
	user, err := h.userSvc.Create(c.Request.Context(), service.CreateUserRequest{
		Username:   req.Username,
		Email:      req.Email,
		Password:   req.Password,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Position:   req.Position,
		EmployeeID: req.EmployeeID,
		Employee:   employeeReq,
		IsActive:   req.IsActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	for _, role := range roles {
		if err := h.roleSvc.AssignToUser(c.Request.Context(), user.ID, role.ID); err != nil {
			response.Error(c, err)
			return
		}
		user.Roles = append(user.Roles, *role)
	}
	response.Created(c, user)
}

// Update godoc
// @Summary      Update user
// @Description  Update user fields. Requires users:update permission.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string          true  "User UUID"
// @Param        body body      updateUserBody  true  "Fields to update"
// @Success      200  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, okID := parseID(c.Param("id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Position  string `json:"position"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	// Өөрийгөө идэвхгүй болгохыг хориглоно — админ өөрөө системээс гарцаагүй
	// хаагдах (lockout) тохиолдлоос сэргийлнэ.
	if req.IsActive != nil && !*req.IsActive && isSelf(c, id) {
		response.Error(c, domain.ErrForbidden)
		return
	}
	// ИМЭЙЛ СОЛИХ нь /auth/forgot-password-оор бүртгэлийг гартаа авах зам тул
	// зөвхөн имэйл ҮНЭХЭЭР өөрчлөгдөж байвал "эрхээр хамрах" шалгалт хийнэ.
	// Нэр, албан тушаал, идэвхтэй эсэхийг засах нь эрх нэмэгдүүлэхгүй — иймд
	// хэрэглэгч удирдах эрхтэй ажилтан тэднийг чөлөөтэй засна.
	if req.Email != "" && !isSelf(c, id) {
		current, err := h.userSvc.GetByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, err)
			return
		}
		if !strings.EqualFold(strings.TrimSpace(req.Email), current.Email) {
			outranks, err := h.callerOutranksTarget(c, id)
			if err != nil {
				response.Error(c, err)
				return
			}
			if !outranks {
				response.Error(c, domain.ErrForbidden)
				return
			}
		}
	}
	user, err := h.userSvc.Update(c.Request.Context(), id, service.UpdateUserRequest{
		Username:  req.Username,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Position:  req.Position,
		IsActive:  req.IsActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// Delete godoc
// @Summary      Delete user (soft)
// @Description  Soft-deletes a user by setting deleted_at. Requires users:delete permission.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, okID := parseID(c.Param("id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	// Өөрийгөө устгахыг хориглоно (админ өөрийгөө хаах эрсдэл).
	if isSelf(c, id) {
		response.Error(c, domain.ErrForbidden)
		return
	}
	if err := h.userSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// ChangePassword godoc
// @Summary      Reset a user's password
// @Description  Admin sets a new password for the user. Requires users:update permission.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string                 true  "User UUID"
// @Param        body body      changePasswordBody     true  "New password"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id}/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, okID := parseID(c.Param("id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req struct {
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	// Өөрөөсөө өндөр эрхтэй хэрэглэгчийн нууц үгийг сэргээхийг хориглоно —
	// эс тэгвээс зөвхөн users:update эрхтэй хэрэглэгч админаар нэвтрэх боломжтой.
	if !isSelf(c, id) {
		outranks, err := h.callerOutranksTarget(c, id)
		if err != nil {
			response.Error(c, err)
			return
		}
		if !outranks {
			response.Error(c, domain.ErrForbidden)
			return
		}
	}
	if err := h.userSvc.ChangePassword(c.Request.Context(), id, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	l := i18n.Detect(c.GetHeader("Accept-Language"))
	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: i18n.T(l, i18n.MsgUpdated),
	})
}

// AssignRole godoc
// @Summary      Assign a role to a user
// @Description  Requires users:update permission. Caller can only grant roles whose permissions the caller already holds.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string           true  "User UUID"
// @Param        body body      assignRoleBody   true  "Role to assign"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id}/roles [post]
func (h *UserHandler) AssignRole(c *gin.Context) {
	userID, okID := parseID(c.Param("id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req struct {
		RoleID string `json:"role_id" validate:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	roleID, okID := parseID(req.RoleID)
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	// Өөрийн ролийг өөрөө өөрчлөхийг хориглоно — users:update эрхтэй хэрэглэгч
	// өөртөө admin роль олгож эрхээ нэмэгдүүлэхээс сэргийлнэ.
	if isSelf(c, userID) {
		response.Error(c, domain.ErrForbidden)
		return
	}
	role, err := h.roleSvc.GetByID(c.Request.Context(), roleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.callerCanGrantRole(c, role) {
		response.Error(c, domain.ErrForbidden)
		return
	}
	if err := h.roleSvc.AssignToUser(c.Request.Context(), userID, roleID); err != nil {
		response.Error(c, err)
		return
	}
	// Роль нь JWT-д агуулагддаг тул шинэ эрх шууд хүчинтэй болохын тулд
	// хэрэглэгчийн одоогийн токенуудыг хүчингүй болгоно.
	if err := h.userSvc.InvalidateSessions(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}
	l := i18n.Detect(c.GetHeader("Accept-Language"))
	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: i18n.T(l, i18n.MsgAssigned),
	})
}

// RemoveRole godoc
// @Summary      Remove a role from a user
// @Description  Requires users:update permission. Caller can only revoke roles whose permissions the caller already holds.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id      path  string  true  "User UUID"
// @Param        role_id path  string  true  "Role UUID"
// @Success      204
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Router       /users/{id}/roles/{role_id} [delete]
func (h *UserHandler) RemoveRole(c *gin.Context) {
	userID, okID := parseID(c.Param("id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	roleID, okID := parseID(c.Param("role_id"))
	if !okID {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	// Өөрөөсөө роль хуулахыг хориглоно (өөрийгөө хаахаас сэргийлнэ).
	if isSelf(c, userID) {
		response.Error(c, domain.ErrForbidden)
		return
	}
	role, err := h.roleSvc.GetByID(c.Request.Context(), roleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	// Олгох эрхгүй роль-ыг хураах эрх ч байхгүй — эс тэгвээс бага эрхтэй
	// хэрэглэгч админаас роль хуулж болно.
	if !h.callerCanGrantRole(c, role) {
		response.Error(c, domain.ErrForbidden)
		return
	}
	if err := h.roleSvc.RemoveFromUser(c.Request.Context(), userID, roleID); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.userSvc.InvalidateSessions(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
