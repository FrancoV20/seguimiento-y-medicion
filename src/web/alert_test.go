package web

import (
	"net/url"
	"strings"
	"testing"
)

func TestMensajes_FormatoPorTipoIgualEnTodasLasPantallas(t *testing.T) {
	app := newFormsApp(t)
	client := app.client(true)
	cases := []struct{ kind, class, label, role string }{
		{"success", "alert alert-success", "Éxito:", "status"},
		{"error", "alert alert-error", "Error:", "alert"},
		{"warning", "alert alert-warning", "Advertencia:", "status"},
	}
	for _, tc := range cases {
		want := `<div class="` + tc.class + `" role="` + tc.role + `"><strong class="alert-label">` +
			tc.label + `</strong> <span class="alert-text">Mensaje de prueba</span></div>`
		for _, pagina := range []string{"pagina", "form"} {
			t.Run(tc.kind+"/"+pagina, func(t *testing.T) {
				_, body := app.get(t, client, "/prueba/alerta?tipo="+tc.kind+"&pagina="+pagina+
					"&texto="+url.QueryEscape("Mensaje de prueba"))
				if !strings.Contains(body, want) {
					t.Errorf("no se encontró el formato esperado:\n%s", want)
				}
			})
		}
	}
}