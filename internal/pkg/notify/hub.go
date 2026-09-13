// Package notify — SSE-ээр бодит цагийн мэдэгдэл түгээх in-memory hub.
// Нэг instance-тэй байршуулалтад зориулагдсан; олон instance болвол
// Redis pub/sub зэрэг гадаад broker-оор солигдох Publisher интерфейстэй.
package notify

import (
	"strconv"
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
	mu sync.RWMutex
	// Түлхүүр нь "<эх сурвалж>:<user_id>" — зөвхөн user_id-гаар түлхүүрлэвэл
	// authdb ба appdb-ийн ижил дугаартай хоёр хэрэглэгч нэг суваг хуваалцаж,
	// нөгөөгийнхөө мэдэгдлийг харна (domain.UserSource*-ыг үз).
	subs map[string]map[chan *domain.Notification]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan *domain.Notification]struct{})}
}

// subKey — захиалгын түлхүүр. Хоосон эх сурвалжийг "auth" гэж үзнэ.
func subKey(source string, userID domain.ID) string {
	if source == "" {
		source = domain.UserSourceAuth
	}
	return source + ":" + strconv.Itoa(int(userID))
}

// Subscribe — хэрэглэгчийн стрийм суваг нээнэ. Хязгаар хэтэрсэн бол nil суваг
// буцаана (дуудагч нь холболтыг татгалзана). unsubscribe-ийг заавал дуудна.
func (h *Hub) Subscribe(source string, userID domain.ID) (<-chan *domain.Notification, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := subKey(source, userID)

	if len(h.subs[key]) >= maxSubsPerUser {
		return nil, func() {}
	}
	ch := make(chan *domain.Notification, chanBuffer)
	if h.subs[key] == nil {
		h.subs[key] = make(map[chan *domain.Notification]struct{})
	}
	h.subs[key][ch] = struct{}{}

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if set, ok := h.subs[key]; ok {
			if _, ok := set[ch]; ok {
				delete(set, ch)
				close(ch)
			}
			if len(set) == 0 {
				delete(h.subs, key)
			}
		}
	}
}

// Publish — мэдэгдлийг хүлээн авагчийн бүх идэвхтэй холболт руу түгээнэ.
// Блоклохгүй: удаан уншигчийн дүүрсэн сувгийг алгасна.
func (h *Hub) Publish(n *domain.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[subKey(n.RecipientSource, n.RecipientID)] {
		select {
		case ch <- n:
		default:
		}
	}
}
