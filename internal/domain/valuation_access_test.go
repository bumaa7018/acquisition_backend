package domain

import "testing"

func TestCanWriteValuationStream(t *testing.T) {
	main, indep, other := ID(2), ID(1), ID(9)
	owners := ValuationStreamOwners{ProfessionalOrgID: &main, IndependentOrgID: &indep}
	org := func(id ID) ValuationCaller { return ValuationCaller{ValuationOrg: true, OrgID: &id} }

	cases := []struct {
		name   string
		caller ValuationCaller
		vType  string
		owners ValuationStreamOwners
		want   bool
	}{
		{"үндсэн байгууллага үндсэн урсгалд", org(main), ValuationTypeAsset, owners, true},
		{"үндсэн байгууллага хөндлөнгийн урсгалд", org(main), ValuationTypeIndependent, owners, false},
		{"үндсэн байгууллага МИКА урсгалд", org(main), ValuationTypeMika, owners, false},
		{"хөндлөнгийн байгууллага өөрийн урсгалд", org(indep), ValuationTypeIndependent, owners, true},
		{"хөндлөнгийн байгууллага үндсэн урсгалд", org(indep), ValuationTypeAsset, owners, false},
		{"холбогдоогүй байгууллага", org(other), ValuationTypeAsset, owners, false},
		{"хөндлөнгийн байгууллага тохируулаагүй нэгж талбар", org(indep), ValuationTypeIndependent, ValuationStreamOwners{ProfessionalOrgID: &main}, false},
		{"байгууллагагүй токен", ValuationCaller{ValuationOrg: true}, ValuationTypeAsset, owners, false},
		{"МИКА өөрийн урсгалд", ValuationCaller{Mika: true}, ValuationTypeMika, owners, true},
		{"МИКА үндсэн урсгалд", ValuationCaller{Mika: true}, ValuationTypeAsset, owners, false},
		{"дотоод ажилтан (админ, ахлах)", ValuationCaller{}, ValuationTypeAsset, owners, false},
		{"дотоод ажилтан МИКА урсгалд", ValuationCaller{}, ValuationTypeMika, owners, false},
	}
	for _, tc := range cases {
		if got := CanWriteValuationStream(tc.caller, tc.vType, tc.owners); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsAdminRoleName(t *testing.T) {
	for _, n := range []string{"admin", "ADMIN", " Admin ", "Админ", "АДМИН", "land_admin", "x_Админ"} {
		if !IsAdminRoleName(n) {
			t.Errorf("%q нь админы нэр гэж танигдах ёстой", n)
		}
	}
	for _, n := range []string{"administrator", "senior_specialist", "admins", "МИКА", "admin_renamed", ""} {
		if IsAdminRoleName(n) {
			t.Errorf("%q нь админы нэр биш", n)
		}
	}
}

func TestValuationOrgScopeCanRead(t *testing.T) {
	main := ValuationOrgScope{MainOrg: true}
	indep := ValuationOrgScope{IndependentParcels: map[string]bool{"P1": true}}
	cases := []struct {
		name   string
		scope  ValuationOrgScope
		parcel string
		vType  string
		want   bool
	}{
		{"үндсэн байгууллага asset бүх нэгж талбар", main, "P9", "asset", true},
		{"хоосон урсгал = asset", main, "P9", "", true},
		{"үндсэн байгууллага independent харахгүй", main, "P1", "independent", false},
		{"үндсэн байгууллага mika харахгүй", main, "P1", "mika", false},
		{"хөндлөнгийн өөрийн нэгж талбар", indep, "P1", "independent", true},
		{"хөндлөнгийн өөр нэгж талбар", indep, "P2", "independent", false},
		{"хөндлөнгийн asset харахгүй", indep, "P1", "asset", false},
		{"хөндлөнгийн хоосон урсгал (=asset) харахгүй", indep, "P1", "", false},
		{"хөндлөнгийн mika харахгүй", indep, "P1", "mika", false},
		{"танигдаагүй урсгал", main, "P1", "other", false},
	}
	for _, tc := range cases {
		if got := tc.scope.CanRead(tc.parcel, tc.vType); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
