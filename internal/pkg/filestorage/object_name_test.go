package filestorage

import (
	"strings"
	"testing"
)

// MinIO зарим тэмдэгтийг зөвшөөрдөггүй ("Objectname contains unsupported
// characters") тул ХАДГАЛАХ нэр цэвэршсэн, ХАРУУЛАХ нэр эх хэвээр байх ёстой.
func TestStoredFileName_SanitizesUnsupportedCharacters(t *testing.T) {
	cases := []struct {
		in       string
		wantName string // харуулах нэр (өөрчлөгдөхгүй)
		badRunes string // хадгалах нэрэнд БАЙХГҮЙ байх ёстой тэмдэгтүүд
	}{
		// filepath.Base нь замын хэсгийг (/) таслана — ХАРУУЛАХ нэр нь файлын нэр.
		{`Тайлан: 2026 "эцсийн".pdf`, `Тайлан: 2026 "эцсийн".pdf`, `:"`},
		{`акт*?<>|.pdf`, `акт*?<>|.pdf`, `*?<>|`},
		{"хүчингүй\tболгох.pdf", "хүчингүй\tболгох.pdf", "\t"},
	}
	for _, c := range cases {
		clean, stored := storedFileName(c.in, false)
		if clean != c.wantName {
			t.Fatalf("харуулах нэр өөрчлөгдсөн: %q → %q", c.wantName, clean)
		}
		if strings.ContainsAny(stored, c.badRunes) {
			t.Fatalf("хадгалах нэрэнд зөвшөөрөгдөөгүй тэмдэгт үлдсэн: %q", stored)
		}
		if strings.TrimSpace(stored) == "" {
			t.Fatalf("хадгалах нэр хоосон болсон: %q", c.in)
		}
	}
}

// unique=true үед давхардлаас сэргийлэх uuid угтвар нэмэгдэнэ.
func TestStoredFileName_UniquePrefix(t *testing.T) {
	_, a := storedFileName("акт.pdf", true)
	_, b := storedFileName("акт.pdf", true)
	if a == b {
		t.Fatal("unique нэр давхардлаа")
	}
	if !strings.HasSuffix(a, "_акт.pdf") {
		t.Fatalf("эх нэр хадгалагдаагүй: %q", a)
	}
}
