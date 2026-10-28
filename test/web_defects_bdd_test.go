package test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type defectBDDScenario struct {
	db           *sql.DB
	service      *defect.Service
	projectID    int64
	activeSprint int64
}

func givenDefectBDDScenario(t *testing.T) defectBDDScenario {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite el escenario con PostgreSQL")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("no se pudo abrir PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}

	stamp := time.Now().UnixNano()
	var scenario defectBDDScenario
	scenario.db = db
	insertID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
			t.Fatalf("no se pudo preparar el escenario BDD: %v", err)
		}
		return id
	}
	scenario.projectID = insertID(
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU06 BDD %d", stamp),
	)
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM user_stories WHERE project_id = $1`, scenario.projectID); err != nil {
			t.Errorf("no se pudieron borrar las historias BDD: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), `DELETE FROM sprints WHERE project_id = $1`, scenario.projectID); err != nil {
			t.Errorf("no se pudieron borrar los Sprints BDD: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), `DELETE FROM projects WHERE id = $1`, scenario.projectID); err != nil {
			t.Errorf("no se pudo borrar el proyecto BDD: %v", err)
		}
	})
	scenario.activeSprint = insertID(
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, 3, 'Detección', '2026-10-01', '2026-10-14', 'Activo') RETURNING id`,
		scenario.projectID,
	)
	scenario.service = defect.NewService(defect.NewPostgreSQLStore(db))
	return scenario
}

func (s defectBDDScenario) insertSprint(t *testing.T, number int, status string) int64 {
	t.Helper()
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, $2, $3, '2026-10-01', '2026-10-14', $4) RETURNING id`,
		s.projectID, number, fmt.Sprintf("Sprint %d", number), status,
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint BDD: %v", err)
	}
	return id
}

func (s defectBDDScenario) insertStory(t *testing.T, title, status string, sprintID *int64) int64 {
	t.Helper()
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
		 VALUES ($1, $2, 'Descripción BDD', 'Media', $3, $4) RETURNING id`,
		s.projectID, title, status, sprintID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear la historia BDD: %v", err)
	}
	return id
}

// Scenario: el defecto queda vinculado a la historia que lo originó.
//
//	Given una historia del proyecto y un Sprint activo de detección
//	When el Product Builder registra un defecto con severidad y descripción
//	Then el defecto queda abierto, vinculado a esa historia y al Sprint activo
func TestScenarioDefectIsLinkedToRelatedStory(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	storyID := scenario.insertStory(t, "BDD historia relacionada", "En progreso", &scenario.activeSprint)
	input := defect.RegisterInput{
		ProjectID:   scenario.projectID,
		StoryID:     storyID,
		Description: "El botón de guardar no responde",
		Severity:    defect.SeverityHigh,
	}

	registered, err := scenario.service.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("el defecto válido debía registrarse: %v", err)
	}

	var projectID, storedStoryID, sprintID int64
	var description, severity, status string
	if err := scenario.db.QueryRow(
		`SELECT project_id, story_id, detection_sprint_id, description, severity, status
		 FROM defects WHERE id = $1`,
		registered.ID,
	).Scan(&projectID, &storedStoryID, &sprintID, &description, &severity, &status); err != nil {
		t.Fatalf("no se pudo consultar el defecto BDD: %v", err)
	}
	if projectID != scenario.projectID || storedStoryID != storyID || sprintID != scenario.activeSprint ||
		description != input.Description || severity != string(input.Severity) || status != "Abierto" {
		t.Errorf("trazabilidad persistida incorrecta: proyecto=%d historia=%d sprint=%d descripción=%q severidad=%q estado=%q",
			projectID, storedStoryID, sprintID, description, severity, status)
	}
}

