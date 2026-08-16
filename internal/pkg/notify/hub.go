// Package notify — SSE-ээр бодит цагийн мэдэгдэл түгээх in-memory hub.
// Нэг instance-тэй байршуулалтад зориулагдсан; олон instance болвол
// Redis pub/sub зэрэг гадаад broker-оор солигдох Publisher интерфейстэй.
package notify

import (
	"sync"

	"acquisition_backend/internal/domain"
)

const (
	// Нэг хэрэглэгчийн зэрэг нээх стрийм холболтын дээд хязгаар —
	// таб бүр нэг холболт нээдэг тул үүнээс их нь нөөц шавхалтын шинж.
	maxSubsPerUser = 8
	// Хүлээн авагчийн сувгийн буфер. Дүүрсэн үед шинэ мэдэгдлийг алгасна —
	// хэрэглэгч жагсаалтыг нээхэд DB-ээс бүрэн уншигдах тул мэдээлэл алдагдахгүй.
	chanBuffer = 16
)

type Hub struct {
	mu   sync.RWMutex
	subs map[domain.ID]map[chan *domain.Notification]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[domain.ID]map[chan *domain.Notification]struct{})}
}

// Subscribe — хэрэглэгчийн стрийм суваг нээнэ. Хязгаар хэтэрсэн бол nil суваг
// буцаана (дуудагч нь холболтыг татгалзана). unsubscribe-ийг заавал дуудна.
func (h *Hub) Subscribe(userID domain.ID) (<-chan *domain.Notification, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.subs[userID]) >= maxSubsPerUser {
		return nil, func() {}
	}
	ch := make(chan *domain.Notification, chanBuffer)
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan *domain.Notification]struct{})
	}
	h.subs[userID][ch] = struct{}{}

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if set, ok := h.subs[userID]; ok {
			if _, ok := set[ch]; ok {
				delete(set, ch)
				close(ch)
			}
			if len(set) == 0 {
				delete(h.subs, userID)
			}
		}
	}
}

// Publish — мэдэгдлийг хүлээн авагчийн бүх идэвхтэй холболт руу түгээнэ.
// Блоклохгүй: удаан уншигчийн дүүрсэн сувгийг алгасна.
func (h *Hub) Publish(n *domain.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[n.RecipientID] {
		select {
		case ch <- n:
		default:
		}
	}
}
