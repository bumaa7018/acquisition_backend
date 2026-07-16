package postgres

import (
	"context"
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{db: db}
}

const notificationColumns = `id, recipient_id, actor_id, actor_name, type, title, body,
	resource_type, resource_id, acquisition_id, parcel_id, is_read, created_at, read_at`

func (r *NotificationRepo) CreateBatch(ctx context.Context, ns []*domain.Notification) error {
	if len(ns) == 0 {
		return nil
	}
	// Нэг INSERT-ээр багцалж бичнэ (мэдэгдэл нэг үйлдлээс олон хүлээн авагчид үүсдэг).
	var (
		sb   strings.Builder
		args []any
	)
	sb.WriteString(`INSERT INTO notification (` + notificationColumns + `) VALUES `)
	for i, n := range ns {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i * 14
		sb.WriteString("(")
		for j := 1; j <= 14; j++ {
			if j > 1 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, "$%d", base+j)
		}
		sb.WriteString(")")
		args = append(args,
			n.ID, n.RecipientID, n.ActorID, n.ActorName, n.Type, n.Title, n.Body,
			n.ResourceType, n.ResourceID, n.AcquisitionID, n.ParcelID,
			n.IsRead, n.CreatedAt, n.ReadAt,
		)
	}
	_, err := r.db.Exec(ctx, sb.String(), args...)
	return err
}

func (r *NotificationRepo) ListByRecipient(ctx context.Context, recipientID uuid.UUID, onlyUnread bool, page, pageSize int) ([]*domain.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	where := "WHERE recipient_id = $1"
	if onlyUnread {
		where += " AND is_read = FALSE"
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM notification "+where, recipientID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		"SELECT "+notificationColumns+" FROM notification "+where+
			" ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		recipientID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []*domain.Notification
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(
			&n.ID, &n.RecipientID, &n.ActorID, &n.ActorName, &n.Type, &n.Title, &n.Body,
			&n.ResourceType, &n.ResourceID, &n.AcquisitionID, &n.ParcelID,
			&n.IsRead, &n.CreatedAt, &n.ReadAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		"SELECT count(*) FROM notification WHERE recipient_id = $1 AND is_read = FALSE",
		recipientID).Scan(&count)
	return count, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, recipientID, id uuid.UUID) error {
	res, err := r.db.Exec(ctx,
		"UPDATE notification SET is_read = TRUE, read_at = now() WHERE id = $1 AND recipient_id = $2 AND is_read = FALSE",
		id, recipientID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		// Байхгүй эсвэл өөр хэрэглэгчийнх эсвэл аль хэдийн уншсан — аль ч тохиолдолд
		// давхар мэдээлэл алдагдахгүй тул амжилттайд тооцно (idempotent).
		var exists bool
		if err := r.db.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM notification WHERE id = $1 AND recipient_id = $2)",
			id, recipientID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return domain.ErrNotFound
		}
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, recipientID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE notification SET is_read = TRUE, read_at = now() WHERE recipient_id = $1 AND is_read = FALSE",
		recipientID)
	return err
}
