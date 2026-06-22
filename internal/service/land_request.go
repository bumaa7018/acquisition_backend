package service

import "time"

type CreateLandAcquisitionRequest struct {
	PlanCode        string
	StartDate       time.Time
	EndDate         time.Time
	ShapefileData   []byte
	CreatedBy       string
	AcquisitionName string
	ImplementingOrg string
	Reason          string
	ResponsibleOrg  string
	FundingSource   string
}

type UpdateLandAcquisitionRequest struct {
	PlanCode        string     // хоосон бол одоогийн план хэвээр үлдэнэ
	StartDate       *time.Time // nil бол өөрчлөхгүй
	EndDate         *time.Time // nil бол өөрчлөхгүй
	ClearEndDate    bool       // true = end_date-г NULL болгоно
	ShapefileData   []byte     // nil бол геометр өөрчлөхгүй
	UpdatedBy       string
	AcquisitionName string
	ImplementingOrg string
	Reason          string
	ResponsibleOrg  string
	FundingSource   string
}
