package domain

import (
	"time"

	"github.com/google/uuid"
)

// Plan статус (auth DB)
const (
	PlanStatusDraft    = 1 // 1=DRAFT
	PlanStatusActive   = 2 // 2=ACTIVE
	PlanStatusArchived = 3 // 3=ARCHIVED
)

// LandAcquisition статус (main DB)
const (
	LandAcquisitionStatusNew       = 1 // Шинэ
	LandAcquisitionStatusField     = 2 // Хээрийн судалгаа
	LandAcquisitionStatusVerified  = 3 // Баталгаажсан
	LandAcquisitionStatusCancelled = 4 // Цуцлагдсан
)

// AcquisitionStatusLabels — статусын монгол нэрс
var AcquisitionStatusLabels = map[int]string{
	LandAcquisitionStatusNew:       "Шинэ",
	LandAcquisitionStatusField:     "Хээрийн судалгаа",
	LandAcquisitionStatusVerified:  "Баталгаажсан",
	LandAcquisitionStatusCancelled: "Цуцлагдсан",
}

// AcquisitionStatusOrder — дэс дугаарын эрэмбэ
var AcquisitionStatusOrder = []int{
	LandAcquisitionStatusNew,
	LandAcquisitionStatusField,
	LandAcquisitionStatusVerified,
	LandAcquisitionStatusCancelled,
}

// StatusOption — боломжтой явцын мэдээлэл
type StatusOption struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// Plan — газар зохион байгуулалтын төлөвлөгөө (auth DB-д хадгалагдана)
type Plan struct {
	ID          uuid.UUID
	PlanCode    string
	Name        string
	BoundaryWKT string // WKT текст хэлбэрт
	AreaM2      float64
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SourceParcel — нэгж талбарын эх мэдээлэл (auth DB-д хадгалагдана)
type SourceParcel struct {
	ID               uuid.UUID
	ParcelID         string
	PlanCode         string
	GeometryWKT      string
	AU1Code          string
	AU2Code          string
	AU3Code          string
	RightType        int
	Landuse          string
	AreaM2           float64
	HolderLastName   string
	HolderName       string
	HolderRegisterNo string
	HolderPhone      string
	HolderEmail      string
	HolderType       string
	HolderCivilID    string
	AppNo            string
	DecisionNo       string
	DecisionDate     *time.Time
	ContractNo       string
	ContractDate     *time.Time
	CertificateNo    string
	CertificateDate  *time.Time
	CreatedAt        time.Time
}

// LandAcquisition — газар чөлөөлөлтын мэдээлэл (main DB-д хадгалагдана)
type LandAcquisition struct {
	ID              uuid.UUID
	PlanID          *uuid.UUID
	PlanCode        string
	PlanName        string
	PlanGeomWKT     string
	PlanAreaM2      float64
	GeometryWKT     string
	AreaM2          float64
	Status          int
	StartDate       *time.Time
	EndDate         *time.Time
	AcquisitionName string
	ImplementingOrg string
	Reason          string
	ResponsibleOrg  string
	FundingSource   string
	CreatedAt       time.Time
	CreatedBy       string
	UpdatedAt       time.Time
	UpdatedBy       string
	AUs             []LandAcquisitionAU
	Parcels         []*Parcel
	ParcelCount     int64
}

// LandAcquisitionAU — чөлөөлөлт ↔ захиргааны нэгж холбоос
type LandAcquisitionAU struct {
	LandAcquisitionID uuid.UUID
	AU1Code           string
	AU1Name           string
	AU2Code           string
	AU2Name           string
	AU3Code           string
	AU3Name           string
}

// LandAcquisitionFilter — газар чөлөөлөлтийн жагсаалт шүүх болон хуудаслах параметр
type LandAcquisitionFilter struct {
	PlanCode      string
	Status        int // 0 = шүүлт хийхгүй; 1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED
	AU3Code       string
	StartDateFrom *time.Time
	StartDateTo   *time.Time
	Page          int
	PageSize      int
}

// ParcelFilter — нэгж талбарын жагсаалт шүүх болон хуудаслах параметр
const (
	RightTypeUse       = 1 // Ашиглах
	RightTypeLease     = 2 // Эзэмших
	RightTypeOwnership = 3 // Өмчлөх
)

type ParcelFilter struct {
	ParcelID  string
	AU1Code   string
	AU2Code   string
	AU3Code   string
	RightType int
	Landuse   string
	Page      int
	PageSize  int
}

// Parcel — нэгж талбар (main DB-д хадгалагдана)
type Parcel struct {
	ID                 uuid.UUID
	ParcelID           string
	OldParcelID        string
	AcquisitionID      uuid.UUID
	ValidFrom          *time.Time
	ValidTill          *time.Time
	AU1Code            string
	AU2Code            string
	AU3Code            string
	GeometryWKT        string
	AcquisitionGeomWKT string  // чөлөөлөлтийн бүстэй давхцах хэсгийн геометр
	AcquisitionAreaM2  float64 // давхцах хэсгийн талбай (м²)
	RightType          int
	Landuse            string
	AreaM2             float64
	CompensationPaid   bool
	CreatedAt          time.Time
	CreatedBy          string
}

// ParcelDetailInfo — parcel_detail + right_holder холбосон мэдээлэл
type ParcelDetailInfo struct {
	RightType        int
	HolderLastName   string
	HolderName       string
	HolderRegisterNo string
	HolderPhone      string
	HolderEmail      string
	HolderType       string
	HolderCivilID    string
	AppNo            string
	AppTimestamps    *time.Time
	DecisionNo       string
	DecisionDate     *time.Time
	ContractNo       string
	ContractDate     *time.Time
	CertificateNo    string
	CertificateDate  *time.Time
}

// ParcelFull — нэгж талбарын дэлгэрэнгүй мэдээлэл (parcel + parcel_detail + right_holder)
type ParcelFull struct {
	Parcel
	Detail *ParcelDetailInfo
}

// AcquisitionProgress — газар чөлөөлөлтийн статус өөрчлөлтийн түүх
type AcquisitionProgress struct {
	ID            uuid.UUID
	AcquisitionID uuid.UUID
	FromStatus    int
	ToStatus      int
	Note          string
	ChangedBy     string
	ChangedAt     time.Time
}

// Document — үндсэн баримт бичиг (URL-аар хадгалагдана)
type Document struct {
	ID         uuid.UUID
	Name       string
	FileType   string
	FileURL    string
	SizeBytes  int64
	Note       string
	UploadedBy string
	UploadedAt time.Time
}

// GlobalParcel — parcel joined with acquisition info
type GlobalParcel struct {
	Parcel
	AcquisitionName   string
	PlanCode          string
	AcquisitionStatus int
	StartDate         *time.Time
	EndDate           *time.Time
}

// GlobalParcelFilter — глобал нэгж талбарын жагсаалт шүүх болон хуудаслах параметр
type GlobalParcelFilter struct {
	ParcelID        string
	RightType       int
	Landuse         string
	AU3Code         string
	AcquisitionName string
	AcquisitionID   string
	PlanCode        string
	Status          int
	Page            int
	PageSize        int
}

// ParcelPayment — нэгж талбарт хийсэн төлбөрийн бичлэг
type ParcelPayment struct {
	ID        uuid.UUID
	ParcelID  uuid.UUID
	Amount    float64
	Currency  string
	PaidAt    *time.Time
	Note      string
	CreatedAt time.Time
	CreatedBy string
}

