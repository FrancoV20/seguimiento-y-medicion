package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

func TestCloseFormCalculatesPendingStoriesSummary(t *testing.T) {
	board := BoardView{
		ID:     15,
		Number: 3,
		Label:  "Sprint 3",
		Groups: []BoardGroup{
			{
				Status: sprint.EnSprint,
				Stories: []BoardStory{
					{ID: 3, Title: "Historia En Sprint", StoryPoints: closeFormIntPointer(2), Status: sprint.EnSprint},
				},
			},
			{
				Status: sprint.InProgress,
				Stories: []BoardStory{
					{ID: 4, Title: "Historia En progreso", StoryPoints: closeFormIntPointer(8), Status: sprint.InProgress},
				},
			},
			{
				Status: sprint.Done,
				Stories: []BoardStory{
					{ID: 1, Title: "Historia Hecho 1", StoryPoints: closeFormIntPointer(3), Status: sprint.Done},
					{ID: 2, Title: "Historia Hecho 2", StoryPoints: closeFormIntPointer(5), Status: sprint.Done},
				},
			},
		},
		PlannedPoints:   18,
		CompletedPoints: 8,
	}

	got := buildCloseSummary(board)

	if len(got.Completed) != 2 {
		t.Errorf("historias completadas = %d; se esperaban 2", len(got.Completed))
	}
	if len(got.ToReturn) != 2 {
		t.Errorf("historias que serán devueltas a Pendiente = %d; se esperaban exactamente 2", len(got.ToReturn))
	}
	if got.PlannedPoints != 18 {
		t.Errorf("Story Points planificados = %d; se esperaban 18", got.PlannedPoints)
	}
	if got.CompletedPoints != 8 {
		t.Errorf("Story Points completados = %d; se esperaban 8", got.CompletedPoints)
	}
}

func TestCloseTemplateRendersActiveSummaryAndEscapedStoryTitle(t *testing.T) {
	server := NewServer(quietLogger())
	board := BoardView{
		ID:    21,
		Label: "Sprint 4",
		Groups: []BoardGroup{{
			Status: sprint.Done,
			Stories: []BoardStory{{
				ID:          7,
				Title:       "<script>alert('x')</script>",
				Status:      sprint.Done,
				StoryPoints: closeFormIntPointer(5),
			}},
		}},
		PlannedPoints:   5,
		CompletedPoints: 5,
	}
	recorder := httptest.NewRecorder()
	server.Renderer().Page(recorder, httptest.NewRequest(http.MethodGet, "/cierre", nil), "close.html", PageData{
		Title: "Cierre de Sprint",
		Data: CloseSprintViewData{
			HasActiveSprint: true,
			Summary:         buildCloseSummary(board),
		},
	})

	body := recorder.Body.String()
	for _, expected := range []string{
		"Sprint 4", "1", "5 de 5", `name="sprint_id" value="21"`,
		"Cancelar y volver al Tablero", "&lt;script&gt;",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("la plantilla de cierre debía contener %q", expected)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Errorf("el título de la historia se renderizó como HTML sin escapar")
	}
}

func TestCloseTemplateWithoutActiveSprintHasNoCloseForm(t *testing.T) {
	server := NewServer(quietLogger())
	recorder := httptest.NewRecorder()
	server.Renderer().Page(recorder, httptest.NewRequest(http.MethodGet, "/cierre", nil), "close.html", PageData{
		Title: "Cierre de Sprint",
		Data: CloseSprintViewData{
			NoActiveSprint: MsgNoSprintToClose,
		},
	})

	body := recorder.Body.String()
	if !strings.Contains(body, MsgNoSprintToClose) {
		t.Errorf("la plantilla debía mostrar %q", MsgNoSprintToClose)
	}
	if strings.Contains(body, "Cerrar Sprint") || strings.Contains(body, `name="sprint_id"`) {
		t.Errorf("la plantilla sin Sprint activo mostró un formulario de cierre")
	}
}

func closeFormIntPointer(value int) *int {
	return &value
}
