// Package actorctx — нэвтэрсэн хэрэглэгчийн (actor) мэдээллийг request
// context-оор service давхаргад дамжуулна. Gin-ий context key-үүд
// c.Request.Context()-д ордоггүй тул мэдэгдэл зэрэг туслах үйлдэлд
// хэн үйлдэл хийснийг service түвшинд эндээс уншина.
package actorctx

import (
	"acquisition_backend/internal/domain"

	"context"
)

type Actor struct {
	ID domain.ID
	// Source — actor-ийн бүртгэл аль санд байгаа (domain.UserSource*).
	// ID нь authdb ба appdb хоёрт МӨРГӨЛДӨХ int32 тул actor-ийг өөр хүнтэй
	// (жишээ нь мэдэгдлээс "өөрийгөө хасах" шалгалтад) андуурахгүйн тулд
	// ЗӨВХӨН ID-тай ХОСООР нь хэрэглэнэ. Хоосон бол UserSourceAuth.
	Source    string
	Name      string
	Position  string
	Roles     []string
	IP        string
	UserAgent string
}

// SourceOrAuth — хоосон эх сурвалжийг дотоод ажилтан гэж үзнэ.
func (a Actor) SourceOrAuth() string {
	if a.Source == "" {
		return domain.UserSourceAuth
	}
	return a.Source
}

type ctxKey struct{}

func With(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

func From(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(ctxKey{}).(Actor)
	return a, ok && a.ID != 0
}
