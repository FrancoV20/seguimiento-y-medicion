package web

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

func TestParseBacklogFormDefaultsActionAndFiltersAcceptanceCriteria(t *testing.T) {
	request := httptest.NewRequest("POST", "/backlog", strings.NewReader(url.Values{
		BacklogFieldTitle:       {"Historia de prueba"},
		BacklogFieldDescription: {"Descripción de prueba"},
		BacklogFieldPriority:    {"Media"},
		BacklogFieldCriterion:   {" Criterio uno ", "", "   ", "Criterio dos"},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := parseBacklogForm(request)

	if got.Action != "guardar" {
		t.Errorf("action = %q, want default %q", got.Action, "guardar")
	}
	if got.Form == nil {
		t.Fatal("Form = nil, want submitted values preserved")
	}
	if got.Form.Values[BacklogFieldTitle] != "Historia de prueba" {
		t.Errorf("title value = %q, want submitted value", got.Form.Values[BacklogFieldTitle])
	}
	if !reflect.DeepEqual(got.Criteria, []string{" Criterio uno ", "", "   ", "Criterio dos"}) {
		t.Errorf("criteria fields = %#v, want all submitted values preserved", got.Criteria)
	}
	if !reflect.DeepEqual(got.AcceptanceCriteria, []string{"Criterio uno", "Criterio dos"}) {
		t.Errorf("acceptance criteria = %#v, want trimmed nonblank criteria", got.AcceptanceCriteria)
	}
}

func TestApplyBacklogFormActionAddsAndRemovesCriteriaWithoutSaving(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		criteria []string
		want     []string
		wantSave bool
	}{
		{
			name:     "add criterion",
			action:   "agregar_criterio",
			criteria: []string{"primero"},
			want:     []string{"primero", ""},
		},
		{
			name:     "remove selected criterion",
			action:   "quitar_criterio:1",
			criteria: []string{"primero", "segundo", "tercero"},
			want:     []string{"primero", "tercero"},
		},
		{
			name:     "out of range index leaves criteria unchanged",
			action:   "quitar_criterio:4",
			criteria: []string{"primero", "segundo"},
			want:     []string{"primero", "segundo"},
		},
		{
			name:     "non numeric index leaves criteria unchanged",
			action:   "quitar_criterio:abc",
			criteria: []string{"primero", "segundo"},
			want:     []string{"primero", "segundo"},
		},
		{
			name:     "cannot remove last criterion",
			action:   "quitar_criterio:0",
			criteria: []string{"conservar"},
			want:     []string{"conservar"},
		},
		{
			name:     "unknown action defaults to save",
			action:   "accion-desconocida",
			criteria: []string{"criterio"},
			want:     []string{"criterio"},
			wantSave: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := &backlogForm{Action: tt.action, Criteria: append([]string(nil), tt.criteria...)}

			shouldSave := form.applyAction()

			if shouldSave != tt.wantSave {
				t.Errorf("applyAction() shouldSave = %t, want %t", shouldSave, tt.wantSave)
			}
			if !reflect.DeepEqual(form.Criteria, tt.want) {
				t.Errorf("criteria after action = %#v, want %#v", form.Criteria, tt.want)
			}
		})
	}
}

func TestParseBacklogFormOptionalStoryPointsAndEstimatedHours(t *testing.T) {
	tests := []struct {
		name         string
		points       string
		hours        string
		wantPoints   *int
		wantHours    *float64
		wantHoursErr bool
	}{
		{name: "both estimates empty", points: "", hours: ""},
		{name: "hours with comma decimal", points: "5", hours: "1,5", wantPoints: backlogFormIntPointer(5), wantHours: backlogFormFloatPointer(1.5)},
		{name: "hours with dot decimal", points: "3", hours: "1.5", wantPoints: backlogFormIntPointer(3), wantHours: backlogFormFloatPointer(1.5)},
		{name: "non numeric hours remain a field error", points: "3", hours: "muchas", wantPoints: backlogFormIntPointer(3), wantHoursErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/backlog", strings.NewReader(url.Values{
				BacklogFieldStoryPoints:    {tt.points},
				BacklogFieldEstimatedHours: {tt.hours},
				BacklogFieldCriterion:      {"criterio"},
			}.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			got := parseBacklogForm(request)

			if !equalOptionalInt(got.StoryPoints, tt.wantPoints) {
				t.Errorf("StoryPoints = %v, want %v", got.StoryPoints, tt.wantPoints)
			}
			if !equalOptionalFloat(got.EstimatedHours, tt.wantHours) {
				t.Errorf("EstimatedHours = %v, want %v", got.EstimatedHours, tt.wantHours)
			}
			if got.Form == nil {
				t.Fatal("Form = nil, want form values retained")
			}
			if got.Form.Values[BacklogFieldEstimatedHours] != tt.hours {
				t.Errorf("hours field value = %q, want original input %q",
					got.Form.Values[BacklogFieldEstimatedHours], tt.hours)
			}
			_, hasHoursError := got.Form.Errors[BacklogFieldEstimatedHours]
			if hasHoursError != tt.wantHoursErr {
				t.Errorf("hours field error present = %t, want %t", hasHoursError, tt.wantHoursErr)
			}
		})
	}
}

