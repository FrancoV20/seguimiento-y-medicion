# Modelo de datos: HU-10 – Cálculo de Métricas de Calidad y Avance

## Esquema

Sin migración ni tablas nuevas. HU-10 solo lee. Los nombres marcados con **(T001)** salen de los documentos de HU-06 y
HU-15 y se confirman en el código cuando esas historias estén en `main`.

| Tabla | Columnas usadas | Origen |
|---|---|---|
| `sprints` | `id`, `project_id`, `number`, `status` ('Activo' o 'Finalizado') | HU-04, en `main` |
| `sprint_story_results` **(T001)** | `sprint_id`, `story_id`, `final_status`, `story_points` | HU-15 |
| `defects` **(T001)** | `project_id`, `status` ('Abierto' o 'Resuelto'), `detection_sprint_id`, `resolution_sprint_id` | HU-06 y HU-19 |

## Dominio (`src/metrics`)

### `quality.go`

```go
var ErrSprintNotFound = errors.New("sprint not found") // nuevo; no existe otro en src/metrics

type QualityStoryResult struct {
    FinalStatus string // "Hecho", "En progreso" o "En Sprint"
    StoryPoints *int   // nil = sin estimar
}

// QualityData son los datos crudos que lee el puerto.
type QualityData struct {
    SprintID        int64
    Status          string // estado del Sprint
    Stories         []QualityStoryResult
    DefectsDetected int
    DefectsResolved int
}

// SprintQuality es el resultado del cálculo; lo consumen la web, HU-08 y HU-09.
type SprintQuality struct {
    SprintID             int64
    StoriesPlanned       int
    StoriesCompleted     int
    CompletionPercentage string // "75.00" o "No disponible"
    PointsPlanned        int
    PointsCompleted      int
    StoriesWithoutPoints int
    DefectsDetected      int
    DefectsResolved      int
}

func calculateSprintQuality(data QualityData) SprintQuality
```

Reglas de `calculateSprintQuality`:

- `StoriesPlanned = len(Stories)`; `StoriesCompleted` cuenta las de `FinalStatus == "Hecho"`.
- `CompletionPercentage`:
  - `"No disponible"` si `StoriesPlanned == 0`;
  - si no, `roundToTwoDecimals(big.NewRat(completadas*100, planificadas)).FloatString(2)` (`roundToTwoDecimals` es la
    función de HU-07).
- `PointsPlanned` suma los `StoryPoints` no nulos. `PointsCompleted` suma los no nulos de las "Hecho".
- `StoriesWithoutPoints` cuenta los `StoryPoints == nil`, incluidas las "Hecho" sin estimación (casos límite).
- `DefectsDetected` y `DefectsResolved` se copian sin cambios.

### `quality_store.go`

```go
type QualityStore interface {
    // LoadSprintQuality devuelve ErrSprintNotFound si el Sprint no existe en el proyecto.
    // Si el Sprint no está "Finalizado", devuelve solo SprintID y Status.
    LoadSprintQuality(ctx context.Context, projectID, sprintID int64) (QualityData, error)
}
```

### `quality_service.go`

```go
type QualityService struct{ store QualityStore }

func NewQualityService(store QualityStore) *QualityService

func (s *QualityService) SprintQuality(ctx context.Context, projectID, sprintID int64) (SprintQuality, error)
```

1. `s.store == nil` → error (como `Service` de HU-07).
2. `projectID <= 0 || sprintID <= 0` → `ErrSprintNotFound`, sin llamar al almacén.
3. `LoadSprintQuality`. Un `ErrSprintNotFound` se devuelve sin cambios; cualquier otro error se envuelve con `%w`,
   para que la web pueda distinguir "base caída".
4. `Status != "Finalizado"` → `fmt.Errorf("%w: %d", ErrSprintNotFinished, sprintID)` (`ErrSprintNotFinished` es el
   de HU-07).
5. `calculateSprintQuality(data)`.

No guarda nada.

## Lectura (`postgres_quality_store.go`)

`BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})`, `defer tx.Rollback()`:

```sql
-- 1. Sprint del proyecto (sin fila → ErrSprintNotFound)
SELECT status FROM sprints WHERE id = $1 AND project_id = $2;

-- 2. Solo si status = 'Finalizado': registro de cierre (T001: nombres de HU-15)
SELECT final_status, story_points
FROM sprint_story_results
WHERE sprint_id = $1
ORDER BY story_id;

-- 3. Solo si status = 'Finalizado': defectos (T001: nombres de HU-06)
SELECT COUNT(*) FILTER (WHERE detection_sprint_id = $1),
       COUNT(*) FILTER (WHERE status = 'Resuelto' AND resolution_sprint_id = $1)
FROM defects
WHERE project_id = $2
  AND (detection_sprint_id = $1 OR resolution_sprint_id = $1);

COMMIT;
```

