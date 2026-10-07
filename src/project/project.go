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
	if len(integrantes) == 0 {
		return Proyecto{}, errors.New("la lista de integrantes es obligatoria")
	}

	return Proyecto{
		Nombre:      nombre,
		Integrantes: integrantes,
		FechaInicio: fechaInicio,
		FechaFin:    fechaFin,
	}, nil
}
