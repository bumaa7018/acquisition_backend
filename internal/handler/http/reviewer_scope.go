package http

import (
	"context"
	"net/url"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Санхүүгийн мэргэжилтэн ба МИКА — нөхөх олговрын ХЯНАЛТЫН гадаад роль.
//
// Тэд хуваарилалтгүйгээр чөлөөлөлт/нэгж талбарыг УНШДАГ (RequireAssignedOrSenior-ийн
// supervisoryReadRoles) ч зөвхөн ҮНЭЛГЭЭТЭЙ холбоотой хэсгийг харах ёстой —
// frontend нь тэдэнд «Ерөнхий» ба «Нөхөх олговор» табыг л нээдэг. Өмнө нь
// backend бүх өгөгдлийг (эзэмшигчийн регистр/утас, иргэний үнэмлэх зэрэг бүх
// хавсралт, нийгмийн судалгаа, мэдэгдэл) буцааж, нуух ажлыг frontend-д үлдээдэг
// байв. Энэ файл тэр хязгаарыг backend дээр тогтооно.

// reviewerOnlyCaller — дуудагч санхүү/МИКА бөгөөд дотоод ахлах/админ биш,
// үнэлгээний байгууллагын ажилтан биш (тэр нь өөрийн /prof хаалттай).
func reviewerOnlyCaller(c *gin.Context) bool {
	if middleware.IsValuationOrgCaller(c) || middleware.IsSenior(c) {
		return false
	}
	return ctxHasRole(c,
		"finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү",
		"mika", "МИКА")
}

// reviewerDocumentTypes — хяналтын роль уншиж болох нэгж талбарын хавсралтын
// төрлүүд (document_type.type): үнэлгээний тайлан, түүний эх Excel, ажлын
// зураг. Нөхөх олговрын табд эдгээр л харагддаг.
var reviewerDocumentTypes = map[string]bool{
	"valuation_report": true,
	"valuation_source": true,
	"work_photo":       true,
}

// reviewerDocFilter — хавсралт хяналтын ролийн харах төрлийнх эсэхийг шалгах
// функц. Төрлийг хүсэлт дотор кэшлэнэ (жагсаалтад нэг төрөл олон удаа давтагдана).
// Төрөлгүй, эсвэл төрөл нь олдоогүй хавсралт ХААЛТТАЙ (fail-closed).
func reviewerDocFilter(ctx context.Context, svc service.LandService) func(*domain.Document) bool {
	cache := map[int]bool{}
	return func(d *domain.Document) bool {
		if d == nil || d.DocumentTypeID == 0 {
			return false
		}
		if v, ok := cache[d.DocumentTypeID]; ok {
			return v
		}
		dt, err := svc.GetDocumentType(ctx, d.DocumentTypeID)
		visible := err == nil && dt != nil && reviewerDocumentTypes[dt.Type]
		cache[d.DocumentTypeID] = visible
		return visible
	}
}

// documentMatchesKey — хавсралтын file_url нь файлын системийн түлхүүртэй
// (`parcel/<uuid>/<нэр>`) таарах эсэх. file_url нь `/api/files/<escape хийсэн түлхүүр>`.
func documentMatchesKey(d *domain.Document, key string) bool {
	if d == nil || key == "" {
		return false
	}
	u := d.FileURL
	if decoded, err := url.PathUnescape(u); err == nil {
		u = decoded
	}
	return strings.HasSuffix(u, "/"+strings.TrimPrefix(key, "/"))
}

// redactParcelForReviewer — нэгж талбарын дэлгэрэнгүйгээс хяналтын ролийн
// хэрэггүй ХУВИЙН ба эрх зүйн мэдээллийг хасна: эзэмшигчид (регистр, утас,
// имэйл), барьцаа, шүүхийн шийдвэр, нэхэмжлэл, УБЕГ-ийн өмчлөл.
func redactParcelForReviewer(r *parcelFullResponse) {
	r.Holders = []parcelHolderResponse{}
	r.Mortgages = []domain.ParcelMortgage{}
	r.CourtDecisions = []domain.ParcelCourtDecision{}
	r.Invoices = []domain.ParcelInvoice{}
	r.Ownerships = []domain.ParcelOwnership{}
	if r.Detail != nil {
		r.Detail.HolderLastName = ""
		r.Detail.HolderName = ""
		r.Detail.HolderRegisterNo = ""
		r.Detail.HolderPhone = ""
		r.Detail.HolderEmail = ""
		r.Detail.HolderType = ""
		r.Detail.HolderCivilID = ""
	}
}

// ── Үнэлгээний байгууллагын хавсралтын эзэмшил ──────────────────────────────

// documentUploader — хавсралтын uploaded_by утга. Үнэлгээний байгууллагын
// ажилтны оруулсныг БАЙГУУЛЛАГААР тэмдэглэнэ (domain.ValuationOrgUploader):
// хоёр сангийн хэрэглэгчийн дугаар мөргөлддөг тул зөвхөн дугаараар эзэмшлийг
// тогтоох боломжгүй, мөн тухайн байгууллагын аль ч ажилтан өөрсдийн
// оруулсныг устгах ёстой.
func documentUploader(c *gin.Context) string {
	if middleware.IsValuationOrgCaller(c) {
		if orgID, ok := middleware.GetCallerOrgID(c); ok {
			return domain.ValuationOrgUploader(orgID)
		}
	}
	return contextUserIDString(c)
}

// profOwnsDocument — үнэлгээний байгууллага ЗӨВХӨН өөрийн оруулсан хавсралтыг
// устгана. Өмнө нь /prof замаар чөлөөлөлт/нэгж талбарын ямар ч хавсралтыг
// (дотоод ажилтны оруулсан гэрээ, иргэний бичиг баримт г.м.) устгаж чаддаг байв.
// Байгууллагын ажилтан биш дуудагчид хамаарахгүй (true).
func profOwnsDocument(c *gin.Context, docs []*domain.Document, docID uuid.UUID) bool {
	if !middleware.IsValuationOrgCaller(c) {
		return true
	}
	orgID, ok := middleware.GetCallerOrgID(c)
	if !ok {
		return false
	}
	for _, d := range docs {
		if d.ID == docID {
			return d.UploadedBy == domain.ValuationOrgUploader(orgID)
		}
	}
	return false
}

func (h *LandHandler) canDeleteAcquisitionDocument(c *gin.Context, acqID, docID uuid.UUID) bool {
	if !middleware.IsValuationOrgCaller(c) {
		return true
	}
	docs, err := h.svc.ListAcquisitionDocuments(c.Request.Context(), acqID)
	if err != nil {
		response.Error(c, err)
		return false
	}
	if !profOwnsDocument(c, docs, docID) {
		response.Error(c, domain.ErrForbidden)
		return false
	}
	return true
}

func (h *LandHandler) canDeleteParcelDocument(c *gin.Context, parcelID, docID uuid.UUID) bool {
	if !middleware.IsValuationOrgCaller(c) {
		return true
	}
	docs, err := h.svc.ListParcelDocuments(c.Request.Context(), parcelID)
	if err != nil {
		response.Error(c, err)
		return false
	}
	if profOwnsDocument(c, docs, docID) || h.legacyValuationDocument(c, docs, docID) {
		return true
	}
	response.Error(c, domain.ErrForbidden)
	return false
}

// legacyValuationDocument — ХУУЧИН хэлбэрээр (uploaded_by = зөвхөн хэрэглэгчийн
// дугаар) хадгалагдсан ҮНЭЛГЭЭНИЙ хавсралт (тайлан, эх Excel, ажлын зураг).
//
// Байгууллагын тэмдэг (domain.ValuationOrgUploader) нэвтрэхээс өмнө оруулсан
// хавсралтын эзнийг тогтоох боломжгүй (хоёр сангийн дугаар мөргөлддөг).
// Тэдгээрийг бүрэн хаавал үнэлгээний байгууллага өөрийн хуучин тайлангаа
// сольж чадахгүй, илгээлт унана. Иймд зөвхөн ҮНЭЛГЭЭНИЙ төрлийн хуучин
// хавсралтыг зөвшөөрнө — гэрээ, иргэний бичиг баримт зэрэг бусад бүх төрөл
// хаалттай хэвээр. Шинээр оруулсан хавсралт бүгд тэмдэгтэй тул энэ зам аажмаар
// хэрэглэгдэхээ болино.
func (h *LandHandler) legacyValuationDocument(c *gin.Context, docs []*domain.Document, docID uuid.UUID) bool {
	for _, d := range docs {
		if d.ID != docID {
			continue
		}
		if !isLegacyUploader(d.UploadedBy) {
			return false
		}
		return reviewerDocFilter(c.Request.Context(), h.svc)(d)
	}
	return false
}

// isLegacyUploader — uploaded_by нь хуучин хэлбэрийн цэвэр тоо (хэрэглэгчийн дугаар) эсэх.
func isLegacyUploader(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ── Хэлтсийн хүрээ (нэгтгэл, тайлан, хяналтын самбар) ───────────────────────

// noDepartment — хэлтэсгүй (fail-closed) дуудагчийн шүүлтийн утга: ямар ч
// ангилалд тохирохгүй тул нэгтгэл хоосон гарна.
var noDepartment = -1

// scopedDepartment — нэгтгэл/тайлангийн шүүлтэд тавих хэлтэс.
//
// nil  — системийн админ ба үнэлгээний байгууллага (тэдний хүрээ өөр хаалтаар).
// &id  — бусад дотоод хэрэглэгч (санхүү, МИКА-г оруулаад) өөрийн хэлтэс.
// &-1  — хэлтэсгүй дуудагч: юу ч харахгүй.
//
// ЯАГААД: чөлөөлөлт/нэгж талбарын `:id` замууд RequireAcquisitionDepartment-
// ээр хумигддаг ч хяналтын самбар, Хронос, тайлан, судалгааны жагсаалт бүх
// хэлтсийн өгөгдлийг нэгтгэж буцаадаг байв.
func scopedDepartment(c *gin.Context) *int {
	dept, restricted := middleware.DataDepartmentScope(c)
	if !restricted {
		return nil
	}
	if dept == nil {
		return &noDepartment
	}
	return dept
}
