# Modelo de datos: HU-08 – Visualización de Dashboard del Proyecto

## Esquema

Sin migración ni tablas nuevas. HU-08 solo lee (la única escritura es la de HU-07 en `sprint_metrics`, research.md
§10). Los nombres marcados con **(T001)** salen de los documentos de HU-06, HU-10, HU-15 y HU-25 y se confirman en el
código cuando esas historias estén en `main`.

| Tabla | Columnas usadas | Quién la lee | Origen |
|---|---|---|---|
| `user_stories` | `project_id`, `status` | `DashboardCatalog` | HU-02 y HU-04, en `main` |
| `sprints` | `id`, `project_id`, `number`, `status` ('Activo' o 'Finalizado') | `DashboardCatalog`, Tablero, HU-10, HU-07 | HU-04, en `main` |
| `defects` **(T001)** | `project_id`, `status` ('Abierto' o 'Resuelto') | `DashboardCatalog` | HU-06 |
| `sprint_story_assignments`, `effort_logs` | — (las lee HU-07 y el Tablero) | HU-07, HU-14 | en `main` |
| `sprint_story_results` **(T001)** | — (la lee HU-10) | HU-10 | HU-15 |
| `sprint_metrics` | — (la escribe HU-07) | HU-07 | en `main` |

## Dominio (`src/metrics/velocity_summary.go`, nuevo)

```go
// VelocitySummary resume la velocidad de los Sprints finalizados de un proyecto.
type VelocitySummary struct {
    SprintCount int
    Total       int    // suma de los Story Points completados
    Average     string // "10.50", o "No disponible" si no hay Sprints
}

func SummarizeVelocity(points []int) VelocitySummary
```

Reglas:

- `SprintCount = len(points)`; `Total` = suma de `points`.
- `Average`:
  - "No disponible" si `SprintCount == 0`;
  - si no, `roundToTwoDecimals(big.NewRat(int64(Total), int64(SprintCount))).FloatString(2)` (`roundToTwoDecimals`
    es la función de HU-07, mitad hacia arriba).
- No guarda nada ni lee la base. No se edita ningún archivo existente de `src/metrics`.

Casos de la prueba de tabla:

| `points` | `Total` | `Average` |
|---|---|---|
| `[]` | 0 | "No disponible" |
| `[5]` | 5 | "5.00" |
| `[8, 13]` | 21 | "10.50" |
| `[21, 21, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5]` | 92 | "7.67" (92 ÷ 12 = 7,666…) |
| `[1, 0, 0, 0, 0, 0, 0, 0]` | 1 | "0.13" (0,125) |
| `[2, 0, 0]` | 2 | "0.67" |
| `[0, 0]` | 0 | "0.00" |

## Pantalla (`src/web`)

### `dashboard_catalog.go`

```go
// DashboardStoryCounts son las historias del proyecto por estado.
type DashboardStoryCounts struct {
    Pending    int // "Pendiente"
    InSprint   int // "En Sprint"
    InProgress int // "En progreso"
    Done       int // "Hecho"
}

// DashboardSprint es un Sprint finalizado del proyecto.
type DashboardSprint struct {
    ID     int64
    Number int
}

type DashboardSummary struct {
    Stories     DashboardStoryCounts
    OpenDefects int
    Finished    []DashboardSprint // todos los finalizados, number ASC
}

// DashboardCatalog lee los datos propios del Dashboard (solo lectura).
type DashboardCatalog interface {
    DashboardSummary(ctx context.Context, projectID int64) (DashboardSummary, error)
}

// DashboardDeviationSource lo implementa *metrics.Service (HU-07).
type DashboardDeviationSource interface {
    CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
}
```

Interfaces que se reutilizan sin redefinir:

- `BoardCatalog` (HU-14, `board_catalog.go`): `ActiveBoard(ctx, projectID) (BoardView, bool, error)`.
- `QualityReader` **(T001)** (HU-10, `quality_catalog.go`): `SprintQuality(ctx, projectID, sprintID)
  (metrics.SprintQuality, error)`.

### `postgres_dashboard_catalog.go`

`PostgreSQLDashboardCatalog` y `NewPostgreSQLDashboardCatalog(db *sql.DB)`. `BeginTx(ctx, &sql.TxOptions{ReadOnly:
true, Isolation: sql.LevelRepeatableRead})`, `defer tx.Rollback()`:

