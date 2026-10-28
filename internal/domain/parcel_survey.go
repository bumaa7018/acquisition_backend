package domain

import (
	"time"

	"github.com/google/uuid"
)

// Шилжин суурьшсан байршлын төрөл (parcel_survey.relocation_type).
const (
	RelocationGerDistrict = "ger_district" // Гэр хороолол
	RelocationApartment   = "apartment"    // Орон сууц
)

// SurveyCountMax — тоон хариултын дээд утга. Тоо нь хязгааргүй (маягтад «+»-ээр
// нэмдэг); энэ нь зөвхөн smallint багана болон алдаатай орцоос хамгаалах хязгаар.
const SurveyCountMax = 999

// SurveyCountBucketMax — статистикт энэ утгаас дээшхийг нэг ангилалд («3+») нэгтгэнэ.
const SurveyCountBucketMax = 3

// ParcelSurvey — чөлөөлөгдсөн газрын судалгаа. Нэгж талбарт НЭГ бүртгэл.
//
// Нэгж талбар үүсэхэд хоосон бүртгэл АВТОМАТААР үүснэ (DB trigger, migration
// 000057). Тоон хариултууд 1..SurveyCountMax, хариулаагүй бол nil.
type ParcelSurvey struct {
	ID       uuid.UUID
	ParcelID uuid.UUID
	// ChimneyCount — тухайн нэгж талбарт утаат яндан хэд байсан.
	ChimneyCount *int
	// ToiletCount — нүхэн жорлон хэдэн удаа гаргаж байсан.
	ToiletCount *int
	// CohabitantCount — гэр сууцанд хэдэн хамтран амьдрагч байсан.
	CohabitantCount *int
	// RelocationType — нөлөөлөлд өртсөний дараа хаана оршин суусан.
	RelocationType    string
	RelocationAddress string
	Note              string
	CreatedAt         time.Time
	CreatedBy         string
	UpdatedAt         time.Time
	UpdatedBy         string
}

// ParcelSurveyRow — жагсаалтын мөр: судалгаа + нэгж талбар, чөлөөлөлтийн мэдээлэл.
type ParcelSurveyRow struct {
	ParcelSurvey
	ParcelCode      string
	AcquisitionID   uuid.UUID
	AcquisitionName string
	AU3Code         string
}

type ParcelSurveyFilter struct {
	// AcquisitionIDs — эдгээр чөлөөлөлтийн нэгж талбарууд (хоосон бол бүгд).
	AcquisitionIDs []uuid.UUID
	// ParcelCode — нэгж талбарын дугаар (ILIKE).
	ParcelCode string
	// Answered — true: бөглөсөн, false: хоосон, nil: бүгд. Бөглөсөн = аль нэг
	// хариулт эсвэл тайлбартай.
	Answered *bool
	Page     int
	PageSize int
}

// ParcelSurveySummary — судалгааны дүнгийн статистик (Хяналтын самбар / жагсаалт).
//
// *Counts map-ийн түлхүүр нь хариулт ("1", "2", "3" — 3 нь 3 ба түүнээс дээш); RelocationCounts-ийнх нь
// RelocationType. Хариулаагүй мөр тооцогдохгүй.
type ParcelSurveySummary struct {
	// Total — бүх (автоматаар үүссэн) судалгаа; Answered — тэдгээрээс бөглөсөн.
	Total            int64
	Answered         int64
	ChimneyCounts    map[string]int64
	ToiletCounts     map[string]int64
	CohabitantCounts map[string]int64
	RelocationCounts map[string]int64
}
