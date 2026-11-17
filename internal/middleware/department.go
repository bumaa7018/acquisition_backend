package middleware

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IsSystemAdmin — БҮХ хэлтсийн хэрэглэгч, мэдээллийг харах цорын ганц роль.
// Бусад дотоод хэрэглэгч (хэлтсийн дарга, ажилтан) зөвхөн өөрийн хэлтсийнхийг.
func IsSystemAdmin(c *gin.Context) bool {
	rolesAny, _ := c.Get(CtxRoles)
	roles, _ := rolesAny.([]string)
	for _, r := range roles {
		if domain.IsSystemAdminLevel(r) {
			return true
		}
	}
	return false
}

// CallerDepartmentID — дуудагчийн хэлтэс (токены department_id). Хэлтэсгүй бол nil.
func CallerDepartmentID(c *gin.Context) *domain.ID {
	raw, ok := c.Get(CtxDepartmentID)
	if !ok {
		return nil
	}
	id, ok := raw.(domain.ID)
	if !ok {
		return nil
	}
	return &id
}

// DataDepartmentScope — чөлөөлөлт/нэгж талбарын өгөгдлийг хэлтсээр хязгаарлах
// эсэх. restricted=false: системийн админ (бүх хэлтэс), санхүүгийн
// мэргэжилтэн (бүх хэлтэс — IsCrossDepartmentFinance) эсвэл үнэлгээний
// байгууллагын ажилтан (өөрийн байгууллагын хаалтаар хязгаарлагдана).
// restricted=true бөгөөд deptID nil бол дуудагч юу ч харахгүй (fail-closed).
func DataDepartmentScope(c *gin.Context) (deptID *int, restricted bool) {
	if IsSystemAdmin(c) || IsValuationOrgCaller(c) || IsCrossDepartmentFinance(c) {
		return nil, false
	}
	if id := CallerDepartmentID(c); id != nil {
		v := int(*id)
		return &v, true
	}
	return nil, true
}

// AcquisitionDepartmentChecker — чөлөөлөлт / нэгж талбарын хариуцсан хэлтэс.
type AcquisitionDepartmentChecker interface {
	AcquisitionDepartmentID(ctx context.Context, acquisitionID uuid.UUID) (*int, error)
	ParcelDepartmentID(ctx context.Context, parcelID uuid.UUID) (*int, error)
}

// RequireAcquisitionDepartment — `:id` (чөлөөлөлтийн UUID)-тай маршрутад
// системийн админаас бусад дотоод хэрэглэгч ЗӨВХӨН өөрийн хэлтсийн
// чөлөөлөлтөд хандана. `:id`-гүй маршрутад (жагсаалт) хамаарахгүй — тэнд
// handler шүүлт тавина.
func RequireAcquisitionDepartment(svc AcquisitionDepartmentChecker) gin.HandlerFunc {
	return requireDepartment(func(ctx context.Context, id uuid.UUID) (*int, error) {
		return svc.AcquisitionDepartmentID(ctx, id)
	})
}

// RequireParcelDepartment — `:id` нь нэгж талбарын UUID байх маршрутад.
func RequireParcelDepartment(svc AcquisitionDepartmentChecker) gin.HandlerFunc {
	return requireDepartment(func(ctx context.Context, id uuid.UUID) (*int, error) {
		return svc.ParcelDepartmentID(ctx, id)
	})
}

func requireDepartment(lookup func(context.Context, uuid.UUID) (*int, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		if idStr == "" {
			c.Next()
			return
		}
		mine, restricted := DataDepartmentScope(c)
		if !restricted {
			c.Next()
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			// Буруу UUID — handler 400 буцаана.
			c.Next()
			return
		}
		if mine == nil {
			response.Abort(c, 403, i18n.ErrAcqOutsideDepartment)
			return
		}
		theirs, err := lookup(c.Request.Context(), id)
		if err != nil || theirs == nil || *theirs != *mine {
			response.Abort(c, 403, i18n.ErrAcqOutsideDepartment)
			return
		}
		c.Next()
	}
}

// IsCrossDepartmentFinance — санхүүгийн мэргэжилтэн БҮХ хэлтсийн чөлөөлөлт,
// нэгж талбарыг харна (хэрэглэгчийн шийдвэр, 2026-11-17): нөхөх олговрыг
// байгууллага даяар хянаж баталдаг төв үүрэг.
//
// Зөвхөн санхүүгийн үүргээр нь: ахлах/хэлтсийн дарга роль ХАМТ эзэмшвэл энэ
// чөлөөлөл үйлчлэхгүй — эс бөгөөс ахлахын бичих эрх бүх хэлтэст тархана.
// Санхүүгийн өгөгдлийн хасалт (эзэмшигчийн мэдээлэл, үнэлгээний бус хавсралт)
// болон бичих эрх (зөвхөн батлах/буцаах) өөрчлөгдөхгүй — хүрээ л өргөснө.
// МИКА хэлтсээр хязгаарлагдсан хэвээр.
func IsCrossDepartmentFinance(c *gin.Context) bool {
	return hasAnyRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү") && !IsSenior(c)
}
