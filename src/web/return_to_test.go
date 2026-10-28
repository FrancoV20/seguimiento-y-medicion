package web

import "testing"

func TestSanitizeReturnTo(t *testing.T) {
	sections := []Section{
		{Name: "Proyectos", Path: "/proyectos"},
		{Name: "Backlog", Path: "/backlog"},
		{Name: "Sprints", Path: "/sprints"},
	}
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"vacio", "", "/"},
		{"inicio", "/", "/"},
		{"seccion registrada", "/backlog", "/backlog"},
		{"subruta de seccion registrada", "/backlog/12", "/backlog/12"},
		{"con consulta", "/backlog?estado=Pendiente", "/backlog?estado=Pendiente"},
		{"con fragmento", "/sprints#activo", "/sprints#activo"},
		{"doble barra inicial", "//otro.com", "/"},
		{"doble barra interna", "/backlog//x", "/"},
		{"url absoluta https", "https://otro.com", "/"},
		{"esquema sin dobles barras", "http:/otro.com", "/"},
		{"javascript", "javascript:alert(1)", "/"},
		{"contrabarra inicial", `\\otro.com`, "/"},
		{"barra y contrabarra", `/\otro.com`, "/"},
		{"salto de linea", "/backlog\r\nSet-Cookie: x=1", "/"},
		{"caracter nulo", "/backlog\x00", "/"},
		{"con espacio", "/backlog x", "/"},
		{"espacio inicial", " /backlog", "/"},
		{"sin barra inicial", "backlog", "/"},
		{"seccion no registrada", "/otra", "/"},
		{"prefijo parecido", "/backlogx", "/"},
		{"pantalla de seleccion", "/contexto/proyecto", "/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeReturnTo(tc.raw, sections); got != tc.want {
				t.Errorf("SanitizeReturnTo(%q) = %q, se esperaba %q", tc.raw, got, tc.want)
			}
		})
	}
}
