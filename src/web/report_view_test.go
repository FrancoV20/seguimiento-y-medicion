package web

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

// HU-09, T027: direcciones, nombres de archivo, confirmación y armado de las vistas de "Reportes".

func TestReportView(t *testing.T) {
	if got := reportPDFPath(7); got != "/reportes/sprints/7/pdf" {
		t.Errorf("reportPDFPath(7) = %q", got)
	}
	if got := reportConfirmPath(7); got != "/reportes/sprints/7/pdf?confirmar=si" {
		t.Errorf("reportConfirmPath(7) = %q", got)
	}
	if got := reportFileName(2); got != "reporte-sprint-2.pdf" {
		t.Errorf("reportFileName(2) = %q", got)
	}
	if got := reportProjectFileName(5); got != "reporte-proyecto-5.pdf" {
		t.Errorf("reportProjectFileName(5) = %q", got)
	}
	if reportProjectPath != "/reportes/proyecto/pdf" {
		t.Errorf("reportProjectPath = %q", reportProjectPath)
	}

	t.Run("reportConfirmed", func(t *testing.T) {
		for query, want := range map[string]bool{
			"?confirmar=si": true,
			"?confirmar=SI": false,
			"?confirmar=1":  false,
			"?confirmar=no": false,
			"?confirmar=":   false,
			"":              false,
		} {
			r := httptest.NewRequest("GET", "/reportes/sprints/7/pdf"+query, nil)
			if got := reportConfirmed(r); got != want {
				t.Errorf("reportConfirmed(%q) = %v, se esperaba %v", query, got, want)
			}
		}
	})

	t.Run("lista con Sprints", func(t *testing.T) {
		got := buildReportListData([]ReportSprintOption{
			{ID: 9, Number: 3, Goal: "Exportar e informar", Active: true},
			{ID: 8, Number: 2, Goal: "Gestionar perfiles y reportes"},
		})
		want := reportListData{
			Sprints: []reportSprintRow{
				{Label: "Sprint 3", Goal: "Exportar e informar", Status: ReportStatusActive, ExportURL: "/reportes/sprints/9/pdf"},
				{Label: "Sprint 2", Goal: "Gestionar perfiles y reportes", Status: ReportStatusFinished, ExportURL: "/reportes/sprints/8/pdf"},
			},
			ProjectURL: "/reportes/proyecto/pdf",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("lista = %+v, se esperaba %+v", got, want)
		}
	})

	t.Run("lista vacía", func(t *testing.T) {
		got := buildReportListData(nil)
		if len(got.Sprints) != 0 || got.NoSprints != MsgReportNoSprints || got.ProjectURL != "/reportes/proyecto/pdf" {
			t.Errorf("lista vacía = %+v", got)
		}
	})

	t.Run("advertencia de un Sprint en curso", func(t *testing.T) {
		got := sprintWarningData(ReportSprintOption{ID: 9, Number: 3, Active: true})
		want := reportWarningData{
			Message:     "El Sprint 3 sigue en curso: el reporte puede estar incompleto",
			GenerateURL: "/reportes/sprints/9/pdf?confirmar=si",
			CancelURL:   "/reportes",
		}
		if got != want {
			t.Errorf("advertencia = %+v, se esperaba %+v", got, want)
		}
	})
}
