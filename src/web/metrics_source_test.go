package web

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-25 (T008): el adaptador MetricsSource sobre el servicio real de HU-07 y PostgreSQL real.
// Los Sprints se cierran con el servicio real de HU-15.

func TestHU07MetricsSource_VelocidadYDesviacionConValoresConocidos(t *testing.T) {
	db := openContextTestDB(t)
	// Dado un Sprint con dos historias "Hecho" (5 y 3 Story Points; 4 y 6 horas estimadas;
	// 5 y 7 horas reales) y una "En progreso" con horas que HU-15 devuelve al Backlog.
	f := newMetricsProject(t, db, "valores conocidos")
	sprintID := f.activeSprint(1, "Sprint con datos conocidos", "2026-10-01", "2026-10-14")
	h1 := f.story(sprintID, "Hecha 1", 5, sprint.Done, "4")
	h2 := f.story(sprintID, "Hecha 2", 3, sprint.Done, "6")
	devuelta := f.story(sprintID, "Devuelta al Backlog", 13, sprint.InProgress, "100")
	f.effort(h1, "5")
	f.effort(h2, "7")
	f.effort(devuelta, "20")
	f.close(sprintID)

	// Cuando el adaptador pide las métricas a HU-07.
	got, err := NewHU07MetricsSource(db).Metrics(context.Background(), f.ProjectID, sprintID)

	// Entonces la velocidad es 8, la desviación es la de HU-07 para 10 estimadas y 12 reales
	// (20 %) y la historia devuelta no cuenta.
	if err != nil {
		t.Fatalf("Metrics devolvió error: %v", err)
	}
	want := SprintMetrics{Velocity: 8, Deviation: "20.00", PartialUsed: 2, PartialTotal: 2}
	if got != want {
		t.Errorf("Metrics = %+v; se esperaba %+v", got, want)
	}
	assertNoMetricsSnapshot(t, db, sprintID)
}

func TestHU07MetricsSource_SinHorasEstimadas(t *testing.T) {
	db := openContextTestDB(t)
	// Dado un Sprint con una historia "Hecho" de 5 Story Points, sin horas estimadas y con horas reales.
	f := newMetricsProject(t, db, "sin horas estimadas")
	sprintID := f.activeSprint(1, "Sprint sin estimaciones", "2026-10-01", "2026-10-14")
	h := f.story(sprintID, "Hecha sin estimar", 5, sprint.Done, "")
	f.effort(h, "3")
	f.close(sprintID)

	got, err := NewHU07MetricsSource(db).Metrics(context.Background(), f.ProjectID, sprintID)

	// Entonces la velocidad igual se informa y la desviación es "No disponible" de HU-07.
	if err != nil {
		t.Fatalf("Metrics devolvió error: %v", err)
	}
	if got.Velocity != 5 {
		t.Errorf("velocidad = %d; se esperaba 5", got.Velocity)
	}
	if got.Deviation != hu07NoData(t) {
		t.Errorf("desviación = %q; se esperaba %q", got.Deviation, hu07NoData(t))
	}
	if got.PartialUsed != 0 || got.PartialTotal != 1 {
		t.Errorf("X de Y = %d de %d; se esperaba 0 de 1", got.PartialUsed, got.PartialTotal)
	}
	assertNoMetricsSnapshot(t, db, sprintID)
}

