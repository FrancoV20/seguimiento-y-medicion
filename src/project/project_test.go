package project

import (
	"testing"
	"time"
)

func TestNuevoProyecto_RechazaCamposObligatoriosVacios(t *testing.T) {
	fechaInicio := time.Date(2026, time.October, 6, 0, 0, 0, 0, time.UTC)
	fechaFin := fechaInicio.AddDate(0, 0, 14)

	pruebas := []struct {
		nombre         string
		nombreProyecto string
		integrantes    []string
	}{
		{
			nombre:         "rechaza nombre vacío",
			nombreProyecto: "",
			integrantes:    []string{"Ana"},
		},
		{
			nombre:         "rechaza integrantes ausentes",
			nombreProyecto: "Proyecto académico",
			integrantes:    nil,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			// Given / Arrange: se prepara un proyecto con un campo obligatorio inválido.
			// When / Act: se intenta crear el proyecto.
			_, err := NuevoProyecto(prueba.nombreProyecto, prueba.integrantes, fechaInicio, fechaFin)

			// Then / Assert: la creación debe rechazarse con un error explícito.
			if err == nil {
				t.Fatal("se esperaba un error al faltar un campo obligatorio")
			}
		})
	}
}
