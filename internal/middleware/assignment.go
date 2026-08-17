package middleware

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AssignmentChecker — IsAssigned шалгах интерфейс (LandService-ийг тойруулан дамжуулахад ашиглана)
type AssignmentChecker interface {
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID domain.ID) (bool, error)
}

// AcquisitionLockChecker — баталгаажсан чөлөөлөлтийн хаалтыг шалгах интерфейс
// (routes where :id = acquisition_id)
type AcquisitionLockChecker interface {
	IsAcquisitionLocked(ctx context.Context, id uuid.UUID) (bool, error)
}

// ParcelLockChecker — нэгж талбарын чөлөөлөлтийн хаалтыг шалгах интерфейс
// (routes where :id = parcel_id)
type ParcelLockChecker interface {
	IsParcelAcquisitionLocked(ctx context.Context, parcelID uuid.UUID) (bool, error)
}

// ProfMemberChecker — professional_org хэрэглэгч тухайн чөлөөлөлтийн
// үндсэн байгуулга эсвэл хөндлөнгийн байгуулга мөн эсэхийг шалгах интерфейс
type ProfMemberChecker interface {
	IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, orgID domain.ID) (bool, error)
}

// ProfParcelMemberChecker — professional_org хэрэглэгч parcel-ээр дамжуулан тухайн
// чөлөөлөлтийн үндсэн байгуулга эсвэл хөндлөнгийн байгуулга мөн эсэхийг шалгана.
type ProfParcelMemberChecker interface {
	IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, orgID domain.ID) (bool, error)
}

// хяналтын (байгууллага даяар харах эрхтэй) ролиуд — санхүүгийн мэргэжилтэн, МИКА
var supervisoryReadRoles = map[string]bool{
	"finance_specialist":     true,
	"Санхүүгийн мэргэжилтэн": true,
	"Санхүү":                 true,
	"mika":                   true,
	"МИКА":                   true,
	"decision_registration":  true,
	"Захирамж бүртгэл":       true,
}

