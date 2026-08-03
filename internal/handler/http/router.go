package http

import (
	"time"

	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(
	healthChecker HealthChecker,
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
	// notificationSvc — заавал бус (variadic) тул хуучин дуудалтууд өөрчлөгдөхгүй;
	// өгөөгүй үед мэдэгдлийн маршрутууд бүртгэгдэхгүй.
	optionalSvcs ...any,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(cors.Default())
	// JSON зэрэг файл бус хүсэлтийн биеийг 5MB-ээр хязгаарлана (файл байршуулах
	// multipart handler-ууд өөрсдийн хязгаараа тогтооно).
	r.Use(middleware.BodySizeLimit(5 << 20))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Файлыг API тараахаа больсон — бүх файл файлын систем (MinIO) дээр байж,
	// frontend-ийн /api/files proxy-гоор тарааагдана. Хуучин `/mock-files`
	// static route-ыг зориуд хассан: API нь файл хадгалах/тараах үүрэггүй.

	healthH := NewHealthHandler(healthChecker)
	r.GET("/health", healthH.Live)
	r.GET("/health/ready", healthH.Ready)

	authH := NewAuthHandler(authSvc)
	userH := NewUserHandler(userSvc, roleSvc)
	roleH := NewRoleHandler(roleSvc, permSvc)
	permH := NewPermissionHandler(permSvc)
	landH := NewLandHandler(landSvc)
	planH := NewPlanHandler(planSvc)
	acquisitionProgressStatusH := NewAcquisitionProgressStatusHandler(acquisitionProgressStatusSvc)
	parcelStatusH := NewParcelStatusHandler(parcelStatusSvc)
	documentTypeH := NewDocumentTypeHandler(documentTypeSvc)
	parcelWorkflowH := NewParcelWorkflowHandler(landSvc)
	acquisitionWorkflowH := NewAcquisitionWorkflowHandler(landSvc)
	acquisitionCategoryH := NewAcquisitionCategoryHandler(landSvc)
	assetTypeConfigH := NewAssetTypeConfigHandler(landSvc)

	v1 := r.Group("/api/v1")

	// Нээлттэй бүртгэл (`POST /auth/register`) ЗОРИУД байхгүй: энэ нь дотоод
	// систем тул хэрэглэгчийг зөвхөн эрх бүхий ажилтан `POST /users`
	// (`users:create`) -ээр үүсгэнэ. Нээлттэй байх үед хэн ч идэвхтэй аккаунт
	// нээж хүчинтэй JWT аван permission шалгадаггүй маршрутууд руу орно.
	auth := v1.Group("/auth")
	auth.Use(middleware.RateLimit(30, time.Minute))
	{
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.RefreshToken)
		auth.POST("/logout", authH.Logout)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)
	}

	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtManager))
	{
		var notificationSvc service.NotificationService
		var auditSvc service.AuditService
		var clientRequestSvc service.ClientRequestService
		var organizationSvc service.OrganizationService
		var departmentSvc service.DepartmentService
		var positionSvc service.PositionService
		var personSvc service.PersonService
		var employeeSvc service.EmployeeService
		for _, svc := range optionalSvcs {
			switch v := svc.(type) {
			case service.NotificationService:
				notificationSvc = v
			case service.AuditService:
				auditSvc = v
			case service.ClientRequestService:
				clientRequestSvc = v
			case service.OrganizationService:
				organizationSvc = v
			case service.DepartmentService:
				departmentSvc = v
			case service.PositionService:
				positionSvc = v
			case service.PersonService:
				personSvc = v
			case service.EmployeeService:
				employeeSvc = v
			}
		}

		// Мэдэгдэл — нэвтэрсэн хэрэглэгч бүр зөвхөн ӨӨРИЙН мэдэгдэлтэй ажиллана
		// (recipient шүүлт repo түвшинд); нэмэлт permission шаардлагагүй.
		if notificationSvc != nil {
			notificationH := NewNotificationHandler(notificationSvc)
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", notificationH.List)
				notifications.GET("/unread-count", notificationH.UnreadCount)
				notifications.GET("/stream", notificationH.Stream)
				notifications.POST("/read-all", notificationH.MarkAllRead)
				notifications.POST("/:id/read", notificationH.MarkRead)
			}
		}
		if auditSvc != nil {
			auditH := NewAuditHandler(auditSvc)
			protected.GET("/audit-logs", middleware.RequirePermission("audit:read"), auditH.List)
		}
		// Бусад системрүү (гадаад API) илгээсэн HTTP хүсэлтийн бүртгэл.
		if clientRequestSvc != nil {
			clientRequestH := NewClientRequestHandler(clientRequestSvc)
			protected.GET("/client-requests", middleware.RequirePermission("audit:read"), clientRequestH.List)
		}
		if organizationSvc != nil && departmentSvc != nil && positionSvc != nil && personSvc != nil && employeeSvc != nil {
			hrH := NewHRHandler(organizationSvc, departmentSvc, positionSvc, personSvc, employeeSvc)
			denyExternal := middleware.DenyAnyRole("professional_org", "mika", "finance_specialist")
			adminOnly := middleware.RequireAnyRole("admin")

			organizations := protected.Group("/organizations")
			organizations.Use(denyExternal)
			{
				organizations.GET("", middleware.RequirePermission("hr:read"), hrH.ListOrganizations)
				organizations.GET("/:id", middleware.RequirePermission("hr:read"), hrH.GetOrganization)
				organizations.POST("", middleware.RequirePermission("hr:create"), adminOnly, hrH.CreateOrganization)
				organizations.PUT("/:id", middleware.RequirePermission("hr:update"), adminOnly, hrH.UpdateOrganization)
				organizations.DELETE("/:id", middleware.RequirePermission("hr:delete"), adminOnly, hrH.DeleteOrganization)
			}

			departments := protected.Group("/departments")
			departments.Use(denyExternal)
			{
				departments.GET("", middleware.RequirePermission("hr:read"), hrH.ListDepartments)
				departments.POST("", middleware.RequirePermission("hr:create"), adminOnly, hrH.CreateDepartment)
				departments.PUT("/:id", middleware.RequirePermission("hr:update"), adminOnly, hrH.UpdateDepartment)
				departments.DELETE("/:id", middleware.RequirePermission("hr:delete"), adminOnly, hrH.DeleteDepartment)
			}

			positions := protected.Group("/positions")
			positions.Use(denyExternal)
			{
				positions.GET("", middleware.RequirePermission("hr:read"), hrH.ListPositions)
				positions.POST("", middleware.RequirePermission("hr:create"), adminOnly, hrH.CreatePosition)
				positions.PUT("/:id", middleware.RequirePermission("hr:update"), adminOnly, hrH.UpdatePosition)
				positions.DELETE("/:id", middleware.RequirePermission("hr:delete"), adminOnly, hrH.DeletePosition)
			}

			persons := protected.Group("/persons")
			persons.Use(denyExternal)
			{
				persons.GET("", middleware.RequirePermission("hr:read"), hrH.ListPersons)
				persons.POST("", middleware.RequirePermission("hr:create"), adminOnly, hrH.CreatePerson)
				persons.PUT("/:id", middleware.RequirePermission("hr:update"), adminOnly, hrH.UpdatePerson)
				persons.DELETE("/:id", middleware.RequirePermission("hr:delete"), adminOnly, hrH.DeletePerson)
			}

			employees := protected.Group("/employees")
			employees.Use(denyExternal)
			{
				employees.GET("", middleware.RequirePermission("hr:read"), hrH.ListEmployees)
				employees.GET("/:id", middleware.RequirePermission("hr:read"), hrH.GetEmployee)
				employees.POST("", middleware.RequirePermission("hr:create"), adminOnly, hrH.CreateEmployee)
				employees.PUT("/:id", middleware.RequirePermission("hr:update"), adminOnly, hrH.UpdateEmployee)
				employees.DELETE("/:id", middleware.RequirePermission("hr:delete"), adminOnly, hrH.DeleteEmployee)
			}
		}

		users := protected.Group("/users")
		{
			users.GET("/me", userH.GetMe)
			users.GET("", middleware.RequirePermission("users:read"), userH.List)
			users.GET("/:id", middleware.RequirePermission("users:read"), userH.GetByID)
			users.POST("", middleware.RequirePermission("users:create"), userH.Create)
			users.PUT("/:id", middleware.RequirePermission("users:update"), userH.Update)
			users.DELETE("/:id", middleware.RequirePermission("users:delete"), userH.Delete)
			// Нууц үг сэргээх + роль олгох/хураах. Хоёулаа users:update эрх
			// шаардана; handler дотор нь эрх нэмэгдүүлэлт (privilege escalation)
			// болон өөрийгөө хаахаас (self-lockout) хамгаалах шалгалт бий.
			users.PUT("/:id/password", middleware.RequirePermission("users:update"), userH.ChangePassword)
			users.POST("/:id/roles", middleware.RequirePermission("users:update"), userH.AssignRole)
			users.DELETE("/:id/roles/:role_id", middleware.RequirePermission("users:update"), userH.RemoveRole)
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
			acquisitionCategories.GET("", acquisitionCategoryH.List)
			acquisitionCategories.POST("", middleware.RequirePermission("admin:create"), acquisitionCategoryH.Create)
			acquisitionCategories.PUT("/:id", middleware.RequirePermission("admin:update"), acquisitionCategoryH.Update)
			acquisitionCategories.DELETE("/:id", middleware.RequirePermission("admin:delete"), acquisitionCategoryH.Delete)
		}

		// Газар чөлөөлөлт
		assignedOrSenior := middleware.RequireAssignedOrSenior(landSvc)
		// Үнэлгээ бичих маршрутууд: МИКА хуваарилалтгүйгээр өөрийн (mika) урсгалдаа
		// бичих эрхтэй — handler-ууд valuation_type-ийг "mika" болгож хатуу тогтооно.
		valuationEditor := middleware.RequireAssignedOrSenior(landSvc, "mika", "МИКА")
		notLocked := middleware.RequireAcquisitionNotLocked(landSvc)
		notParcelLocked := middleware.RequireParcelAcquisitionNotLocked(landSvc)
		profOrgMember := middleware.RequireProfOrgMember(landSvc)
		profParcelMember := middleware.RequireProfParcelMember(landSvc)
		// /prof/land-acquisitions/:id/parcels/:parcel_id/* — parcel түвшний гишүүнчлэл:
		// хөндлөнгийн байгууллага зөвхөн өөрийн parcel дээр үйлдэл хийнэ (M3).
		profParcelMemberByParam := middleware.RequireProfParcelMemberParam(landSvc, "parcel_id")
		// /prof routes: GET-г оролцуулан бүх хүсэлтийг хаадаг хаалтууд
		profNotLocked := middleware.RequireProfAcquisitionNotLocked(landSvc)
		profNotParcelLocked := middleware.RequireProfParcelAcquisitionNotLocked(landSvc)
		land := protected.Group("/land-acquisitions")
		land.Use(notLocked)
		{
			land.GET("", middleware.DenyAnyRole("professional_org"), middleware.RequireAnyPermission("land:read", "compensation:read"), landH.List)
			land.GET("/suggest", middleware.DenyAnyRole("professional_org"), middleware.RequireAnyPermission("land:read", "compensation:read"), landH.Suggest)
			land.POST("", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:create"), landH.Create)
			// Зөвшөөрөгдсөн ажилтан эсвэл ахлах мэргэжилтэн шалгах middleware
			land.GET("/:id", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.GetByID)
			land.GET("/:id/parcels", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListParcels)
			land.GET("/:id/assets", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListAssets)
			land.GET("/:id/compensations", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListCompensations)
			land.GET("/:id/parcels/:parcel_id", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.GetParcelByID)
			land.PUT("/:id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.Update)
			land.DELETE("/:id", middleware.RequirePermission("land:delete"), landH.Delete)
			// Assignee routes
			land.GET("/:id/assignees", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetAssignees)
			land.PUT("/:id/assignees", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.SetAssignees)
			land.POST("/:id/assets", middleware.RequireAnyPermission("land:create", "compensation:create"), valuationEditor, landH.CreateAsset)
			land.PUT("/:id/assets/:asset_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateAsset)
			land.DELETE("/:id/assets/:asset_id", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.DeleteAsset)
			land.POST("/:id/assets/:asset_id/photos", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.UploadAssetPhoto)
			land.GET("/:id/assets/:asset_id/specs", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListAssetSpecs)
			land.POST("/:id/assets/:asset_id/specs", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.UpsertAssetSpecs)
			land.GET("/:id/assets/:asset_id/calculations", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListAssetCalculations)
			land.POST("/:id/assets/:asset_id/calculations", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.UpsertAssetCalculations)
			land.GET("/:id/land-valuation", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.GetLandValuation)
			land.POST("/:id/land-valuation", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.UpsertLandValuation)
			land.DELETE("/:id/land-valuation", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.DeleteLandValuation)
			land.POST("/:id/valuation-import", middleware.RequireAnyPermission("land:create", "compensation:create"), valuationEditor, landH.ImportValuation)
			land.POST("/:id/compensations", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.CreateCompensation)
			land.PUT("/:id/compensations/:comp_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UpdateCompensation)
			land.DELETE("/:id/compensations/:comp_id", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.DeleteCompensation)
			land.GET("/:id/compensations/:comp_id/history", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListCompensationHistory)
			land.POST("/:id/compensations/:comp_id/approve", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү"), middleware.RequireAnyPermission("land:update", "compensation:update"), landH.ApproveCompensation)
			land.POST("/:id/compensations/:comp_id/reject", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү"), middleware.RequireAnyPermission("land:update", "compensation:update"), landH.RejectCompensation)
			// Үнэлгээний тайлан — МИКА өөрийн баталгаажсан урсгалдаа мөн хавсаргана (valuationEditor)
			land.POST("/:id/compensations/:comp_id/report", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, landH.UploadCompensationReport)
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
			// Нөхөх олговрын үнэлгээний илгээх/зөвшөөрөх төлөв (санхүү зөвшөөрөх/буцаах).
			// POST-ийн ролийн шалгалт handler дотор (approve/return=санхүү, submit=мэрг.байгуулга/ахлах).
			land.GET("/:id/parcels/:parcel_id/valuation-status", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, parcelWorkflowH.GetValuationSubmission)
			land.GET("/:id/parcels/:parcel_id/valuation-status-history", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, parcelWorkflowH.ListValuationSubmissionHistory)
			land.POST("/:id/parcels/:parcel_id/valuation-status", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "senior_specialist", "Ахлах мэргэжилтэн", "mika", "МИКА"), middleware.RequireAnyPermission("land:update", "compensation:update"), parcelWorkflowH.TransitionValuationSubmission)
			land.GET("/:id/progress", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListProgress)
			land.GET("/:id/boundary-history", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListBoundaryHistory)
			land.GET("/:id/available-statuses", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetAvailableStatuses)
			land.POST("/:id/advance", middleware.RequirePermission("land:update"), assignedOrSenior, landH.AdvanceStatus)
			land.POST("/:id/funding-sources", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:create"), landH.CreateFundingSource)
			land.PUT("/:id/funding-sources/:src_id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.UpdateFundingSource)
			land.DELETE("/:id/funding-sources/:src_id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.DeleteFundingSource)
			land.GET("/:id/parcels/:parcel_id/representatives", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListAuthorizedRepresentatives)
			land.POST("/:id/parcels/:parcel_id/representatives", middleware.RequirePermission("land:update"), assignedOrSenior, landH.CreateAuthorizedRepresentative)
			land.DELETE("/:id/parcels/:parcel_id/representatives/:rep_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteAuthorizedRepresentative)
			land.GET("/:id/documents", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListAcquisitionDocuments)
			land.POST("/:id/documents", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UploadAcquisitionDocument)
			land.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteAcquisitionDocument)
			// Дроны ортофото (.tif) — GeoServer дээр зураг тус бүр өөрийн давхарга
			// болж бүртгэгдэнэ. Нэг хүсэлтэд НЭГ файл; refresh нь frontend
			// зураг байршуулсны дараа автоматаар дуудагдана.
			land.GET("/:id/drone-images", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListDroneImages)
			// Дроны зураг нь ШУУД файлын систем руу байршина — API зөвхөн
			// зөвшөөрөл гаргаж (upload-url), дараа нь объектыг шалгаж бүртгэнэ.
			land.POST("/:id/drone-images/upload-url", middleware.RequirePermission("land:update"), assignedOrSenior, landH.CreateDroneUploadURL)
			land.POST("/:id/drone-images", middleware.RequirePermission("land:update"), assignedOrSenior, landH.RegisterDroneImage)
			land.POST("/:id/drone-images/refresh", middleware.RequirePermission("land:update"), assignedOrSenior, landH.RefreshDroneImages)
			land.DELETE("/:id/drone-images/:image_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteDroneImage)
			land.PUT("/:id/professional-org", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.SetAcquisitionProfessionalOrg)
			land.PATCH("/:id/parcels/:parcel_id/independent-org", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн", "professional_org"), middleware.RequirePermission("land:update"), landH.SetParcelIndependentOrg)
		}

		prof := protected.Group("/prof")
		prof.Use(middleware.RequireOnlyRole("professional_org"))
		{
			prof.GET("/construction-types", middleware.RequirePermission("compensation:read"), landH.ListConstructionTypes)
			prof.GET("/asset-spec-types", middleware.RequirePermission("compensation:read"), assetTypeConfigH.ListSpecTypes)
			prof.GET("/asset-calc-types", middleware.RequirePermission("compensation:read"), assetTypeConfigH.ListCalcTypes)

			profLand := prof.Group("/land-acquisitions")
			{
				profLand.GET("", middleware.RequirePermission("compensation:read"), landH.ListMyAcquisitions)

				// /:id/* — тухайн чөлөөлөлтийн гишүүн мөн эсэхийг шалгана
				profLandItem := profLand.Group("/:id")
				profLandItem.Use(profNotLocked)
				profLandItem.Use(profOrgMember)
				{
					profLandItem.GET("", middleware.RequirePermission("compensation:read"), landH.GetByID)
					profLandItem.GET("/parcels", middleware.RequirePermission("compensation:read"), landH.ListEvaluationParcels)
					profLandItem.GET("/parcels/:parcel_id", middleware.RequirePermission("compensation:read"), profParcelMemberByParam, landH.GetParcelByID)
					profLandItem.PATCH("/parcels/:parcel_id/independent-org", middleware.RequirePermission("compensation:update"), profParcelMemberByParam, landH.SetParcelIndependentOrg)
					// Нэгж талбарын статус
					profLandItem.GET("/parcels/:parcel_id/available-statuses", middleware.RequirePermission("compensation:read"), profParcelMemberByParam, parcelWorkflowH.GetAvailableStatuses)
					profLandItem.PATCH("/parcels/:parcel_id/status", middleware.RequirePermission("compensation:update"), profParcelMemberByParam, parcelWorkflowH.UpdateParcelStatus)
					profLandItem.GET("/parcels/:parcel_id/status-history", middleware.RequirePermission("compensation:read"), profParcelMemberByParam, parcelWorkflowH.ListParcelStatusHistory)
					// Нөхөх олговрын үнэлгээний илгээх төлөв (мэрг. байгууллага илгээнэ)
					profLandItem.GET("/parcels/:parcel_id/valuation-status", middleware.RequirePermission("compensation:read"), profParcelMemberByParam, parcelWorkflowH.GetValuationSubmission)
					profLandItem.POST("/parcels/:parcel_id/valuation-status", middleware.RequirePermission("compensation:update"), profParcelMemberByParam, parcelWorkflowH.TransitionValuationSubmission)
					profLandItem.GET("/parcels/:parcel_id/valuation-status-history", middleware.RequirePermission("compensation:read"), profParcelMemberByParam, parcelWorkflowH.ListValuationSubmissionHistory)
					// Эрх эзэмшигч төлөөлөгч
					profLandItem.GET("/parcels/:parcel_id/representatives", middleware.RequirePermission("compensation:read"), profParcelMemberByParam, landH.ListAuthorizedRepresentatives)
					profLandItem.POST("/parcels/:parcel_id/representatives", middleware.RequirePermission("compensation:update"), profParcelMemberByParam, landH.CreateAuthorizedRepresentative)
					profLandItem.DELETE("/parcels/:parcel_id/representatives/:rep_id", middleware.RequirePermission("compensation:update"), profParcelMemberByParam, landH.DeleteAuthorizedRepresentative)
					// Байгааламж — үнэлгээ (valuation:create/update)
					profLandItem.GET("/assets", middleware.RequirePermission("compensation:read"), landH.ListAssets)
					profLandItem.POST("/assets", middleware.RequirePermission("compensation:create"), landH.CreateAsset)
					profLandItem.DELETE("/assets/:asset_id", middleware.RequirePermission("compensation:update"), landH.DeleteAsset)
					profLandItem.POST("/assets/:asset_id/photos", middleware.RequirePermission("compensation:update"), landH.UploadAssetPhoto)
					profLandItem.GET("/assets/:asset_id/specs", middleware.RequirePermission("compensation:read"), landH.ListAssetSpecs)
					profLandItem.POST("/assets/:asset_id/specs", middleware.RequirePermission("compensation:update"), landH.UpsertAssetSpecs)
					profLandItem.GET("/assets/:asset_id/calculations", middleware.RequirePermission("compensation:read"), landH.ListAssetCalculations)
					profLandItem.POST("/assets/:asset_id/calculations", middleware.RequirePermission("compensation:update"), landH.UpsertAssetCalculations)
					// Газрын үнэлгээ (valuation:update)
					profLandItem.GET("/land-valuation", middleware.RequirePermission("compensation:read"), landH.GetLandValuation)
					profLandItem.POST("/land-valuation", middleware.RequirePermission("compensation:update"), landH.UpsertLandValuation)
					profLandItem.DELETE("/land-valuation", middleware.RequirePermission("compensation:update"), landH.DeleteLandValuation)
					profLandItem.POST("/valuation-import", middleware.RequirePermission("compensation:create"), landH.ImportValuation)
					// Нөхөх олговор — үнэлгээ (valuation:create/update)
					profLandItem.GET("/compensations", middleware.RequirePermission("compensation:read"), landH.ListCompensations)
					profLandItem.POST("/compensations", middleware.RequirePermission("compensation:create"), landH.CreateCompensation)
					profLandItem.DELETE("/compensations/:comp_id", middleware.RequirePermission("compensation:update"), landH.DeleteCompensation)
					profLandItem.GET("/compensations/:comp_id/history", middleware.RequirePermission("compensation:read"), landH.ListCompensationHistory)
					profLandItem.POST("/compensations/:comp_id/report", middleware.RequirePermission("compensation:update"), landH.UploadCompensationReport)
					profLandItem.POST("/compensations/:comp_id/grant", middleware.RequirePermission("compensation:update"), landH.CreateCompensationGrant)
					profLandItem.PUT("/compensations/:comp_id/grant", middleware.RequirePermission("compensation:update"), landH.UpdateCompensationGrant)
					profLandItem.DELETE("/compensations/:comp_id/grant", middleware.RequirePermission("compensation:update"), landH.DeleteCompensationGrant)
					// Баримт бичиг (чөлөөлөлт)
					profLandItem.GET("/documents", middleware.RequirePermission("compensation:read"), landH.ListAcquisitionDocuments)
					profLandItem.POST("/documents", middleware.RequirePermission("compensation:update"), landH.UploadAcquisitionDocument)
					profLandItem.DELETE("/documents/:doc_id", middleware.RequirePermission("compensation:update"), landH.DeleteAcquisitionDocument)
				}
			}

			// Нэгж талбарын баримт бичиг — хөрөнгийн зураг (үнэлгээний хэсэг)
			profParcels := prof.Group("/parcels")
			profParcels.Use(profNotParcelLocked)
			profParcels.Use(profParcelMember)
			{
				profParcels.GET("/:id/documents", middleware.RequirePermission("compensation:read"), landH.ListParcelDocuments)
				profParcels.POST("/:id/documents", middleware.RequirePermission("compensation:update"), landH.UploadParcelDocument)
				profParcels.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("compensation:update"), landH.DeleteParcelDocument)
			}
		}

		// Газар зохион байгуулалтын төлөвлөвлөгөө
		plans := protected.Group("/plans")
		{
			plans.GET("/search", middleware.RequireAnyPermission("land:read", "compensation:read"), planH.Search)
			plans.GET("/suggest", middleware.RequireAnyPermission("land:read", "compensation:read"), planH.Suggest)
		}

		// Чөлөөлөлтийн явцын статус
		acquisitionProgressStatuses := protected.Group("/acquisition-progress-statuses")
		{
			acquisitionProgressStatuses.GET("", acquisitionProgressStatusH.List)
			acquisitionProgressStatuses.GET("/:id", acquisitionProgressStatusH.GetByID)
			acquisitionProgressStatuses.POST("", middleware.RequirePermission("admin:create"), acquisitionProgressStatusH.Create)
			acquisitionProgressStatuses.PUT("/:id", middleware.RequirePermission("admin:update"), acquisitionProgressStatusH.Update)
			acquisitionProgressStatuses.DELETE("/:id", middleware.RequirePermission("admin:delete"), acquisitionProgressStatusH.Delete)
		}

		// Нэгж талбарын статус
		parcelStatuses := protected.Group("/parcel-statuses")
		{
			parcelStatuses.GET("", parcelStatusH.List)
			parcelStatuses.GET("/:id", parcelStatusH.GetByID)
			parcelStatuses.POST("", middleware.RequirePermission("admin:create"), parcelStatusH.Create)
			parcelStatuses.PUT("/:id", middleware.RequirePermission("admin:update"), parcelStatusH.Update)
			parcelStatuses.DELETE("/:id", middleware.RequirePermission("admin:delete"), parcelStatusH.Delete)
		}

		// Баримт бичгийн төрөл
		documentTypes := protected.Group("/document-types")
		{
			documentTypes.GET("", documentTypeH.List)
			documentTypes.GET("/:id", documentTypeH.GetByID)
			documentTypes.POST("", middleware.RequirePermission("admin:create"), documentTypeH.Create)
			documentTypes.PUT("/:id", middleware.RequirePermission("admin:update"), documentTypeH.Update)
			documentTypes.DELETE("/:id", middleware.RequirePermission("admin:delete"), documentTypeH.Delete)
		}

		// Дашбоард — дотоод хэрэглэгчид бүрэн, гадаад үнэлгээ/санхүүгийн ролиуд
		// backend scope-оор зөвхөн өөрт хамаарах мэдээллээ авна.
		protected.GET("/dashboard",
			middleware.RequirePermissionOrRole("land:read", "professional_org", "mika", "finance_specialist"),
			landH.Dashboard)

		// Тайлангийн маршрут
		report := protected.Group("/report")
		{
			report.GET("/download", middleware.RequirePermission("land:read"), landH.ListReportParcels)
			report.GET("/summary", middleware.RequirePermission("land:read"), landH.ReportSummary)
		}

		// Нэгж талбарын ажлын урсгал
		parcelWorkflow := protected.Group("/parcel-workflow")
		{
			parcelWorkflow.GET("", parcelWorkflowH.List)
			parcelWorkflow.POST("", middleware.RequirePermission("admin:create"), parcelWorkflowH.Create)
			parcelWorkflow.DELETE("/:id", middleware.RequirePermission("admin:delete"), parcelWorkflowH.Delete)
		}

		// Газар чөлөөлөлтийн ажлын урсгал
		acquisitionWorkflow := protected.Group("/acquisition-workflow")
		{
			acquisitionWorkflow.GET("", acquisitionWorkflowH.List)
			acquisitionWorkflow.GET("/statuses", acquisitionWorkflowH.ListStatuses)
			acquisitionWorkflow.POST("", middleware.RequirePermission("admin:create"), acquisitionWorkflowH.Create)
			acquisitionWorkflow.DELETE("/:id", middleware.RequirePermission("admin:delete"), acquisitionWorkflowH.Delete)
		}

		// Байгааламжийн тооцоолол / чанарын мэдээллийн төрлүүд
		assetSpecTypes := protected.Group("/asset-spec-types")
		{
			assetSpecTypes.GET("", assetTypeConfigH.ListSpecTypes)
			assetSpecTypes.POST("", middleware.RequirePermission("admin:create"), assetTypeConfigH.CreateSpecType)
			assetSpecTypes.PUT("/:id", middleware.RequirePermission("admin:update"), assetTypeConfigH.UpdateSpecType)
			assetSpecTypes.DELETE("/:id", middleware.RequirePermission("admin:delete"), assetTypeConfigH.DeleteSpecType)
		}

		assetCalcTypes := protected.Group("/asset-calc-types")
		{
			assetCalcTypes.GET("", assetTypeConfigH.ListCalcTypes)
			assetCalcTypes.POST("", middleware.RequirePermission("admin:create"), assetTypeConfigH.CreateCalcType)
			assetCalcTypes.PUT("/:id", middleware.RequirePermission("admin:update"), assetTypeConfigH.UpdateCalcType)
			assetCalcTypes.DELETE("/:id", middleware.RequirePermission("admin:delete"), assetTypeConfigH.DeleteCalcType)
		}

		// Глобал нөхөн төлбөрийн маршрут
		protected.GET("/compensations", middleware.RequirePermission("land:read"), landH.ListAllCompensations)
		protected.GET("/compensations/:id", middleware.RequirePermission("land:read"), landH.GetGlobalCompensation)

		// Глобал нэгж талбарын маршрут
		parcelAssignedOrSenior := middleware.RequireParcelAssignedOrSenior(landSvc)
		parcels := protected.Group("/parcels")
		parcels.Use(notParcelLocked)
		{
			parcels.GET("", middleware.RequirePermission("land:read"), landH.ListGlobalParcels)
			parcels.GET("/:id/payments", middleware.RequirePermission("land:read"), landH.ListParcelPayments)
			parcels.POST("/:id/payments", middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.AddParcelPayment)
			// Үнэлгээний тайлан Нөхөх олговор табд харагддаг тул гадаад ролиуд (санхүү/МИКА)
			// мөн уншина; мэрг. байгууллага өөрийн /prof (гишүүнчлэлтэй) замаа ашиглана.
			parcels.GET("/:id/documents", middleware.DenyAnyRole("professional_org"), middleware.RequireAnyPermission("land:read", "compensation:read"), landH.ListParcelDocuments)
			parcels.POST("/:id/documents", middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.UploadParcelDocument)
			parcels.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.DeleteParcelDocument)
		}

		// Нэгж талбарын статус солих + боломжтой статусууд + түүх
		protected.GET("/land-acquisitions/:id/parcels/:parcel_id/available-statuses",
			middleware.RequirePermission("land:read"), assignedOrSenior, parcelWorkflowH.GetAvailableStatuses)
		protected.PATCH("/land-acquisitions/:id/parcels/:parcel_id/status",
			middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, parcelWorkflowH.UpdateParcelStatus)
		protected.GET("/land-acquisitions/:id/parcels/:parcel_id/status-history",
			middleware.RequirePermission("land:read"), assignedOrSenior, parcelWorkflowH.ListParcelStatusHistory)
	}

	return r
}
