package role

import (
	"errors"
	"testing"
)

// HU-28, T005: regla pura de los roles del proyecto (escenario 11).

func TestParseRole(t *testing.T) {
	valid := []struct {
		in   string
		want Role
	}{
		{"Scrum Master", RoleScrumMaster},
		{"  Scrum Master ", RoleScrumMaster},
		{"Integrante", RoleMember},
		{"\tIntegrante\n", RoleMember},
	}
	for _, tc := range valid {
		got, err := ParseRole(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("ParseRole(%q) = %q, %v; se esperaba %q sin error", tc.in, got, err, tc.want)
		}
	}

	for _, in := range []string{"scrum master", "SM", "Product Owner", "", "   ", "integrante", "Scrum  Master"} {
		got, err := ParseRole(in)
		if !errors.Is(err, ErrInvalidRole) {
			t.Errorf("ParseRole(%q) error = %v; se esperaba ErrInvalidRole", in, err)
		}
		if got != "" {
			t.Errorf("ParseRole(%q) = %q; con error se esperaba el rol vacío", in, got)
		}
	}
}

func TestCanManage(t *testing.T) {
	cases := []struct {
		in   Role
		want bool
	}{
		{RoleScrumMaster, true},
		{RoleMember, false},
		{"", false},
		{"Product Owner", false},
	}
	for _, tc := range cases {
		if got := CanManage(tc.in); got != tc.want {
			t.Errorf("CanManage(%q) = %v; se esperaba %v", tc.in, got, tc.want)
		}
	}
	if RoleScrumMaster != "Scrum Master" || RoleMember != "Integrante" {
		t.Errorf("textos de los roles = %q y %q; se esperaba \"Scrum Master\" e \"Integrante\"", RoleScrumMaster, RoleMember)
	}
}