```sql
-- 1. Historias por estado
SELECT COUNT(*) FILTER (WHERE status = 'Pendiente'),
       COUNT(*) FILTER (WHERE status = 'En Sprint'),
       COUNT(*) FILTER (WHERE status = 'En progreso'),
       COUNT(*) FILTER (WHERE status = 'Hecho')
FROM user_stories
WHERE project_id = $1;

-- 2. Defectos abiertos (T001: nombres de HU-06)
SELECT COUNT(*) FROM defects WHERE project_id = $1 AND status = 'Abierto';

-- 3. Sprints finalizados
SELECT id, number FROM sprints
WHERE project_id = $1 AND status = 'Finalizado'
ORDER BY number;

COMMIT;
```

Cualquier error se devuelve envuelto (`fmt.Errorf("dashboard summary %d: %w", projectID, err)`), como
`PostgreSQLBoardCatalog`; `Fail` decide 500 o 503.

### `dashboard_chart.go`

```go
const (
    dashboardChartHeight = 120 // alto del área de barras
    dashboardBarWidth    = 40
    dashboardBarGap      = 16
)

// dashboardBarHeights devuelve el alto de cada barra, proporcional al mayor valor.
func dashboardBarHeights(values []int, maxHeight int) []int

type dashboardBar struct {
    Label  string // "Pendiente" o "Sprint 3"
    Value  int
    X, Y   int    // esquina superior izquierda dentro del SVG
    Height int
}

type dashboardChart struct {
    AriaLabel string // descripción accesible con todos los valores
    Width     int    // len(Bars) × (ancho + separación) + separación
    Height    int    // dashboardChartHeight + 30 (margen y etiquetas)
    Bars      []dashboardBar
}

func newDashboardChart(title string, labels []string, values []int) dashboardChart
```

Reglas de `dashboardBarHeights`:

- Lista vacía → lista vacía.
- Máximo 0 → todos 0 (no se divide).
- Si no, `alto = (valor × maxHeight × 2 + máximo) ÷ (2 × máximo)` (división entera: mitad hacia arriba), y un valor
  mayor que 0 nunca da menos de 1.

| `values` (maxHeight 120) | Alturas |
|---|---|
| `[2, 1, 2, 3]` | `[80, 40, 80, 120]` |
| `[8, 13]` | `[74, 120]` (8 × 120 ÷ 13 = 73,85) |
| `[0, 0, 0, 0]` | `[0, 0, 0, 0]` |
| `[1, 1000]` | `[1, 120]` (0,12 sube a 1) |
| `[]` | `[]` |

Geometría: la barra `i` va en `X = separación + i × (ancho + separación)`, `Y = 10 + dashboardChartHeight − alto`; la
etiqueta, debajo de la línea base. `AriaLabel`: "Historias por estado: Pendiente 2, En Sprint 1, En progreso 2,
Hecho 3" y "Velocidad por Sprint: Sprint 1, 8 Story Points; Sprint 2, 13 Story Points".

### `dashboard_view.go`

Textos (únicos en el paquete; ninguno existe hoy):

| Identificador | Texto |
|---|---|
| `TitleDashboard` | "Dashboard" |
| `MsgDashboardNoData` | "Aún no hay datos suficientes para calcular métricas" |
| `MsgDashboardNoActiveSprint` | "No hay un Sprint en curso" |
| `msgDashboardActiveSprint` | "Sprint %d en curso" |
| `msgDashboardSprintPoints` | "%d Story Points planificados y %d completados" |
| `msgDashboardTotalVelocity` | "Velocidad acumulada: %d Story Points" |
| `msgDashboardAverageVelocity` | "Promedio: %s Story Points por Sprint" |
| `MsgDashboardLastTenSprints` | "Se muestran los últimos 10 Sprints finalizados" |
| `msgDashboardDeviation` | "Desviación de esfuerzo del Sprint %d: %s" |
| `MsgDashboardOneOpenDefect` | "1 defecto abierto" |
| `msgDashboardOpenDefects` | "%d defectos abiertos" |
| `MsgDashboardNoOpenDefects` | "No hay defectos abiertos" |
| `titleDashboardStoriesChart` | "Historias por estado" |
| `titleDashboardVelocityChart` | "Velocidad por Sprint" |

Funciones:

- `dashboardDecimalText("10.50")` → "10,50".
- `dashboardDeviationText` **(T001: o la de HU-25)**: "20.00" → "+20,00 %"; "-8.50" → "-8,50 %"; "0.00" → "0,00 %";
  "No disponible" sin cambios.
- `dashboardOpenDefectsText(n)`: 0 → `MsgDashboardNoOpenDefects`; 1 → `MsgDashboardOneOpenDefect`; n →
  `msgDashboardOpenDefects`.
