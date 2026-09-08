package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
)

const maxOwnershipResponseBytes = 4 << 20

// OwnershipInfoClient — УБЕГ-ийн (Улсын бүртгэлийн ерөнхий газар) газар
// өмчлөлийн бүртгэлийг XYP гарцаар уншина.
//
// Энэ нь ГУС-ийн дундын сервис (ParcelInfoClient) БИШ, тусдаа хаягтай гарц
// тул тохиргоо (XYP_BASE_URL) ба клиент нь салангид байна.
type OwnershipInfoClient interface {
	// ListParcelOwnerships — нэгж талбарын дугаараар өмчлөгчид ба өмчлөлийн
	// түүхийн мөрүүд. Бүртгэл байхгүй үед хоосон жагсаалт (алдаа биш).
	ListParcelOwnerships(ctx context.Context, parcelID string) ([]domain.ParcelOwnership, error)
}

type httpOwnershipInfoClient struct {
	baseURL string
	http    *http.Client
}

// NewHTTPOwnershipInfoClient — baseURL нь XYP гарцын ЯЗГУУР
// (ж: http://192.168.15.224:8050/api/geo). Хоосон үед клиент буцахгүй —
// дуудагч nil эсэхээр шалгана.
func NewHTTPOwnershipInfoClient(baseURL string) OwnershipInfoClient {
	return &httpOwnershipInfoClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		// ХУР гарц ГУС-ийн дундын сервисээс УДААН хариулдаг (бүртгэлгүй
		// талбар дээр хариу огт эргэхгүй тохиолдол ч бий) тул 10 секундээс урт.
		// Гэхдээ frontend-ийн 30 секундын хязгаараас БАГА байх ёстой: эс бөгөөс
		// browser эрт таслаж, серверийн монгол мессеж хэрэглэгчид хүрэхгүй.
		http: &http.Client{Timeout: 20 * time.Second},
	}
}

// ownershipRow — XYP-ийн хариултын мөр. Талбарын нэрс эх системийн бичиглэлээр
// (галиглал) ирдэг тул ЗӨВХӨН энд харагдана — цаашид domain нэрээр дамжина.
type ownershipRow struct {
	Ognoo             string  `json:"ognoo"`
	UBD               string  `json:"ubd"`
	Zoriulalt         string  `json:"zoriulalt"`
	Hemjee            string  `json:"hemjee"`
	PrprtyParcelID    string  `json:"prprty_parcel_id"`
	Registr           string  `json:"registr"`
	Ner               string  `json:"ner"`
	Ovog              string  `json:"ovog"`
	Umchlugch         string  `json:"umchlugch"`
	UmchCode          string  `json:"umch_code"`
	Uilchilgee        string  `json:"uilchilgee"`
	ServiceCode       string  `json:"service_code"`
	AimagCityName     string  `json:"aimag_city_name"`
	AimagCode         string  `json:"aimag_code"`
	SoumDistrictName  string  `json:"soum_district_name"`
	SoumCode          string  `json:"soum_code"`
	BagKhorooName     string  `json:"bag_khoroo_name"`
	BagCode           string  `json:"bag_code"`
	AddressStreetName string  `json:"address_street_name"`
	BairNumber        *string `json:"bair_number"`
	Toot              *string `json:"toot"`
	FullAddress       string  `json:"full_address"`
}

func (c *httpOwnershipInfoClient) ListParcelOwnerships(
	ctx context.Context, parcelID string,
) ([]domain.ParcelOwnership, error) {
	parcelID = strings.TrimSpace(parcelID)
	if c.baseURL == "" || parcelID == "" {
		return nil, domain.ErrInvalidInput
	}

	q := url.Values{}
	q.Set("parcel_id", parcelID)
	reqURL := c.baseURL + "/xyp/ownership/citizen/parcel?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOwnershipResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxOwnershipResponseBytes {
		return nil, fmt.Errorf("response is too large")
	}
	// Бүртгэл байхгүй нэгж талбар — хоосон жагсаалт (алдаа биш).
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrInternalServer
	}

	// status=false үед data хоосон/байхгүй ирдэг — үүнийг ч хоосон гэж үзнэ.
	var envelope struct {
		Status bool           `json:"status"`
		Data   []ownershipRow `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.ParcelOwnership, 0, len(envelope.Data))
	for _, row := range envelope.Data {
		out = append(out, row.toDomain())
	}
	return out, nil
}

func (r ownershipRow) toDomain() domain.ParcelOwnership {
	return domain.ParcelOwnership{
		SourceParcelID:    strings.TrimSpace(r.PrprtyParcelID),
		RecordDate:        strings.TrimSpace(r.Ognoo),
		CertificateNo:     strings.TrimSpace(r.UBD),
		LanduseName:       strings.TrimSpace(r.Zoriulalt),
		Area:              parseOwnershipArea(r.Hemjee),
		RegisterNo:        strings.TrimSpace(r.Registr),
		FirstName:         strings.TrimSpace(r.Ner),
		LastName:          strings.TrimSpace(r.Ovog),
		OwnershipType:     strings.TrimSpace(r.Umchlugch),
		OwnershipCode:     strings.TrimSpace(r.UmchCode),
		ServiceName:       strings.TrimSpace(r.Uilchilgee),
		ServiceCode:       strings.TrimSpace(r.ServiceCode),
		AimagCityName:     strings.TrimSpace(r.AimagCityName),
		AimagCode:         strings.TrimSpace(r.AimagCode),
		SoumDistrictName:  strings.TrimSpace(r.SoumDistrictName),
		SoumCode:          strings.TrimSpace(r.SoumCode),
		BagKhorooName:     strings.TrimSpace(r.BagKhorooName),
		BagCode:           strings.TrimSpace(r.BagCode),
		AddressStreetName: strings.TrimSpace(r.AddressStreetName),
		BairNumber:        derefTrim(r.BairNumber),
		Toot:              derefTrim(r.Toot),
		FullAddress:       strings.TrimSpace(r.FullAddress),
	}
}

// parseOwnershipArea — hemjee нь ТЕКСТЭЭР ирдэг ("589"). Тоо болгож чадахгүй
// үед 0 буцна: хадгалалт бүтэн үлдэх нь нэг талбарын форматаас чухал.
func parseOwnershipArea(v string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}

// derefTrim — XYP-д хоосон талбар null-аар ирдэг (bair_number, toot).
func derefTrim(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
