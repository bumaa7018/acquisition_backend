package domain

import "testing"

// Цэсний код өөрчлөгдсөн ч эрхийн түлхүүр ХЭВЭЭР байх ёстой — энэ бол
// A0NNNNN дугаарлалт нэвтрүүлэхэд эвдрэхээс сэргийлэх гол баталгаа.
func TestMenuPermissionKeyStaysResourceScoped(t *testing.T) {
	cases := map[string]string{
		"A010101": "land:update",
		"A010102": "compensation:read",
		"A010103": "decision:create",
		"A010104": "users:delete",
		"A010107": "hr:structure:create",
		"A010109": "audit:read",
	}
	actions := map[string]string{
		"A010101": "update",
		"A010102": "read",
		"A010103": "create",
		"A010104": "delete",
		"A010107": "structure:create",
		"A010109": "read",
	}
	for code, want := range cases {
		if got := MenuPermissionKey(code, actions[code]); got != want {
			t.Errorf("MenuPermissionKey(%q, %q) = %q, хүссэн %q", code, actions[code], got, want)
		}
	}
}

func TestMenuCodeResourceRoundTrip(t *testing.T) {
	for code, resource := range menuCodeToResource {
		if got := MenuResource(code); got != resource {
			t.Errorf("MenuResource(%q) = %q, хүссэн %q", code, got, resource)
		}
		if got := MenuCodeForResource(resource); got != code {
			t.Errorf("MenuCodeForResource(%q) = %q, хүссэн %q", resource, got, code)
		}
	}
}

// Танихгүй код — өөр системийн цэс — саадгүй дамжина.
func TestUnknownMenuCodePassesThrough(t *testing.T) {
	if got := MenuResource("B990123"); got != "B990123" {
		t.Errorf("MenuResource(танихгүй) = %q, кодоо буцаах ёстой", got)
	}
	if got := MenuCodeForResource("weather"); got != "weather" {
		t.Errorf("MenuCodeForResource(танихгүй) = %q, нөөцөө буцаах ёстой", got)
	}
}

// router.go-д ашиглагддаг нөөц бүр кодтой байх ёстой; эс тэгвээс seed нь
// тэр эрхийг ямар ч цэсэнд холбож чадахгүй бөгөөд эрх чимээгүй алга болно.
func TestEveryRoutedResourceHasMenuCode(t *testing.T) {
	routed := []string{
		"land", "compensation", "decision",
		"users", "roles", "permissions", "hr", "admin", "audit",
	}
	for _, resource := range routed {
		if _, ok := resourceToMenuCode[resource]; !ok {
			t.Errorf("%q нөөцөд цэсний код алга", resource)
		}
	}
}