// RequireAssignedOrSenior — ахлах мэргэжилтэн эсвэл
// тухайн чөлөөлөлтэд хуваарилагдсан ажилтан байвал зөвшөөрнө, эс бөгөөс 403 буцаана.
// Санхүүгийн мэргэжилтэн болон МИКА хяналтын үүрэгтэй тул хуваарилалтгүйгээр
// ЗӨВХӨН унших (GET/HEAD/OPTIONS) хүсэлтэд нэвтэрнэ — бичих үйлдэл нь
// RequirePermission/RequireAnyRole хаалтуудаараа хамгаалагдсан хэвээр.
// extraWriteRoles — нэмэлтээр бичих эрхтэй нэвтрэх ролиуд (ж: үнэлгээний
// маршрутуудад МИКА өөрийн урсгалдаа бичдэг).
func RequireAssignedOrSenior(svc AssignmentChecker, extraWriteRoles ...string) gin.HandlerFunc {
	extra := make(map[string]bool, len(extraWriteRoles))
	for _, r := range extraWriteRoles {
		extra[r] = true
	}
	return func(c *gin.Context) {
		method := c.Request.Method
		isRead := method == "GET" || method == "HEAD" || method == "OPTIONS"

		// Ахлах мэргэжилтэн бол шалгуур шаардахгүй; санхүү/МИКА зөвхөн унших үед.
		rolesAny, _ := c.Get(CtxRoles)
		if roles, ok := rolesAny.([]string); ok {
			for _, r := range roles {
				if r == "admin" || r == "Админ" || r == "senior_specialist" || r == "Ахлах мэргэжилтэн" || (isRead && supervisoryReadRoles[r]) || extra[r] {
					c.Next()
					return
				}
			}
		}

		acqID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		userIDRaw, _ := c.Get(CtxUserID)
		userID, ok := userIDRaw.(domain.ID)
		if !ok {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		assigned, err := svc.IsAssigned(c.Request.Context(), acqID, userID)
		if err != nil || !assigned {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		c.Next()
	}
}

// isSenior — контекстоос ахлах мэргэжилтэн эсэхийг шалгана
func IsSenior(c *gin.Context) bool {
	rolesAny, _ := c.Get(CtxRoles)
	roles, ok := rolesAny.([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == "admin" || r == "Админ" || r == "senior_specialist" || r == "Ахлах мэргэжилтэн" {
			return true
		}
	}
	return false
}

// GetCallerID — контекстоос userID авна
func GetCallerID(c *gin.Context) (domain.ID, bool) {
	raw, _ := c.Get(CtxUserID)
	id, ok := raw.(domain.ID)
	return id, ok
}

// GetCallerOrgID — контекстоос хэрэглэгчийн ХАРЬЯА байгууллагын ID авна.
// Ажилтны бүртгэлгүй (иймд байгууллагад харьяалагдаагүй) хэрэглэгчид ok=false.
func GetCallerOrgID(c *gin.Context) (domain.ID, bool) {
	raw, _ := c.Get(CtxOrgID)
	id, ok := raw.(domain.ID)
	return id, ok
}

// GetCallerName — контекстоос хэрэглэгчийн нэр авна
func GetCallerName(c *gin.Context) string {
	raw, _ := c.Get(CtxUserName)
	name, _ := raw.(string)
	return name
}

// ParcelAssignmentChecker — хэрэглэгч parcel-ийн харгалзах чөлөөлөлтөд
// хуваарилагдсан эсэхийг шалгах интерфейс (routes where :id = parcel_id).
type ParcelAssignmentChecker interface {
	IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID domain.ID) (bool, error)
}

// RequireParcelAssignedOrSenior — /parcels/:id/* маршрутуудад: ахлах
// мэргэжилтэн эсвэл тухайн parcel-ийн чөлөөлөлтөд хуваарилагдсан ажилтан л нэвтэрнэ.
//
// RequireAssignedOrSenior-ийн адилаар санхүү/МИКА хяналтын үүрэгтэй тул
// хуваарилалтгүйгээр ЗӨВХӨН унших (GET/HEAD/OPTIONS) хүсэлтэд нэвтэрнэ.
// Энэ чөлөөлөл байхгүй бол нөхөх олговрын таб дахь үнэлгээний тайлан тэдэнд
// хаагдана; байхгүй үед харин хуваарилагдаагүй мэргэжилтэн нэгж талбарын
// баримт/төлбөрийг глобал замаар уншиж, чөлөөлөлтийн замын хаалтыг тойрдог байв.
func RequireParcelAssignedOrSenior(svc ParcelAssignmentChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		isRead := method == "GET" || method == "HEAD" || method == "OPTIONS"

		rolesAny, _ := c.Get(CtxRoles)
		if roles, ok := rolesAny.([]string); ok {
			for _, r := range roles {
				if r == "admin" || r == "Админ" || r == "senior_specialist" || r == "Ахлах мэргэжилтэн" ||
					(isRead && supervisoryReadRoles[r]) {
					c.Next()
					return
				}
			}
		}

		parcelID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}
		userIDRaw, _ := c.Get(CtxUserID)
		userID, ok := userIDRaw.(domain.ID)
		if !ok {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}
		assigned, err := svc.IsAssignedToParcel(c.Request.Context(), parcelID, userID)
		if err != nil || !assigned {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}
		c.Next()
	}
}

// RequireAcquisitionNotLocked — баталгаажсан (status=3) чөлөөлөлтийн бичих
// хүсэлтийг 423 Locked-ээр буцаана. GET/HEAD/OPTIONS болон `:id`-гүй замуудад
// шалгуур хэрэглэхгүй.
func RequireAcquisitionNotLocked(svc AcquisitionLockChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}
		idStr := c.Param("id")
		if idStr == "" {
			c.Next()
			return
		}
		acqID, err := uuid.Parse(idStr)
		if err != nil {
			c.Next()
			return
		}
		locked, err := svc.IsAcquisitionLocked(c.Request.Context(), acqID)
		if err != nil {
			// Олдоогүй эсвэл DB алдаа — handler-д шилжүүлнэ
			c.Next()
			return
		}
		if locked {
			response.Abort(c, 423, i18n.ErrAcqLocked)
			return
		}
		c.Next()
	}
}

// RequireParcelAcquisitionNotLocked — /parcels/:id/* маршрутад ашиглана.
// `:id` нь parcel_id — харгалзах чөлөөлөлтийн статусыг шалгана.
func RequireParcelAcquisitionNotLocked(svc ParcelLockChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}
		idStr := c.Param("id")
		if idStr == "" {
			c.Next()
			return
		}
		parcelID, err := uuid.Parse(idStr)
		if err != nil {
			c.Next()
			return
		}
		locked, err := svc.IsParcelAcquisitionLocked(c.Request.Context(), parcelID)
		if err != nil {
			c.Next()
			return
		}
		if locked {
			response.Abort(c, 423, i18n.ErrAcqLocked)
			return
		}
		c.Next()
	}
}

