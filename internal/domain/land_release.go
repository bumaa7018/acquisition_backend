package domain

import (
	"errors"
	"time"
)

// «Буулгалтын мэдээлэл» — shapefile-аар оруулсан газрууд. Үндсэн нэгж
// талбар/чөлөөлөлтөөс ТУСДАА хүснэгтүүдэд (land_release*) хадгалагдана.

// Газрын төлөв.
const (
	LandReleaseStatusPlanning  = 1 // Төлөвлөж байгаа
	LandReleaseStatusReleased  = 2 // Захирамж гарч чөлөөлсөн
	LandReleaseStatusCancelled = 3 // Хүчингүй
)

// LandReleaseStatusLabels — төлөвийн нэр.
var LandReleaseStatusLabels = map[int]string{
	LandReleaseStatusPlanning:  "Төлөвлөж байгаа",
	LandReleaseStatusReleased:  "Захирамж гарч чөлөөлсөн",
	LandReleaseStatusCancelled: "Хүчингүй",
}

// LandReleaseTransitionAllowed — Төлөвлөж байгаа → Чөлөөлсөн / Хүчингүй л зөвшөөрнө.
func LandReleaseTransitionAllowed(from, to int) bool {
	return from == LandReleaseStatusPlanning &&
		(to == LandReleaseStatusReleased || to == LandReleaseStatusCancelled)
}

var (
	// ErrLandReleaseDecreeRequired — «Захирамж гарч чөлөөлсөн» төлөвт захирамжийн
	// дугаар ба файл заавал.
	ErrLandReleaseDecreeRequired = errors.New("land release decree number and file are required")
	// ErrLandReleaseTransition — зөвшөөрөгдөөгүй төлөвийн шилжилт.
	ErrLandReleaseTransition = errors.New("land release status transition not allowed")
	// ErrLandReleaseNoFiles — shapefile оруулаагүй.
	ErrLandReleaseNoFiles = errors.New("land release shapefile is required")
	// ErrLandReleasePhoto — зургийн файл (PDF) буруу/хоосон.
	ErrLandReleasePhoto = errors.New("land release photo must be a PDF")
	// ErrLandReleaseStructureType — буулгасан байгууламжийн төрөл сонгоогүй/олдсонгүй.
	ErrLandReleaseStructureType = errors.New("land release structure type is required")
)

// LandReleaseStructureType — буулгасан байгууламжийн төрөл (хэрэглэгч оруулахдаа
// шинээр нэмж болно).
type LandReleaseStructureType struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// LandReleasePhoto — газрын зураг: оруулсан зургуудыг (frontend) нэгтгэсэн НЭГ
// PDF (зураг бүр нэг хуудас). Холбоос хүснэгтээр олон газарт холбогдож болно.
type LandReleasePhoto struct {
	ID        string    `json:"id"`
	FileName  string    `json:"file_name"`
	FileURL   string    `json:"file_url"`
	SizeBytes int64     `json:"size_bytes"`
	PageCount int       `json:"page_count"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	// ReleaseCount — энэ зурагтай холбогдсон газрын тоо (олноор оруулсан бол > 1).
	ReleaseCount int `json:"release_count"`
}

// LandReleaseDecree — захирамж: нэг дугаар, нэг файл.
type LandReleaseDecree struct {
	ID           string     `json:"id"`
	DecreeNumber string     `json:"decree_number"`
	DecreeDate   *time.Time `json:"decree_date,omitempty"`
	FileName     string     `json:"file_name"`
	FileType     string     `json:"file_type"`
	FileURL      string     `json:"file_url"`
	SizeBytes    int64      `json:"size_bytes"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by"`
	// ReleaseCount — энэ захирамжтай холбогдсон газрын тоо.
	ReleaseCount int `json:"release_count"`
}

// LandRelease — нэг газар.
type LandRelease struct {
	ID          string  `json:"id"`
	ReleaseNo   int64   `json:"release_no"`
	ImportID    *string `json:"import_id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Status      int     `json:"status"`
	StatusName  string  `json:"status_name"`
	// Буулгасан байгууламжийн төрөл — 000061-ээс өмнө оруулсанд nil.
	StructureTypeID   *int   `json:"structure_type_id,omitempty"`
	StructureTypeName string `json:"structure_type_name"`
	// Сум/дүүрэг, аймаг/хот — оруулахад байршлаар тодорхойлогдоно (давхцаагүй бол хоосон).
	AU1Code     string    `json:"au1_code"`
	AU1Name     string    `json:"au1_name"`
	AU2Code     string    `json:"au2_code"`
	AU2Name     string    `json:"au2_name"`
	AreaM2      float64   `json:"area_m2"`
	SourceFile  string    `json:"source_file"`
	SourceIndex int       `json:"source_index"`
	GeometryWKT string    `json:"geometry_wkt,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`

	// Захирамж (холбоос хүснэгтээр) — байхгүй бол nil.
	Decree *LandReleaseDecree `json:"decree,omitempty"`
	// Photos — зөвхөн дэлгэрэнгүйд (шинэ нь эхэнд).
	Photos []*LandReleasePhoto `json:"photos,omitempty"`
	// History — зөвхөн дэлгэрэнгүйд.
	History []*LandReleaseHistory `json:"history,omitempty"`
}

