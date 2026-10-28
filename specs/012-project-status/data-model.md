# Modelo de datos: HU-12 – Consulta del Estado de un Proyecto

## Esquema

Sin migración ni tablas nuevas. HU-12 solo lee; no usa la columna `projects.status`.

| Tabla | Columnas usadas | Quién la lee | Origen |
|---|---|---|---|
| `projects` | `id`, `name`, `start_date`, `end_date` | `PostgreSQLProjectStatusCatalog` | HU-01, en `main` |
| `user_stories` | `project_id`, `status` | `PostgreSQLProjectStatusCatalog` | HU-02 y HU-04, en `main` |
| `sprints`, `sprint_story_assignments` | — (las lee el Tablero) | `PostgreSQLBoardCatalog` (HU-14) | HU-04, en `main` |

## Dominio (`src/project/status.go`, nuevo)

```go
// DateStatus es el estado de un proyecto según sus fechas y la fecha de hoy.
type DateStatus string

const (
    StatusNotStarted DateStatus = "No iniciado"
    StatusInProgress DateStatus = "En curso"
    StatusFinished   DateStatus = "Finalizado"
)

// StatusOn calcula el estado comparando días del calendario (research.md §2). No lee el reloj.
func StatusOn(start, end, today time.Time) DateStatus

// ProgressPercentage devuelve done ÷ total × 100, mitad hacia arriba a dos decimales, con punto: "50.00".
// Sin historias (total <= 0) devuelve "0.00".
func ProgressPercentage(done, total int) string
```

Reglas de `StatusOn`:

- Se toma `(año, mes, día)` de cada fecha con `t.Date()`, en su propia zona.
- `today < start` → `StatusNotStarted`; `start <= today <= end` → `StatusInProgress`; `today > end` →
  `StatusFinished`.

| `start` | `end` | `today` | Resultado |
|---|---|---|---|
| 01/10/2026 | 30/11/2026 | 28/10/2026 10:00 | "En curso" («Alfa») |
| 01/12/2026 | 31/01/2027 | 28/10/2026 10:00 | "No iniciado" («Beta») |
| 01/08/2026 | 30/09/2026 | 28/10/2026 10:00 | "Finalizado" («Gamma») |
| 28/10/2026 | 15/12/2026 | 28/10/2026 00:00 y 23:59 | "En curso" («Delta», empieza hoy) |
| 01/09/2026 | 28/10/2026 | 28/10/2026 00:00 y 23:59 | "En curso" («Épsilon», termina hoy) |
| 01/09/2026 | 28/10/2026 | 29/10/2026 00:00 | "Finalizado" (día siguiente al fin) |
| 28/10/2026 | 15/12/2026 | 27/10/2026 23:59 | "No iniciado" (día anterior al inicio) |
| 28/10/2026 (medianoche UTC) | 15/12/2026 | 27/10/2026 22:00 en UTC-3 (28/10 01:00 UTC) | "No iniciado" (cuenta el día local) |

Reglas de `ProgressPercentage`:

- `total <= 0` → `"0.00"`.
- Si no, `centésimas = (done*10000*2 + total) / (2*total)` (división entera: mitad hacia arriba) y
  `fmt.Sprintf("%d.%02d", centésimas/100, centésimas%100)`.

| `done`, `total` | Resultado | Cuenta |
|---|---|---|
| 4, 8 | "50.00" | «Alfa» |
| 3, 3 | "100.00" | «Gamma» |
| 0, 0 | "0.00" | «Beta», «Delta», «Épsilon» |
| 0, 5 | "0.00" | |
| 2, 3 | "66.67" | 66,666… |
| 1, 3 | "33.33" | 33,333… |
| 1, 8 | "12.50" | |
| 4, 7 | "57.14" | 57,142… (HU-09) |
| 1, 6 | "16.67" | 16,666… |
| 7, 11 | "63.64" | 63,636… |
| 1, 32 | "3.13" | 3,125: mitad exacta, sube |

No se edita ningún archivo existente de `src/project`.

## Pantalla (`src/web`)

### `project_status_catalog.go`

```go
// ProjectStatusCounts son las historias del proyecto por estado.
type ProjectStatusCounts struct {
    Pending    int // "Pendiente"
    InSprint   int // "En Sprint"
    InProgress int // "En progreso"
    Done       int // "Hecho"
}

func (c ProjectStatusCounts) Total() int // suma de las cuatro

// ProjectStatusSummary son los datos propios de la pantalla.
type ProjectStatusSummary struct {
    Name      string
    StartDate time.Time
    EndDate   time.Time
    Counts    ProjectStatusCounts
}

// ProjectStatusCatalog lee el proyecto y sus cantidades (solo lectura).
type ProjectStatusCatalog interface {
    // StatusSummary devuelve NotFound si el proyecto no existe.
    StatusSummary(ctx context.Context, projectID int64) (ProjectStatusSummary, error)
}
```

