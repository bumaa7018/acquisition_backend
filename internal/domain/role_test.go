package domain

import "testing"

// Мэргэжлийн байгууллагын роль API-гаар ОЛГОГДОХГҮЙ байх ёстой — тэр роль нь
// хэрэглэгчийн бүтцээс тусдаа бүртгэлд хамаарна. Бусад роль хэвийн олгогдоно.
func TestRoleIsAPIGrantable(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{
			name: "мэргэжлийн байгууллага — хориотой",
			role: Role{Name: "compensation_professional_org", Resource: "compensation"},
			want: false,
		},
		{
			name: "resource-гүй бичигдсэн байсан ч түвшингээр таних",
			role: Role{Name: "professional_org"},
			want: false,
		},
		{"админ", Role{Name: "admin", Resource: "system"}, true},
		{"ахлах мэргэжилтэн", Role{Name: "senior_specialist", Resource: "system"}, true},
		{"МИКА", Role{Name: "compensation_mika", Resource: "compensation"}, true},
		{"санхүү", Role{Name: "compensation_finance_specialist", Resource: "compensation"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsAPIGrantable(); got != tt.want {
				t.Errorf("IsAPIGrantable() = %v, want %v (role=%q)", got, tt.want, tt.role.Name)
			}
		})
	}
}
