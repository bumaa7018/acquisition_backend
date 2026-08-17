package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
)

const maxParcelInfoResponseBytes = 1 << 20
const maxParcelListResponseBytes = 20 << 20
const middlewareUsernameHeader = "X-Middleware-Username"
const middlewarePasswordHeader = "X-Middleware-Password"

type authHeaderContextKey struct{}

// ContextWithAuthorization carries the caller's bearer token to internal middleware calls.
func ContextWithAuthorization(ctx context.Context, authorization string) context.Context {
	if authorization == "" {
		return ctx
	}
	return context.WithValue(ctx, authHeaderContextKey{}, authorization)
}

func authorizationFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authHeaderContextKey{}).(string); ok {
		return v
	}
	return ""
}

// ParcelInfoClient fetches parcel source data from the middleware service.
type ParcelInfoClient interface {
	GetParcelInfo(ctx context.Context, parcelID string) (*domain.SourceParcel, error)
	ListParcelIDsByAcquisition(ctx context.Context, acquisitionWKT string) ([]string, error)
	// ListApplicationPersons — ЗӨВХӨН хүсэлтийн дугаараар (app_no)
	// эзэмшигчдийн мэдээлэл. Нэг хүсэлт дээр хэд хэдэн хүн байж болно.
	//
	// Нэгж талбарын дугаараар хайх нөөц зам БАЙХГҮЙ: нэг талбар дээр хүсэлт
	// олон удаа гардаг тул өөр өөр хүсэлтийн эзэмшигчид хольцолдоно.
	ListApplicationPersons(ctx context.Context, appNo string) ([]*domain.ParcelHolder, error)
	// ListApplicationDocuments — хүсэлтийн дугаараар (app_no) хавсаргасан
	// баримтууд. roles өгвөл баримтын үүргээр (doc_code) нэмж шүүнэ; хоосон
	// бол бүх үүрэг буцна.
	ListApplicationDocuments(ctx context.Context, appNo string, roles []string) ([]*domain.SourceDocument, error)
	// Газрын суурь үнэ, төлбөр, барьцаа, шүүхийн шийдвэр — ГУС-ийн тусдаа гарцууд
	GetParcelBasePrice(ctx context.Context, parcelID string) (*domain.ParcelBasePrice, error)
	ListParcelInvoices(ctx context.Context, parcelID string) ([]*domain.ParcelInvoice, error)
	ListParcelMortgages(ctx context.Context, parcelID string) ([]*domain.ParcelMortgage, error)
	ListCourtDecisions(ctx context.Context, appNo string) ([]*domain.ParcelCourtDecision, error)
	ListParcelMonitorings(ctx context.Context, parcelID string) ([]*domain.ParcelMonitoring, error)
}

type PlanProjectClient interface {
	GetPlanProject(ctx context.Context, code string) (*domain.Plan, error)
}

type MiddlewareCredentials struct {
	Username string
	Password string
}

type httpParcelInfoClient struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

func NewHTTPParcelInfoClient(baseURL string, credentials ...MiddlewareCredentials) ParcelInfoClient {
	var creds MiddlewareCredentials
	if len(credentials) > 0 {
		creds = credentials[0]
	}
	return &httpParcelInfoClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: strings.TrimSpace(creds.Username),
		password: strings.TrimSpace(creds.Password),
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *httpParcelInfoClient) applyRequestHeaders(ctx context.Context, req *http.Request) {
	if authorization := authorizationFromContext(ctx); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if c.username != "" || c.password != "" {
		req.Header.Set(middlewareUsernameHeader, c.username)
		req.Header.Set(middlewarePasswordHeader, c.password)
	}
}

