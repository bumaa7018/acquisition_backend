package legacyimport

import (
	"testing"
	"time"
)

// Архивын нэр нэгтгэх дүрэм — CLI-ийн archive-map.ts-тэй ижил жишээнүүд.
func TestClusterNames(t *testing.T) {
	resolve, clusters := clusterNames(map[string]int{
		"Сэлбэ дэд төв":                    5,
		"СЭЛБЭ ДЭД ТӨВ":                    2,
		"Сонгинохайрхан дүүргийн дэд төв":  4,
		"Сснгинохайрхан дүүргийн дэд төв":  1, // үсгийн алдаа
		"14 дүгээр хороо 2 дугаар блок":    3,
		"14 дүгээр хороо 1 дүгээр блок":    3, // ТОО өөр — ӨӨР төсөл
		"Баянхошуу дэд төвийн хүрээнд зам": 2,
	})
	if got := resolve["СЭЛБЭ ДЭД ТӨВ"]; got != "Сэлбэ дэд төв" {
		t.Fatalf("том/жижиг үсэг нэгтгэгдэнэ: %q", got)
	}
	if got := resolve["Сснгинохайрхан дүүргийн дэд төв"]; got != "Сонгинохайрхан дүүргийн дэд төв" {
		t.Fatalf("үсгийн алдаа нэгтгэгдэнэ: %q", got)
	}
	if resolve["14 дүгээр хороо 2 дугаар блок"] == resolve["14 дүгээр хороо 1 дүгээр блок"] {
		t.Fatal("тоо өөр бол ӨӨР төсөл")
	}
	if len(clusters) != 5 {
		t.Fatalf("5 төсөл хүлээсэн, %d", len(clusters))
	}
}

func TestCategoryForConstructionType(t *testing.T) {
	for _, tc := range []struct {
		label   string
		want    categoryName
		matched bool
	}{
		{"Авто зам", categoryName{General: categoryRoad}, true},
		{"автозам", categoryName{General: categoryRoad}, true},
		{"Гэр хорооллыг орон сууцжуулах төсөл хөтөлбөр", categoryName{General: categoryGer}, true},
		{"Сургууль, цэцэрлэг", categoryName{General: categoryGer, Sub: "Сургууль, цэцэрлэгийн талбай"}, true},
		{"Инженерийн шугам сүлжээ", categoryName{General: categoryGer, Sub: "Инженерийн дэд бүтэц"}, true},
		{"Сансрын төв", fallbackCategory, false},
		{"", fallbackCategory, false},
	} {
		got, matched := categoryForConstructionType(tc.label)
		if got != tc.want || matched != tc.matched {
			t.Errorf("%q: %+v %v, хүлээсэн %+v %v", tc.label, got, matched, tc.want, tc.matched)
		}
	}
}

func TestDuplicateAndSyntheticParcelID(t *testing.T) {
	if got := duplicateParcelID("1802301324", 2, "f|s|1"); got != "D21802301324" {
		t.Fatalf("D2 угтвар: %q", got)
	}
	// 12 тэмдэгтэд багтахгүй бол тогтвортой ERR… дугаар.
	long := duplicateParcelID("180230132400", 3, "f|s|9")
	if long != syntheticParcelID("f|s|9") || len(long) != 12 {
		t.Fatalf("урт дугаар → ERR: %q", long)
	}
	if syntheticParcelID("a|b|1") != syntheticParcelID("a|b|1") || syntheticParcelID("a|b|1") == syntheticParcelID("a|b|2") {
		t.Fatal("ERR дугаар мөрийн түлхүүрээс тогтвортой")
	}
}

func TestGusDefaultDelay(t *testing.T) {
	if c := NewGusClient(GusConfig{}); c.cfg.Delay != 300*time.Millisecond {
		t.Fatalf("ГУС-ын анхдагч завсарлага 300ms байх ёстой, %v", c.cfg.Delay)
	}
	if c := NewGusClient(GusConfig{Delay: time.Second}); c.cfg.Delay != time.Second {
		t.Fatalf("тохируулсан завсарлага хэвээр: %v", c.cfg.Delay)
	}
}
