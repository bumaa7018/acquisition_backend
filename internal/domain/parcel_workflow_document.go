package domain

import "strings"

// RequiredDocument — ажлын урсгалын шилжилтэд ЗААВАЛ байх хавсралтын төрөл.
type RequiredDocument struct {
	DocumentTypeID   int    `json:"document_type_id"`
	DocumentTypeCode string `json:"document_type_code"`
	DocumentTypeName string `json:"document_type_name"`
}

// ParcelStatusRequirement — тухайн нэгж талбарт шаардлагатай хавсралт орсон эсэх.
type ParcelStatusRequirement struct {
	DocumentTypeID   int    `json:"document_type_id"`
	DocumentTypeCode string `json:"document_type_code"`
	DocumentTypeName string `json:"document_type_name"`
	Present          bool   `json:"present"`
	// DocumentName, DocumentURL — орсон (хамгийн сүүлийн) баримт; байхгүй бол хоосон.
	DocumentName string `json:"document_name"`
	DocumentURL  string `json:"document_url"`
}

// RequiredDocumentsMissingError — явцад шилжихэд шаардлагатай хавсралт дутуу (422).
type RequiredDocumentsMissingError struct {
	Names []string
}

func (e *RequiredDocumentsMissingError) Error() string {
	return "Дараах хавсралтыг оруулсны дараа шилжүүлнэ үү: " + strings.Join(e.Names, ", ")
}
