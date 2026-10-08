package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ChronosEvent — нэгж талбарын төлөвийн түүхийн нэг үйл явдал.
type ChronosEvent struct {
	Date     time.Time `json:"date"`
	StatusID int       `json:"status_id"`
	Name     string    `json:"name"`
}

// ChronosParcel — «Хронос» (цаг хугацааны изометр зураг)-ын нэг нэгж талбар.
// Geometry нь EPSG:3857 (метр) GeoJSON — изометр проекц, суурь tile-тай шууд нийцнэ.
type ChronosParcel struct {
	ID              uuid.UUID `json:"id"`
	ParcelID        string    `json:"parcel_id"`
	AcquisitionID   uuid.UUID `json:"acquisition_id"`
	AcquisitionName string    `json:"acquisition_name"`
	StatusID        int       `json:"status_id"`
	StatusName      string    `json:"status_name"`
	StatusColor     string    `json:"status_color"`
	// ReleasedAt — «Захирамжид холбогдоно» (is_released) төлөвт АНХ шилжсэн огноо.
	ReleasedAt *time.Time      `json:"released_at"`
	AreaM2     float64         `json:"area_m2"`
	Geometry   json.RawMessage `json:"geometry"`
	Events     []ChronosEvent  `json:"events"`

	// Нэгж талбар дарахад — эзэмшигч, зориулалт, эрхийн төрөл.
	HolderName  string `json:"holder_name"`
	LanduseName string `json:"landuse_name"`
	RightType   int    `json:"right_type"`
	// Нөхөх олговрын 3 үндсэн дүн (санхүүгийн самбартай ижил ангилал):
	// газар (хөрөнгөгүй), үл хөдлөх (real_state), эд хөрөнгө (бусад хөрөнгө).
	CompLand      float64 `json:"comp_land"`
	CompRealState float64 `json:"comp_real_state"`
	CompProperty  float64 `json:"comp_property"`

	// Сараар шинэчлэгдэх статистикт — сонгосон (үндсэн) урсгалын нөхөх олговор.
	CompAmount     float64    `json:"comp_amount"`
	CompAt         *time.Time `json:"comp_at"`          // анх оруулсан
	CompApprovedAt *time.Time `json:"comp_approved_at"` // бүгд батлагдсан бол сүүлийн баталсан огноо
	// Захирамжийн төсөл (идэвхтэй холбоос).
	DecisionID          *uuid.UUID       `json:"decision_id"`
	DecisionLinkedAt    *time.Time       `json:"decision_linked_at"`
	DecisionConfirmedAt *time.Time       `json:"decision_confirmed_at"`
	Payments            []ChronosPayment `json:"payments"`
}

// ChronosPayment — «Нөхөх олговор олгосон баримт»-ын шат (60/40/100) ба огноо.
type ChronosPayment struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Stage int    `json:"stage"`
}

// ChronosMonth — тухайн сарын явцын өөрчлөлтийн (төлөвийн түүхийн бичлэг) тоо (цагийн гол).
type ChronosMonth struct {
	Month string `json:"month"` // YYYY-MM
	Count int    `json:"count"`
}

// ChronosStatus — нэгж талбарын төлөвийн бүртгэл (өнгө, явцын эрэмбэ).
type ChronosStatus struct {
	ID         int    `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	SortOrder  int    `json:"sort_order"`
	IsReleased bool   `json:"is_released"`
	IsFinal    bool   `json:"is_final"`
}

// ChronosDecision — захирамжийн төсөл (сонгосон сарын байдлаар явцыг тооцох).
type ChronosDecision struct {
	ID          uuid.UUID              `json:"id"`
	CreatedAt   time.Time              `json:"created_at"`
	ConfirmedAt *time.Time             `json:"confirmed_at"`
	Events      []ChronosDecisionEvent `json:"events"`
}

// ChronosDecisionEvent — захирамжийн явц (reviewing / confirming) ба огноо.
type ChronosDecisionEvent struct {
	Type string    `json:"type"`
	Date time.Time `json:"date"`
}

type ChronosData struct {
	Statuses  []ChronosStatus   `json:"statuses"`
	Decisions []ChronosDecision `json:"decisions"`
	Parcels   []*ChronosParcel  `json:"parcels"`
	Months    []ChronosMonth    `json:"months"`
	// Truncated — дээд хязгаараас (ChronosMaxParcels) их нэгж талбар байсан.
	Truncated bool `json:"truncated"`
}

// ChronosMaxParcels — нэг удаад зурах нэгж талбарын дээд тоо (canvas 60fps).
const ChronosMaxParcels = 5000
