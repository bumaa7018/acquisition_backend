package legacyimport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ГУС-ын нэгж талбарын хариу 24 цаг кэшлэгдэнэ — тэр хугацаанд дахин
// асуухгүй, хугацаа дуусмагц ГУС-аас ШИНЭЭР татна (олдоогүй хариу ч мөн адил).
func TestGusCacheExpiresAfter24Hours(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	gus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.URL.Path]++
		mu.Unlock()
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if r.URL.Path == "/parcel/info/1802300001" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"parcel_id": id, "area_m2": 650}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer gus.Close()
	count := func(path string) int {
		mu.Lock()
		defer mu.Unlock()
		return calls[path]
	}

	c := NewGusClient(GusConfig{BaseURL: gus.URL, Username: "u", Password: "p", Delay: time.Millisecond})
	if c.cfg.CacheTTL != 24*time.Hour {
		t.Fatalf("анхдагч кэш 24 цаг байх ёстой: %v", c.cfg.CacheTTL)
	}
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	ctx := context.Background()
	lookup := func(id string) *GusHit {
		t.Helper()
		hit, err := c.ParcelByEitherID(ctx, id, "")
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		return hit
	}

	if hit := lookup("1802300001"); hit == nil {
		t.Fatal("олдох ёстой")
	}
	lookup("9999000001") // олдоогүй (info → ub)
	if count("/parcel/info/1802300001") != 1 || count("/parcel/info/9999000001") != 1 || count("/parcel/ub/9999000001") != 1 {
		t.Fatalf("эхний хайлт: %v", calls)
	}

	// 24 цагаас ӨМНӨ — кэшээс (олдоогүй хариу ч).
	now = now.Add(24*time.Hour - time.Minute)
	lookup("1802300001")
	lookup("9999000001")
	if count("/parcel/info/1802300001") != 1 || count("/parcel/info/9999000001") != 1 {
		t.Fatalf("24 цагаас өмнө ГУС дахин дуудагдав: %v", calls)
	}

	// 24 цаг өнгөрмөгц — ГУС-аас шинээр.
	now = now.Add(time.Minute)
	if hit := lookup("1802300001"); hit == nil {
		t.Fatal("дахин татахад олдох ёстой")
	}
	lookup("9999000001")
	if count("/parcel/info/1802300001") != 2 || count("/parcel/info/9999000001") != 2 || count("/parcel/ub/9999000001") != 2 {
		t.Fatalf("24 цагийн дараа дахин татах ёстой: %v", calls)
	}

	// Ажлын эхэнд хугацаа дууссан бичлэг санах ойгоос хасагдана.
	now = now.Add(25 * time.Hour)
	c.ClearFailures()
	c.cacheMu.Lock()
	left := len(c.cache)
	c.cacheMu.Unlock()
	if left != 0 {
		t.Fatalf("хугацаа дууссан %d бичлэг үлдэв", left)
	}
}

// Тохиргоогоор өөр хугацаа өгч болно.
func TestGusCacheTTLConfigurable(t *testing.T) {
	if c := NewGusClient(GusConfig{CacheTTL: time.Hour}); c.cfg.CacheTTL != time.Hour {
		t.Fatalf("CacheTTL: %v", c.cfg.CacheTTL)
	}
}
