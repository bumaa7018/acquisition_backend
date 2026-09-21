package http

import (
	"net/http"
	"time"

	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/pkg/response"
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

	// Бүртгэлгүй хаяг/метод — gin-ийн анхдагч нь `text/plain` "404 page not
	// found" буцаадаг. Frontend бүх алдааг `{code,error,message}` JSON гэж
	// уншдаг тул тэр биеийг задалж чадалгүй ерөнхий "холбогдоход алдаа" гэсэн
	// төөрөгдүүлсэн мессеж харуулдаг байв. Одоо бусад алдааны адил JSON.
	r.NoRoute(func(c *gin.Context) {
		response.Abort(c, http.StatusNotFound, i18n.ErrRouteNotFound)
	})

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

	// Гадаад API-ийн түлхүүрүүд optionalSvcs-ээр ирнэ (заагаагүй бол хоосон →
	// гадаад гарцууд хаалттай).
	var externalAPIKeys []middleware.ExternalAPIKey
	for _, svc := range optionalSvcs {
		if keys, ok := svc.([]middleware.ExternalAPIKey); ok {
			externalAPIKeys = keys
		}
	}

	v1 := r.Group("/api/v1")

	// Нээлттэй бүртгэл (`POST /auth/register`) ЗОРИУД байхгүй: энэ нь дотоод
	// систем тул хэрэглэгчийг зөвхөн эрх бүхий ажилтан `POST /users`
	// (`users:create`) -ээр үүсгэнэ. Нээлттэй байх үед хэн ч идэвхтэй аккаунт
	// нээж хүчинтэй JWT аван permission шалгадаггүй маршрутууд руу орно.
	// ── ГАДААД системийн API (JWT-гүй, API key-тэй) ─────────────────────────
	//
	// Төлөвлөгөөний 2.8.4: нөхөх олговрын мэдээг бусад систем машинаар уншина.
	// Танилт нь X-API-Key (EXTERNAL_API_KEYS). Түлхүүр тохируулаагүй бол
	// гарцууд ХААЛТТАЙ (middleware өөрөө 503 буцаана).
	//
	// ЯАГААД `protected`-ээс ГАДНА: тэр бүлэг JWT + ValuationOrgScope
	// шаарддаг ба гадаад системд хэрэглэгчийн токен байхгүй.
	external := v1.Group("/external")
	// Гадаад системийн хүсэлтийн урсгалыг хязгаарлана (нэг IP-ээс минутад 120).
	external.Use(middleware.RateLimit(120, time.Minute))
	external.Use(middleware.ExternalAPIKeyAuth(externalAPIKeys))
	{
		externalCompensationH := NewExternalCompensationHandler(landSvc)
		external.GET("/compensations", externalCompensationH.ListCompensations)
	}

	auth := v1.Group("/auth")
	auth.Use(middleware.RateLimit(30, time.Minute))
	{
		auth.POST("/login", authH.Login)
		// Мэргэжлийн байгууллагын нэвтрэлт ТУСДАА эндпойнт: бүртгэл нь өөр
		// санд (appdb.valuation_org_employee) бөгөөд хоёр сангийн user_id
		// мөргөлддөг тул урсгалыг холихгүй.
		auth.POST("/prof/login", authH.LoginProfessionalOrg)
		auth.POST("/refresh", authH.RefreshToken)
		auth.POST("/logout", authH.Logout)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)
	}

	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtManager))
	// Үнэлгээний байгууллагын ажилтан нь системийн үндсэн хэрэглэгчээс ТУСДАА
	// явдаг: түүний хандалтыг sdplatform-ийн роль/эрх БИШ, аппын өөрийн
	// allowlist шийднэ. Хаалтыг маршрут тус бүрд БИШ, бүх хамгаалагдсан замын
	// эхэнд тавьсан нь зориудын — шинээр нэмэгдэх endpoint гадаад байгууллагад
	// анхнаасаа хаалттай байна (deny-by-default).
	protected.Use(middleware.ValuationOrgScope("/api/v1"))
	{
		var notificationSvc service.NotificationService
		var auditSvc service.AuditService
		var clientRequestSvc service.ClientRequestService
		var organizationSvc service.OrganizationService
		var departmentSvc service.DepartmentService
		var positionSvc service.PositionService
		var personSvc service.PersonService
		var employeeSvc service.EmployeeService
		var valuationOrgSvc service.ValuationOrgService
		var decisionSvcs service.DecisionServices
		var menuSvc service.MenuService
		var settingSvc service.SettingService
		for _, svc := range optionalSvcs {
			switch v := svc.(type) {
			case service.DecisionServices:
				decisionSvcs = v
			case service.ValuationOrgService:
				valuationOrgSvc = v
			case service.MenuService:
				menuSvc = v
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
			case service.SettingService:
				settingSvc = v
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
				organizations.POST("", middleware.RequirePermission("hr:structure:create"), adminOnly, hrH.CreateOrganization)
				organizations.PUT("/:id", middleware.RequirePermission("hr:update"), adminOnly, hrH.UpdateOrganization)
				organizations.DELETE("/:id", middleware.RequirePermission("hr:delete"), adminOnly, hrH.DeleteOrganization)
			}

			departments := protected.Group("/departments")
			departments.Use(denyExternal)
			{
				departments.GET("", middleware.RequirePermission("hr:read"), hrH.ListDepartments)
				departments.POST("", middleware.RequirePermission("hr:structure:create"), adminOnly, hrH.CreateDepartment)
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

		// Үнэлгээний (мэргэжлийн) байгууллагын app-owned profile.
		//
		// УНШИХ эрхийг land:read-тэй ажилтанд ч нээнэ: чөлөөлөлт/нэгж талбарт
		// үнэлгээний байгууллага сонгох сонгогч энэ жагсаалтаас тэжээгддэг.
		// БИЧИХ нь HR лавлахын адил зөвхөн admin.
		if valuationOrgSvc != nil {
			valuationOrgH := NewValuationOrgHandler(valuationOrgSvc)
			valuationOrgAdmin := middleware.RequireAnyRole("admin")
			valuationOrgs := protected.Group("/valuation-orgs")
			{
				valuationOrgs.GET("", middleware.RequireAnyPermission("hr:read", "land:read"), valuationOrgH.List)
				valuationOrgs.GET("/:id", middleware.RequireAnyPermission("hr:read", "land:read"), valuationOrgH.GetByID)
				valuationOrgs.POST("", middleware.RequirePermission("hr:create"), valuationOrgAdmin, valuationOrgH.Create)
				valuationOrgs.PUT("/:id", middleware.RequirePermission("hr:update"), valuationOrgAdmin, valuationOrgH.Update)
				valuationOrgs.DELETE("/:id", middleware.RequirePermission("hr:delete"), valuationOrgAdmin, valuationOrgH.Delete)
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
			roles.POST("/:id/permissions", middleware.RequirePermissionOrRole("roles:update", "admin"), roleH.AssignPermission)
			roles.DELETE("/:id/permissions/:permission_id", middleware.RequirePermissionOrRole("roles:update", "admin"), roleH.RemovePermission)
			// Цэс холбох (эрхийн 1-р давхарга) — menuSvc өгсөн үед л.
			if menuSvc != nil {
				menuH := NewMenuHandler(menuSvc, roleSvc)
				roles.GET("/:id/menus", middleware.RequirePermission("roles:read"), menuH.ListByRole)
				roles.POST("/:id/menus", middleware.RequirePermissionOrRole("roles:update", "admin"), menuH.Attach)
				roles.DELETE("/:id/menus/:menu_id", middleware.RequirePermissionOrRole("roles:update", "admin"), menuH.Detach)
				roles.PUT("/:id/menus/:menu_id/permissions", middleware.RequirePermissionOrRole("roles:update", "admin"), menuH.SetRoleMenuPermissions)
				roles.DELETE("/:id/menus/:menu_id/permissions", middleware.RequirePermissionOrRole("roles:update", "admin"), menuH.RemoveRoleMenuPermissions)
			}
		}

		if menuSvc != nil {
			menuH := NewMenuHandler(menuSvc)
			menus := protected.Group("/menus")
			{
				// Хэрэглэгчид ЗӨВХӨН өөрт харагдах цэс — эрх шаардахгүй.
				menus.GET("/me", menuH.Mine)
				menus.GET("", middleware.RequirePermission("roles:read"), menuH.List)
			}
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
		// fieldStage — НЭГЖ ТАЛБАРТАЙ АЖИЛЛАХ үйлдэл зөвхөн "Хээрийн судалгаа"
		// төлөвөөс эхэлнэ. "Шинэ" төлөв нь нэгж талбарын БҮРДЭЛ (татах,
		// хилээс гарсныг хасах, давхардлыг арилгах) хийгддэг шат тул тэнд
		// үнэлгээ/олговор/төлөв бичихийг хаана — эс бөгөөс бүрдэл өөрчлөгдөхөд
		// (хил солих, давхардал арилгах) бичигдсэн өгөгдөл нэгж талбартайгаа
		// хамт устах эрсдэлтэй. Нэгж талбар ТАТАХ маршрутууд (by-acquisition,
		// sync) энэ хаалтад ОРОХГҮЙ — тэр нь яг "Шинэ" төлөвийн ажил.
		fieldStage := middleware.RequireAcquisitionFieldStage(landSvc)
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
			// Шүүлтүүрийн dropdown-ы хөнгөн жагсаалт — НЭВТЭРСЭН БҮХ хэрэглэгчид
			// нээлттэй (роль/permission шалгахгүй). Жагсаалтын шүүлтүүрийг хэн ч
			// харж болно; бодит өгөгдлийн эрх нь жагсаалтын endpoint-ууд дээрээ
			// (GET "", /report/*) шалгагдсан хэвээр.
			land.GET("/filter-options", landH.FilterOptions)
			// Шүүлтүүрийн сум/дүүрэг — мөн нэвтэрсэн БҮХ хэрэглэгчид нээлттэй.
			land.GET("/districts", landH.Districts)
			land.POST("", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:create"), landH.Create)
			// Зөвшөөрөгдсөн ажилтан эсвэл ахлах мэргэжилтэн шалгах middleware
			land.GET("/:id", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.GetByID)
			land.GET("/:id/parcels", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListParcels)
			land.GET("/:id/assets", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListAssets)
			land.GET("/:id/compensations", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListCompensations)
			land.GET("/:id/parcels/:parcel_id", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.GetParcelByID)
			land.PUT("/:id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.Update)
			// Гараас оруулах хилийг ХАДГАЛАЛГҮЙ шалгах (WKT + талбайн зөрүү).
			// Хил солих нь буцаах боломжгүй тул хадгалахаас өмнөх урьдчилсан
			// харагдац — хамгаалалт нь PUT-тэй ЯГ ижил байх ёстой.
			land.POST("/:id/boundary-preview", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.PreviewBoundary)
			// POST/PUT-ийн адил ролийн хаалт: `land:delete`-тэй custom роль
			// хуваарилалтгүйгээр аль ч чөлөөлөлтийг устгах боломжтой байв.
			land.DELETE("/:id", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:delete"), assignedOrSenior, landH.Delete)
			// Assignee routes
			land.GET("/:id/assignees", middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetAssignees)
			land.PUT("/:id/assignees", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.SetAssignees)
			land.POST("/:id/assets", middleware.RequireAnyPermission("land:create", "compensation:create"), valuationEditor, fieldStage, landH.CreateAsset)
			land.PUT("/:id/assets/:asset_id", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.UpdateAsset)
			land.DELETE("/:id/assets/:asset_id", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.DeleteAsset)
			land.POST("/:id/assets/:asset_id/photos", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.UploadAssetPhoto)
			land.GET("/:id/assets/:asset_id/specs", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListAssetSpecs)
			land.POST("/:id/assets/:asset_id/specs", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.UpsertAssetSpecs)
			land.GET("/:id/assets/:asset_id/calculations", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListAssetCalculations)
			land.POST("/:id/assets/:asset_id/calculations", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.UpsertAssetCalculations)
			land.GET("/:id/land-valuation", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.GetLandValuation)
			land.POST("/:id/land-valuation", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.UpsertLandValuation)
			land.DELETE("/:id/land-valuation", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.DeleteLandValuation)
			land.POST("/:id/valuation-import", middleware.RequireAnyPermission("land:create", "compensation:create"), valuationEditor, fieldStage, landH.ImportValuation)
			// Үнэлгээний хүснэгт бүрийн ТАЙЛБАР (уншихад үнэлгээ харах эрх хангалттай)
			land.GET("/:id/valuation-notes", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListValuationNotes)
			land.PUT("/:id/valuation-notes", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.SaveValuationNotes)
			land.POST("/:id/compensations", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.CreateCompensation)
			land.PUT("/:id/compensations/:comp_id", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.UpdateCompensation)
			land.DELETE("/:id/compensations/:comp_id", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.DeleteCompensation)
			land.GET("/:id/compensations/:comp_id/history", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, landH.ListCompensationHistory)
			land.POST("/:id/compensations/:comp_id/approve", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү"), middleware.RequireAnyPermission("land:update", "compensation:update"), landH.ApproveCompensation)
			land.POST("/:id/compensations/:comp_id/reject", middleware.RequireAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү"), middleware.RequireAnyPermission("land:update", "compensation:update"), landH.RejectCompensation)
			// Үнэлгээний тайлан — МИКА өөрийн баталгаажсан урсгалдаа мөн хавсаргана (valuationEditor)
			land.POST("/:id/compensations/:comp_id/report", middleware.RequireAnyPermission("land:update", "compensation:update"), valuationEditor, fieldStage, landH.UploadCompensationReport)
			land.POST("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.CreateCompensationGrant)
			land.PUT("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.UpdateCompensationGrant)
			land.DELETE("/:id/compensations/:comp_id/grant", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.DeleteCompensationGrant)
			// Чөлөөлөх ХИЛЭЭР нэгж талбарыг тодорхойлж бүртгэнэ (дугаар л татна).
			// Дэлгэрэнгүйг дараа нь дугаар тус бүрээр доорх /sync-ээр татна.
			land.POST("/:id/parcels/by-acquisition", middleware.RequirePermission("land:create"), assignedOrSenior, landH.DiscoverParcels)
			land.POST("/:id/parcels/:parcel_id/sync", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncParcel)
			// Хавсралт татах: эх системд байгаа боловч энд байхгүй баримт л
			// нэмэгдэнэ (хадгалагдсан хавсралт хөндөгдөхгүй).
			land.POST("/:id/parcels/:parcel_id/sync/documents", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelDocuments)
			land.POST("/:id/parcels/:parcel_id/sync/holders", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelHolders)
			land.POST("/:id/parcels/:parcel_id/sync/base-price", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelBasePrice)
			land.POST("/:id/parcels/:parcel_id/sync/invoices", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelInvoices)
			land.POST("/:id/parcels/:parcel_id/sync/fee", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelFee)
			land.POST("/:id/parcels/:parcel_id/sync/mortgages", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelMortgages)
			land.POST("/:id/parcels/:parcel_id/sync/court-decisions", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelCourtDecisions)
			land.POST("/:id/parcels/:parcel_id/sync/monitoring", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelMonitorings)
			// УБЕГ-ийн газар өмчлөлийн бүртгэл — ГУС-аас БИШ, ХУР (XYP) гарцаас.
			land.POST("/:id/parcels/:parcel_id/sync/ownership", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SyncParcelOwnerships)
			land.POST("/:id/parcels/:parcel_id/sync/valuation", middleware.RequirePermission("land:create"), assignedOrSenior, landH.SyncValuation)
			land.PATCH("/:id/parcels/:parcel_id/valuation", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.UpdateParcelValuation)
			land.PUT("/:id/parcels/:parcel_id/compensation", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.SetParcelCompensation)
			// ТӨСӨӨЛЛИЙН үнэлгээ — мэргэжилтэн/админ ГААРАС оруулна.
			// Мэргэжлийн байгууллагад зөвхөн ХАРАГДАНА (уншилтын маршрутаар),
			// бичих эрх энд нээгдээгүй.
			land.PATCH("/:id/parcels/:parcel_id/estimated-value", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.SetParcelEstimatedValue)
			// Түүх нь УНШИХ эрхээр (land:read) — санхүү/захирамжийн ажилтан ч
			// үнэлгээ хэрхэн бүрдсэнийг хянах шаардлагатай, засах эрх шаардахгүй.
			land.GET("/:id/parcels/:parcel_id/estimated-value/history", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListParcelEstimatedValueHistory)
			land.PATCH("/:id/parcels/:parcel_id/meta", middleware.RequirePermission("land:update"), assignedOrSenior, fieldStage, landH.UpdateParcelMeta)
			// Нөхөх олговрын үнэлгээний илгээх/зөвшөөрөх төлөв (санхүү зөвшөөрөх/буцаах).
			// POST-ийн ролийн шалгалт handler дотор (approve/return=санхүү, submit=мэрг.байгуулга/ахлах).
			land.GET("/:id/parcels/:parcel_id/valuation-status", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, parcelWorkflowH.GetValuationSubmission)
			land.GET("/:id/parcels/:parcel_id/valuation-status-history", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, parcelWorkflowH.ListValuationSubmissionHistory)
			land.GET("/:id/parcels/:parcel_id/valuation-snapshots", middleware.RequireAnyPermission("land:read", "compensation:read"), assignedOrSenior, parcelWorkflowH.ListValuationSnapshots)
			land.POST("/:id/parcels/:parcel_id/valuation-status", middleware.RequireAnyPermission("land:update", "compensation:update"), middleware.RequireAssignedOrSenior(landSvc, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "mika", "МИКА"), fieldStage, parcelWorkflowH.TransitionValuationSubmission)
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
			// Нөхөн төлбөр хүлээн авах эзэмшигчийг сонгох (үндсэн эзэмшигч олон байх үед)
			land.PUT("/:id/parcels/:parcel_id/payment-recipient", middleware.RequirePermission("land:update"), assignedOrSenior, landH.SetParcelPaymentRecipient)
			land.GET("/:id/documents", middleware.RequirePermission("land:read"), assignedOrSenior, landH.ListAcquisitionDocuments)
			land.POST("/:id/documents", middleware.RequirePermission("land:update"), assignedOrSenior, landH.UploadAcquisitionDocument)
			land.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), assignedOrSenior, landH.DeleteAcquisitionDocument)
			// Нийгэм, эдийн засгийн судалгаа — чөлөөлөлтөд НЭГ бүртгэл
			// (дахин оруулахад орлогдоно). Судалгааны файл нь ЗӨВХӨН PDF.
			land.GET("/:id/socio-survey", middleware.DenyAnyRole("finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "mika", "МИКА"), middleware.RequirePermission("land:read"), assignedOrSenior, landH.GetAcquisitionSocioSurvey)
			land.POST("/:id/socio-survey", middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, landH.SaveAcquisitionSocioSurvey)
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
			// Хөндлөнгийн үнэлгээчнийг ЭНД зөвхөн дотоод ахлах мэргэжилтэн
			// сольно; үнэлгээний байгууллага өөрийн /prof замаа хэрэглэнэ
			// (дотоод маршрутууд тэдэнд allowlist-ээр хаалттай).
			land.PATCH("/:id/parcels/:parcel_id/independent-org", middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн"), middleware.RequirePermission("land:update"), landH.SetParcelIndependentOrg)
		}

		// Үнэлгээний байгууллагын ажлын орон зай.
		//
		// ЭРХ: sdplatform-ийн роль/эрх (professional_org, compensation:*) ЭНД
		// ХЭРЭГЛЭГДЭХГҮЙ — хаалт нь аппын өөрийн бүртгэл (токены
		// `valuation_org` = valuation_org_employee дээр идэвхтэй ажилтан).
		// Өгөгдлийн хүрээг дараагийн хаалтууд шийднэ: profOrgMember нь
		// чөлөөлөлтийн үндсэн байгууллага, profParcelMember* нь нэгж талбарын
		// хөндлөнгийн байгууллага мөн эсэхийг тулгана.
		prof := protected.Group("/prof")
		prof.Use(middleware.RequireValuationOrgCaller())
		{
			prof.GET("/construction-types", landH.ListConstructionTypes)
			prof.GET("/asset-spec-types", assetTypeConfigH.ListSpecTypes)
			prof.GET("/asset-calc-types", assetTypeConfigH.ListCalcTypes)

			profLand := prof.Group("/land-acquisitions")
			{
				profLand.GET("", landH.ListMyAcquisitions)

				// /:id/* — тухайн чөлөөлөлтийн гишүүн мөн эсэхийг шалгана
				profLandItem := profLand.Group("/:id")
				profLandItem.Use(profNotLocked)
				profLandItem.Use(profOrgMember)
				{
					profLandItem.GET("", landH.GetByID)
					profLandItem.GET("/parcels", landH.ListEvaluationParcels)
					profLandItem.GET("/parcels/:parcel_id", profParcelMemberByParam, landH.GetParcelByID)
					profLandItem.PATCH("/parcels/:parcel_id/independent-org", profParcelMemberByParam, landH.SetParcelIndependentOrg)
					// Нэгж талбарын статус
					profLandItem.GET("/parcels/:parcel_id/available-statuses", profParcelMemberByParam, parcelWorkflowH.GetAvailableStatuses)
					// "Шинэ" төлөвт явц солихгүй (үндсэн маршруттай ижил хаалт).
					profLandItem.PATCH("/parcels/:parcel_id/status", profParcelMemberByParam, fieldStage, parcelWorkflowH.UpdateParcelStatus)
					profLandItem.GET("/parcels/:parcel_id/status-history", profParcelMemberByParam, parcelWorkflowH.ListParcelStatusHistory)
					// Нөхөх олговрын үнэлгээний илгээх төлөв (мэрг. байгууллага илгээнэ)
					profLandItem.GET("/parcels/:parcel_id/valuation-status", profParcelMemberByParam, parcelWorkflowH.GetValuationSubmission)
					profLandItem.POST("/parcels/:parcel_id/valuation-status", profParcelMemberByParam, parcelWorkflowH.TransitionValuationSubmission)
					profLandItem.GET("/parcels/:parcel_id/valuation-status-history", profParcelMemberByParam, parcelWorkflowH.ListValuationSubmissionHistory)
					profLandItem.GET("/parcels/:parcel_id/valuation-snapshots", profParcelMemberByParam, parcelWorkflowH.ListValuationSnapshots)
					// Эрх эзэмшигч төлөөлөгч
					profLandItem.GET("/parcels/:parcel_id/representatives", profParcelMemberByParam, landH.ListAuthorizedRepresentatives)
					profLandItem.POST("/parcels/:parcel_id/representatives", profParcelMemberByParam, landH.CreateAuthorizedRepresentative)
					profLandItem.DELETE("/parcels/:parcel_id/representatives/:rep_id", profParcelMemberByParam, landH.DeleteAuthorizedRepresentative)
					profLandItem.PUT("/parcels/:parcel_id/payment-recipient", profParcelMemberByParam, landH.SetParcelPaymentRecipient)
					// Байгааламж — үнэлгээ (valuation:create/update)
					profLandItem.GET("/assets", landH.ListAssets)
					profLandItem.POST("/assets", landH.CreateAsset)
					// Хөрөнгийн мэдээллийг дэлгэцээс шууд засах (нэр, хэмжээ, нэгж үнэ г.м)
					profLandItem.PUT("/assets/:asset_id", landH.UpdateAsset)
					profLandItem.DELETE("/assets/:asset_id", landH.DeleteAsset)
					profLandItem.POST("/assets/:asset_id/photos", landH.UploadAssetPhoto)
					profLandItem.GET("/assets/:asset_id/specs", landH.ListAssetSpecs)
					profLandItem.POST("/assets/:asset_id/specs", landH.UpsertAssetSpecs)
					profLandItem.GET("/assets/:asset_id/calculations", landH.ListAssetCalculations)
					profLandItem.POST("/assets/:asset_id/calculations", landH.UpsertAssetCalculations)
					// Газрын үнэлгээ (valuation:update)
					profLandItem.GET("/land-valuation", landH.GetLandValuation)
					profLandItem.POST("/land-valuation", landH.UpsertLandValuation)
					profLandItem.DELETE("/land-valuation", landH.DeleteLandValuation)
					profLandItem.POST("/valuation-import", landH.ImportValuation)
					// Хүснэгтийн тайлбар — мэрг. байгууллага өөрийн урсгалдаа бичнэ
					profLandItem.GET("/valuation-notes", landH.ListValuationNotes)
					profLandItem.PUT("/valuation-notes", landH.SaveValuationNotes)
					// Нөхөх олговор — үнэлгээ (valuation:create/update)
					profLandItem.GET("/compensations", landH.ListCompensations)
					profLandItem.POST("/compensations", landH.CreateCompensation)
					// Олговрын дүн/тайлбарыг дэлгэцээс засах. Байхгүй байсан тул
					// мэргэжлийн байгууллагын засвар 404 өгдөг байв (үндсэн
					// маршрутад аль хэдийн байсан).
					profLandItem.PUT("/compensations/:comp_id", landH.UpdateCompensation)
					profLandItem.DELETE("/compensations/:comp_id", landH.DeleteCompensation)
					profLandItem.GET("/compensations/:comp_id/history", landH.ListCompensationHistory)
					profLandItem.POST("/compensations/:comp_id/report", landH.UploadCompensationReport)
					profLandItem.POST("/compensations/:comp_id/grant", landH.CreateCompensationGrant)
					profLandItem.PUT("/compensations/:comp_id/grant", landH.UpdateCompensationGrant)
					profLandItem.DELETE("/compensations/:comp_id/grant", landH.DeleteCompensationGrant)
					// Баримт бичиг (чөлөөлөлт)
					profLandItem.GET("/documents", landH.ListAcquisitionDocuments)
					profLandItem.POST("/documents", landH.UploadAcquisitionDocument)
					profLandItem.DELETE("/documents/:doc_id", landH.DeleteAcquisitionDocument)
					profLandItem.GET("/socio-survey", landH.GetAcquisitionSocioSurvey)
				}
			}

			// Нэгж талбарын баримт бичиг — хөрөнгийн зураг (үнэлгээний хэсэг)
			profParcels := prof.Group("/parcels")
			profParcels.Use(profNotParcelLocked)
			profParcels.Use(profParcelMember)
			{
				profParcels.GET("/:id/documents", landH.ListParcelDocuments)
				profParcels.GET("/:id/documents/:doc_id/file", landH.GetParcelDocumentFile)
				profParcels.POST("/:id/documents", landH.UploadParcelDocument)
				profParcels.DELETE("/:id/documents/:doc_id", landH.DeleteParcelDocument)
			}
		}

		// Газар зохион байгуулалтын төлөвлөвлөгөө
		plans := protected.Group("/plans")
		{
			// Үнэлгээний байгууллага эрх (permission) эзэмшихгүй тул түүнийг
			// токеноороо нэвтрүүлнэ — төлөвлөгөөний код/нэр нь шүүлтүүрийн
			// хөнгөн лавлах бөгөөд чөлөөлөлтийн өгөгдөл агуулахгүй.
			planLookup := middleware.AllowValuationOrgOr(
				middleware.RequireAnyPermission("land:read", "compensation:read"))
			plans.GET("/search", planLookup, planH.Search)
			plans.GET("/suggest", planLookup, planH.Suggest)
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

		// Захирамжийн төсөл. Захирамжийн төсөлтэй ажиллах мэргэжилтэн ЗӨВХӨН
		// decision:* эрхээр ажиллана (land:* эрх энд хүчинтэй БИШ). Сонголтын
		// хоёр жагсаалт (Ажлын төрөл, Төсөв) нь бусад lookup-ийн адил admin:*
		// эрхээр удирдагдаж, GET нь нэвтэрсэн хэрэглэгч бүрт нээлттэй.
		if decisionSvcs.Draft != nil {
			decisionH := NewDecisionDraftHandler(decisionSvcs.Draft)

			decisionDrafts := protected.Group("/decision-drafts")
			{
				decisionDrafts.GET("", middleware.RequirePermission("decision:read"), decisionH.List)
				decisionDrafts.GET("/:id", middleware.RequirePermission("decision:read"), decisionH.GetByID)
				decisionDrafts.POST("", middleware.RequirePermission("decision:create"), decisionH.Create)
				decisionDrafts.PUT("/:id", middleware.RequirePermission("decision:update"), decisionH.Update)
				decisionDrafts.DELETE("/:id", middleware.RequirePermission("decision:delete"), decisionH.Delete)
				decisionDrafts.POST("/:id/confirm", middleware.RequirePermission("decision:update"), decisionH.Confirm)
				decisionDrafts.GET("/:id/progress-history", middleware.RequirePermission("decision:read"), decisionH.ListProgressHistory)
				decisionDrafts.POST("/:id/progress", middleware.RequirePermission("decision:update"), decisionH.AddProgress)
				decisionDrafts.GET("/:id/parcels", middleware.RequirePermission("decision:read"), decisionH.ListParcels)
				decisionDrafts.POST("/:id/parcels", middleware.RequirePermission("decision:update"), decisionH.LinkParcel)
				decisionDrafts.DELETE("/:id/parcels/:link_id", middleware.RequirePermission("decision:update"), decisionH.UnlinkParcel)
				decisionDrafts.PATCH("/:id/parcels/:link_id/funding-source", middleware.RequirePermission("decision:update"), decisionH.SetParcelFundingSource)
				// Санхүүгийн эх үүсвэр — төслийг үүсгэсний дараа олноор нэмнэ
				decisionDrafts.GET("/:id/funding-sources", middleware.RequirePermission("decision:read"), decisionH.ListDraftFundingSources)
				decisionDrafts.POST("/:id/funding-sources", middleware.RequirePermission("decision:update"), decisionH.AddFundingSource)
				decisionDrafts.DELETE("/:id/funding-sources/:link_id", middleware.RequirePermission("decision:update"), decisionH.RemoveFundingSource)
			}

			// Нэгж талбарын "Захирамж" таб
			protected.GET("/parcels/:id/decision-drafts",
				middleware.RequirePermission("decision:read"), decisionH.ListByParcel)

			// Санхүүгийн эх үүсвэрийн сонголт (бүх чөлөөлөлтөөр)
			protected.GET("/funding-sources",
				middleware.RequirePermission("decision:read"), decisionH.ListFundingSourceOptions)
			// Гараар шинэ эх үүсвэр бүртгэх (чөлөөлөлтгүй, бие даасан мөр)
			protected.POST("/funding-sources",
				middleware.RequirePermission("decision:create"), decisionH.CreateFundingSourceOption)
		}
		if decisionSvcs.WorkType != nil {
			workTypeH := NewDecisionOptionHandler(decisionSvcs.WorkType)
			workTypes := protected.Group("/decision-work-types")
			{
				workTypes.GET("", workTypeH.List)
				workTypes.GET("/:id", workTypeH.GetByID)
				workTypes.POST("", middleware.RequirePermission("admin:create"), workTypeH.Create)
				workTypes.PUT("/:id", middleware.RequirePermission("admin:update"), workTypeH.Update)
				workTypes.DELETE("/:id", middleware.RequirePermission("admin:delete"), workTypeH.Delete)
			}
		}
		if decisionSvcs.Budget != nil {
			budgetH := NewDecisionOptionHandler(decisionSvcs.Budget)
			budgets := protected.Group("/decision-budgets")
			{
				budgets.GET("", budgetH.List)
				budgets.GET("/:id", budgetH.GetByID)
				budgets.POST("", middleware.RequirePermission("admin:create"), budgetH.Create)
				budgets.PUT("/:id", middleware.RequirePermission("admin:update"), budgetH.Update)
				budgets.DELETE("/:id", middleware.RequirePermission("admin:delete"), budgetH.Delete)
			}
		}

		// Баримт бичгийн төрөл
		// Системийн тохиргоо. Суурь зургийг БҮХ хэрэглэгч уншина (газрын зураг
		// ижил зурагтай байх ёстой), харин СОЛИХ нь admin:update эрхтэйд.
		if settingSvc != nil {
			settingH := NewSettingHandler(settingSvc)
			settings := protected.Group("/settings")
			{
				settings.GET("/basemap", settingH.GetBasemap)
				settings.PUT("/basemap", middleware.RequirePermission("admin:update"), settingH.SaveBasemap)
				settings.DELETE("/basemap", middleware.RequirePermission("admin:update"), settingH.ResetBasemap)
			}
		}

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
		//
		// Үнэлгээний байгууллагын ажилтан эрхгүйгээр (роль/permission-гүй)
		// нэвтэрнэ — handler дотор org_id-гаар ХАТУУ шүүгдэнэ; org_id байхгүй
		// бол 403.
		protected.GET("/dashboard",
			middleware.AllowValuationOrgOr(
				middleware.RequirePermissionOrRole("land:read", "mika", "finance_specialist")),
			landH.Dashboard)

		// Файлын объектын эрх — frontend-ийн `/api/files` proxy түлхүүр тус бүрд
		// энэ маршрутыг дуудна. Эрхийн дүрэм нь handler дотор (объектын эзний
		// чөлөөлөлт/нэгж талбарт буулгаж шалгана) тул энд нэмэлт хаалт хэрэггүй.
		fileAccessH := NewFileAccessHandler(landSvc)
		protected.GET("/files/authorize", fileAccessH.Authorize)

		// Тайлангийн маршрут.
		//
		// ЭРХ: тайлан нь БҮХ чөлөөлөлтийн нэгж талбарыг хуваарилалт үл харгалзан
		// нэгтгэдэг тул `land:read` дангаараа хангалтгүй — хуваарилагдаагүй
		// мэргэжилтэн байгууллага даяарх өгөгдлийг Excel-ээр татаж чаддаг байв.
		// Frontend-ийн ГЧХ тайлан (`/api/report/gchh2`) аль хэдийн зөвхөн ахлах
		// мэргэжилтэнд нээлттэй — backend-ийг мөн тэр бодлогод нийцүүлэв.
		reportSenior := middleware.RequireAnyRole("senior_specialist", "Ахлах мэргэжилтэн")
		report := protected.Group("/report")
		{
			report.GET("/download", reportSenior, middleware.RequirePermission("land:read"), landH.ListReportParcels)
			report.GET("/summary", reportSenior, middleware.RequirePermission("land:read"), landH.ReportSummary)
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

		// Геометрийн туслах тооцоо — WKT хилээс талбай (ГУС-тай ИЖИЛ, UTM).
		// Дуудагч өөрийн хилээ явуулж, түүнийхээ талбайг авч байгаа тул
		// нэмэлт эрхийн шалгалт хэрэггүй — нэвтэрсэн байхад л хангалттай.
		protected.POST("/geometry/area", landH.ComputeGeometryArea)

		// Глобал нэгж талбарын маршрут
		parcelAssignedOrSenior := middleware.RequireParcelAssignedOrSenior(landSvc)
		parcels := protected.Group("/parcels")
		parcels.Use(notParcelLocked)
		{
			parcels.GET("", middleware.RequirePermission("land:read"), landH.ListGlobalParcels)
			// ЖИЧ: :id-тай маршрутуудад parcelAssignedOrSenior ЗААВАЛ байна.
			// Түүнгүйгээр `/land-acquisitions/:id/parcels/:parcel_id`-ийн
			// хуваарилалтын хаалтыг энэ глобал замаар бүрэн тойрч болдог байв
			// (хуваарилагдаагүй мэргэжилтэн бүх нэгж талбарын баримт/төлбөрийг
			// уншина). Санхүү/МИКА-гийн хяналтын УНШИХ эрх middleware дотор хэвээр.
			parcels.GET("/:id/payments", middleware.RequirePermission("land:read"), parcelAssignedOrSenior, landH.ListParcelPayments)
			parcels.POST("/:id/payments", middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.AddParcelPayment)
			// Үнэлгээний тайлан Нөхөх олговор табд харагддаг тул гадаад ролиуд (санхүү/МИКА)
			// мөн уншина; мэрг. байгууллага өөрийн /prof (гишүүнчлэлтэй) замаа ашиглана.
			parcels.GET("/:id/documents", middleware.DenyAnyRole("professional_org"), middleware.RequireAnyPermission("land:read", "compensation:read"), parcelAssignedOrSenior, landH.ListParcelDocuments)
			// Эх системээс татсан хавсралтын файл — API нь эх системийн файл
			// серверээс уншиж дамжуулна (?download=1 бол татна).
			parcels.GET("/:id/documents/:doc_id/file", middleware.DenyAnyRole("professional_org"), middleware.RequireAnyPermission("land:read", "compensation:read"), parcelAssignedOrSenior, landH.GetParcelDocumentFile)
			parcels.POST("/:id/documents", middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.UploadParcelDocument)
			parcels.DELETE("/:id/documents/:doc_id", middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.DeleteParcelDocument)
			// Урьдчилан мэдэгдэх хуудсыг иргэнд имэйлээр хүргүүлэх ба илгээсэн
			// түүх. PDF-ийг frontend загвараас үүсгэж дамжуулна.
			parcels.GET("/:id/notice-emails", middleware.RequireAnyRole("admin"), middleware.RequirePermission("land:read"), parcelAssignedOrSenior, landH.ListParcelNoticeEmails)
			parcels.POST("/:id/notice-emails", middleware.RequireAnyRole("admin"), middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.SendParcelNoticeEmail)
			// E-Mongolia-гаар мэдэгдэл хүргүүлэх (2.8.5). Гадаад холболт МОК.
			parcels.GET("/:id/emongolia-notices", middleware.RequireAnyRole("admin"), middleware.RequirePermission("land:read"), parcelAssignedOrSenior, landH.ListParcelEMongoliaNotices)
			parcels.POST("/:id/emongolia-notices", middleware.RequireAnyRole("admin"), middleware.RequirePermission("land:update"), parcelAssignedOrSenior, landH.SendParcelEMongoliaNotice)
		}

		// Нэгж талбарын статус солих + боломжтой статусууд + түүх
		protected.GET("/land-acquisitions/:id/parcels/:parcel_id/available-statuses",
			middleware.RequirePermission("land:read"), assignedOrSenior, parcelWorkflowH.GetAvailableStatuses)
		// fieldStage: чөлөөлөлт "Шинэ" төлөвт байхад нэгж талбарын ЯВЦ СОЛИХГҮЙ.
		// Тэр шат нь нэгж талбарын БҮРДЭЛ (татах, хилээс гарсныг хасах,
		// давхардал арилгах) хийгддэг — бүрдэл өөрчлөгдөхөд бичигдсэн явц нь
		// нэгж талбартайгаа хамт устах эрсдэлтэй. Үнэлгээ/олговортой ижил
		// хаалт (estimated-value маршрутыг үз).
		protected.PATCH("/land-acquisitions/:id/parcels/:parcel_id/status",
			middleware.RequirePermission("land:update"), assignedOrSenior, notLocked, fieldStage, parcelWorkflowH.UpdateParcelStatus)
		protected.GET("/land-acquisitions/:id/parcels/:parcel_id/status-history",
			middleware.RequirePermission("land:read"), assignedOrSenior, parcelWorkflowH.ListParcelStatusHistory)
	}

	return r
}
