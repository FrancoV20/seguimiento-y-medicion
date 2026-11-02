package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// MEJORA-01 (diseño visual consistente): reglas del diseño común que no deben perderse al agregar
// pantallas. Las expectativas de las demás pruebas no cambian.

func readStaticCSS(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(staticFiles, "static/app.css")
	if err != nil {
		t.Fatalf("no se pudo leer app.css: %v", err)
	}
	return string(b)
}

// Escenarios 1 y 6: un único conjunto de colores, también para los gráficos del Dashboard.
func TestDesign_TokensComunesYGraficos(t *testing.T) {
	css := readStaticCSS(t)
	for _, token := range []string{"--bg:", "--surface:", "--text:", "--muted:", "--accent:", "--font:", "--chart-1:"} {
		if !strings.Contains(css, token) {
			t.Errorf("app.css debe definir el token %q en :root", token)
		}
	}
	if !regexp.MustCompile(`\.dashboard-chart rect\s*\{[^}]*fill:\s*var\(--chart-1\)`).MatchString(css) {
		t.Error("las barras del Dashboard deben usar el color del diseño (var(--chart-1))")
	}
}

// Escenario 3: sin fuentes ni recursos externos.
func TestDesign_SinRecursosExternos(t *testing.T) {
	css := readStaticCSS(t)
	for _, forbidden := range []string{"@import", "url(", "@font-face", "http://", "https://"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("app.css no debe usar %q: los estilos tienen que cargar sin internet", forbidden)
		}
	}
}

// Escenario 4: todo mensaje con color lleva también su etiqueta de texto.
func TestDesign_AlertasConEtiqueta(t *testing.T) {
	alert := regexp.MustCompile(`(?s)<div class="alert alert-[a-z]+"[^>]*>(.*?)</div>`)
	err := fs.WalkDir(templateFiles, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(templateFiles, p)
		if err != nil {
			return err
		}
		for _, m := range alert.FindAllStringSubmatch(string(b), -1) {
			if !strings.Contains(m[1], `class="alert-label"`) {
				t.Errorf("%s: alerta sin etiqueta de texto (alert-label): %s", p, m[0])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("no se pudieron recorrer las plantillas: %v", err)
	}
}

// Escenario 5: en pantallas angostas, las tablas anchas se desplazan dentro de su recuadro.
func TestDesign_PantallaAngosta(t *testing.T) {
	css := readStaticCSS(t)
	media := regexp.MustCompile(`(?s)@media \(max-width: 640px\)\s*\{(.*)\}\s*$`).FindStringSubmatch(css)
	if media == nil {
		t.Fatal("app.css debe tener la sección @media (max-width: 640px)")
	}
	if !regexp.MustCompile(`table\s*\{[^}]*overflow-x:\s*auto`).MatchString(media[1]) {
		t.Error("en pantallas angostas las tablas deben desplazarse dentro de su recuadro (overflow-x: auto)")
	}
}
