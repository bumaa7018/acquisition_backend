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
	planSvc service.PlanService,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(cors.Default())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Static("/mock-files", "uploads/mock-file-server")

	healthH := NewHealthHandler(db)
	r.GET("/health", healthH.Live)
	r.GET("/health/ready", healthH.Ready)

	authH := NewAuthHandler(authSvc)
	userH := NewUserHandler(userSvc)
	roleH := NewRoleHandler(roleSvc)
	permH := NewPermissionHandler(permSvc)
	landH := NewLandHandler(landSvc)
	planH := NewPlanHandler(planSvc)

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
			roles.DELETE("/:id", middleware.RequirePermission("roles:delete"), roleH.Delete)
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
			land.GET("/:id/buildings", middleware.RequirePermission("land:read"), landH.ListBuildings)
			land.POST("/:id/buildings", middleware.RequirePermission("land:create"), landH.CreateBuilding)
			land.PUT("/:id/buildings/:building_id", middleware.RequirePermission("land:update"), landH.UpdateBuilding)
			land.DELETE("/:id/buildings/:building_id", middleware.RequirePermission("land:update"), landH.DeleteBuilding)
			land.GET("/:id/compensations", middleware.RequirePermission("land:read"), landH.ListCompensations)
			land.POST("/:id/compensations", middleware.RequirePermission("land:update"), landH.CreateCompensation)
			land.PUT("/:id/compensations/:comp_id", middleware.RequirePermission("land:update"), landH.UpdateCompensation)
			land.DELETE("/:id/compensations/:comp_id", middleware.RequirePermission("land:update"), landH.DeleteCompensation)
			land.POST("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), landH.CreateCompensationGrant)
			land.PUT("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), landH.UpdateCompensationGrant)
			land.DELETE("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), landH.DeleteCompensationGrant)
			land.GET("/:id/parcels/:parcel_id", middleware.RequirePermission("land:read"), landH.GetParcelByID)
			land.PUT("/:id", middleware.RequirePermission("land:update"), landH.Update)
			land.DELETE("/:id", middleware.RequirePermission("land:delete"), landH.Delete)
			land.POST("/:id/parcels/:parcel_id/sync", middleware.RequirePermission("land:create"), landH.SyncParcel)
			land.PATCH("/:id/parcels/:parcel_id/valuation", middleware.RequirePermission("land:update"), landH.UpdateParcelValuation)
			land.PUT("/:id/parcels/:parcel_id/compensation", middleware.RequirePermission("land:update"), landH.SetParcelCompensation)
			land.GET("/:id/progress", middleware.RequirePermission("land:read"), landH.ListProgress)
			land.GET("/:id/available-statuses", middleware.RequirePermission("land:read"), landH.GetAvailableStatuses)
			land.POST("/:id/advance", middleware.RequirePermission("land:update"), landH.AdvanceStatus)
			land.GET("/:id/documents", middleware.RequirePermission("land:read"), landH.ListAcquisitionDocuments)
			land.POST("/:id/documents", middleware.RequirePermission("land:update"), landH.UploadAcquisitionDocument)
			land.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), landH.DeleteAcquisitionDocument)
		}

		// Газар зохион байгуулалтын төлөвлөвлөгөө
		plans := protected.Group("/plans")
		{
			plans.GET("/search", middleware.RequirePermission("land:read"), planH.Search)
			plans.GET("/suggest", middleware.RequirePermission("land:read"), planH.Suggest)
		}

		// Глобал нэгж талбарын маршрут
		parcels := protected.Group("/parcels")
		{
			parcels.GET("", middleware.RequirePermission("land:read"), landH.ListGlobalParcels)
			parcels.GET("/:id/payments", middleware.RequirePermission("land:read"), landH.ListParcelPayments)
			parcels.POST("/:id/payments", middleware.RequirePermission("land:update"), landH.AddParcelPayment)
			parcels.GET("/:id/documents", middleware.RequirePermission("land:read"), landH.ListParcelDocuments)
			parcels.POST("/:id/documents", middleware.RequirePermission("land:update"), landH.UploadParcelDocument)
			parcels.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), landH.DeleteParcelDocument)
		}
	}

	return r
}