// LandReleaseHistory — төлөвийн түүх.
type LandReleaseHistory struct {
	ID           string    `json:"id"`
	FromStatus   *int      `json:"from_status,omitempty"`
	ToStatus     int       `json:"to_status"`
	DecreeNumber string    `json:"decree_number,omitempty"`
	Note         string    `json:"note"`
	ChangedAt    time.Time `json:"changed_at"`
	ChangedBy    string    `json:"changed_by"`
}

// LandReleaseFilter — жагсаалтын шүүлт. Search нь нэр ЭСВЭЛ захирамжийн дугаар.
type LandReleaseFilter struct {
	Search string
	Status int
	Year   int
	// StructureTypeID — буулгасан байгууламжийн төрөл.
	StructureTypeID int
	// AU2Code — сум/дүүрэг (land_release.au2_code).
	AU2Code  string
	Page     int
	PageSize int
}

// LandReleaseMapItem — газрын зурагт (бүх газар, хөнгөн).
type LandReleaseMapItem struct {
	ID          string `json:"id"`
	ReleaseNo   int64  `json:"release_no"`
	Name        string `json:"name"`
	Status      int    `json:"status"`
	GeometryWKT string `json:"geometry_wkt"`
}

// LandReleaseStats — дээд хэсгийн статистик.
type LandReleaseStats struct {
	Total       int                    `json:"total"`
	TotalAreaM2 float64                `json:"total_area_m2"`
	DecreeCount int                    `json:"decree_count"`
	ByStatus    []LandReleaseStatusRow `json:"by_status"`
	ByYear      []LandReleaseYearRow   `json:"by_year"`
	// ByDistrict — сум/дүүргээр (land_release.au2_code — оруулахад хадгалсан).
	ByDistrict []LandReleaseDistrictRow `json:"by_district"`
	// ByDistrictYear — сум/дүүрэг × он (оны графикт дүүрэг бүр нэг цуврал).
	ByDistrictYear []LandReleaseDistrictYearRow `json:"by_district_year"`
	// Districts — газартай БҮХ сум/дүүрэг (шүүлтгүй, кодоор эрэмбэлсэн):
	// графикийн өнгө шүүлтээс үл хамааран дүүрэгт тогтмол оноогдоно.
	Districts []LandReleaseDistrictRef `json:"districts"`
	// Years — бүх он (шүүлтгүй), оны сонголтод.
	Years []int `json:"years"`
}

// LandReleaseDistrictYearRow — нэг сум/дүүргийн нэг оны дүн (Code хоосон —
// сум/дүүрэгтэй давхцаагүй).
type LandReleaseDistrictYearRow struct {
	Code   string  `json:"code"`
	Year   int     `json:"year"`
	Count  int     `json:"count"`
	AreaM2 float64 `json:"area_m2"`
}

type LandReleaseDistrictRef struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	AU1Name string `json:"au1_name"`
}

// LandReleaseDistrictRow — нэг сум/дүүргийн дүн. Code хоосон бол ямар ч
// сум/дүүрэгтэй давхцаагүй газар.
type LandReleaseDistrictRow struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	AU1Name   string  `json:"au1_name"`
	Count     int     `json:"count"`
	Planning  int     `json:"planning"`
	Released  int     `json:"released"`
	Cancelled int     `json:"cancelled"`
	AreaM2    float64 `json:"area_m2"`
}

type LandReleaseStatusRow struct {
	Status int     `json:"status"`
	Count  int     `json:"count"`
	AreaM2 float64 `json:"area_m2"`
}

// LandReleaseYearRow — он тус бүрийн тоо (захирамжтай бол захирамжийн оноор,
// эс бөгөөс оруулсан оноор).
type LandReleaseYearRow struct {
	Year      int     `json:"year"`
	Planning  int     `json:"planning"`
	Released  int     `json:"released"`
	Cancelled int     `json:"cancelled"`
	AreaM2    float64 `json:"area_m2"`
	Decrees   int     `json:"decrees"`
}

// LandReleaseFeature — оруулах файлаас уншсан нэг газар.
type LandReleaseFeature struct {
	SourceFile  string  `json:"source_file"`
	SourceIndex int     `json:"source_index"`
	GeometryWKT string  `json:"geometry_wkt"`
	AreaM2      float64 `json:"area_m2"`
}

// LandReleaseImport — оруулалт (popup-аар нэг удаа).
type LandReleaseImport struct {
	ID            string
	Name          string
	Description   string
	InitialStatus int
	// StructureTypeID эсвэл StructureTypeName (шинэ төрөл — хадгалахад нэмэгдэнэ).
	StructureTypeID   *int
	StructureTypeName string
	FileNames         []string
	CreatedBy         string
	// Photo — оруулалтын зураг (PDF); байвал бүх газарт холбогдоно.
	Photo *LandReleasePhoto
}