// Scenario: se registra un defecto sobre una historia de un Sprint anterior.
//
//	Given una historia Hecho de un Sprint finalizado y un Sprint activo distinto
//	When el Product Builder registra un defecto asociado a esa historia
//	Then el defecto usa el Sprint activo y la historia conserva estado y asignación
func TestScenarioDefectCanReferenceStoryFromFinishedSprint(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	finishedSprint := scenario.insertSprint(t, 2, "Finalizado")
	storyID := scenario.insertStory(t, "BDD historia terminada", "Hecho", &finishedSprint)

	registered, err := scenario.service.Register(context.Background(), defect.RegisterInput{
		ProjectID:   scenario.projectID,
		StoryID:     storyID,
		Description: "El cálculo final no coincide",
		Severity:    defect.SeverityCritical,
	})
	if err != nil {
		t.Fatalf("se debía aceptar una historia de un Sprint finalizado: %v", err)
	}

	var storyStatus string
	var storySprint sql.NullInt64
	if err := scenario.db.QueryRow(
		`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`, storyID,
	).Scan(&storyStatus, &storySprint); err != nil {
		t.Fatal(err)
	}
	if storyStatus != "Hecho" || !storySprint.Valid || storySprint.Int64 != finishedSprint {
		t.Errorf("la historia debía conservar estado y asignación: estado=%q Sprint=%+v", storyStatus, storySprint)
	}

	var detectionSprintID int64
	if err := scenario.db.QueryRow(
		`SELECT detection_sprint_id FROM defects WHERE id = $1`, registered.ID,
	).Scan(&detectionSprintID); err != nil {
		t.Fatal(err)
	}
	if detectionSprintID != scenario.activeSprint {
		t.Errorf("el Sprint de detección = %d; se esperaba el Sprint activo %d", detectionSprintID, scenario.activeSprint)
	}
}

// Scenario: faltan los campos obligatorios del defecto.
//
//	Given un Product Builder que intenta registrar un defecto
//	When omite la descripción, la severidad y la historia
//	Then el servicio informa los tres errores y no guarda ningún defecto
func TestScenarioMissingRequiredDefectFields(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	before := scenario.defectCount(t)

	_, err := scenario.service.Register(context.Background(), defect.RegisterInput{ProjectID: scenario.projectID})

	var validation defect.ValidationErrors
	if !errors.As(err, &validation) {
		t.Fatalf("se debían devolver errores de validación; obtuvo %v", err)
	}
	if len(validation) != 3 {
		t.Fatalf("se esperaban tres errores simultáneos; obtuvo %d", len(validation))
	}
	for _, field := range []string{defect.FieldDescription, defect.FieldSeverity, defect.FieldStory} {
		found := false
		for _, violation := range validation {
			if violation.Field == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("faltó el error para el campo %q", field)
		}
	}
	if got := scenario.defectCount(t); got != before {
		t.Errorf("la validación no debía guardar defectos; antes=%d después=%d", before, got)
	}
}

// Scenario: una descripción vacía o demasiado larga no se registra.
//
//	Given un Product Builder que carga un defecto
//	When la descripción está vacía o supera los 1000 caracteres
//	Then el servicio rechaza la solicitud sin persistirla
func TestScenarioInvalidDefectDescriptionIsRejected(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	storyID := scenario.insertStory(t, "BDD historia de descripción", "Pendiente", nil)
	before := scenario.defectCount(t)

	for _, tc := range []struct {
		name        string
		description string
		want        error
	}{
		{"vacía", "", defect.ErrDescriptionRequired},
		{"solo espacios", " \t ", defect.ErrDescriptionRequired},
		{"supera el límite", strings.Repeat("x", 1001), defect.ErrDescriptionTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := scenario.service.Register(context.Background(), defect.RegisterInput{
				ProjectID:   scenario.projectID,
				StoryID:     storyID,
				Description: tc.description,
				Severity:    defect.SeverityHigh,
			})
			if !errors.Is(err, tc.want) {
				if got := validationErrorFor(err, defect.FieldDescription); !errors.Is(got, tc.want) {
					t.Errorf("error = %v; se esperaba %v", got, tc.want)
				}
			}
			if got := scenario.defectCount(t); got != before {
				t.Errorf("la descripción inválida no debía guardarse; antes=%d después=%d", before, got)
			}
		})
	}
}

