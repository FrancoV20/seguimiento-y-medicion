# Modelo de datos: HU-16 – Consulta de Sprints Anteriores

## Esquema

Sin migración ni tablas nuevas. HU-16 solo lee; el único efecto en la base es el de HU-07 (vuelve a guardar las
métricas del Sprint).

| Tabla | Columnas usadas | Quién la lee | Origen |
|---|---|---|---|
| `sprints` | `id`, `project_id`, `number`, `goal`, `start_date`, `end_date`, `status` | `PostgreSQLPastSprintCatalog`, HU-07 | HU-04, en `main` |
| `sprint_story_results` | `sprint_id`, `story_id`, `final_status`, `story_points` | `PostgreSQLPastSprintCatalog` | HU-15 (T001: HU-15) |
| `user_stories` | `id`, `title` (y, para HU-07, `status` y `story_points`) | `PostgreSQLPastSprintCatalog`, HU-07 | HU-02, en `main` |
| `sprint_story_assignments` | `sprint_id`, `story_id` | HU-07 | HU-04, en `main` |
| `sprint_metrics` | todas (las escribe HU-07) | — | HU-07, en `main` |

Estados que importan:

| Dato | Valores | Uso en HU-16 |
|---|---|---|
| `sprints.status` | 'Activo', 'Finalizado' | Solo 'Finalizado' es un Sprint anterior |
| `sprint_story_results.final_status` (T001: HU-15) | 'En Sprint', 'En progreso', 'Hecho' | 'Hecho' → Completadas; los demás → No completadas |
| `sprint_story_results.story_points` (T001: HU-15) | entero de la serie o `NULL` | `NULL` → "Sin estimar" |

## Catálogo (`src/web/past_sprint_catalog.go`, nuevo)

```go
// PastSprintSummary es un Sprint finalizado de la lista.
type PastSprintSummary struct {
    ID        int64
    Number    int
    Goal      string
    StartDate time.Time
    EndDate   time.Time
}

// PastSprintStory es una fila del registro de cierre de HU-15 (T001: HU-15).
type PastSprintStory struct {
    StoryID     int64
    Title       string // título actual de la historia
    StoryPoints *int   // al cerrar; nil = "Sin estimar"
    FinalStatus string // "En Sprint", "En progreso" o "Hecho"
}

// PastSprintDetail es un Sprint finalizado con su registro de cierre, ordenado por StoryID.
type PastSprintDetail struct {
    PastSprintSummary
    Stories []PastSprintStory // vacío si el Sprint no tiene registro
}

// PastSprintCatalog lee los Sprints finalizados del proyecto (solo lectura).
type PastSprintCatalog interface {
    // FinishedSprints devuelve los Sprints 'Finalizado' del proyecto, por número descendente.
    FinishedSprints(ctx context.Context, projectID int64) ([]PastSprintSummary, error)
    // SprintDetail devuelve NotFound si el Sprint no existe, es de otro proyecto o no está 'Finalizado'.
    SprintDetail(ctx context.Context, projectID, sprintID int64) (PastSprintDetail, error)
}

// PastSprintMetrics es lo que HU-16 usa de HU-07; *metrics.Service la cumple.
type PastSprintMetrics interface {
    CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
}
```

## Adaptador (`src/web/postgres_past_sprint_catalog.go`, nuevo)

`PostgreSQLPastSprintCatalog` y `NewPostgreSQLPastSprintCatalog(db *sql.DB)`.

`FinishedSprints`, en una sola consulta:

```sql
SELECT id, number, goal, start_date, end_date
FROM sprints
WHERE project_id = $1 AND status = 'Finalizado'
ORDER BY number DESC;
```

`SprintDetail`: `BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})`,
`defer tx.Rollback()`:

```sql
-- 1. Sprint del proyecto, finalizado (sin fila → NotFound)
SELECT id, number, goal, start_date, end_date
FROM sprints
WHERE id = $1 AND project_id = $2 AND status = 'Finalizado';

-- 2. Registro de cierre (T001: HU-15) con el título actual
SELECT r.story_id, us.title, r.story_points, r.final_status
FROM sprint_story_results r
JOIN user_stories us ON us.id = r.story_id
WHERE r.sprint_id = $1
ORDER BY r.story_id;

COMMIT;
```

Errores:

