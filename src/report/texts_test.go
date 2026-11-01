package report

import (
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// HU-09, T011: textos y formatos del reporte de un Sprint (data-model.md, texts.go).

func TestReportTexts(t *testing.T) {
	check := func(t *testing.T, name, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, se esperaba %q", name, got, want)
		}
	}

	t.Run("percentText", func(t *testing.T) {
		check(t, `percentText("75.00")`, percentText("75.00"), "75,00 %")
		check(t, `percentText("0.00")`, percentText("0.00"), "0,00 %")
		check(t, `percentText("No disponible")`, percentText("No disponible"), "No disponible")
	})

	t.Run("deviationText", func(t *testing.T) {
		check(t, `deviationText("6.25")`, deviationText("6.25"), "+6,25 %")
		check(t, `deviationText("-8.50")`, deviationText("-8.50"), "-8,50 %")
		check(t, `deviationText("0.00")`, deviationText("0.00"), "0,00 %")
		check(t, `deviationText("No disponible")`, deviationText("No disponible"), "No disponible")
	})

	t.Run("hoursText", func(t *testing.T) {
		check(t, "hoursText(1700)", hoursText(effort.Hours(1700)), "17")
		check(t, "hoursText(250)", hoursText(effort.Hours(250)), "2,5")
		check(t, "hoursText(25)", hoursText(effort.Hours(25)), "0,25")
		check(t, "hoursText(0)", hoursText(effort.Hours(0)), "0")
	})

	t.Run("dateText", func(t *testing.T) {
		check(t, "dateText", dateText(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)), "01/09/2026")
		check(t, "dateTimeText", dateTimeText(time.Date(2026, 10, 28, 10, 30, 0, 0, time.UTC)), "28/10/2026 10:30")
	})

	t.Run("storyStatusText", func(t *testing.T) {
		check(t, `("Hecho", true)`, storyStatusText("Hecho", true), "Hecho")
		check(t, `("En progreso", true)`, storyStatusText("En progreso", true), "En progreso – volvió al Backlog")
		check(t, `("En Sprint", true)`, storyStatusText("En Sprint", true), "En Sprint – volvió al Backlog")
		check(t, `("En progreso", false)`, storyStatusText("En progreso", false), "En progreso")
	})

	t.Run("withoutHoursText", func(t *testing.T) {
		check(t, "withoutHoursText(0)", withoutHoursText(0), "")
		check(t, "withoutHoursText(1)", withoutHoursText(1), "1 historia sin horas estimadas")
		check(t, "withoutHoursText(2)", withoutHoursText(2), "2 historias sin horas estimadas")
	})

	t.Run("totales de horas", func(t *testing.T) {
		check(t, "estimatedTotalText(2800, 1)", estimatedTotalText(2800, 1), "28 horas estimadas en total (1 historia sin horas estimadas)")
		check(t, "estimatedTotalText(100, 0)", estimatedTotalText(100, 0), "1 hora estimada en total")
		check(t, "estimatedTotalText(0, 2)", estimatedTotalText(0, 2), "0 horas estimadas en total (2 historias sin horas estimadas)")
		check(t, "estimatedTotalText(250, 0)", estimatedTotalText(250, 0), "2,5 horas estimadas en total")
		check(t, "registeredTotalText(2400)", registeredTotalText(2400), "24 horas registradas en total")
		check(t, "registeredTotalText(100)", registeredTotalText(100), "1 hora registrada en total")
		check(t, "registeredTotalText(0)", registeredTotalText(0), "0 horas registradas en total")
		check(t, "memberHoursText", memberHoursText(MemberHours{Name: "Ana", Hours: 1700}), "Ana: 17 h")
	})

	t.Run("defectsSummaryText", func(t *testing.T) {
		check(t, "(2, 1)", defectsSummaryText(2, 1), "2 defectos detectados y 1 resuelto en este Sprint")
		check(t, "(1, 0)", defectsSummaryText(1, 0), "1 defecto detectado y 0 resueltos en este Sprint")
		check(t, "(0, 1)", defectsSummaryText(0, 1), "0 defectos detectados y 1 resuelto en este Sprint")
		check(t, "(50, 20)", defectsSummaryText(50, 20), "50 defectos detectados y 20 resueltos en este Sprint")
		check(t, "(0, 0)", defectsSummaryText(0, 0), "No se registraron defectos en este Sprint")
	})

	t.Run("defectsSoFarText", func(t *testing.T) {
		check(t, "(0)", defectsSoFarText(0), "No se registraron defectos en este Sprint")
		check(t, "(1)", defectsSoFarText(1), "1 defecto detectado hasta el momento")
		check(t, "(3)", defectsSoFarText(3), "3 defectos detectados hasta el momento")
	})
}