func TestHU07MetricsSource_CalculoParcialConXdeY(t *testing.T) {
	db := openContextTestDB(t)
	// Dado un Sprint con tres historias "Hecho": solo una tiene horas estimadas (8) y reales (6).
	f := newMetricsProject(t, db, "cálculo parcial")
	sprintID := f.activeSprint(1, "Sprint parcial", "2026-10-01", "2026-10-14")
	completa := f.story(sprintID, "Hecha con datos", 5, sprint.Done, "8")
	sinEstimar := f.story(sprintID, "Hecha sin estimar", 3, sprint.Done, "")
	f.story(sprintID, "Hecha sin horas reales", 2, sprint.Done, "4")
	f.effort(completa, "6")
	f.effort(sinEstimar, "2")
	f.close(sprintID)

	got, err := NewHU07MetricsSource(db).Metrics(context.Background(), f.ProjectID, sprintID)

	// Entonces HU-07 calcula con 1 de 3 historias y lo informa con su aviso.
	if err != nil {
		t.Fatalf("Metrics devolvió error: %v", err)
	}
	want := SprintMetrics{
		Velocity:     10,
		Deviation:    "-25.00",
		PartialUsed:  1,
		PartialTotal: 3,
		Partial:      true,
		PartialAlert: hu07PartialWarning(t, 1, 3),
	}
	if got != want {
		t.Errorf("Metrics = %+v; se esperaba %+v", got, want)
	}
	assertNoMetricsSnapshot(t, db, sprintID)
}

func TestHU07MetricsSource_SinHistoriasCompletadasVelocidadCero(t *testing.T) {
	db := openContextTestDB(t)
	f := newMetricsProject(t, db, "velocidad cero")

	// Dado un Sprint cerrado por HU-15 sin historias "Hecho" (las incompletas vuelven al Backlog)
	// y otro Sprint cerrado sin ninguna historia.
	conIncompletas := f.activeSprint(1, "Sprint sin terminar nada", "2026-10-01", "2026-10-14")
	enProgreso := f.story(conIncompletas, "En progreso", 5, sprint.InProgress, "6")
	f.story(conIncompletas, "En Sprint", 3, sprint.EnSprint, "2")
	f.effort(enProgreso, "4")
	f.close(conIncompletas)
	vacio := f.finishedSprint(2, "Sprint vacío", "2026-10-15", "2026-10-28")

	for name, sprintID := range map[string]int64{"con incompletas": conIncompletas, "sin historias": vacio} {
		t.Run(name, func(t *testing.T) {
			got, err := NewHU07MetricsSource(db).Metrics(context.Background(), f.ProjectID, sprintID)

			// Entonces la velocidad es cero y la desviación "No disponible", sin error ni alerta.
			if err != nil {
				t.Fatalf("Metrics devolvió error: %v", err)
			}
			want := SprintMetrics{Velocity: 0, Deviation: hu07NoData(t)}
			if got != want {
				t.Errorf("Metrics = %+v; se esperaba %+v", got, want)
			}
			assertNoMetricsSnapshot(t, db, sprintID)
		})
	}
}

func TestHU07MetricsSource_SprintInexistenteOActivoDevuelveError(t *testing.T) {
	db := openContextTestDB(t)
	f := newMetricsProject(t, db, "errores")
	activo := f.activeSprint(1, "Sprint en curso", "2026-10-01", "2026-10-14")
	f.story(activo, "Hecha del Sprint activo", 5, sprint.Done, "4")

	// Un Sprint activo o inexistente no tiene métricas: HU-07 lo rechaza y el adaptador lo informa.
	for name, sprintID := range map[string]int64{"activo": activo, "inexistente": MaxID} {
		t.Run(name, func(t *testing.T) {
			got, err := NewHU07MetricsSource(db).Metrics(context.Background(), f.ProjectID, sprintID)
			if err == nil {
				t.Fatalf("Metrics(%s) = %+v sin error; se esperaba un error", name, got)
			}
			if errors.Is(err, errMetricsNotImplemented) {
				t.Fatalf("Metrics(%s) todavía no está implementado", name)
			}
		})
	}
	assertNoMetricsSnapshot(t, db, activo)
}

// assertNoMetricsSnapshot verifica que consultar las métricas no guardó nada en sprint_metrics
// (HU-25 no guarda métricas: SC-006).
func assertNoMetricsSnapshot(t *testing.T, db *sql.DB, sprintID int64) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sprint_metrics WHERE sprint_id = $1`, sprintID).Scan(&count); err != nil {
		t.Fatalf("no se pudo consultar sprint_metrics: %v", err)
	}
	if count != 0 {
		t.Errorf("se guardaron %d filas en sprint_metrics para el Sprint %d; la pantalla no debe guardar métricas", count, sprintID)
	}
}
