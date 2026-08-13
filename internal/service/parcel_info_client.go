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
	// ListApplicationPersons — хүсэлтийн дугаараар (app_no) эзэмшигчдийн
	// мэдээлэл. Нэг хүсэлт дээр хэд хэдэн хүн байж болно. appNo хоосон бол
	// нэгж талбарын дугаараар хайна.
	ListApplicationPersons(ctx context.Context, appNo, parcelID string) ([]*domain.ParcelHolder, error)
}

type PlanProjectClient interface {
	GetPlanProject(ctx context.Context, code string) (*domain.Plan, error)
}

type httpParcelInfoClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPParcelInfoClient(baseURL string) ParcelInfoClient {
	return &httpParcelInfoClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
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
	if authorization := authorizationFromContext(ctx); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

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
func (c *httpParcelInfoClient) ListApplicationPersons(ctx context.Context, appNo, parcelID string) ([]*domain.ParcelHolder, error) {
	appNo = strings.TrimSpace(appNo)
	parcelID = strings.TrimSpace(parcelID)
	if c.baseURL == "" || (appNo == "" && parcelID == "") {
		return nil, domain.ErrInvalidInput
	}

	query := url.Values{}
	if appNo != "" {
		query.Set("app_no", appNo)
	} else {
		query.Set("parcel_id", parcelID)
	}

	reqURL := c.baseURL + "/application/persons?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if authorization := authorizationFromContext(ctx); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

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
	if authorization := authorizationFromContext(ctx); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

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

func (c *httpParcelInfoClient) GetPlanProject(ctx context.Context, code string) (*domain.Plan, error) {
	if c.baseURL == "" || code == "" {
		return nil, domain.ErrInvalidInput
	}

	reqURL := c.baseURL + "/plan/project?code=" + url.QueryEscape(code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if authorization := authorizationFromContext(ctx); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

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
		ProjectID    string  `json:"project_id"`
		Code         string  `json:"code"`
		Name         string  `json:"name"`
		PlanTypeName string  `json:"plan_type_name"`
		StartDate    *string `json:"start_date"`
		EndDate      *string `json:"end_date"`
		StartData    *string `json:"start_data"`
		EndData      *string `json:"end_data"`
		ApprovedDate *string `json:"approved_date"`
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
		ProjectID:    project.ProjectID,
		Code:         project.Code,
		PlanCode:     project.Code,
		Name:         project.Name,
		PlanTypeName: project.PlanTypeName,
		StartData:    project.StartData,
		EndData:      project.EndData,
		ApprovedDate: project.ApprovedDate,
	}, nil
}
