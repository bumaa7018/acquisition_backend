package http

import (
	"time"

	"github.com/your-org/go-api-template/internal/domain"
)

type landAcquisitionResponse struct {
	ID              string           `json:"id"`
	PlanCode        string           `json:"plan_code"`
	PlanName        string           `json:"plan_name"`
	GeometryWKT     string           `json:"geometry_wkt"`
	AreaM2          float64          `json:"area_m2"`
	Status          int              `json:"status"`
	StartDate       *string          `json:"start_date"`
	EndDate         *string          `json:"end_date"`
	AcquisitionName string           `json:"acquisition_name"`
	ImplementingOrg string           `json:"implementing_org"`
	Reason          string           `json:"reason"`
	ResponsibleOrg  string           `json:"responsible_org"`
	FundingSource   string           `json:"funding_source"`
	CreatedAt       string           `json:"created_at"`
	CreatedBy       string           `json:"created_by"`
	ParcelCount     int64            `json:"parcel_count"`
	AUs             []auResponse     `json:"aus"`
	Parcels         []parcelResponse `json:"parcels"`
}

type auResponse struct {
	AU1Code string `json:"au1_code"`
	AU1Name string `json:"au1_name"`
	AU2Code string `json:"au2_code"`
	AU2Name string `json:"au2_name"`
	AU3Code string `json:"au3_code"`
	AU3Name string `json:"au3_name"`
}

type parcelResponse struct {
	ID                string  `json:"id"`
	ParcelID          string  `json:"parcel_id"`
	AU1Code           string  `json:"au1_code"`
	AU2Code           string  `json:"au2_code"`
	AU3Code           string  `json:"au3_code"`
	RightType         int     `json:"right_type"`
	Landuse           string  `json:"landuse"`
	AreaM2            float64 `json:"area_m2"`
	AcquisitionAreaM2 float64 `json:"acquisition_area_m2"`
	CompensationPaid  bool    `json:"compensation_paid"`
	GeometryWKT       string  `json:"geometry_wkt,omitempty"`
}

type parcelFullResponse struct {
	ID                 string            `json:"id"`
	ParcelID           string            `json:"parcel_id"`
	OldParcelID        string            `json:"old_parcel_id,omitempty"`
	AcquisitionID      string            `json:"acquisition_id"`
	ValidFrom          *string           `json:"valid_from"`
	ValidTill          *string           `json:"valid_till"`
	AU1Code            string            `json:"au1_code"`
	AU2Code            string            `json:"au2_code"`
	AU3Code            string            `json:"au3_code"`
	GeometryWKT        string            `json:"geometry_wkt"`
	AcquisitionGeomWKT string            `json:"acquisition_geom_wkt"`
	AcquisitionAreaM2  float64           `json:"acquisition_area_m2"`
	RightType          int               `json:"right_type"`
	Landuse            string            `json:"landuse"`
	AreaM2             float64           `json:"area_m2"`
	CreatedAt          string            `json:"created_at"`
	CreatedBy          string            `json:"created_by"`
	Detail             *parcelDetailInfo `json:"detail,omitempty"`
}

type parcelDetailInfo struct {
	RightType        int     `json:"right_type"`
	HolderLastName   string  `json:"holder_last_name"`
	HolderName       string  `json:"holder_name"`
	HolderRegisterNo string  `json:"holder_register_no"`
	HolderPhone      string  `json:"holder_phone"`
	HolderEmail      string  `json:"holder_email"`
	HolderType       string  `json:"holder_type"`
	HolderCivilID    string  `json:"holder_civil_id"`
	AppNo            string  `json:"app_no"`
	DecisionNo       string  `json:"decision_no"`
	DecisionDate     *string `json:"decision_date"`
	ContractNo       string  `json:"contract_no"`
	ContractDate     *string `json:"contract_date"`
	CertificateNo    string  `json:"certificate_no"`
	CertificateDate  *string `json:"certificate_date"`
}

type progressResponse struct {
	ID         string `json:"id"`
	FromStatus int    `json:"from_status"`
	ToStatus   int    `json:"to_status"`
	Note       string `json:"note"`
	ChangedBy  string `json:"changed_by"`
	ChangedAt  string `json:"changed_at"`
}

type documentResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	FileType   string `json:"file_type"`
	FileURL    string `json:"file_url"`
	SizeBytes  int64  `json:"size_bytes"`
	Note       string `json:"note"`
	UploadedBy string `json:"uploaded_by"`
	UploadedAt string `json:"uploaded_at"`
}

func toDocumentResponse(d *domain.Document) documentResponse {
	return documentResponse{
		ID:         d.ID.String(),
		Name:       d.Name,
		FileType:   d.FileType,
		FileURL:    d.FileURL,
		SizeBytes:  d.SizeBytes,
		Note:       d.Note,
		UploadedBy: d.UploadedBy,
		UploadedAt: d.UploadedAt.Format(time.RFC3339),
	}
}

type globalParcelResponse struct {
	ID                string  `json:"id"`
	ParcelID          string  `json:"parcel_id"`
	AU1Code           string  `json:"au1_code"`
	AU2Code           string  `json:"au2_code"`
	AU3Code           string  `json:"au3_code"`
	RightType         int     `json:"right_type"`
	Landuse           string  `json:"landuse"`
	AreaM2            float64 `json:"area_m2"`
	AcquisitionAreaM2 float64 `json:"acquisition_area_m2"`
	CompensationPaid  bool    `json:"compensation_paid"`
	AcquisitionID     string  `json:"acquisition_id"`
	AcquisitionName   string  `json:"acquisition_name"`
	PlanCode          string  `json:"plan_code"`
	AcquisitionStatus int     `json:"acquisition_status"`
	StartDate         *string `json:"start_date"`
	EndDate           *string `json:"end_date"`
}

