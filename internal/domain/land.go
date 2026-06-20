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
	LandAcquisitionStatusNew       = 1 // 1=NEW
	LandAcquisitionStatusActive    = 2 // 2=ACTIVE
	LandAcquisitionStatusClosed    = 3 // 3=CLOSED
	LandAcquisitionStatusCancelled = 4 // 4=CANCELLED
)

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
	RightType        string
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
	ID             uuid.UUID
	PlanID         *uuid.UUID
	PlanCode       string
	PlanGeomWKT    string
	PlanAreaM2     float64
	GeometryWKT    string
	AreaM2         float64
	Status         int
	StartDate      *time.Time
	EndDate        *time.Time
	ProjectName    string
	ImplementingOrg string
	Reason          string
	ResponsibleOrg  string
	FundingSource   string
	CreatedAt      time.Time
	CreatedBy      string
	UpdatedAt      time.Time
	UpdatedBy      string
	AUs            []LandAcquisitionAU
	Parcels        []*Parcel
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
type ParcelFilter struct {
	ParcelID  string
	AU1Code   string
	AU2Code   string
	AU3Code   string
	RightType string
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
	RightType          string
	Landuse            string
	AreaM2             float64
	CreatedAt          time.Time
	CreatedBy          string
}

// ParcelDetailInfo — parcel_detail + right_holder холбосон мэдээлэл
type ParcelDetailInfo struct {
	RightType        string
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
