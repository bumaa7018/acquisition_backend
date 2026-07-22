package service

// backgroundWriteSem нь audit/client_request зэрэг "fire-and-forget" (лог)
// DB бичилтийн зэрэгцээ гүйцэтгэлийг хязгаарлана. Үүнгүй бол урсгал огцом
// өсөхөд (жишээ нь секундэд мянга мянган хүсэлт) адил тооны goroutine зэрэг
// гарч, бодит хүсэлт үйлчлэх асуулгатай ижил pgx connection pool-ийг
// булаацалдаж, аль алиныг нь удаашруулна.
var backgroundWriteSem = make(chan struct{}, 20)

// tryAcquireBackgroundSlot ээлж дүүрээгүй бол release функц + true буцаана.
// Дүүрсэн үед false буцаана — дуудагч тал бичихээ алгасаж, warning-оор л
// тэмдэглэнэ (хэзээ ч блоклохгүй, бодит хүсэлтийг удаашруулахгүй).
func tryAcquireBackgroundSlot() (release func(), ok bool) {
	select {
	case backgroundWriteSem <- struct{}{}:
		return func() { <-backgroundWriteSem }, true
	default:
		return nil, false
	}
}