type parcelPaymentResponse struct {
	ID        string  `json:"id"`
	ParcelID  string  `json:"parcel_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	PaidAt    *string `json:"paid_at"`
	Note      string  `json:"note"`
	CreatedAt string  `json:"created_at"`
	CreatedBy string  `json:"created_by"`
}

func toGlobalParcelResponse(p *domain.GlobalParcel) globalParcelResponse {
	r := globalParcelResponse{
		ID:                p.ID.String(),
		ParcelID:          p.ParcelID,
		AU1Code:           p.AU1Code,
		AU2Code:           p.AU2Code,
		AU3Code:           p.AU3Code,
		RightType:         p.RightType,
		Landuse:           p.Landuse,
		AreaM2:            p.AreaM2,
		AcquisitionAreaM2: p.AcquisitionAreaM2,
		CompensationPaid:  p.CompensationPaid,
		AcquisitionID:     p.AcquisitionID.String(),
		AcquisitionName:   p.AcquisitionName,
		PlanCode:          p.PlanCode,
		AcquisitionStatus: p.AcquisitionStatus,
	}
	if p.StartDate != nil {
		s := p.StartDate.Format("2006-01-02")
		r.StartDate = &s
	}
	if p.EndDate != nil {
		e := p.EndDate.Format("2006-01-02")
		r.EndDate = &e
	}
	return r
}

func toLandAcqResponse(acq *domain.LandAcquisition) landAcquisitionResponse {
	r := landAcquisitionResponse{
		ID:              acq.ID.String(),
		PlanCode:        acq.PlanCode,
		PlanName:        acq.PlanName,
		GeometryWKT:     acq.GeometryWKT,
		AreaM2:          acq.AreaM2,
		Status:          acq.Status,
		AcquisitionName: acq.AcquisitionName,
		ImplementingOrg: acq.ImplementingOrg,
		Reason:          acq.Reason,
		ResponsibleOrg:  acq.ResponsibleOrg,
		FundingSource:   acq.FundingSource,
		CreatedAt:       acq.CreatedAt.Format(time.RFC3339),
		CreatedBy:       acq.CreatedBy,
		ParcelCount:     acq.ParcelCount,
	}
	if acq.StartDate != nil {
		s := acq.StartDate.Format("2006-01-02")
		r.StartDate = &s
	}
	if acq.EndDate != nil {
		e := acq.EndDate.Format("2006-01-02")
		r.EndDate = &e
	}
	for _, au := range acq.AUs {
		r.AUs = append(r.AUs, auResponse{
			AU1Code: au.AU1Code, AU1Name: au.AU1Name,
			AU2Code: au.AU2Code, AU2Name: au.AU2Name,
			AU3Code: au.AU3Code, AU3Name: au.AU3Name,
		})
	}
	for _, p := range acq.Parcels {
		r.Parcels = append(r.Parcels, parcelResponse{
			ID: p.ID.String(), ParcelID: p.ParcelID,
			AU1Code: p.AU1Code, AU2Code: p.AU2Code, AU3Code: p.AU3Code,
			RightType: p.RightType, Landuse: p.Landuse, AreaM2: p.AreaM2,
		})
	}
	return r
}

func toParcelFullResponse(pf *domain.ParcelFull) parcelFullResponse {
	r := parcelFullResponse{
		ID:                 pf.ID.String(),
		ParcelID:           pf.ParcelID,
		OldParcelID:        pf.OldParcelID,
		AcquisitionID:      pf.AcquisitionID.String(),
		AU1Code:            pf.AU1Code,
		AU2Code:            pf.AU2Code,
		AU3Code:            pf.AU3Code,
		GeometryWKT:        pf.GeometryWKT,
		AcquisitionGeomWKT: pf.AcquisitionGeomWKT,
		AcquisitionAreaM2:  pf.AcquisitionAreaM2,
		RightType:          pf.RightType,
		Landuse:            pf.Landuse,
		AreaM2:             pf.AreaM2,
		CreatedAt:          pf.CreatedAt.Format(time.RFC3339),
		CreatedBy:          pf.CreatedBy,
	}
	if pf.ValidFrom != nil {
		s := pf.ValidFrom.Format("2006-01-02")
		r.ValidFrom = &s
	}
	if pf.ValidTill != nil {
		s := pf.ValidTill.Format("2006-01-02")
		r.ValidTill = &s
	}
	if pf.Detail != nil {
		d := pf.Detail
		di := &parcelDetailInfo{
			RightType:        d.RightType,
			HolderLastName:   d.HolderLastName,
			HolderName:       d.HolderName,
			HolderRegisterNo: d.HolderRegisterNo,
			HolderPhone:      d.HolderPhone,
			HolderEmail:      d.HolderEmail,
			HolderType:       d.HolderType,
			HolderCivilID:    d.HolderCivilID,
			AppNo:            d.AppNo,
			DecisionNo:       d.DecisionNo,
			ContractNo:       d.ContractNo,
			CertificateNo:    d.CertificateNo,
		}
		if d.DecisionDate != nil {
			s := d.DecisionDate.Format("2006-01-02")
			di.DecisionDate = &s
		}
		if d.ContractDate != nil {
			s := d.ContractDate.Format("2006-01-02")
			di.ContractDate = &s
		}
		if d.CertificateDate != nil {
			s := d.CertificateDate.Format("2006-01-02")
			di.CertificateDate = &s
		}
		r.Detail = di
	}
	return r
}
