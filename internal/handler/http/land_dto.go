package http

import (
	"time"

	"github.com/your-org/go-api-template/internal/domain"
)

type landAcquisitionResponse struct {
	ID                   string           `json:"id"`
	PlanCode             string           `json:"plan_code"`
	PlanName             string           `json:"plan_name"`
	GeometryWKT          string           `json:"geometry_wkt"`
	AreaM2               float64          `json:"area_m2"`
	Status               int              `json:"status"`
	StartDate            *string          `json:"start_date"`
	EndDate              *string          `json:"end_date"`
	AcquisitionName      string           `json:"acquisition_name"`
	ImplementingOrg      string           `json:"implementing_org"`
	Reason               string           `json:"reason"`
	ResponsibleOrg       string           `json:"responsible_org"`
	FundingSource        string           `json:"funding_source"`
	ConstructionTypeID   *int             `json:"construction_type_id"`
	ConstructionTypeName string           `json:"construction_type_name"`
	DecreeNumber         string           `json:"decree_number"`
	DecreeDate           *string          `json:"decree_date"`
	CreatedAt            string           `json:"created_at"`
	CreatedBy            string           `json:"created_by"`
	ParcelCount          int64            `json:"parcel_count"`
	AUs                  []auResponse     `json:"aus"`
	Parcels              []parcelResponse `json:"parcels"`
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
	ID                string   `json:"id"`
	ParcelID          string   `json:"parcel_id"`
	AU1Code           string   `json:"au1_code"`
	AU2Code           string   `json:"au2_code"`
	AU3Code           string   `json:"au3_code"`
	RightType         int      `json:"right_type"`
	Landuse           string   `json:"landuse"`
	AreaM2            float64  `json:"area_m2"`
	AcquisitionAreaM2 float64  `json:"acquisition_area_m2"`
	RemainingAreaM2   *float64 `json:"remaining_area_m2"`
	CompensationPaid  bool     `json:"compensation_paid"`
	DbChanged         bool     `json:"db_changed"`
	ChangedParcelID   string   `json:"changed_parcel_id"`
	GeometryWKT       string   `json:"geometry_wkt,omitempty"`
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
	RemainingAreaM2    *float64          `json:"remaining_area_m2"`
	RightType          int               `json:"right_type"`
	Landuse            string            `json:"landuse"`
	AreaM2             float64           `json:"area_m2"`
	DbChanged          bool              `json:"db_changed"`
	ChangedParcelID    string            `json:"changed_parcel_id"`
	CreatedAt          string            `json:"created_at"`
	CreatedBy          string            `json:"created_by"`
	Detail             *parcelDetailInfo `json:"detail,omitempty"`
}

