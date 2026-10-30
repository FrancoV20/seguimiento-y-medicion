package poker

import (
	"errors"
	"slices"
	"testing"
)

// HU-05b, T003: regla pura de divergencia (escenarios 2, 3, 4 y 5). Nivel = posición del valor en
// Series + 1; hay divergencia si nivel(mayor) − nivel(menor) > 2.

func TestSpread(t *testing.T) {
	cases := []struct {
		name      string
		values    []int
		low, high int
		divergent bool
	}{
		{"Login 2-13-5: niveles 2 y 6", []int{2, 13, 5}, 2, 13, true},
		{"Pagos 3-5-8: niveles 3 y 5, límite exacto", []int{3, 5, 8}, 3, 8, false},
		{"Reportes 3-13-3: niveles 3 y 6", []int{3, 13, 3}, 3, 13, true},
		{"Perfil 5-5: todos iguales", []int{5, 5}, 5, 5, false},
		{"un solo voto", []int{8}, 8, 8, false},
		{"1 y 3: niveles 1 y 3", []int{1, 3}, 1, 3, false},
		{"1 y 5: niveles 1 y 4", []int{1, 5}, 1, 5, true},
		{"55 y 89: niveles 9 y 10", []int{55, 89}, 55, 89, false},
		{"1 y 89: niveles 1 y 10", []int{1, 89}, 1, 89, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Spread(c.values)
			if err != nil {
				t.Fatalf("Spread(%v) devolvió error: %v", c.values, err)
			}
			want := VoteSpread{Low: c.low, High: c.high, Divergent: c.divergent}
			if got != want {
				t.Errorf("Spread(%v) = %+v, se esperaba %+v", c.values, got, want)
			}
		})
	}
}

func TestSpread_Errores(t *testing.T) {
	for name, values := range map[string][]int{"nil": nil, "vacío": {}} {
		if _, err := Spread(values); !errors.Is(err, ErrNoVotes) {
			t.Errorf("%s: se esperaba ErrNoVotes, se obtuvo %v", name, err)
		}
	}
	for _, values := range [][]int{{4}, {2, 0}} {
		if _, err := Spread(values); !errors.Is(err, ErrValueNotInSeries) {
			t.Errorf("%v: se esperaba ErrValueNotInSeries, se obtuvo %v", values, err)
		}
	}
}

func TestSpread_NoModificaLosValores(t *testing.T) {
	values := []int{13, 2, 5}
	if _, err := Spread(values); err != nil {
		t.Fatalf("Spread devolvió error: %v", err)
	}
	if want := []int{13, 2, 5}; !slices.Equal(values, want) {
		t.Errorf("Spread modificó el slice recibido: %v, se esperaba %v", values, want)
	}
}
