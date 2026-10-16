package backlog

import (
	"errors"
	"testing"
)

func intPtr(v int) *int { return &v }

func TestValidateStoryPoints_AceptaValoresDeLaSerie(t *testing.T) {
	for _, v := range []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89} {
		if err := ValidateStoryPoints(intPtr(v)); err != nil {
			t.Errorf("el valor %d debería ser válido, se obtuvo: %v", v, err)
		}
	}
}

func TestValidateStoryPoints_RechazaValoresFueraDeLaSerie(t *testing.T) {
	for _, v := range []int{-1, 0, 4, 6, 7, 10, 90, 100} {
		if err := ValidateStoryPoints(intPtr(v)); !errors.Is(err, ErrInvalidStoryPoints) {
			t.Errorf("el valor %d debería rechazarse, se obtuvo: %v", v, err)
		}
	}
}

func TestValidateStoryPoints_PermiteAusenciaDeEstimacion(t *testing.T) {
	if err := ValidateStoryPoints(nil); err != nil {
		t.Errorf("la ausencia de Story Points debe permitirse, se obtuvo: %v", err)
	}
}
