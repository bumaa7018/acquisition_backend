package service

// Системийн тохиргоо. Одоогоор ЗӨВХӨН газрын зургийн СУУРЬ зураг
// (төлөвлөгөөний 2.6.1: агаар, сансрын зургийг дэд системд шинэчлэх).
//
// Тохируулаагүй үед (мөр байхгүй) nil буцна — дэлгэц өөрийн үндсэн суурь
// зургаа хэрэглэнэ. Иймд "тохиргоо байхгүй" нь АЛДАА биш.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type SettingService interface {
	// GetBasemap — тохируулсан суурь зураг. Тохируулаагүй бол (nil, nil).
	GetBasemap(ctx context.Context) (*domain.Basemap, error)
	// SaveBasemap — хаяг, төрлийг шалгаад хадгална.
	SaveBasemap(ctx context.Context, in domain.Basemap, updatedBy string) (*domain.Basemap, error)
	// ResetBasemap — тохиргоог устгаж үндсэн зураг рүү буцаана.
	ResetBasemap(ctx context.Context) error
}

type settingService struct {
	repo repository.SettingRepository
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &settingService{repo: repo}
}

func (s *settingService) GetBasemap(ctx context.Context) (*domain.Basemap, error) {
	rec, err := s.repo.Get(ctx, domain.SettingKeyBasemap)
	if err == domain.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		slog.Error("суурь зургийн тохиргоо уншихад алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	basemap := &domain.Basemap{}
	if err := json.Unmarshal([]byte(rec.Value), basemap); err != nil {
		// Гараар эвдэрсэн JSON — газрын зургийг унагахгүйн тулд үндсэн зураг рүү.
		slog.Error("суурь зургийн тохиргоо задлагдсангүй", "error", err)
		return nil, nil
	}
	updatedAt := rec.UpdatedAt
	basemap.UpdatedAt = &updatedAt
	basemap.UpdatedBy = rec.UpdatedBy
	return basemap, nil
}

func (s *settingService) SaveBasemap(
	ctx context.Context, in domain.Basemap, updatedBy string,
) (*domain.Basemap, error) {
	clean, err := validateBasemap(in)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(clean)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	if err := s.repo.Upsert(ctx, domain.SettingKeyBasemap, string(payload), updatedBy); err != nil {
		slog.Error("суурь зургийн тохиргоо хадгалахад алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	return s.GetBasemap(ctx)
}

func (s *settingService) ResetBasemap(ctx context.Context) error {
	if err := s.repo.Delete(ctx, domain.SettingKeyBasemap); err != nil {
		slog.Error("суурь зургийн тохиргоо устгахад алдаа", "error", err)
		return domain.ErrInternalServer
	}
	return nil
}

// validateBasemap — хаяг, төрлийн шалгалт.
//
// Хаягийг browser ӨӨРӨӨ дуудах тул схемийг http/https-ээр хязгаарлана
// (javascript:, data: зэрэг хаяг дэлгэц дээр гүйцэтгэгдэхээс сэргийлнэ).
func validateBasemap(in domain.Basemap) (domain.Basemap, error) {
	out := domain.Basemap{
		Type:        strings.TrimSpace(strings.ToLower(in.Type)),
		URL:         strings.TrimSpace(in.URL),
		Enabled:     in.Enabled,
		Layer:       strings.TrimSpace(in.Layer),
		MaxZoom:     in.MaxZoom,
		Extent:      in.Extent,
		Attribution: strings.TrimSpace(in.Attribution),
		Name:        strings.TrimSpace(in.Name),
	}

	// Хоосон хаягтай тохиргоо = ҮНДСЭН суурь зураг (унтраасан).
	if out.URL == "" {
		out.Enabled = false
		if out.Type == "" {
			out.Type = domain.BasemapTypeXYZ
		}
		return out, nil
	}

	switch out.Type {
	case domain.BasemapTypeXYZ, domain.BasemapTypeWMS, domain.BasemapTypeImage:
	default:
		return out, domain.ErrInvalidBasemapType
	}

	parsed, err := url.Parse(out.URL)
	if err != nil || parsed.Host == "" {
		return out, domain.ErrInvalidBasemapURL
	}
	// Харьцангуй зам (/api/...) ч зөвшөөрөгдөнө — өөрийн proxy-гоор дамжуулах
	// тохиолдол байдаг. Тэр үед Host хоосон байх тул дээрх шалгалт схемтэй
	// хаягт л хамаарна.
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return out, domain.ErrInvalidBasemapURL
	}

	switch out.Type {
	case domain.BasemapTypeXYZ:
		// Tile сервисийн URL-д ЗААВАЛ орлуулагч байх ёстой, эс бөгөөс бүх
		// тайл нэг зураг болж газрын зураг "хөлдөнө".
		lower := strings.ToLower(out.URL)
		hasXY := strings.Contains(lower, "{x}") &&
			(strings.Contains(lower, "{y}") || strings.Contains(lower, "{-y}"))
		if !hasXY || !strings.Contains(lower, "{z}") {
			return out, domain.ErrInvalidBasemapTileURL
		}
	case domain.BasemapTypeWMS:
		if out.Layer == "" {
			return out, domain.ErrBasemapLayerRequired
		}
	case domain.BasemapTypeImage:
		// Тогтмол зургийг ХААНА тавихыг extent л тодорхойлно.
		if len(out.Extent) != 4 {
			return out, domain.ErrBasemapExtentRequired
		}
		if out.Extent[0] >= out.Extent[2] || out.Extent[1] >= out.Extent[3] {
			return out, domain.ErrBasemapExtentRequired
		}
	}

	if out.MaxZoom < 0 || out.MaxZoom > 24 {
		out.MaxZoom = 0
	}
	if out.Type != domain.BasemapTypeImage {
		out.Extent = nil
	}
	if out.Type != domain.BasemapTypeWMS {
		out.Layer = ""
	}
	return out, nil
}
