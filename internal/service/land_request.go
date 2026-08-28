package service

import "time"

// Чөлөөлөлтийн ХИЛ нь ГАРААС оруулагддаггүй: PlanCode-оор олдсон
// төлөвлөгөөний нэгж талбарын хилийг ХУУЛЖ авна (shapefile илгээх зам
// бүрмөсөн хаагдсан).
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
	PlanCode          string
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
