package legacyimport

import (
	"strings"
	"testing"
)

func TestParseOwnerNames(t *testing.T) {
	cases := []struct {
		in   string
		want string // «овог/нэр» · …; хоосон бол алдаа хүлээнэ
	}{
		{"Б.Бат А.Цэцэг", "Б/Бат · А/Цэцэг"},
		{"Б Бат А.Цэцэг", "Б/Бат · А/Цэцэг"},
		{"Б\nБат А.Цэцэг", "Б/Бат · А/Цэцэг"},
		{"Б.Бат АнарынЦэцэг", "Б/Бат · Анарын/Цэцэг"},
		{"Б.Бат, А.Цэцэг", "Б/Бат · А/Цэцэг"},
		{"Б.Бат; А.Цэцэг", "Б/Бат · А/Цэцэг"},
		{"Ц.БАТ Доржийн Гомо А.Цэцэг", "Ц/БАТ · Доржийн/Гомо · А/Цэцэг"},
		{"Сүрэнхорлоогийн Саранчимэг", "Сүрэнхорлоогийн/Саранчимэг"},
		{"Батын Болд Доржийн Сүх", "Батын/Болд · Доржийн/Сүх"},
		{"Б. Бат-Эрдэнэ", "Б/Бат-Эрдэнэ"},
		{"Б. БатЭрдэнэ", "Б/БатЭрдэнэ"},
		{"Саранчимэг", "/Саранчимэг"},
		{"Тэгш хотхон ХХК, Б.Бат", "/Тэгш хотхон ХХК · Б/Бат"},
		{"", ""},
		{"Саранчимэг, Б.Бат", "/Саранчимэг · Б/Бат"}, // «,» хүн тусгаарлана
		{"Б Бат; Цэцэг", "Б/Бат · /Цэцэг"},           // «;» хүн тусгаарлана
		{"Батын Болд Доржийн Сүх, Б.Бат", "Батын/Болд · Доржийн/Сүх · Б/Бат"},
		{"Б, Бат", "/Б · /Бат"}, // «,»-ийн хоёр талд хоёр хүн
		{"Бат Дорж, Цэцэг", "Бат/Дорж · /Цэцэг"},
		{"Б Бат Цэцэг", ""}, // хосгүй үг
		{"Б А.Цэцэг", ""},   // «Б»-ийн нэр тодорхойгүй
	}
	for _, c := range cases {
		got, err := parseOwnerNames(c.in)
		parts := make([]string, len(got))
		for i, o := range got {
			parts[i] = o.lastName + "/" + o.name
		}
		s := strings.Join(parts, " · ")
		switch {
		case c.want == "" && c.in != "" && err == nil:
			t.Errorf("%q: алдаа хүлээсэн, ирсэн %q", c.in, s)
		case c.want != "" && (err != nil || s != c.want):
			t.Errorf("%q: %q хүлээсэн, ирсэн %q (%v)", c.in, c.want, s, err)
		}
	}
}

func TestOwnerIssues(t *testing.T) {
	cases := []struct {
		name, register, code string
	}{
		{"Сүрэнхорлоогийн Саранчимэг", "ХЛ56072067", ""},
		{"Батын Болд, Доржийн Сүх", "УБ12345678, УБ87654321", ""},
		{"Ц.БАТ Доржийн Гомо А.Цэцэг", "АА12345678 АА12345679 АА12345670", ""},
		{"", "", "owner_missing"},
		{"", "УБ12345678", "owner_missing"},
		{"Сүрэнхорлоогийн Саранчимэг", "", "owner_no_register"},
		{"Батын Болд, Доржийн Сүх", "УБ12345678", "owner_mismatch"},
		{"Ц.БАТ Доржийн Гомо А.Цэцэг", "АА12345678 АА12345679", "owner_mismatch"},
		{"Б Бат Цэцэг", "", "owner_mismatch"},
	}
	for _, c := range cases {
		var warns []Issue
		for _, i := range ownerIssues("4932803", &ReportRow{HolderName: c.name, HolderRegisterNo: c.register}) {
			if i.Level == "warning" {
				warns = append(warns, i)
			}
		}
		switch {
		case c.code == "" && len(warns) != 0:
			t.Errorf("%q / %q: анхааруулгагүй байх ёстой, ирсэн %+v", c.name, c.register, warns)
		case c.code != "" && (len(warns) != 1 || warns[0].Code != c.code):
			t.Errorf("%q / %q: %s хүлээсэн, ирсэн %+v", c.name, c.register, c.code, warns)
		}
	}
	// 3 өмчлөгч — регистр дарааллаараа.
	owners, err := splitOwners("Ц.БАТ Доржийн Гомо А.Цэцэг", "АА12345678 АА12345679 АА12345670")
	if err != nil || len(owners) != 3 || owners[1].lastName != "Доржийн" || owners[1].registerNo != "АА12345679" {
		t.Errorf("3 өмчлөгч хүлээсэн: %+v %v", owners, err)
	}
}
