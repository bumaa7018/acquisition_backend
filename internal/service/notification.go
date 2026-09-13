package service

import (
	"context"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/notify"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

// source нь ЗААВАЛ (domain.UserSource*): recipientID нь authdb ба appdb хоёрт
// мөргөлддөг тул эх сурвалжгүйгээр өөр хэрэглэгчийн мэдэгдэл уншигдана.
type NotificationService interface {
	List(ctx context.Context, source string, recipientID domain.ID, onlyUnread bool, page, pageSize int) ([]*domain.Notification, int64, error)
	UnreadCount(ctx context.Context, source string, recipientID domain.ID) (int64, error)
	MarkRead(ctx context.Context, source string, recipientID domain.ID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, source string, recipientID domain.ID) error
	// Subscribe — SSE стрийм суваг. Хязгаар хэтэрвэл nil суваг буцаана.
	Subscribe(source string, userID domain.ID) (<-chan *domain.Notification, func())
	// Notify — мэдэгдлийг олон хүлээн авагчид үүсгэж, DB-д хадгалаад
	// идэвхтэй SSE холболтууд руу шууд түгээнэ. Actor өөрөө хүлээн авагчдаас
	// хасагдана, давхардсан хүлээн авагч нэг л мэдэгдэл авна.
	//
	// Хүлээн авагчид нь ДОТООД ажилтнууд (domain.UserSourceAuth) — одоогийн
	// бүх дуудагч ийм. Үнэлгээний байгууллагын ажилтанд мэдэгдэл хүргэх бол
	// NotifyTo-г эх сурвалжтай нь дуудна.
	Notify(ctx context.Context, recipients []domain.ID, n domain.Notification)
	NotifyTo(ctx context.Context, source string, recipients []domain.ID, n domain.Notification)
}

type notificationService struct {
	repo repository.NotificationRepository
	hub  *notify.Hub
}

func NewNotificationService(repo repository.NotificationRepository, hub *notify.Hub) NotificationService {
	return &notificationService{repo: repo, hub: hub}
}

func (s *notificationService) List(ctx context.Context, source string, recipientID domain.ID, onlyUnread bool, page, pageSize int) ([]*domain.Notification, int64, error) {
	return s.repo.ListByRecipient(ctx, source, recipientID, onlyUnread, page, pageSize)
}

func (s *notificationService) UnreadCount(ctx context.Context, source string, recipientID domain.ID) (int64, error) {
	return s.repo.UnreadCount(ctx, source, recipientID)
}

func (s *notificationService) MarkRead(ctx context.Context, source string, recipientID domain.ID, id uuid.UUID) error {
	return s.repo.MarkRead(ctx, source, recipientID, id)
}

func (s *notificationService) MarkAllRead(ctx context.Context, source string, recipientID domain.ID) error {
	return s.repo.MarkAllRead(ctx, source, recipientID)
}

func (s *notificationService) Subscribe(source string, userID domain.ID) (<-chan *domain.Notification, func()) {
	return s.hub.Subscribe(source, userID)
}

// Notify — дотоод ажилтнууд руу (domain.UserSourceAuth).
func (s *notificationService) Notify(ctx context.Context, recipients []domain.ID, n domain.Notification) {
	s.NotifyTo(ctx, domain.UserSourceAuth, recipients, n)
}

func (s *notificationService) NotifyTo(ctx context.Context, source string, recipients []domain.ID, n domain.Notification) {
	if source == "" {
		source = domain.UserSourceAuth
	}
	n.RecipientSource = source
	// Хүлээн авагчдыг цэвэрлэх: давхардал ба actor-ийг хасна.
	seen := make(map[domain.ID]struct{}, len(recipients))
	var rows []*domain.Notification
	now := time.Now()
	for _, rid := range recipients {
		if rid == 0 {
			continue
		}
		if n.ActorID != nil && rid == *n.ActorID {
			continue
		}
		if _, dup := seen[rid]; dup {
			continue
		}
		seen[rid] = struct{}{}
		row := n // хуулбар
		row.ID = uuid.New()
		row.RecipientID = rid
		row.IsRead = false
		row.CreatedAt = now
		row.ReadAt = nil
		rows = append(rows, &row)
	}
	if len(rows) == 0 {
		return
	}

	// Мэдэгдэл нь туслах үйлдэл — үндсэн бизнес үйлдлийг унагаахгүй,
	// алдааг зөвхөн логлоно. Хүсэлтийн ctx аль хэдийн дууссан байж болох тул
	// хадгалалтад өөрийн богино timeout-той ctx ашиглана.
	go func() {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.CreateBatch(saveCtx, rows); err != nil {
			slog.Error("notification save failed", "type", n.Type, "recipients", len(rows), "error", err)
			return
		}
		for _, row := range rows {
			s.hub.Publish(row)
		}
	}()
}
