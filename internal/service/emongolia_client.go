package service

// E-Mongolia-ийн мэдэгдлийн гарц (төлөвлөгөөний 2.8.5).
//
// ТҮР ЗУУРЫН МОК: гадаад системийн интеграцийн заавар/хаяг гарааг хүлээж
// байгаа тул одоогоор мок клиент ажиллана. Мок нь илгээлтийг ЛОГЛОЖ, лавлагааны
// дугаар (MOCK-…) буцаах ба түүх дээр `status = "mocked"` гэж бүртгэгдэнэ —
// иймд "иргэнд хүрсэн" гэж эндүүрэх боломжгүй.
//
// Бодит холболт нэмэгдэхэд ЗӨВХӨН энэ файлын httpEMongoliaClient-ийг
// хэрэгжүүлж, EMONGOLIA_BASE_URL-ийг тохируулна — service/handler хөндөгдөхгүй.

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// EMongoliaNoticeRequest — гадаад систем рүү явуулах мэдэгдэл.
type EMongoliaNoticeRequest struct {
	// RegisterNo — хүлээн авагч иргэний регистрийн дугаар (E-Mongolia-ийн танигч)
	RegisterNo    string
	RecipientName string
	Subject       string
	Message       string
	// ParcelCode — нэгж талбарын дугаар (лавлагаанд)
	ParcelCode string
	FileName   string
	FileData   []byte
}

// EMongoliaNoticeResult — гадаад системийн хариу.
type EMongoliaNoticeResult struct {
	// ExternalRef — гадаад системийн лавлагааны дугаар.
	ExternalRef string
	// Mocked — МОК хэрэгжүүлэлт (бодит систем рүү явуулаагүй).
	Mocked bool
}

type EMongoliaClient interface {
	SendNotice(ctx context.Context, req EMongoliaNoticeRequest) (*EMongoliaNoticeResult, error)
}

// mockEMongoliaClient — гадаад холболтгүй үеийн ТҮР хэрэгжүүлэлт.
type mockEMongoliaClient struct{}

// NewMockEMongoliaClient — мок клиент (EMONGOLIA_BASE_URL тохируулаагүй үед).
func NewMockEMongoliaClient() EMongoliaClient { return mockEMongoliaClient{} }

func (mockEMongoliaClient) SendNotice(
	_ context.Context, req EMongoliaNoticeRequest,
) (*EMongoliaNoticeResult, error) {
	ref := "MOCK-" + strings.ToUpper(uuid.NewString()[:8])
	// Мок ч гэсэн бүх илгээлт ЛОГЛОГДОНО: дараа нь бодит холболт ороход
	// ямар мэдэгдэл ямар иргэн рүү явахыг тулгаж шалгах боломжтой.
	slog.Warn("E-Mongolia МОК: мэдэгдэл гадаад систем рүү илгээгдээгүй",
		"register_no", req.RegisterNo,
		"parcel_code", req.ParcelCode,
		"file_name", req.FileName,
		"file_bytes", len(req.FileData),
		"external_ref", ref,
	)
	return &EMongoliaNoticeResult{ExternalRef: ref, Mocked: true}, nil
}