// RequireProfAcquisitionNotLocked — /prof/land-acquisitions/:id/* маршрутуудад ашиглана.
// GET-г оролцуулан БҮРТГЭЛ HTTP method-ийг шалгана: баталгаажсан (status=3)
// чөлөөлөлтийн бүх хүсэлтийг 423-аар буцаана.
func RequireProfAcquisitionNotLocked(svc AcquisitionLockChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		idStr := c.Param("id")
		if idStr == "" {
			c.Next()
			return
		}
		acqID, err := uuid.Parse(idStr)
		if err != nil {
			c.Next()
			return
		}
		locked, err := svc.IsAcquisitionLocked(c.Request.Context(), acqID)
		if err != nil {
			c.Next()
			return
		}
		if locked {
			response.Abort(c, 423, i18n.ErrAcqLocked)
			return
		}
		c.Next()
	}
}

// RequireProfParcelAcquisitionNotLocked — /prof/parcels/:id/* маршрутуудад ашиглана.
// GET-г оролцуулан БҮРТГЭЛ HTTP method-ийг шалгана: парцелийн харгалзах
// чөлөөлөлт баталгаажсан бол 423-аар буцаана.
func RequireProfParcelAcquisitionNotLocked(svc ParcelLockChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		idStr := c.Param("id")
		if idStr == "" {
			c.Next()
			return
		}
		parcelID, err := uuid.Parse(idStr)
		if err != nil {
			c.Next()
			return
		}
		locked, err := svc.IsParcelAcquisitionLocked(c.Request.Context(), parcelID)
		if err != nil {
			c.Next()
			return
		}
		if locked {
			response.Abort(c, 423, i18n.ErrAcqLocked)
			return
		}
		c.Next()
	}
}

// RequireProfOrgMember — /prof/land-acquisitions/:id/* маршрутуудад ашиглана.
// Хэрэглэгчийн ХАРЬЯА БАЙГУУЛЛАГА (JWT-ийн org_id) нь тухайн чөлөөлөлтийн
// үндсэн байгууллага (professional_org_id) эсвэл аль нэгэн нэгж талбарын
// хараат бус байгууллага (independent_org_id) мөн эсэхийг шалгана. Таарахгүй
// бол 403.
//
// Өмнө нь хэрэглэгчийн ID-тай тулгадаг байсан тул нэг байгууллагад ганц
// хэрэглэгч байж чаддаг байв; одоо тухайн байгууллагын аль ч ажилтан ажиллана.
func RequireProfOrgMember(svc ProfMemberChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		acqID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		orgID, ok := GetCallerOrgID(c)
		if !ok {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		member, err := svc.IsProfMemberOfAcquisition(c.Request.Context(), acqID, orgID)
		if err != nil || !member {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		c.Next()
	}
}

// RequireProfParcelMember — /prof/parcels/:id/* маршрутуудад ашиглана.
// `:id` нь parcel UUID бөгөөд professional_org хэрэглэгч тухайн parcel-ийн
// чөлөөлөлтөд хамаарах эсэхийг шалгана.
func RequireProfParcelMember(svc ProfParcelMemberChecker) gin.HandlerFunc {
	return RequireProfParcelMemberParam(svc, "id")
}

// RequireProfParcelMemberParam — parcel UUID-г өгсөн path параметрээс уншиж,
// professional_org хэрэглэгч тухайн parcel-д хамаарах (чөлөөлөлтийн үндсэн
// байгууллага эсвэл тухайн parcel-ийн хөндлөнгийн байгууллага) эсэхийг шалгана.
// /prof/land-acquisitions/:id/parcels/:parcel_id/* маршрутуудад "parcel_id"-ээр
// ашиглаж, зэргэлдээ parcel руу халдахаас сэргийлнэ.
func RequireProfParcelMemberParam(svc ProfParcelMemberChecker, param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		parcelID, err := uuid.Parse(c.Param(param))
		if err != nil {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		orgID, ok := GetCallerOrgID(c)
		if !ok {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		member, err := svc.IsProfMemberOfParcel(c.Request.Context(), parcelID, orgID)
		if err != nil || !member {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		c.Next()
	}
}

var _ = domain.ErrForbidden // ensure domain import used