- Sin fila en el paso 1 → `NotFound(fmt.Errorf("past sprint %d: %w", sprintID, sql.ErrNoRows))`.
- Cualquier otro error se devuelve envuelto (`fmt.Errorf("past sprint %d: %w", sprintID, err)`), y `Fail` decide
  entre 500 y 503.

## Vista (`src/web/past_sprint_view.go`, nuevo)

| Identificador | Texto |
|---|---|
| `TitlePastSprints` | "Sprints anteriores" |
| `MsgPastSprintsEmpty` | "Aún no hay Sprints anteriores" |
| `msgPastSprintLabel` | "Sprint %d" |
| `msgPastSprintGoal` | "Objetivo: %s" |
| `msgPastSprintDates` | "Del %s al %s" |
| `msgPastSprintVelocity` | "Velocidad: %d Story Points" |
| `MsgPastSprintAllCompleted` | "Todas las historias se completaron" |
| `MsgPastSprintNoneCompleted` | "Ninguna historia se completó" |
| `MsgPastSprintNoRecord` | "No hay registro de las historias de este Sprint" |
| `LabelPastSprintUnestimated` | "Sin estimar" |
| `LabelPastSprintReturned` | "volvió al Backlog" |
| `LinkPastSprintsBack` | "Volver a Sprints anteriores" |
| encabezados | "Completadas", "No completadas", "Historia", "Story Points", "Estado al cerrar" |

```go
type pastSprintItem struct {
    Label string // "Sprint 2"
    Goal  string // "Gestionar perfiles y reportes"
    Dates string // "Del 01/09/2026 al 14/09/2026"
    Link  string // "/sprints-anteriores/{id}"
}

type pastSprintListView struct {
    Sprints []pastSprintItem // por número descendente
    Empty   string           // MsgPastSprintsEmpty si no hay Sprints
}

type pastSprintRow struct {
    Title       string
    StoryPoints string // "5" o "Sin estimar"
    FinalStatus string // "Hecho", "En progreso" o "En Sprint"
    Returned    string // LabelPastSprintReturned en las no completadas; vacío en las completadas
}

type pastSprintDetailView struct {
    Label    string // "Sprint 2"
    Goal     string // "Objetivo: Gestionar perfiles y reportes"
    Dates    string // "Del 01/09/2026 al 14/09/2026"
    Velocity string // "Velocidad: 10 Story Points"

    NoRecord      string          // MsgPastSprintNoRecord si no hay filas; entonces no hay tablas
    Completed     []pastSprintRow // por StoryID
    NoneCompleted string          // MsgPastSprintNoneCompleted si hay registro y no hay completadas
    Pending       []pastSprintRow // por StoryID
    AllCompleted  string          // MsgPastSprintAllCompleted si hay registro y no hay no completadas
    Back          string          // LinkPastSprintsBack
}
```

Funciones puras:

- `pastSprintDateText(t time.Time) string` → "01/09/2026" (`02/01/2006`).
- `pastSprintPointsText(p *int) string` → "5" o "Sin estimar".
- `buildPastSprintListView(sprints []PastSprintSummary) pastSprintListView`.
- `buildPastSprintDetailView(detail PastSprintDetail, velocity int) pastSprintDetailView`: separa
  `detail.Stories` por `FinalStatus == string(sprint.Done)` y conserva el orden.

Casos de `buildPastSprintDetailView`:

| Datos | Completadas | No completadas | Mensajes | Velocidad |
|---|---|---|---|---|
| Sprint 2 de «Alfa» | «Login» 5, «Tablero» 3, «Perfil» 2 | «Reportes» 8, En progreso, volvió al Backlog | — | 10 |
| Sprint 1 de «Alfa» | «Alta» 3 | — | "Todas las historias se completaron" | 3 |
| Sprint 1 de «Gamma» | — | «Migración» 5, En progreso; «Respaldo» Sin estimar, En Sprint; las dos con volvió al Backlog | "Ninguna historia se completó" | 0 |
| Sprint 2 de «Gamma» (sin registro) | — | — | "No hay registro de las historias de este Sprint" | 2 |

## Handlers (`src/web/past_sprint_handlers.go`, nuevo)

`RegisterPastSprints(s *Server, catalog PastSprintCatalog, calc PastSprintMetrics) error`:

