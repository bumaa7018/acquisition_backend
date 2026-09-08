package domain

import "time"

// SettingKeyBasemap — газрын зургийн суурь зургийн тохиргооны түлхүүр.
const SettingKeyBasemap = "basemap"

// SettingRecord — app_setting-ийн НЭГ мөр (утга нь JSON текст).
type SettingRecord struct {
	Key       string
	Value     string
	UpdatedAt time.Time
	UpdatedBy string
}

// Суурь зургийн эх сурвалжийн ТӨРЛҮҮД.
//
//   - BasemapTypeXYZ   — тайлын (tile) сервис: URL-д {x}/{y}/{z} орлуулагчтай
//     (ж: https://.../tile/{z}/{y}/{x}). Хамгийн өргөн хэрэглэгддэг.
//   - BasemapTypeWMS   — растер зураг үйлдвэрлэгч WMS сервис: GetMap-ийг
//     дэлгэцийн хүрээгээр дуудна (layer нэр ЗААВАЛ).
//   - BasemapTypeImage — ТОГТМОЛ нэг зураг (georeferenced): дэлгэцэд өөрийн
//     хүрээгээр (extent) тавигдана.
const (
	BasemapTypeXYZ   = "xyz"
	BasemapTypeWMS   = "wms"
	BasemapTypeImage = "image"
)

// Basemap — суурь зургийн тохиргоо.
//
// Enabled=false эсвэл URL хоосон бол дэлгэц ӨГӨГДМӨЛ суурь зургаа хэрэглэнэ
// (тохиргоо "байхгүй"-тэй адил).
type Basemap struct {
	Type    string `json:"type"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
	// Layer — ЗӨВХӨН WMS-д (GetMap-ийн layers параметр).
	Layer string `json:"layer,omitempty"`
	// MaxZoom — tile сервисийн хамгийн ойрын түвшин (0 = заагаагүй).
	MaxZoom int `json:"max_zoom,omitempty"`
	// Extent — ЗӨВХӨН image төрөлд: [minLon, minLat, maxLon, maxLat] (WGS84).
	Extent []float64 `json:"extent,omitempty"`
	// Attribution — зургийн эх сурвалжийн бичиглэл (газрын зургийн буланд).
	Attribution string `json:"attribution,omitempty"`
	// Name — тохиргооны танигдах нэр (ж: "Агаарын зураг 2026").
	Name      string     `json:"name,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UpdatedBy string     `json:"updated_by,omitempty"`
}
