package service

import "time"

// ҮҮСГЭХ үед чөлөөлөлтийн ХИЛ нь ГАРААС оруулагддаггүй: PlanCode-оор олдсон
// төлөвлөгөөний нэгж талбарын хилийг ХУУЛЖ авна (үүсгэх урсгалд shapefile
// хүлээж авахгүй). Гараас оруулах зам зөвхөн ЗАСВАРЛАХ (Update) үед нээлттэй —
// UpdateLandAcquisitionRequest.ShapefileData-г үз.
type CreateLandAcquisitionRequest struct {
	PlanCode          string
	StartDate         time.Time
	EndDate           *time.Time
	CreatedBy         string
	AcquisitionName   string
	ImplementingOrg   string
	Reason            string
	ResponsibleOrg    string
	FundingSource     string
	GeneralCategoryID *int
	SubCategoryID     *int
}

type UpdateLandAcquisitionRequest struct {
	// PlanCode — хоосон бол хил хөндөгдөхгүй. Утга өгвөл тэр дугаараар
	// төлөвлөгөөг ДАХИН хайж, олдсон төлөвлөгөөний хилийг чөлөөлөлтийн ШИНЭ
	// хил болгож тохируулна (одоогийнхтой ижил дугаар байсан ч дахин хуулна).
	PlanCode string
	// ShapefileData — ЗАСВАРЛАХ үед гараас оруулсан хилийн .shp файлын байт.
	// Хээрийн хэмжилтээр төлөвлөгөөний хил засагдсан үед хэрэглэнэ: хил нь
	// файлаас уншигдана, харин төлөвлөгөөний мэдээлэл (plan_code, plan_geom)
	// ХӨНДӨГДӨХГҮЙ — тэр нь жишиг хэвээр үлдэж, талбайн зөрүү түүнээс
	// хэмжигдэнэ (maxBoundaryAreaDeviation).
	//
	// PlanCode-той ЗЭРЭГ өгвөл ErrBoundarySourceConflict.
	ShapefileData     []byte
	StartDate         *time.Time // nil бол өөрчлөхгүй
	EndDate           *time.Time // nil бол өөрчлөхгүй
	ClearEndDate      bool       // true = end_date-г NULL болгоно
	AreaM2            *float64   // nil бол өөрчлөхгүй; хил солиход хилээс тооцоолно
	UpdatedBy         string
	AcquisitionName   string
	ImplementingOrg   string
	Reason            string
	ResponsibleOrg    string
	FundingSource     string
	GeneralCategoryID *int
	SubCategoryID     *int
}
