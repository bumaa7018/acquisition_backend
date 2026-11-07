package service

import (
	"testing"

	"acquisition_backend/internal/domain"
)

func TestCreateUserRequest_CheckDepartment(t *testing.T) {
	mine, other := domain.ID(7), domain.ID(9)
	cases := []struct {
		name string
		req  CreateUserRequest
		dept *domain.ID
		want error
	}{
		{"админ: хэлтэсгүй ч болно", CreateUserRequest{}, nil, nil},
		{"админ бус: хэлтэсгүй бол татгалзана", CreateUserRequest{RequireDepartment: true}, nil, domain.ErrDepartmentRequired},
		{"админ бус: хэлтэстэй", CreateUserRequest{RequireDepartment: true}, &other, nil},
		{"дарга: өөрийн хэлтэс", CreateUserRequest{RequireDepartment: true, ScopeDepartmentID: &mine}, &mine, nil},
		{"дарга: өөр хэлтэс", CreateUserRequest{RequireDepartment: true, ScopeDepartmentID: &mine}, &other, domain.ErrOutsideDepartment},
		{"дарга: хэлтэсгүй ажилтан", CreateUserRequest{ScopeDepartmentID: &mine}, nil, domain.ErrDepartmentRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.req.checkDepartment(tc.dept); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
