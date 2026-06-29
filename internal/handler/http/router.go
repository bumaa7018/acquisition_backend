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
	acquisitionProgressStatusSvc service.AcquisitionProgressStatusService,
	parcelStatusSvc service.ParcelStatusService,
	documentTypeSvc service.DocumentTypeService,
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
	acquisitionProgressStatusH := NewAcquisitionProgressStatusHandler(acquisitionProgressStatusSvc)
	parcelStatusH := NewParcelStatusHandler(parcelStatusSvc)
	documentTypeH := NewDocumentTypeHandler(documentTypeSvc)
	parcelWorkflowH := NewParcelWorkflowHandler(landSvc)
	acquisitionWorkflowH := NewAcquisitionWorkflowHandler(landSvc)
	acquisitionCategoryH := NewAcquisitionCategoryHandler(landSvc)

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

		// Lookup
		protected.GET("/construction-types", middleware.RequirePermission("land:read"), landH.ListConstructionTypes)

		// Ангилал CRUD
		acquisitionCategories := protected.Group("/acquisition-categories")
		{
			acquisitionCategories.GET("", middleware.RequirePermission("land:read"), acquisitionCategoryH.List)
			acquisitionCategories.POST("", middleware.RequirePermission("land:create"), acquisitionCategoryH.Create)
			acquisitionCategories.PUT("/:id", middleware.RequirePermission("land:update"), acquisitionCategoryH.Update)
			acquisitionCategories.DELETE("/:id", middleware.RequirePermission("land:delete"), acquisitionCategoryH.Delete)
		}

		// Газар чөлөөлөлт
		assignedOrSenior := middleware.RequireAssignedOrSenior(landSvc)
		land := protected.Group("/land-acquisitions")
		{
			land.GET("", middleware.RequirePermission("land:read"), landH.List)
			land.GET("/suggest", middleware.RequirePermission("land:read"), landH.Suggest)
			land.POST("", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:create"), landH.Create)
			// Зөвшөөрөгдсөн ажилтан эсвэл ахлах мэргэжилтэн шалгах middleware
			land.GET("/:id", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetByID)
			land.GET("/:id/parcels", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListParcels)
			land.GET("/:id/assets", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListAssets)
			land.GET("/:id/compensations", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListCompensations)
			land.GET("/:id/parcels/:parcel_id", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetParcelByID)
			land.PUT("/:id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.Update)
			land.DELETE("/:id", middleware.RequirePermission("land:delete"), landH.Delete)
			// Assignee routes
			land.GET("/:id/assignees", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetAssignees)
			land.PUT("/:id/assignees", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.SetAssignees)
			land.POST("/:id/assets", middleware.RequirePermission("land:create"), assignedOrSenior, landH.CreateAsset)
			land.PUT("/:id/assets/:asset_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateAsset)
			land.DELETE("/:id/assets/:asset_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteAsset)
			land.POST("/:id/compensations", middleware.RequirePermission("land:update"), assignedOrSenior, landH.CreateCompensation)
			land.PUT("/:id/compensations/:comp_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateCompensation)
			land.DELETE("/:id/compensations/:comp_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteCompensation)
			land.GET("/:id/compensations/:comp_id/history", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListCompensationHistory)
			land.POST("/:id/compensations/:comp_id/approve", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү"), middleware.RequirePermission("land:update"), landH.ApproveCompensation)
			land.POST("/:id/compensations/:comp_id/reject", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү"), middleware.RequirePermission("land:update"), landH.RejectCompensation)
			land.POST("/:id/compensations/:comp_id/report", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UploadCompensationReport)
			land.POST("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), assignedOrSenior, landH.CreateCompensationGrant)
			land.PUT("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateCompensationGrant)
			land.DELETE("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteCompensationGrant)
			land.POST("/:id/parcels/:parcel_id/sync", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncParcel)
			land.POST("/:id/parcels/:parcel_id/sync/contract-act", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncContractAct)
			land.POST("/:id/parcels/:parcel_id/sync/valuation", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncValuation)
			land.POST("/:id/parcels/:parcel_id/sync/settlement-act", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncSettlementAct)
			land.POST("/:id/parcels/:parcel_id/sync/location-valuation", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncLocationValuation)
			land.POST("/:id/parcels/:parcel_id/sync/monitoring", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncMonitoring)
			land.PATCH("/:id/parcels/:parcel_id/valuation", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateParcelValuation)
			land.PUT("/:id/parcels/:parcel_id/compensation", middleware.RequirePermission("land:update"), assignedOrSenior, landH.SetParcelCompensation)
			land.PATCH("/:id/parcels/:parcel_id/meta", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateParcelMeta)
			land.GET("/:id/progress", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListProgress)
			land.GET("/:id/boundary-history", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListBoundaryHistory)
			land.GET("/:id/available-statuses", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetAvailableStatuses)
			land.POST("/:id/advance", middleware.RequirePermission("land:update"), assignedOrSenior, landH.AdvanceStatus)
			land.GET("/:id/funding-sources", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListFundingSources)
			land.POST("/:id/funding-sources", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:create"), landH.CreateFundingSource)
			land.PUT("/:id/funding-sources/:src_id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.UpdateFundingSource)
			land.DELETE("/:id/funding-sources/:src_id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.DeleteFundingSource)
			land.GET("/:id/parcels/:parcel_id/representatives", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListAuthorizedRepresentatives)
			land.POST("/:id/parcels/:parcel_id/representatives", middleware.RequirePermission("land:update"), assignedOrSenior, landH.CreateAuthorizedRepresentative)
			land.DELETE("/:id/parcels/:parcel_id/representatives/:rep_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteAuthorizedRepresentative)
			land.GET("/:id/documents", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListAcquisitionDocuments)
			land.POST("/:id/documents", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UploadAcquisitionDocument)
			land.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteAcquisitionDocument)
		}

		// Газар зохион байгуулалтын төлөвлөвлөгөө
		plans := protected.Group("/plans")
		{
			plans.GET("/search", middleware.RequirePermission("land:read"), planH.Search)
			plans.GET("/suggest", middleware.RequirePermission("land:read"), planH.Suggest)
		}

		// Чөлөөлөлтийн явцын статус
		acquisitionProgressStatuses := protected.Group("/acquisition-progress-statuses")
		{
			acquisitionProgressStatuses.GET("", middleware.RequirePermission("land:read"), acquisitionProgressStatusH.List)
			acquisitionProgressStatuses.GET("/:id", middleware.RequirePermission("land:read"), acquisitionProgressStatusH.GetByID)
			acquisitionProgressStatuses.POST("", middleware.RequirePermission("land:create"), acquisitionProgressStatusH.Create)
			acquisitionProgressStatuses.PUT("/:id", middleware.RequirePermission("land:update"), acquisitionProgressStatusH.Update)
			acquisitionProgressStatuses.DELETE("/:id", middleware.RequirePermission("land:delete"), acquisitionProgressStatusH.Delete)
		}

		// Нэгж талбарын статус
		parcelStatuses := protected.Group("/parcel-statuses")
		{
			parcelStatuses.GET("", middleware.RequirePermission("land:read"), parcelStatusH.List)
			parcelStatuses.GET("/:id", middleware.RequirePermission("land:read"), parcelStatusH.GetByID)
			parcelStatuses.POST("", middleware.RequirePermission("land:create"), parcelStatusH.Create)
			parcelStatuses.PUT("/:id", middleware.RequirePermission("land:update"), parcelStatusH.Update)
			parcelStatuses.DELETE("/:id", middleware.RequirePermission("land:delete"), parcelStatusH.Delete)
		}

		// Баримт бичгийн төрөл
		documentTypes := protected.Group("/document-types")
		{
			documentTypes.GET("", middleware.RequirePermission("land:read"), documentTypeH.List)
			documentTypes.GET("/:id", middleware.RequirePermission("land:read"), documentTypeH.GetByID)
			documentTypes.POST("", middleware.RequirePermission("land:create"), documentTypeH.Create)
			documentTypes.PUT("/:id", middleware.RequirePermission("land:update"), documentTypeH.Update)
			documentTypes.DELETE("/:id", middleware.RequirePermission("land:delete"), documentTypeH.Delete)
		}

		// Дашбоард
		protected.GET("/dashboard", middleware.RequirePermission("land:read"), landH.Dashboard)

		// Тайлангийн маршрут
		report := protected.Group("/report")
		{
			report.GET("/download", middleware.RequirePermission("land:read"), landH.ListReportParcels)
		}

		// Нэгж талбарын ажлын урсгал
		parcelWorkflow := protected.Group("/parcel-workflow")
		{
			parcelWorkflow.GET("", middleware.RequirePermission("land:read"), parcelWorkflowH.List)
			parcelWorkflow.POST("", middleware.RequirePermission("land:create"), parcelWorkflowH.Create)
			parcelWorkflow.DELETE("/:id", middleware.RequirePermission("land:delete"), parcelWorkflowH.Delete)
		}

		// Газар чөлөөлөлтийн ажлын урсгал
		acquisitionWorkflow := protected.Group("/acquisition-workflow")
		{
			acquisitionWorkflow.GET("", middleware.RequirePermission("land:read"), acquisitionWorkflowH.List)
			acquisitionWorkflow.GET("/statuses", middleware.RequirePermission("land:read"), acquisitionWorkflowH.ListStatuses)
			acquisitionWorkflow.POST("", middleware.RequirePermission("land:create"), acquisitionWorkflowH.Create)
			acquisitionWorkflow.DELETE("/:id", middleware.RequirePermission("land:delete"), acquisitionWorkflowH.Delete)
		}

		// Глобал нөхөн төлбөрийн маршрут
		protected.GET("/compensations", middleware.RequirePermission("land:read"), landH.ListAllCompensations)
		protected.GET("/compensations/:id", middleware.RequirePermission("land:read"), landH.GetGlobalCompensation)

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

		// Нэгж талбарын статус солих + боломжтой статусууд + түүх
		protected.GET("/land-acquisitions/:id/parcels/:parcel_id/available-statuses",
			middleware.RequirePermission("land:read"), assignedOrSenior, parcelWorkflowH.GetAvailableStatuses)
		protected.PATCH("/land-acquisitions/:id/parcels/:parcel_id/status",
			middleware.RequirePermission("land:update"), assignedOrSenior, parcelWorkflowH.UpdateParcelStatus)
		protected.GET("/land-acquisitions/:id/parcels/:parcel_id/status-history",
			middleware.RequirePermission("land:read"), assignedOrSenior, parcelWorkflowH.ListParcelStatusHistory)
	}

	return r
}
