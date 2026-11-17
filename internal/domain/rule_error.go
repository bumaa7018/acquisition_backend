package domain

// RuleError — бизнесийн дүрэм зөрчсөн хүсэлт (422). Message нь хэрэглэгчид
// шууд харагдах тайлбар (юу зөрчигдсөн, яах ёстой).
type RuleError struct {
	Message string
}

func (e *RuleError) Error() string { return e.Message }

// NewRuleError — тайлбартай RuleError үүсгэнэ.
func NewRuleError(msg string) error { return &RuleError{Message: msg} }