Se reutiliza sin redefinir `BoardCatalog` (HU-14): `ActiveBoard(ctx, projectID) (BoardView, bool, error)`.

### `postgres_project_status_catalog.go`

`PostgreSQLProjectStatusCatalog` y `NewPostgreSQLProjectStatusCatalog(db *sql.DB)`.
`BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})`, `defer tx.Rollback()`:

```sql
-- 1. Proyecto (sin fila → NotFound)
SELECT name, start_date, end_date FROM projects WHERE id = $1;

-- 2. Historias por estado
SELECT COUNT(*) FILTER (WHERE status = 'Pendiente'),
       COUNT(*) FILTER (WHERE status = 'En Sprint'),
       COUNT(*) FILTER (WHERE status = 'En progreso'),
       COUNT(*) FILTER (WHERE status = 'Hecho')
FROM user_stories
WHERE project_id = $1;

COMMIT;
```

Cualquier otro error se devuelve envuelto (`fmt.Errorf("project status %d: %w", projectID, err)`); `Fail` decide 500
o 503.

### `project_status_view.go`

| Identificador | Texto |
|---|---|
| `TitleProjectStatus` | "Estado del proyecto" |
| `msgProjectStatusDates` | "Del %s al %s" |
| `msgProjectStatusState` | "Estado: %s" |
| `msgProjectStatusProgress` | "%d de %d historias completadas (%s)" |
| `msgProjectStatusSprint` | "%s en curso" (con `BoardView.Label`: "Sprint 2 en curso") |
| `msgProjectStatusSprintGoal` | "Objetivo: %s" |
| `MsgProjectStatusNoSprint` | "No hay un Sprint en curso" |
| etiquetas de estado de historia | "Pendiente", "En Sprint", "En progreso", "Hecho" |

Funciones:

- `projectStatusDateText(t time.Time) string` → "01/10/2026" (`02/01/2006`).
- `projectStatusPercentText("50.00")` → "50,00 %".
- `buildProjectStatusView(summary, board, found, today) projectStatusView`: llama a `project.StatusOn(StartDate,
  EndDate, today)` y a `project.ProgressPercentage(Counts.Done, Counts.Total())`.

```go
type projectStatusRow struct {
    Label string // "Pendiente"
    Count int
}

type projectStatusView struct {
    Name     string
    Dates    string             // "Del 01/10/2026 al 30/11/2026"
    State    string             // "Estado: En curso"
    Progress string             // "4 de 8 historias completadas (50,00 %)"
    Rows     []projectStatusRow // cuatro filas, en orden, con ceros

    HasSprint   bool
    SprintLabel string // "Sprint 2 en curso"
    SprintGoal  string // "Objetivo: Pagos online"
    SprintDates string // "Del 19/10/2026 al 01/11/2026"
    NoSprint    string // MsgProjectStatusNoSprint si no hay Sprint activo
}
```

### `project_status_handlers.go`

`RegisterProjectStatus(s *Server, catalog ProjectStatusCatalog, board BoardCatalog, now func() time.Time) error`:

- `now == nil` → `time.Now`; `catalog` o `board` `nil` → error sin registrar;
- registra `Section{Name: TitleProjectStatus, Path: "/estado", RequiresProject: true}` con
  `Routes{"GET /estado": …}`.

Pasos de `GET /estado` (`projectIDFrom(r)` de HU-14):

1. `catalog.StatusSummary(ctx, projectID)`. Error → `Fail` (404 si es `NotFound`).
2. `board.ActiveBoard(ctx, projectID)`. Error → `Fail`.
3. `buildProjectStatusView(summary, board, found, now())` y `Page(w, r, "project_status.html", …)` con 200.

Nada se escribe en la respuesta antes del paso 3.

### `templates/project_status.html`

- Nombre del proyecto, fechas y estado.
- "Avance": la línea "X de Y historias completadas (P %)" y una tabla Estado/Historias con las cuatro filas.
- "Sprint en curso": etiqueta, objetivo y fechas, o `NoSprint`.
- Sin formularios ni `<script>`.

## Respuestas

| Caso | Código | Contenido |
|---|---|---|
| Proyecto activo | 200 | Estado, avance, historias por estado y Sprint en curso (o "No hay un Sprint en curso") |
| Proyecto activo que ya no existe | 404 | Página de HU-20a |
| Falla del catálogo o del Tablero | 500 o 503 | Página de HU-20a, sin datos parciales |
| Sin proyecto activo | 303 | `/contexto/proyecto?volver_a=%2Festado` |
| `POST` u otro método | 405 | Página de HU-20a |

## Contrato que se ofrece (HU-09)

```go
estado := project.StatusOn(inicio, fin, hoy)           // project.DateStatus: "No iniciado", "En curso", "Finalizado"
porcentaje := project.ProgressPercentage(hechas, total) // "57.14"; "0.00" sin historias
```
