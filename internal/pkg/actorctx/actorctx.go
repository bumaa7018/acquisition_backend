// Package actorctx — нэвтэрсэн хэрэглэгчийн (actor) мэдээллийг request
// context-оор service давхаргад дамжуулна. Gin-ий context key-үүд
// c.Request.Context()-д ордоггүй тул мэдэгдэл зэрэг туслах үйлдэлд
// хэн үйлдэл хийснийг service түвшинд эндээс уншина.
package actorctx

import (
	"context"

	"github.com/google/uuid"
)

type Actor struct {
	ID        uuid.UUID
	Name      string
	Position  string
	Roles     []string
	IP        string
	UserAgent string
}

type ctxKey struct{}

func With(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

func From(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(ctxKey{}).(Actor)
	return a, ok && a.ID != uuid.Nil
}