Cualquier error de lectura se devuelve envuelto (`fmt.Errorf("load sprint quality %d: %w", ...)`).

## Pantalla (`src/web`)

### `quality_catalog.go`

```go
type QualitySprintOption struct {
    ID     int64
    Number int
}

type QualitySprintList struct {
    Finished     []QualitySprintOption // number DESC
    ActiveNumber int                   // 0 = no hay Sprint activo
}

type QualityCatalog interface {
    QualitySprints(ctx context.Context, projectID int64) (QualitySprintList, error)
}

// QualityReader lo implementa *metrics.QualityService.
type QualityReader interface {
    SprintQuality(ctx context.Context, projectID, sprintID int64) (metrics.SprintQuality, error)
}
```

### `postgres_quality_catalog.go`

```sql
SELECT id, number, status
FROM sprints
WHERE project_id = $1 AND status IN ('Activo', 'Finalizado')
ORDER BY number DESC;
```

Las filas "Finalizado" van a `Finished`; la "Activo" (como mucho una, FIX-HU-04) va a `ActiveNumber`. Si falla la
base, devuelve `Unavailable`.

### Modelo de vista (`quality_view.go`)

```go
type qualityPageData struct {
    Sprints    []qualitySprintLink // Number, URL ("/calidad?sprint=ID"), Current
    ActiveHint string              // MsgQualityActiveHint con el número, o ""
    NoSprints  string              // MsgQualityNoSprints, o ""
    Selected   *qualityView        // nil si no hay Sprints finalizados
}

type qualityView struct {
    SprintNumber     int
    StoriesText      string // "3 de 4 historias completadas"
    Percentage       string // "75,00 %" o "No disponible"
    PointsPlanned    int
    PointsCompleted  int
    PointsWarning    string // "" si StoriesWithoutPoints == 0
    DefectsDetected  int
    DefectsResolved  int
}
```

- `selectQualitySprint(raw string, list QualitySprintList) (QualitySprintOption, bool)` usa `parseID(raw)`. Si el
  valor está en `Finished`, elige ese Sprint; si no, el primero. Devuelve `false` si la lista está vacía.
- `qualityPercentText("75.00")` devuelve `"75,00 %"`, y `"No disponible"` queda igual.
- `qualityWithoutPointsWarning(n)` devuelve el singular con `n == 1` y el plural con `n > 1`.

### Textos (`quality_view.go`)

| Constante | Texto |
|---|---|
| `TitleQuality` | "Calidad y avance" |
| `MsgQualityNoSprints` | "Aún no hay Sprints finalizados" |
| `MsgQualityActiveHint` | "El Sprint %d sigue en curso; sus métricas se calculan al cerrarlo" |
| `MsgQualityNotAvailable` | "No disponible" |
| `msgQualityWithoutPointsOne` | "1 historia sin Story Points no se suma a los planificados" |
| `msgQualityWithoutPointsMany` | "%d historias sin Story Points no se suman a los planificados" |
| `msgQualityStories` | "%d de %d historias completadas" |

### Handler (`quality_handlers.go`)

`RegisterQuality(s *Server, reader QualityReader, catalog QualityCatalog) error` registra
`Section{Name: "Calidad y avance", Path: "/calidad", RequiresProject: true}` con `Routes{"GET /calidad": ...}`.
Devuelve un error si alguno de sus argumentos es `nil`, como `RegisterBacklog`.

1. `active := ActiveContextFrom(ctx)`. `RequireContext` ya garantiza el proyecto.
2. `catalog.QualitySprints`. Si falla, `Fail` y nada más.
3. Sin finalizados: 200 con `NoSprints` y `ActiveHint`.
4. `selectQualitySprint`, y después `reader.SprintQuality(ctx, projectID, elegido.ID)`. Si falla, `Fail`; no se
   muestran números parciales.
5. 200 con la lista, `ActiveHint` y `Selected`.

| Caso | Respuesta |
|---|---|
| Con finalizados | 200 |
| Sin finalizados | 200 con `MsgQualityNoSprints` |
| `sprint` inválido, activo, inexistente o ajeno | 200 con el más reciente |
| Falla del catálogo o del cálculo | `Fail`: 500, o 503 si la base no responde |
| Sin proyecto | 303 de HU-20b |
| `POST` u otro método | 405 de HU-20a |
