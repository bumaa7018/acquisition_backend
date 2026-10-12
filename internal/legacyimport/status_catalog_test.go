package legacyimport

import "testing"

// SHP-ийн төлөвийн код системийн кодтой тулгагдана; таараагүй / кодгүй нь
// эхний төлөвт орно, таараагүйг (unmatched) кодгүйгээс ялгана.
func TestStatusCatalogMapCode(t *testing.T) {
	c := &statusCatalog{
		first: statusEntry{id: 101, name: "Нөлөөлөлд өртсөн"},
		byCode: map[string]statusEntry{
			"101": {id: 101, name: "Нөлөөлөлд өртсөн"},
			"109": {id: 109, name: "Чөлөөлсөн"},
			"110": {id: 110, name: "Админы нэмсэн"},
		},
	}
	code := func(v int) *int { return &v }
	cases := []struct {
		name string
		in   *int
		want statusMapping
	}{
		{"системийн код", code(109), statusMapping{status: 109, mapped: true, label: "Чөлөөлсөн"}},
		{"админы нэмсэн код", code(110), statusMapping{status: 110, mapped: true, label: "Админы нэмсэн"}},
		{"хуучин код таарахгүй", code(4), statusMapping{status: 101, unmatched: true, label: "Нөлөөлөлд өртсөн"}},
		{"кодгүй", nil, statusMapping{status: 101, label: "Нөлөөлөлд өртсөн"}},
	}
	for _, tc := range cases {
		if got := c.mapCode(tc.in); got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
