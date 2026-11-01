package web

import "testing"

// T021 (HU-10): formato del porcentaje, advertencia de historias sin Story Points y elección del Sprint.

func TestQualityView(t *testing.T) {
	t.Run("qualityPercentText", func(t *testing.T) {
		cases := []struct{ in, want string }{
			{"75.00", "75,00 %"},
			{"0.00", "0,00 %"},
			{"100.00", "100,00 %"},
			{"66.67", "66,67 %"},
			{"No disponible", "No disponible"},
		}
		for _, c := range cases {
			if got := qualityPercentText(c.in); got != c.want {
				t.Errorf("qualityPercentText(%q) = %q; se esperaba %q", c.in, got, c.want)
			}
		}
		if MsgQualityNotAvailable != "No disponible" {
			t.Errorf("MsgQualityNotAvailable = %q; se esperaba \"No disponible\"", MsgQualityNotAvailable)
		}
	})

	t.Run("advertencia de historias sin Story Points", func(t *testing.T) {
		cases := []struct {
			n    int
			want string
		}{
			{0, ""},
			{1, "1 historia sin Story Points no se suma a los planificados"},
			{2, "2 historias sin Story Points no se suman a los planificados"},
			{5, "5 historias sin Story Points no se suman a los planificados"},
		}
		for _, c := range cases {
			if got := qualityWithoutPointsWarning(c.n); got != c.want {
				t.Errorf("qualityWithoutPointsWarning(%d) = %q; se esperaba %q", c.n, got, c.want)
			}
		}
	})

	t.Run("textos de la pantalla", func(t *testing.T) {
		texts := map[string][2]string{
			"TitleQuality":         {TitleQuality, "Calidad y avance"},
			"MsgQualityNoSprints":  {MsgQualityNoSprints, "Aún no hay Sprints finalizados"},
			"MsgQualityActiveHint": {MsgQualityActiveHint, "El Sprint %d sigue en curso; sus métricas se calculan al cerrarlo"},
		}
		for name, pair := range texts {
			if pair[0] != pair[1] {
				t.Errorf("%s = %q; se esperaba %q", name, pair[0], pair[1])
			}
		}
	})

	t.Run("selectQualitySprint", func(t *testing.T) {
		list := QualitySprintList{
			Finished:     []QualitySprintOption{{ID: 30, Number: 3}, {ID: 20, Number: 2}, {ID: 7, Number: 1}},
			ActiveNumber: 4,
		}
		latest := QualitySprintOption{ID: 30, Number: 3}
		cases := []struct {
			name string
			raw  string
			want QualitySprintOption
		}{
			{"vacío", "", latest},
			{"no numérico", "abc", latest},
			{"cero", "0", latest},
			{"negativo", "-1", latest},
			{"con ceros a la izquierda", "007", latest},
			{"con espacios", " 20", latest},
			{"mayor que el máximo", "99999999999999999999", latest},
			{"el Sprint activo (no está en la lista)", "40", latest},
			{"de otro proyecto (no está en la lista)", "90", latest},
			{"válido", "20", QualitySprintOption{ID: 20, Number: 2}},
			{"el más antiguo", "7", QualitySprintOption{ID: 7, Number: 1}},
		}
		for _, c := range cases {
			got, ok := selectQualitySprint(c.raw, list)
			if !ok || got != c.want {
				t.Errorf("%s (%q): (%+v, %t); se esperaba (%+v, true)", c.name, c.raw, got, ok, c.want)
			}
		}

		if got, ok := selectQualitySprint("20", QualitySprintList{ActiveNumber: 4}); ok {
			t.Errorf("sin Sprints finalizados debía devolver false; devolvió (%+v, true)", got)
		}
	})
}