- `catalog` o `calc` `nil` → error sin registrar;
- registra `Section{Name: TitlePastSprints, Path: "/sprints-anteriores", RequiresProject: true}` con
  `Routes{"GET /sprints-anteriores": …, "GET /sprints-anteriores/{id}": …}`.

`GET /sprints-anteriores` (`projectIDFrom(r)` de HU-14):

1. `catalog.FinishedSprints(ctx, projectID)`. Error → `Fail`.
2. `buildPastSprintListView` y `Page(w, r, "past_sprints.html", …)` con 200.

`GET /sprints-anteriores/{id}`:

1. `id := parseID(r.PathValue("id"))`; si es 0 → `Fail(NotFound(...))`, sin consultar la base.
2. `catalog.SprintDetail(ctx, projectID, id)`. Error → `Fail` (404 si es `NotFound`).
3. `calc.CalculateMetrics(id)`. Error → `Fail`.
4. `buildPastSprintDetailView(detail, m.Velocity)` y `Page(w, r, "past_sprint.html", …)` con 200.

En los dos handlers no se escribe nada antes del último paso.

## Plantillas

- `templates/past_sprints.html`: título "Sprints anteriores"; una lista con un enlace por Sprint ("Sprint N"), su
  objetivo y sus fechas; o `Empty`.
- `templates/past_sprint.html`: "Sprint N", objetivo, fechas y velocidad. Después, `NoRecord`, o las dos secciones:
  - "Completadas": tabla Historia / Story Points / Estado al cerrar, o `NoneCompleted`;
  - "No completadas": la misma tabla con la columna "volvió al Backlog", o `AllCompleted`.

  Al final, el enlace `Back`.
- Ninguna tiene formularios, botones ni `<script>`.

## Respuestas

| Caso | Código | Contenido |
|---|---|---|
| Lista con Sprints finalizados | 200 | Un elemento por Sprint, por número descendente |
| Lista sin Sprints finalizados | 200 | "Aún no hay Sprints anteriores" |
| Detalle de un Sprint finalizado del proyecto | 200 | Encabezado, velocidad y tablas o mensajes |
| `{id}` inválido | 404 | Página de HU-20a, sin consultar la base |
| `{id}` inexistente, de otro proyecto o en curso | 404 | Página de HU-20a, sin llamar a HU-07 |
| Falla del catálogo o de HU-07 | 500 o 503 | Página de HU-20a, sin datos parciales |
| Sin proyecto activo | 303 | `/contexto/proyecto?volver_a=%2Fsprints-anteriores` (o `%2Fsprints-anteriores%2F{id}`) |
| `POST`, `PUT`, `DELETE`… en las dos direcciones | 405 | Página de HU-20a |

## Datos de prueba

Los de "Datos de ejemplo" de [spec.md](spec.md), cargados como los deja HU-15 (research.md §10):

| Proyecto | Sprint | `status` | Registro de cierre (T001: HU-15) | Asignaciones | Historias (estado actual) |
|---|---|---|---|---|---|
| «Alfa» | 1 "Dar de alta usuarios", 18/08–31/08/2026 | Finalizado | «Alta» Hecho 3 | «Alta» | «Alta» Hecho 3 |
| «Alfa» | 2 "Gestionar perfiles y reportes", 01/09–14/09/2026 | Finalizado | «Login» Hecho 5, «Tablero» Hecho 3, «Perfil» Hecho 2, «Reportes» En progreso 8 | «Login», «Tablero», «Perfil» | las tres Hecho; «Reportes» Pendiente 8, sin `active_sprint_id` |
| «Alfa» | 3 "Exportar e informar", 15/09–28/09/2026 | Activo | — | «Exportar», «Ayuda» | «Exportar» En progreso 5; «Ayuda» En Sprint, sin estimar |
| «Beta» | — | — | — | — | — |
| «Gamma» | 1 "Migrar datos", 01/09–14/09/2026 | Finalizado | «Migración» En progreso 5, «Respaldo» En Sprint `NULL` | — | las dos Pendiente |
| «Gamma» | 2 "Auditar accesos", 15/09–28/09/2026 | Finalizado | — (sin filas) | «Auditoría» | «Auditoría» Hecho 2 |

Proyectos: «Alfa» del 18/08/2026 al 30/11/2026; «Beta» del 01/10/2026 al 31/12/2026; «Gamma» del 01/09/2026 al
31/12/2026. Las historias se crean en el orden de la tabla de la spec.
