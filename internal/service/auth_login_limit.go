package service

import (
	"strings"
	"sync"
	"time"
)

// Нэвтрэлтийн brute-force хамгаалалт — USERNAME тутам амжилтгүй оролдлогыг
// тоолно. ЯАГААД IP биш USERNAME: browser нь Next-ийн rewrite proxy-гоор
// backend-т хүрдэг тул backend бүх хэрэглэгчийг Next container-ийн НЭГ IP гэж
// хардаг. Иймд IP-д суурилсан хязгаар (эсвэл X-Forwarded-For) найдваргүй —
// XFF-г солиод тойрч болно (аудитаар батлагдсан). Username-д суурилах нь
// топологиос хамааралгүй бөгөөд нэг дансыг чиглэсэн нууц үг таах халдлагыг
// шууд хаана.
const (
	loginMaxFailures = 10              // цонхонд зөвшөөрөх амжилтгүй оролдлого
	loginWindow      = 5 * time.Minute // цонхны урт
	loginGCThreshold = 10000           // map энэ хэмжээнээс өсвөл хуучныг цэвэрлэнэ
)

type loginAttempt struct {
	failures    int
	windowStart time.Time
}

// loginLimiter — process-ийн санах ой дахь энгийн тоолуур. Дахин асахад
// тэглэгдэнэ (DoS/brute-force сааруулах зорилготой, нягт бүртгэл биш).
type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
	now      func() time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{attempts: map[string]*loginAttempt{}, now: time.Now}
}

func normalizeLoginKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// blocked — оролдлого хийхээс ӨМНӨ дуудна: цонхны хязгаарт хүрсэн үү.
func (l *loginLimiter) blocked(username string) bool {
	key := normalizeLoginKey(username)
	if key == "" {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.attempts[key]
	if a == nil {
		return false
	}
	if l.now().Sub(a.windowStart) > loginWindow {
		delete(l.attempts, key)
		return false
	}
	return a.failures >= loginMaxFailures
}

// recordFailure — амжилтгүй нэвтрэлт (буруу нууц үг эсвэл олдоогүй хэрэглэгч).
func (l *loginLimiter) recordFailure(username string) {
	key := normalizeLoginKey(username)
	if key == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	a := l.attempts[key]
	if a == nil || now.Sub(a.windowStart) > loginWindow {
		l.attempts[key] = &loginAttempt{failures: 1, windowStart: now}
		if len(l.attempts) > loginGCThreshold {
			for k, v := range l.attempts {
				if now.Sub(v.windowStart) > loginWindow {
					delete(l.attempts, k)
				}
			}
		}
		return
	}
	a.failures++
}

// reset — амжилттай нэвтэрсний дараа тоолуурыг цэвэрлэнэ.
func (l *loginLimiter) reset(username string) {
	key := normalizeLoginKey(username)
	if key == "" {
		return
	}
	l.mu.Lock()
	delete(l.attempts, key)
	l.mu.Unlock()
}
