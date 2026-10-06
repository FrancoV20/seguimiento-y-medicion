package project

import (
	"errors"
	"time"
)

type Proyecto struct {
	Nombre      string
	Integrantes []string
	FechaInicio time.Time
	FechaFin    time.Time
}

func NuevoProyecto(nombre string, integrantes []string, fechaInicio, fechaFin time.Time) (Proyecto, error) {
	if nombre == "" {
		return Proyecto{}, errors.New("el nombre del proyecto no puede estar vacío")
	}

	return Proyecto{
		Nombre:      nombre,
		Integrantes: integrantes,
		FechaInicio: fechaInicio,
		FechaFin:    fechaFin,
	}, nil
}