type parcelDetailInfo struct {
	RightType        int      `json:"right_type"`
	HolderLastName   string   `json:"holder_last_name"`
	HolderName       string   `json:"holder_name"`
	HolderRegisterNo string   `json:"holder_register_no"`
	HolderPhone      string   `json:"holder_phone"`
	HolderEmail      string   `json:"holder_email"`
	HolderType       string   `json:"holder_type"`
	HolderCivilID    string   `json:"holder_civil_id"`
	AppNo            string   `json:"app_no"`
	DecisionNo       string   `json:"decision_no"`
	DecisionDate     *string  `json:"decision_date"`
	ContractNo       string   `json:"contract_no"`
	ContractDate     *string  `json:"contract_date"`
	CertificateNo    string   `json:"certificate_no"`
	CertificateDate  *string  `json:"certificate_date"`
	ValuationZone    string   `json:"valuation_zone"`
	BasePricePerHa   *float64 `json:"base_price_per_ha"`
	AuctionCoeff     *float64 `json:"auction_coeff"`
	AuctionPrice     *float64 `json:"auction_price"`
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

type dashboardResponse struct {
	Acquisitions   []landAcquisitionResponse  `json:"acquisitions"`
	ParcelStatuses []*domain.ParcelStatus     `json:"parcel_statuses"`
	ReportRows     []reportParcelRowResponse  `json:"report_rows"`
	Parcels        []globalParcelResponse     `json:"parcels"`
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
	Status            int     `json:"status"`
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

type assetResponse struct {
	ID            string  `json:"id"`
	AcquisitionID string  `json:"acquisition_id"`
	ParcelID      string  `json:"parcel_id"`
	AssetNumber   string  `json:"asset_number"`
	AssetType     string  `json:"asset_type"`
	AssetName     string  `json:"asset_name"`
	FloorCount    int     `json:"floor_count"`
	AreaM2        float64 `json:"area_m2"`
	OwnerName     string  `json:"owner_name"`
	Address       string  `json:"address"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func toAssetResponse(b *domain.Asset) assetResponse {
	return assetResponse{
		ID:            b.ID.String(),
		AcquisitionID: b.AcquisitionID.String(),
		ParcelID:      b.ParcelID,
		AssetNumber:   b.AssetNumber,
		AssetType:     b.AssetType,
		AssetName:     b.AssetName,
		FloorCount:    b.FloorCount,
		AreaM2:        b.AreaM2,
		OwnerName:     b.OwnerName,
		Address:       b.Address,
		Notes:         b.Notes,
		CreatedAt:     b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     b.UpdatedAt.Format(time.RFC3339),
	}
}

type compensationGrantResponse struct {
	ID             string  `json:"id"`
	AcquisitionID  string  `json:"acquisition_id"`
	CompensationID string  `json:"compensation_id"`
	Amount         float64 `json:"amount"`
	GrantDate      *string `json:"grant_date"`
	Note           string  `json:"note"`
	LandAreaM2     float64 `json:"land_area_m2"`
	LandPrice      float64 `json:"land_price"`
	LandLocation   string  `json:"land_location"`
	LandPurpose    string  `json:"land_purpose"`
	LandUseType    string  `json:"land_use_type"`
	ParcelNumber   string  `json:"parcel_number"`
	DecreeNumber   string  `json:"decree_number"`
	CreatedAt      string  `json:"created_at"`
	CreatedBy      string  `json:"created_by"`
	UpdatedAt      string  `json:"updated_at"`
}

func toCompensationGrantResponse(g *domain.CompensationGrant) compensationGrantResponse {
	r := compensationGrantResponse{
		ID:             g.ID.String(),
		AcquisitionID:  g.AcquisitionID.String(),
		CompensationID: g.CompensationID.String(),
		Amount:         g.Amount,
		Note:           g.Note,
		LandAreaM2:     g.LandAreaM2,
		LandPrice:      g.LandPrice,
		LandLocation:   g.LandLocation,
		LandPurpose:    g.LandPurpose,
		LandUseType:    g.LandUseType,
		ParcelNumber:   g.ParcelNumber,
		DecreeNumber:   g.DecreeNumber,
		CreatedAt:      g.CreatedAt.Format(time.RFC3339),
		CreatedBy:      g.CreatedBy,
		UpdatedAt:      g.UpdatedAt.Format(time.RFC3339),
	}
	if g.GrantDate != nil {
		s := g.GrantDate.Format("2006-01-02")
		r.GrantDate = &s
	}
	return r
}

type compensationResponse struct {
	ID               string                     `json:"id"`
	AcquisitionID    string                     `json:"acquisition_id"`
	TargetType       string                     `json:"target_type"`
	ParcelID         string                     `json:"parcel_id"`
	AssetID          *string                    `json:"asset_id"`
	CompensationType string                     `json:"compensation_type"`
	CoveragePercent  float64                    `json:"coverage_percent"`
	Amount           float64                    `json:"amount"`
	CompensationDate *string                    `json:"compensation_date"`
	Note             string                     `json:"note"`
	Grant            *compensationGrantResponse `json:"grant,omitempty"`
	CreatedAt        string                     `json:"created_at"`
	CreatedBy        string                     `json:"created_by"`
	UpdatedAt        string                     `json:"updated_at"`
}

func toCompensationResponse(c *domain.Compensation) compensationResponse {
	r := compensationResponse{
		ID:               c.ID.String(),
		AcquisitionID:    c.AcquisitionID.String(),
		TargetType:       c.TargetType,
		ParcelID:         c.ParcelID,
		CompensationType: c.CompensationType,
		CoveragePercent:  c.CoveragePercent,
		Amount:           c.Amount,
		Note:             c.Note,
		CreatedAt:        c.CreatedAt.Format(time.RFC3339),
		CreatedBy:        c.CreatedBy,
		UpdatedAt:        c.UpdatedAt.Format(time.RFC3339),
	}
	if c.AssetID != nil {
		s := c.AssetID.String()
		r.AssetID = &s
	}
	if c.CompensationDate != nil {
		s := c.CompensationDate.Format("2006-01-02")
		r.CompensationDate = &s
	}
	if c.Grant != nil {
		gr := toCompensationGrantResponse(c.Grant)
		r.Grant = &gr
	}
	return r
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
		Status:            p.Status,
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
		ID:                   acq.ID.String(),
		PlanCode:             acq.PlanCode,
		PlanName:             acq.PlanName,
		GeometryWKT:          acq.GeometryWKT,
		AreaM2:               acq.AreaM2,
		Status:               acq.Status,
		AcquisitionName:      acq.AcquisitionName,
		ImplementingOrg:      acq.ImplementingOrg,
		Reason:               acq.Reason,
		ResponsibleOrg:       acq.ResponsibleOrg,
		FundingSource:        acq.FundingSource,
		ConstructionTypeID:   acq.ConstructionTypeID,
		ConstructionTypeName: acq.ConstructionTypeName,
		DecreeNumber:         acq.DecreeNumber,
		CreatedAt:            acq.CreatedAt.Format(time.RFC3339),
		CreatedBy:            acq.CreatedBy,
		ParcelCount:          acq.ParcelCount,
	}
	if acq.StartDate != nil {
		s := acq.StartDate.Format("2006-01-02")
		r.StartDate = &s
	}
	if acq.EndDate != nil {
		e := acq.EndDate.Format("2006-01-02")
		r.EndDate = &e
	}
	if acq.DecreeDate != nil {
		s := acq.DecreeDate.Format("2006-01-02")
		r.DecreeDate = &s
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
			AcquisitionAreaM2: p.AcquisitionAreaM2, RemainingAreaM2: p.RemainingAreaM2,
			CompensationPaid: p.CompensationPaid, DbChanged: p.DbChanged,
			ChangedParcelID: p.ChangedParcelID,
		})
	}
	return r
}

func toParcelResponse(p *domain.Parcel) parcelResponse {
	return parcelResponse{
		ID:                p.ID.String(),
		ParcelID:          p.ParcelID,
		AU1Code:           p.AU1Code,
		AU2Code:           p.AU2Code,
		AU3Code:           p.AU3Code,
		RightType:         p.RightType,
		Landuse:           p.Landuse,
		AreaM2:            p.AreaM2,
		AcquisitionAreaM2: p.AcquisitionAreaM2,
		RemainingAreaM2:   p.RemainingAreaM2,
		CompensationPaid:  p.CompensationPaid,
		DbChanged:         p.DbChanged,
		ChangedParcelID:   p.ChangedParcelID,
		GeometryWKT:       p.GeometryWKT,
	}
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
		RemainingAreaM2:    pf.RemainingAreaM2,
		RightType:          pf.RightType,
		Landuse:            pf.Landuse,
		AreaM2:             pf.AreaM2,
		DbChanged:          pf.DbChanged,
		ChangedParcelID:    pf.ChangedParcelID,
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
		di.ValuationZone = d.ValuationZone
		di.BasePricePerHa = d.BasePricePerHa
		di.AuctionCoeff = d.AuctionCoeff
		di.AuctionPrice = d.AuctionPrice
		r.Detail = di
	}
	return r
}

type reportParcelRowResponse struct {
	ParcelID             string  `json:"parcel_id"`
	AreaM2               float64 `json:"area_m2"`
	AcquisitionAreaM2    float64 `json:"acquisition_area_m2"`
	RemainingAreaM2      float64 `json:"remaining_area_m2"`
	RightType            int     `json:"right_type"`
	DbChanged            bool    `json:"db_changed"`
	ChangedParcelID      string  `json:"changed_parcel_id"`
	AcquisitionID        string  `json:"acquisition_id"`
	AcquisitionName      string  `json:"acquisition_name"`
	PlanCode             string  `json:"plan_code"`
	ConstructionTypeName string  `json:"construction_type_name"`
	DecreeNumber         string  `json:"decree_number"`
	DecreeDate           *string `json:"decree_date"`
	HolderLastName       string  `json:"holder_last_name"`
	HolderName           string  `json:"holder_name"`
	HolderRegisterNo     string  `json:"holder_register_no"`
	LandComp             float64 `json:"land_comp"`
	RealStateComp        float64 `json:"real_state_comp"`
	PropertyComp         float64 `json:"property_comp"`
	TotalComp            float64 `json:"total_comp"`
}

func toReportParcelRowResponse(r *domain.ReportParcelRow) reportParcelRowResponse {
	resp := reportParcelRowResponse{
		ParcelID:             r.ParcelID,
		AreaM2:               r.AreaM2,
		AcquisitionAreaM2:    r.AcquisitionAreaM2,
		RemainingAreaM2:      r.RemainingAreaM2,
		RightType:            r.RightType,
		DbChanged:            r.DbChanged,
		ChangedParcelID:      r.ChangedParcelID,
		AcquisitionID:        r.AcquisitionID.String(),
		AcquisitionName:      r.AcquisitionName,
		PlanCode:             r.PlanCode,
		ConstructionTypeName: r.ConstructionTypeName,
		DecreeNumber:         r.DecreeNumber,
		HolderLastName:       r.HolderLastName,
		HolderName:           r.HolderName,
		HolderRegisterNo:     r.HolderRegisterNo,
		LandComp:             r.LandComp,
		RealStateComp:        r.RealStateComp,
		PropertyComp:         r.PropertyComp,
		TotalComp:            r.LandComp + r.RealStateComp + r.PropertyComp,
	}
	if r.DecreeDate != nil {
		s := r.DecreeDate.Format("2006-01-02")
		resp.DecreeDate = &s
	}
	return resp
}
