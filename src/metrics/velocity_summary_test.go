package metrics

import (
	"fmt"
	"testing"
)

// HU-08 (T005) [E5, E6]: velocidad acumulada y promedio de los Sprints finalizados, sin base de
// datos. El promedio usa el redondeo de HU-07 (mitad hacia arriba, dos decimales) y se devuelve
// con punto; la coma la pone la pantalla.
func TestSummarizeVelocity(t *testing.T) {
	tests := []struct {
		name        string
		points      []int
		wantTotal   int
		wantAverage string
	}{
		{name: "sin Sprints finalizados", points: []int{}, wantTotal: 0, wantAverage: "No disponible"},
		{name: "sin Sprints (nil)", points: nil, wantTotal: 0, wantAverage: "No disponible"},
		{name: "un Sprint", points: []int{5}, wantTotal: 5, wantAverage: "5.00"},
		{name: "escenario 5: Sprints de 8 y 13", points: []int{8, 13}, wantTotal: 21, wantAverage: "10.50"},
		{name: "doce Sprints, periódico", points: []int{21, 21, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}, wantTotal: 92, wantAverage: "7.67"},
		{name: "mitad hacia arriba (0,125)", points: []int{1, 0, 0, 0, 0, 0, 0, 0}, wantTotal: 1, wantAverage: "0.13"},
		{name: "redondeo hacia arriba (0,666…)", points: []int{2, 0, 0}, wantTotal: 2, wantAverage: "0.67"},
		{name: "Sprints sin puntos completados", points: []int{0, 0}, wantTotal: 0, wantAverage: "0.00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeVelocity(tc.points)

			want := VelocitySummary{SprintCount: len(tc.points), Total: tc.wantTotal, Average: tc.wantAverage}
			if got != want {
				t.Errorf("SummarizeVelocity(%v) = %+v; se esperaba %+v", tc.points, got, want)
			}
		})
	}
}

// SummarizeVelocity no modifica la lista que recibe.
func TestSummarizeVelocity_NoModificaLaEntrada(t *testing.T) {
	points := []int{8, 13}
	SummarizeVelocity(points)
	if fmt.Sprint(points) != "[8 13]" {
		t.Errorf("SummarizeVelocity modificó la entrada: %v", points)
	}
}
