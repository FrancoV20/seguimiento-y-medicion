package poker

import (
	"errors"
	"strconv"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// HU-05a, T006-T007: serie de votación.

func TestParseValue_Serie(t *testing.T) {
	for _, v := range []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89} {
		got, err := ParseValue(strconv.Itoa(v))
		if err != nil || got != v {
			t.Errorf("ParseValue(%d) = (%d, %v), se esperaba (%d, nil)", v, got, err, v)
		}
	}
}

func TestParseValue_Vacio(t *testing.T) {
	if _, err := ParseValue(""); !errors.Is(err, ErrValueRequired) {
		t.Errorf("se esperaba ErrValueRequired, se obtuvo %v", err)
	}
}

func TestParseValue_FueraDeSerie(t *testing.T) {
	for _, raw := range []string{"0", "4", "90", "-1", "abc", "05", " 5", "5.0"} {
		if _, err := ParseValue(raw); !errors.Is(err, ErrValueNotInSeries) {
			t.Errorf("ParseValue(%q): se esperaba ErrValueNotInSeries, se obtuvo %v", raw, err)
		}
	}
}

// La serie de votación es la misma que acepta HU-02 para los Story Points.
func TestSeries_CoincideConStoryPoints(t *testing.T) {
	if len(Series) != 10 {
		t.Fatalf("se esperaban 10 valores, hay %d", len(Series))
	}
	for i, v := range Series {
		if i > 0 && Series[i-1] >= v {
			t.Errorf("la serie debe ser creciente: %v", Series)
		}
		v := v
		if err := backlog.ValidateStoryPoints(&v); err != nil {
			t.Errorf("HU-02 rechaza %d: %v", v, err)
		}
		if !InSeries(v) {
			t.Errorf("InSeries(%d) debía ser verdadero", v)
		}
	}
	for _, v := range []int{0, 4, 90} {
		v := v
		if err := backlog.ValidateStoryPoints(&v); !errors.Is(err, backlog.ErrInvalidStoryPoints) {
			t.Errorf("HU-02 debía rechazar %d, se obtuvo %v", v, err)
		}
		if InSeries(v) {
			t.Errorf("InSeries(%d) debía ser falso", v)
		}
	}
}
