package service

import "time"

type CreateLandAcquisitionRequest struct {
	PlanCode      string
	StartDate     time.Time
	EndDate       time.Time
	ShapefileData []byte
	CreatedBy     string
}

type UpdateLandAcquisitionRequest struct {
	PlanCode      string     // хоосон бол одоогийн план хэвээр үлдэнэ
	StartDate     *time.Time // nil бол өөрчлөхгүй
	EndDate       *time.Time // nil бол өөрчлөхгүй
	ShapefileData []byte     // nil бол геометр өөрчлөхгүй
	UpdatedBy     string
}