- `buildDashboardView(summary, board, found, velocities, deviation) dashboardView`.

```go
type dashboardVelocityRow struct {
    Label  string // "Sprint 1"
    Points int
}

type dashboardView struct {
    NoData bool // proyecto sin historias, sin Sprints y sin defectos: solo MsgDashboardNoData

    StoriesChart dashboardChart
    StoryRows    []dashboardBar // misma lista que el gráfico, para la tabla

    HasActiveSprint bool
    ActiveSprint    string // "Sprint 3 en curso"
    ActiveCounts    [3]int // En Sprint, En progreso, Hecho (len de BoardView.Groups)
    ActivePoints    string // "11 Story Points planificados y 2 completados"

    HasFinished    bool          // false: bloques de velocidad y desviación con MsgDashboardNoData
    VelocityChart  dashboardChart // últimos 10, number ASC
    VelocityRows   []dashboardVelocityRow
    LastTenNotice  string         // MsgDashboardLastTenSprints si hay más de 10, si no vacío
    TotalVelocity  string         // "Velocidad acumulada: 21 Story Points"
    AverageVelocity string        // "Promedio: 10,50 Story Points por Sprint"
    Deviation      string         // "Desviación de esfuerzo del Sprint 2: +20,00 %"
    PartialWarning string         // SprintMetrics.Warning si IsPartial, si no vacío

    OpenDefects string // "3 defectos abiertos"
}
```

Reglas del armado:

- `NoData` = historias en 0, `Finished` vacío y sin Sprint activo. No hace falta mirar los defectos: todo defecto
  tiene una historia del proyecto (`story_id NOT NULL`, HU-06), así que sin historias no puede haber defectos.
- Las barras de velocidad son las de los últimos 10 de `Finished` (por número), en orden ascendente; `Total` y
  `Average` usan todos los de `Finished`.
- El Sprint de la desviación es el último de `Finished`.

### `dashboard_handlers.go`

`RegisterDashboard(s *Server, catalog DashboardCatalog, board BoardCatalog, quality QualityReader, deviation
DashboardDeviationSource) error`:

- si falta algún argumento, devuelve un error sin registrar;
- registra `Section{Name: TitleDashboard, Path: "/dashboard", RequiresProject: true}` con
  `Routes{"GET /dashboard": …}`.

Pasos de `GET /dashboard` (`projectIDFrom(r)` de HU-14):

1. `catalog.DashboardSummary(ctx, projectID)`. Error → `Fail`.
2. `board.ActiveBoard(ctx, projectID)`. Error → `Fail`.
3. Si es `NoData`, renderizar solo el mensaje (200).
4. Por cada `Finished`: `quality.SprintQuality(ctx, projectID, id)` → `PointsCompleted`. Error → `Fail`.
5. Si hay finalizados: `deviation.CalculateMetrics(último.ID)`. Error → `Fail`.
6. `metrics.SummarizeVelocity(velocidades)`, `buildDashboardView` y `Page(w, r, "dashboard.html", …)` con 200.

Nada se escribe en la respuesta antes del paso 6.

### `templates/dashboard.html`

- Mensaje único si `NoData`.
- Si no, cinco bloques en este orden: historias por estado (SVG + tabla), Sprint en curso, velocidad (SVG + tabla,
  acumulada, promedio y aviso de los últimos 10), desviación (con aviso parcial) y defectos abiertos. Sin Sprints
  finalizados, los bloques de velocidad y desviación se reemplazan por `MsgDashboardNoData`.
- `<svg role="img" aria-label="{{.AriaLabel}}" viewBox="0 0 {{.Width}} {{.Height}}">` con un `<rect>` y un `<text>`
  por barra.
- Sin `<script>` ni formularios.

## Respuestas

| Caso | Código | Contenido |
|---|---|---|
| Con datos | 200 | Cinco bloques |
| Proyecto sin datos | 200 | Solo "Aún no hay datos suficientes para calcular métricas" |
| Falla de lectura o de cálculo | 500 o 503 | Página de error de HU-20a |
| Sin proyecto activo | 303 | `/contexto/proyecto?volver_a=%2Fdashboard` |
| `POST` u otro método | 405 | Página de HU-20a |

## Contrato que se consume de HU-10 **(T001)**

```go
q, err := quality.SprintQuality(ctx, projectID, sprintID) // q.PointsCompleted
```

`SprintQuality` devuelve `ErrSprintNotFinished` o `ErrSprintNotFound` solo para Sprints activos o ajenos; HU-08
nunca los pide, y si aparecieran van a `Fail`.