func TestBacklogFormOptionsMatchHU02Validation(t *testing.T) {
	wantPriorities := []string{
		string(backlog.PriorityHigh),
		string(backlog.PriorityMedium),
		string(backlog.PriorityLow),
	}
	if !reflect.DeepEqual(backlogPriorities(), wantPriorities) {
		t.Fatalf("priorities = %#v, want %#v", backlogPriorities(), wantPriorities)
	}
	wantStoryPoints := []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}
	if !reflect.DeepEqual(backlogStoryPointValues(), wantStoryPoints) {
		t.Fatalf("Story Points = %#v, want %#v", backlogStoryPointValues(), wantStoryPoints)
	}

	newRequest := func(priority backlog.Priority, points *int) backlog.CreateStoryRequest {
		return backlog.CreateStoryRequest{
			ProjectID:          1,
			Title:              "Historia válida",
			Description:        "Descripción válida",
			Priority:           priority,
			AcceptanceCriteria: []string{"Criterio válido"},
			StoryPoints:        points,
		}
	}
	for _, priority := range backlogPriorities() {
		store := backlog.NewInMemoryBacklogStore(1)
		if _, err := backlog.NewBacklogService(store).CreateStory(
			newRequest(backlog.Priority(priority), backlogFormIntPointer(5)),
		); err != nil {
			t.Errorf("HU-02 rejected listed priority %q: %v", priority, err)
		}
	}
	for _, points := range backlogStoryPointValues() {
		store := backlog.NewInMemoryBacklogStore(1)
		if _, err := backlog.NewBacklogService(store).CreateStory(
			newRequest(backlog.PriorityHigh, backlogFormIntPointer(points)),
		); err != nil {
			t.Errorf("HU-02 rejected listed Story Points %d: %v", points, err)
		}
	}

	store := backlog.NewInMemoryBacklogStore(1)
	if _, err := backlog.NewBacklogService(store).CreateStory(newRequest("Urgente", backlogFormIntPointer(5))); err == nil {
		t.Error("HU-02 accepted invalid priority")
	}
	store = backlog.NewInMemoryBacklogStore(1)
	if _, err := backlog.NewBacklogService(store).CreateStory(newRequest(backlog.PriorityHigh, backlogFormIntPointer(4))); err == nil {
		t.Error("HU-02 accepted Story Points outside the allowed series")
	}
}

func backlogFormIntPointer(value int) *int           { return &value }
func backlogFormFloatPointer(value float64) *float64 { return &value }

func equalOptionalInt(got, want *int) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}

func equalOptionalFloat(got, want *float64) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}
