package domain

// InUseError — тохиргооны бичлэг (хавсралтын төрөл, захирамжийн сонголт г.м.)
// өөр бүртгэлд ашиглагдаж байгаа тул устгах боломжгүй (409). Message нь
// хэрэглэгчид шууд харагдах тайлбар (юунд, хэдэн удаа ашиглагдсан).
type InUseError struct {
	Message string
}

func (e *InUseError) Error() string { return e.Message }

// SystemDocumentTypeCodes — КОД нь системийн үйлдэлд шууд ашиглагддаг
// хавсралтын төрөл (үнэлгээний тайлан/эх сурвалж/ажлын зураг, хурлын
// тэмдэглэл, олговор олгосон баримт, «Бусад»). Эдгээрийг устгах эсвэл кодыг
// солих нь тухайн үйлдлийг эвдэнэ.
var SystemDocumentTypeCodes = map[string]bool{
	"valuation_report":         true,
	"valuation_source":         true,
	"work_photo":               true,
	"meeting_minutes":          true,
	CompensationReceiptDocType: true,
	"other":                    true,
}