// Scenario: no se registra un defecto sin Sprint activo.
//
//	Given un proyecto cuyos Sprints están finalizados
//	When se intenta registrar un defecto válido
//	Then el registro se rechaza y no crea defectos
func TestScenarioDefectRequiresActiveSprint(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	storyID := scenario.insertStory(t, "BDD historia sin Sprint activo", "Pendiente", nil)
	if _, err := scenario.db.Exec(
		`UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, scenario.activeSprint,
	); err != nil {
		t.Fatal(err)
	}
	before := scenario.defectCount(t)

	_, err := scenario.service.Register(context.Background(), defect.RegisterInput{
		ProjectID:   scenario.projectID,
		StoryID:     storyID,
		Description: "No se debe registrar",
		Severity:    defect.SeverityHigh,
	})

	if !errors.Is(err, defect.ErrNoActiveSprint) {
		t.Errorf("se esperaba el rechazo por falta de Sprint activo; obtuvo %v", err)
	}
	if got := scenario.defectCount(t); got != before {
		t.Errorf("no debía guardarse ningún defecto; antes=%d después=%d", before, got)
	}
}

// Scenario: una historia de otro proyecto no se asocia al defecto.
//
//	Given una historia que pertenece a otro proyecto
//	When se intenta registrarla como historia relacionada
//	Then el servicio la rechaza y no guarda el defecto
func TestScenarioDefectCannotReferenceStoryFromAnotherProject(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	var otherProjectID, otherStoryID int64
	if err := scenario.db.QueryRow(
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU06 BDD otro proyecto %d", time.Now().UnixNano()),
	).Scan(&otherProjectID); err != nil {
		t.Fatalf("no se pudo crear el segundo proyecto: %v", err)
	}
	t.Cleanup(func() {
		if _, err := scenario.db.Exec(`DELETE FROM user_stories WHERE id = $1`, otherStoryID); err != nil {
			t.Errorf("no se pudo borrar la historia BDD: %v", err)
		}
		if _, err := scenario.db.Exec(`DELETE FROM projects WHERE id = $1`, otherProjectID); err != nil {
			t.Errorf("no se pudo borrar el proyecto BDD: %v", err)
		}
	})
	if err := scenario.db.QueryRow(
		`INSERT INTO user_stories (project_id, title, description, priority, status)
		 VALUES ($1, 'BDD historia externa', 'Descripción', 'Media', 'Pendiente') RETURNING id`,
		otherProjectID,
	).Scan(&otherStoryID); err != nil {
		t.Fatalf("no se pudo crear la historia del segundo proyecto: %v", err)
	}
	before := scenario.defectCount(t)

	_, err := scenario.service.Register(context.Background(), defect.RegisterInput{
		ProjectID:   scenario.projectID,
		StoryID:     otherStoryID,
		Description: "No se debe vincular",
		Severity:    defect.SeverityHigh,
	})

	if !errors.Is(err, defect.ErrStoryNotFound) {
		t.Errorf("se esperaba el rechazo de historia ajena; obtuvo %v", err)
	}
	if got := scenario.defectCount(t); got != before {
		t.Errorf("no debía guardarse ningún defecto; antes=%d después=%d", before, got)
	}
}

type failingBDDDefectStore struct{ err error }

func (s failingBDDDefectStore) Create(context.Context, defect.RegisterInput) (defect.Registered, error) {
	return defect.Registered{}, s.err
}

// Scenario: un error al guardar no devuelve un registro exitoso.
//
//	Given un defecto válido y un almacenamiento que falla
//	When se solicita registrar el defecto
//	Then el servicio devuelve el fallo y no devuelve identificador de registro
func TestScenarioDefectStorageFailureDoesNotSucceed(t *testing.T) {
	scenario := givenDefectBDDScenario(t)
	storyID := scenario.insertStory(t, "BDD historia para falla", "Pendiente", nil)
	before := scenario.defectCount(t)
	service := defect.NewService(failingBDDDefectStore{err: errors.New("fallo simulado")})

	registered, err := service.Register(context.Background(), defect.RegisterInput{
		ProjectID:   scenario.projectID,
		StoryID:     storyID,
		Description: "Defecto válido",
		Severity:    defect.SeverityHigh,
	})

	if err == nil {
		t.Fatal("el fallo de almacenamiento debía propagarse")
	}
	if registered.ID != 0 || registered.SprintNumber != 0 {
		t.Errorf("no debía devolverse un registro exitoso: %+v", registered)
	}
	if got := scenario.defectCount(t); got != before {
		t.Errorf("la falla no debía guardar defectos; antes=%d después=%d", before, got)
	}
}

func (s defectBDDScenario) defectCount(t *testing.T) int {
	t.Helper()
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM defects WHERE project_id = $1`, s.projectID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func validationErrorFor(err error, field string) error {
	var violations defect.ValidationErrors
	if !errors.As(err, &violations) {
		return err
	}
	for _, violation := range violations {
		if violation.Field == field {
			return violation.Err
		}
	}
	return nil
}
