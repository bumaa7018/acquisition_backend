package http

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/your-org/go-api-template/internal/middleware"
	"github.com/your-org/go-api-template/internal/pkg/jwt"
	"github.com/your-org/go-api-template/internal/service"
)

func NewRouter(
	db *pgxpool.Pool,
	jwtManager *jwt.Manager,
	authSvc service.AuthService,
	userSvc service.UserService,
	roleSvc service.RoleService,
	permSvc service.PermissionService,
	landSvc service.LandService,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(cors.Default())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	healthH := NewHealthHandler(db)
	r.GET("/health", healthH.Live)
	r.GET("/health/ready", healthH.Ready)

	authH := NewAuthHandler(authSvc)
	userH := NewUserHandler(userSvc)
	roleH := NewRoleHandler(roleSvc)
	permH := NewPermissionHandler(permSvc)
	landH := NewLandHandler(landSvc)

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.RefreshToken)
		auth.POST("/logout", authH.Logout)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)
	}

	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtManager))
	{
		users := protected.Group("/users")
		{
			users.GET("/me", userH.GetMe)
			users.GET("", middleware.RequirePermission("users:read"), userH.List)
			users.GET("/:id", middleware.RequirePermission("users:read"), userH.GetByID)
			users.POST("", middleware.RequirePermission("users:create"), userH.Create)
			users.PUT("/:id", middleware.RequirePermission("users:update"), userH.Update)
			users.DELETE("/:id", middleware.RequirePermission("users:delete"), userH.Delete)
		}

		roles := protected.Group("/roles")
		{
			roles.GET("", middleware.RequirePermission("roles:read"), roleH.List)
			roles.GET("/:id", middleware.RequirePermission("roles:read"), roleH.GetByID)
			roles.POST("", middleware.RequirePermission("roles:create"), roleH.Create)
			roles.PUT("/:id", middleware.RequirePermission("roles:update"), roleH.Update)
			roles.DELETE("/:id", middleware.RequirePermission("roles:update"), roleH.Delete)
			roles.POST("/:id/permissions", middleware.RequirePermission("roles:update"), roleH.AssignPermission)
			roles.DELETE("/:id/permissions/:permission_id", middleware.RequirePermission("roles:update"), roleH.RemovePermission)
		}

		permissions := protected.Group("/permissions")
		{
			permissions.GET("", middleware.RequirePermission("permissions:read"), permH.List)
			permissions.POST("", middleware.RequirePermission("permissions:create"), permH.Create)
			permissions.DELETE("/:id", middleware.RequirePermission("permissions:delete"), permH.Delete)
		}

		// Газар чөлөөлөлт
		land := protected.Group("/land-acquisitions")
		{
			land.GET("", middleware.RequirePermission("land:read"), landH.List)
			land.POST("", middleware.RequirePermission("land:create"), landH.Create)
			land.GET("/:id", middleware.RequirePermission("land:read"), landH.GetByID)
			land.GET("/:id/parcels", middleware.RequirePermission("land:read"), landH.ListParcels)
			land.GET("/:id/parcels/:parcel_id", middleware.RequirePermission("land:read"), landH.GetParcelByID)
			land.PUT("/:id", middleware.RequirePermission("land:update"), landH.Update)
			land.DELETE("/:id", middleware.RequirePermission("land:delete"), landH.Delete)
			land.POST("/:id/parcels/:parcel_id/sync", middleware.RequirePermission("land:create"), landH.SyncParcel)
		}
	}

	return r
}
