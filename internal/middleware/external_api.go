package middleware

// ГАДААД системийн API-ийн танилт (API key).
//
// Яагаад JWT БИШ: эдгээр гарц нь ХҮН биш, өөр СИСТЕМД (ТЕГ, ШШГЕГ, үнийн
// мэдээний сан г.м.) зориулагдсан. Тэдэнд системийн хэрэглэгч үүсгэх нь эрх,
// роль, нууц үг сэргээх урсгалыг тэдний зүгээс шаарддаг тул тохирохгүй.
//
// Түлхүүрүүд .env-д (EXTERNAL_API_KEYS) "нэр:түлхүүр" хэлбэрээр бүртгэгдэнэ.
// Тохируулаагүй бол гарцууд БҮРЭН ХААЛТТАЙ (deny-by-default) — санамсаргүй
// нээлттэй үлдэхээс сэргийлнэ.

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ExternalAPIKey — тохируулсан НЭГ түлхүүр.
type ExternalAPIKey struct {
	// Label — логт бичигдэх системийн нэр (түлхүүр өөрөө ХЭЗЭЭ Ч логлогдохгүй).
	Label string
	Key   string
}

const (
	externalAPIKeyHeader   = "X-API-Key"
	externalClientCtxKey   = "external_client"
	externalAuthorizePrefx = "bearer "
)

// ExternalAPIKeyAuth — X-API-Key (эсвэл Authorization: Bearer <key>) толгойгоор
// гадаад системийг танина.
//
// Түлхүүрийг ТОГТМОЛ ХУГАЦААНД харьцуулна (subtle.ConstantTimeCompare) —
// урттай/эрт зөрөх зөрүүгээр түлхүүрийг тааварлах боломжийг хаана.
func ExternalAPIKeyAuth(keys []ExternalAPIKey) gin.HandlerFunc {
	// Хэшийг урьдчилан бодно: харьцуулалт нь ижил урттай болж, түлхүүрийн
	// урт ч зөрүүлэхгүй.
	type hashedKey struct {
		label string
		sum   [32]byte
	}
	hashed := make([]hashedKey, 0, len(keys))
	for _, k := range keys {
		key := strings.TrimSpace(k.Key)
		if key == "" {
			continue
		}
		label := strings.TrimSpace(k.Label)
		if label == "" {
			label = "external"
		}
		hashed = append(hashed, hashedKey{label: label, sum: sha256.Sum256([]byte(key))})
	}

	return func(c *gin.Context) {
		// Тохируулаагүй бол гарц ХААЛТТАЙ.
		if len(hashed) == 0 {
			response.Abort(c, http.StatusServiceUnavailable, i18n.ErrExternalAPIDisabled)
			return
		}

		presented := strings.TrimSpace(c.GetHeader(externalAPIKeyHeader))
		if presented == "" {
			if auth := strings.TrimSpace(c.GetHeader("Authorization")); len(auth) > len(externalAuthorizePrefx) &&
				strings.EqualFold(auth[:len(externalAuthorizePrefx)], externalAuthorizePrefx) {
				presented = strings.TrimSpace(auth[len(externalAuthorizePrefx):])
			}
		}
		if presented == "" {
			response.Error(c, domain.ErrUnauthorized)
			return
		}

		sum := sha256.Sum256([]byte(presented))
		for _, k := range hashed {
			if subtle.ConstantTimeCompare(sum[:], k.sum[:]) == 1 {
				c.Set(externalClientCtxKey, k.label)
				c.Next()
				return
			}
		}
		response.Error(c, domain.ErrUnauthorized)
	}
}

// ExternalClient — танигдсан гадаад системийн нэр (логт бичихэд).
func ExternalClient(c *gin.Context) string {
	if v, ok := c.Get(externalClientCtxKey); ok {
		if label, ok := v.(string); ok {
			return label
		}
	}
	return ""
}