func (c *httpParcelInfoClient) GetParcelInfo(ctx context.Context, parcelID string) (*domain.SourceParcel, error) {
	if c.baseURL == "" || parcelID == "" {
		return nil, domain.ErrInvalidInput
	}

	reqURL := c.baseURL + "/parcel/info/" + url.PathEscape(parcelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.applyRequestHeaders(ctx, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParcelInfoResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxParcelInfoResponseBytes {
		return nil, fmt.Errorf("parcel info response is too large")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrInternalServer
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 {
		return nil, domain.ErrNotFound
	}

	// Хариулт нь эх системийн SQL-ийн баганыг шууд дамжуулдаг тул хэлбэрийн
	// зөрүүг тэвчих тусгай бүтцээр уншина (parcel_info_payload.go-г үз).
	var payload parcelInfoPayload
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, err
	}
	src := payload.toSourceParcel(parcelID)
	if src.ParcelID == "" {
		return nil, domain.ErrNotFound
	}
	return src, nil
}

// ListApplicationPersons — middleware /application/persons. Нэгж талбарын
// дэлгэрэнгүйгээс гарсан app_no-гоор эзэмшигчдийг (олон хүн) татна.
func (c *httpParcelInfoClient) ListApplicationPersons(ctx context.Context, appNo string) ([]*domain.ParcelHolder, error) {
	appNo = strings.TrimSpace(appNo)
	if c.baseURL == "" || appNo == "" {
		return nil, domain.ErrInvalidInput
	}

	query := url.Values{}
	query.Set("app_no", appNo)

	reqURL := c.baseURL + "/application/persons?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.applyRequestHeaders(ctx, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParcelInfoResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxParcelInfoResponseBytes {
		return nil, fmt.Errorf("application persons response is too large")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrInternalServer
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 {
		return nil, nil
	}

	var persons []struct {
		MainApplicant  bool   `json:"main_applicant"`
		Name           string `json:"name"`
		FirstName      string `json:"first_name"`
		PersonRegister string `json:"person_register"`
		Phone          string `json:"phone"`
		EmailAddress   string `json:"email_address"`
		PersonType     string `json:"person_type"`
	}
	if err := json.Unmarshal(envelope.Data, &persons); err != nil {
		return nil, err
	}

	holders := make([]*domain.ParcelHolder, 0, len(persons))
	for _, p := range persons {
		name := strings.TrimSpace(p.Name)
		firstName := strings.TrimSpace(p.FirstName)

		holder := &domain.ParcelHolder{
			MainApplicant: p.MainApplicant,
			// bs_person.name нь овог/эцгийн нэр, first_name нь өөрийн нэр.
			LastName: name,
			Name:     firstName,
			// Хуулийн этгээд үед first_name хоосон ирдэг — байгууллагын бүтэн
			// нэрийг хоёр талбарт хуулахгүй, зөвхөн Name-д бичнэ.
			RegisterNo: strings.TrimSpace(p.PersonRegister),
			Phone:      strings.TrimSpace(p.Phone),
			Email:      strings.TrimSpace(p.EmailAddress),
			PersonType: strings.TrimSpace(p.PersonType),
			AppNo:      appNo,
		}
		if firstName == "" {
			holder.LastName, holder.Name = "", name
		}
		if holder.Name == "" && holder.LastName == "" && holder.RegisterNo == "" {
			continue // хоосон мөр
		}
		holders = append(holders, holder)
	}
	return holders, nil
}

// ListApplicationDocuments — middleware /application/documents. Нэгж талбарын
// дэлгэрэнгүйгээс гарсан app_no-гоор хавсралтуудыг татна.
func (c *httpParcelInfoClient) ListApplicationDocuments(ctx context.Context, appNo string, roles []string) ([]*domain.SourceDocument, error) {
	appNo = strings.TrimSpace(appNo)
	if c.baseURL == "" || appNo == "" {
		return nil, domain.ErrInvalidInput
	}

	query := url.Values{}
	query.Set("app_no", appNo)
	for _, role := range roles {
		if role = strings.TrimSpace(role); role != "" {
			query.Add("role", role)
		}
	}

	reqURL := c.baseURL + "/application/documents?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.applyRequestHeaders(ctx, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParcelInfoResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxParcelInfoResponseBytes {
		return nil, fmt.Errorf("application documents response is too large")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrInternalServer
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 {
		return nil, nil
	}

	// Эх системийн баганууд шууд дамждаг тул кодууд тоо ч, текст ч байж болно.
	var payload []struct {
		AppNo   flexString `json:"app_no"`
		DocName flexString `json:"doc_name"`
		DocCode flexString `json:"doc_code"`
		DocID   flexString `json:"doc_id"`
		FileURL flexString `json:"file_url"`
		FtpID   flexString `json:"ftp_id"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, err
	}

	docs := make([]*domain.SourceDocument, 0, len(payload))
	for _, p := range payload {
		// Баримтын дугааргүй мөр давхардлыг таних боломжгүй тул хэрэггүй.
		docID := p.DocID.valueNoZero()
		if docID == "" {
			continue
		}
		docs = append(docs, &domain.SourceDocument{
			AppNo:   firstNonEmpty(p.AppNo.value(), appNo),
			DocName: p.DocName.value(),
			DocCode: p.DocCode.value(),
			DocID:   docID,
			FileURL: p.FileURL.value(),
			FtpID:   p.FtpID.valueNoZero(),
		})
	}
	return docs, nil
}

func (c *httpParcelInfoClient) ListParcelIDsByAcquisition(ctx context.Context, acquisitionWKT string) ([]string, error) {
	if c.baseURL == "" || acquisitionWKT == "" {
		return nil, domain.ErrInvalidInput
	}

	payload, err := json.Marshal(struct {
		Geometry string `json:"geometry"`
	}{
		Geometry: acquisitionWKT,
	})
	if err != nil {
		return nil, err
	}

	reqURL := c.baseURL + "/parcels/by/acquisition"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyRequestHeaders(ctx, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParcelListResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxParcelListResponseBytes {
		return nil, fmt.Errorf("parcel list response is too large")
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrInternalServer
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 {
		return nil, nil
	}

	var parcelIDs []string
	if err := json.Unmarshal(envelope.Data, &parcelIDs); err != nil {
		return nil, err
	}
	return parcelIDs, nil
}

// getJSON — ГУС-ийн дундын сервисээс JSON уншиж, envelope-ийн data-г out руу
// задална. 404 үед notFoundNil=true бол (nil, nil) буцна (хоосон жагсаалт).
func (c *httpParcelInfoClient) getJSON(ctx context.Context, path string, out any, notFoundNil bool) error {
	if c.baseURL == "" {
		return domain.ErrInvalidInput
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.applyRequestHeaders(ctx, req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParcelInfoResponseBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxParcelInfoResponseBytes {
		return fmt.Errorf("response is too large")
	}
	if resp.StatusCode == http.StatusNotFound {
		if notFoundNil {
			return nil
		}
		return domain.ErrNotFound
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return domain.ErrInternalServer
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if len(envelope.Data) == 0 {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}

func (c *httpParcelInfoClient) GetParcelBasePrice(ctx context.Context, parcelID string) (*domain.ParcelBasePrice, error) {
	parcelID = strings.TrimSpace(parcelID)
	if parcelID == "" {
		return nil, domain.ErrInvalidInput
	}
	bp := &domain.ParcelBasePrice{}
	if err := c.getJSON(ctx, "/parcel/base-price/"+url.PathEscape(parcelID), bp, false); err != nil {
		return nil, err
	}
	return bp, nil
}

func (c *httpParcelInfoClient) ListParcelInvoices(ctx context.Context, parcelID string) ([]*domain.ParcelInvoice, error) {
	parcelID = strings.TrimSpace(parcelID)
	if parcelID == "" {
		return nil, domain.ErrInvalidInput
	}
	var out []*domain.ParcelInvoice
	if err := c.getJSON(ctx, "/parcel/invoices/"+url.PathEscape(parcelID), &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *httpParcelInfoClient) ListParcelMortgages(ctx context.Context, parcelID string) ([]*domain.ParcelMortgage, error) {
	parcelID = strings.TrimSpace(parcelID)
	if parcelID == "" {
		return nil, domain.ErrInvalidInput
	}
	q := url.Values{}
	q.Set("parcel_id", parcelID)
	var out []*domain.ParcelMortgage
	if err := c.getJSON(ctx, "/application/mortgages?"+q.Encode(), &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *httpParcelInfoClient) ListCourtDecisions(ctx context.Context, appNo string) ([]*domain.ParcelCourtDecision, error) {
	appNo = strings.TrimSpace(appNo)
	if appNo == "" {
		return nil, domain.ErrInvalidInput
	}
	q := url.Values{}
	q.Set("app_no", appNo)
	var out []*domain.ParcelCourtDecision
	if err := c.getJSON(ctx, "/application/court-decisions?"+q.Encode(), &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *httpParcelInfoClient) ListParcelMonitorings(ctx context.Context, parcelID string) ([]*domain.ParcelMonitoring, error) {
	parcelID = strings.TrimSpace(parcelID)
	if parcelID == "" {
		return nil, domain.ErrInvalidInput
	}
	var out []*domain.ParcelMonitoring
	if err := c.getJSON(ctx, "/parcel/monitoring/"+url.PathEscape(parcelID), &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *httpParcelInfoClient) GetPlanProject(ctx context.Context, code string) (*domain.Plan, error) {
	if c.baseURL == "" || code == "" {
		return nil, domain.ErrInvalidInput
	}

	reqURL := c.baseURL + "/plan/project?code=" + url.QueryEscape(code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.applyRequestHeaders(ctx, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParcelInfoResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxParcelInfoResponseBytes {
		return nil, fmt.Errorf("plan project response is too large")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrInternalServer
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 {
		return nil, domain.ErrNotFound
	}

	var project struct {
		// ParcelID нь тоогоор ч ирж болно (эх багана numeric).
		ParcelID     flexString `json:"parcel_id"`
		ProjectID    flexString `json:"project_id"`
		Code         string     `json:"code"`
		Name         string     `json:"name"`
		PlanTypeName string     `json:"plan_type_name"`
		// Gazner — төлөвлөгөөний нэгж талбар дээрх бүтээн байгуулалтын ажил.
		Gazner string `json:"gazner"`
		// GeometryWKT — төлөвлөгөөний нэгж талбарын ХИЛ
		// (data_plan.pl_project_parcel.polygon_geom). Хоосон байж болно.
		GeometryWKT  flexString `json:"geometry_wkt"`
		StartDate    *string    `json:"start_date"`
		EndDate      *string    `json:"end_date"`
		StartData    *string    `json:"start_data"`
		EndData      *string    `json:"end_data"`
		ApprovedDate *string    `json:"approved_date"`
	}
	if err := json.Unmarshal(envelope.Data, &project); err != nil {
		return nil, err
	}
	if project.Code == "" {
		return nil, domain.ErrNotFound
	}

	if project.StartData == nil {
		project.StartData = project.StartDate
	}
	if project.EndData == nil {
		project.EndData = project.EndDate
	}

	return &domain.Plan{
		ProjectID:    project.ProjectID.value(),
		Code:         project.Code,
		PlanCode:     project.Code,
		Name:         project.Name,
		PlanTypeName: project.PlanTypeName,
		// Хариултад дугаар ирээгүй бол хайсан дугаараар бүртгэнэ.
		ParcelID:     firstNonEmpty(project.ParcelID.value(), code),
		Gazner:       strings.TrimSpace(project.Gazner),
		BoundaryWKT:  sanitizeWKT(project.GeometryWKT.value()),
		StartData:    project.StartData,
		EndData:      project.EndData,
		ApprovedDate: project.ApprovedDate,
	}, nil
}
