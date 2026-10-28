# Modelo de datos: HU-09 – Exportación de Reportes en PDF

## Esquema

Sin migración ni tablas nuevas. HU-09 solo lee (la única escritura es la de HU-07 en `sprint_metrics`, research.md
§10). Los nombres marcados con **(T001)** salen de los documentos de HU-06, HU-08, HU-10, HU-15 y HU-17, o del
comportamiento descrito para HU-12, y se confirman en el código cuando esas historias estén en `main`.

| Tabla | Columnas usadas | Quién la lee | Origen |
|---|---|---|---|
| `projects` | `id`, `name`, `start_date`, `end_date` | `report.PostgreSQLStore` | HU-01, en `main` |
| `project_members` | `project_id`, `member_id` | `report.PostgreSQLStore` (reporte del proyecto) | HU-01, en `main` |
| `sprints` | `id`, `project_id`, `number`, `goal`, `start_date`, `end_date`, `status` ('Activo' o 'Finalizado') | `report.PostgreSQLStore`, `PostgreSQLReportCatalog` | HU-04, en `main` |
| `user_stories` | `id`, `title`, `status`, `story_points`, `estimated_hours` | `report.PostgreSQLStore` | HU-02, HU-04 y HU-17 **(T001)**, en `main` |
| `sprint_story_assignments` | `sprint_id`, `story_id` | `report.PostgreSQLStore` (Sprint en curso) | HU-04, en `main` |
| `sprint_story_results` **(T001)** | `sprint_id`, `story_id`, `final_status`, `story_points` | `report.PostgreSQLStore` (Sprint finalizado) | HU-15 |
| `effort_logs` | `story_id`, `member_id`, `hours` | `report.PostgreSQLStore` | HU-03, en `main` |
| `members` | `id`, `display_name` | `report.PostgreSQLStore` | HU-01, en `main` |
| `defects` **(T001)** | `id`, `project_id`, `story_id`, `description`, `severity`, `status`, `detection_sprint_id`, `resolution_sprint_id` | `report.PostgreSQLStore` | HU-06 y HU-19 |
| `sprint_metrics` | — (la escribe HU-07) | HU-07 | en `main` |

## Generador (`src/pdf`, nuevo)

### `document.go`

```go
type Font int

const (
    Regular Font = iota // Helvetica      (/F1)
    Bold                // Helvetica-Bold (/F2)
)

// Column es una columna de tabla: título y ancho en puntos.
type Column struct {
    Title string
    Width float64
}

var (
    ErrTableTooWide   = errors.New("pdf: table columns exceed the page width")
    ErrRowColumnCount = errors.New("pdf: row does not match the table columns")
)

type Document struct { /* páginas con sus operaciones de dibujo; posición actual */ }

func New() *Document                                     // A4 vertical, una página vacía
func (d *Document) Text(font Font, size float64, text string)   // párrafo; corta por ancho; pasa de página
func (d *Document) Space(points float64)                         // espacio vertical
func (d *Document) Table(columns []Column, rows [][]string) error // encabezado repetido en cada página
func (d *Document) PageCount() int
func (d *Document) Bytes(footer func(page, total int) string) ([]byte, error)
```

Reglas:

- Página A4: `/MediaBox [0 0 595.28 841.89]`; márgenes 50 pt (izquierda, derecha y arriba) y 60 pt (abajo); ancho
  útil 495 pt.
- `Text` corta el texto con `wrapText` al ancho útil; interlineado = 1,25 × tamaño. Si la línea no entra, abre una
  página nueva.
- `Table` valida antes de dibujar: suma de anchos > 495 → `ErrTableTooWide`; una fila con otra cantidad de celdas →
  `ErrRowColumnCount`. Dibuja el encabezado en Bold 9 y las filas en Regular 9; cada celda se corta al ancho de su
  columna menos 4 pt de relleno; el alto de la fila es el de la celda con más líneas. Si la fila no entra, abre una
  página y repite el encabezado. Una fila más alta que una página entera se corta entre páginas línea por línea.
- `Bytes` agrega en cada página el texto de `footer(página, total)` en Regular 9, centrado a 30 pt del borde
  inferior, y serializa (research.md §5). Devuelve un error solo si una operación previa dejó un error guardado.

### `winansi.go`

```go
func encodeWinAnsi(s string) []byte // research.md §3
func escapeLiteral(b []byte) []byte // "(" → "\(", ")" → "\)", "\" → "\\"

// Literal devuelve el texto tal como queda dentro de una cadena del PDF (codificado y escapado).
// Lo usan las pruebas para verificar los bytes de la codificación.
func Literal(s string) []byte

// ExtractText devuelve el texto de todas las cadenas "(...) Tj" del documento, decodificado a UTF-8 y unido con un
// espacio, en el orden en que se dibujan (research.md §19). Lo usan las pruebas de src/report, src/web y test/.
func ExtractText(b []byte) (string, error)
```

`ExtractText` devuelve un error si no encuentra la `xref` o un flujo de contenido. Una fila de tabla queda como el
texto de sus celdas unidas con un espacio: "Sprint 1 Finalizado 1 de 1 3 de 3 3 0,00 % 0 detectados y 0 resueltos".

| Entrada | Bytes |
|---|---|
| `"Página"` | `50 E1 67 69 6E 61` |
| `"Año – 5 €"` | `41 F1 6F 20 96 20 35 20 80` |
| `"“Hola” ‘x’ …"` | `93 48 6F 6C 61 94 20 91 78 92 20 85` |
| `"«Alfa» — ok"` | `AB 41 6C 66 61 BB 20 97 20 6F 6B` |
| `"a\tb\nc"` | `61 20 62 20 63` |
| `"😀 Ω"` | `3F 20 3F` |
| `"Login (v2) C:\x"` escapado | `Login \(v2\) C:\\x` |

### `widths.go`

```go
var helveticaWidths, helveticaBoldWidths [256]int // milésimas, métricas AFM estándar

func textWidth(font Font, size float64, encoded []byte) float64
func wrapText(font Font, size, width float64, text string) [][]byte // líneas ya codificadas
```

| Caso | Resultado |
|---|---|
| `textWidth(Regular, 10, "Página")` | 31,13 (667 + 556 + 556 + 222 + 556 + 556 = 3113 milésimas) |
| `textWidth(Bold, 10, "Página")` | 32,79 (667 + 556 + 611 + 278 + 611 + 556 = 3279 milésimas) |
| `wrapText` de un texto que entra | una línea igual al texto |
| `wrapText` de varias palabras más anchas que el ancho | varias líneas, cada una ≤ ancho, sin perder palabras |
| `wrapText` de una palabra más ancha que el ancho | la palabra cortada por caracteres |
| `wrapText("")` | una línea vacía |

### `writer.go`

Serialización de research.md §5: encabezado, objetos numerados, flujos con `/Length` exacto, `xref`, `trailer`,
`startxref` y `%%EOF`.

## Reporte (`src/report`, nuevo)

### `model.go`

```go
var (
    ErrSprintNotFound     = errors.New("report: sprint not found")
    ErrProjectNotFound    = errors.New("report: project not found")
    ErrInconsistentReport = errors.New("report: sprint data changed while reading")
)

type SprintInfo struct {
    ProjectName string
    Number      int
    Goal        string
    StartDate   time.Time
    EndDate     time.Time
    Status      string // "Activo" o "Finalizado"
}

type Story struct {
    ID             int64
    Title          string
    Status         string        // final (registro de cierre) o actual (en curso)
    StoryPoints    *int          // nil = sin estimar
    EstimatedHours *effort.Hours // nil = sin estimar
    ActualHours    effort.Hours  // suma de todos sus registros de esfuerzo
}

type MemberHours struct {
    Name  string // members.display_name
    Hours effort.Hours
}

type Defect struct {
    Description      string
    Severity         string
    Status           string // "Abierto" o "Resuelto"
    StoryTitle       string
    ResolutionSprint int    // número; 0 = sin Sprint de resolución
}

// Data son los datos propios que lee el puerto.
type Data struct {
    Sprint  SprintInfo
    Stories []Story       // por ID ascendente
    Members []MemberHours // por nombre, después por id
    Defects []Defect      // detectados en el Sprint, por id ascendente
}

// SprintReport es el reporte armado; Render lo convierte en PDF.
type SprintReport struct {
    Data        Data
    Partial     bool                   // Sprint en curso
    Metrics     *metrics.SprintMetrics // HU-07; nil si Partial
    Quality     *metrics.SprintQuality // HU-10 (T001); nil si Partial
    GeneratedAt time.Time
}
```

### `project_model.go`

```go
type StoryCounts struct {
    Pending, InSprint, InProgress, Done int // "Pendiente", "En Sprint", "En progreso", "Hecho"
}

type ProjectSprint struct {
    ID     int64
    Number int
    Status string // "Activo" o "Finalizado"
}

type OpenDefect struct {
    Description     string
    Severity        string
    StoryTitle      string
    DetectionSprint int // número
}

// ProjectData son los datos propios del reporte del proyecto que lee el puerto.
type ProjectData struct {
    ID                  int64
    Name                string
    StartDate, EndDate  time.Time
    Members             []string          // display_name de project_members, por nombre
    Stories             StoryCounts       // estado actual de todas las historias del proyecto
    Sprints             []ProjectSprint   // por número ascendente
    EstimatedTotal      effort.Hours      // suma de las horas estimadas informadas
    WithoutEstimate     int               // historias sin horas estimadas
    RegisteredTotal     effort.Hours      // suma de effort_logs de las historias del proyecto
    MemberHours         []MemberHours     // por nombre, después por id
    DefectsOpen         int
    DefectsResolved     int
    OpenDefects         []OpenDefect      // los "Abierto", por id
}

// ProjectSprintRow es una fila de la tabla de Sprints.
type ProjectSprintRow struct {
    Number  int
    Active  bool                   // Sprint en curso: sin Quality ni Metrics
    Quality *metrics.SprintQuality // HU-10 (T001)
    Metrics *metrics.SprintMetrics // HU-07
}

// ProjectReport es el reporte del proyecto armado; RenderProject lo convierte en PDF.
type ProjectReport struct {
    Data        ProjectData
    Status      string                  // HU-12 (T001): "No iniciado", "En curso" o "Finalizado"
    Progress    string                  // HU-12 (T001): porcentaje de avance, "57.14" o "0.00"
    Partial     bool                    // hay un Sprint en curso
    Sprints     []ProjectSprintRow
    Velocity    metrics.VelocitySummary // HU-08 (T001)
    GeneratedAt time.Time
}
```

### `store.go` y `sources.go`

```go
type Store interface {
    // LoadSprintReport devuelve ErrSprintNotFound si el Sprint no existe en el proyecto.
    LoadSprintReport(ctx context.Context, projectID, sprintID int64) (Data, error)
    // LoadProjectReport devuelve ErrProjectNotFound si el proyecto no existe.
    LoadProjectReport(ctx context.Context, projectID int64) (ProjectData, error)
}

type MetricsSource interface { // *metrics.Service (HU-07)
    CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
}

type QualitySource interface { // *metrics.QualityService (HU-10, T001)
    SprintQuality(ctx context.Context, projectID, sprintID int64) (metrics.SprintQuality, error)
}
```

### `service.go`

```go
type Service struct { /* store, metrics, quality, now */ }

func NewService(store Store, metricsSrc MetricsSource, quality QualitySource, now func() time.Time) *Service // now nil = time.Now
func (s *Service) Build(ctx context.Context, projectID, sprintID int64) (SprintReport, error)
func (s *Service) SprintPDF(ctx context.Context, projectID, sprintID int64) ([]byte, error) // Build + Render
```

Pasos de `Build`:

1. Alguna dependencia `nil` → error.
2. `projectID <= 0 || sprintID <= 0` → `ErrSprintNotFound`, sin llamar al almacén.
3. `LoadSprintReport`. `ErrSprintNotFound` se devuelve sin cambios; otro error, con
   `fmt.Errorf("load sprint report %d: %w", sprintID, err)`.
4. `Status == "Finalizado"`:
   - `CalculateMetrics(sprintID)`; error → `fmt.Errorf("sprint metrics %d: %w", …)`;
   - `SprintQuality(ctx, projectID, sprintID)`; error → `fmt.Errorf("sprint quality %d: %w", …)`;
   - `len(Stories) != StoriesPlanned` o `len(Defects) != DefectsDetected` → `ErrInconsistentReport`.
5. Si no: `Partial = true`, sin llamar a HU-07 ni a HU-10.
6. `GeneratedAt = now()`.

### `project_service.go`

```go
func (s *Service) BuildProject(ctx context.Context, projectID int64) (ProjectReport, error)
func (s *Service) ProjectPDF(ctx context.Context, projectID int64) ([]byte, error) // BuildProject + RenderProject
```

Pasos de `BuildProject` (research.md §18):

1. Alguna dependencia `nil` → error; `projectID <= 0` → `ErrProjectNotFound`, sin llamar al almacén.
2. `LoadProjectReport`. `ErrProjectNotFound` sin cambios; otro error, con
   `fmt.Errorf("load project report %d: %w", projectID, err)`.
3. `now := s.now()`. `Status` con la regla de estado de HU-12 **(T001)** (`StartDate`, `EndDate`, `now`); `Progress`
   con la regla de avance de HU-12 **(T001)** (`Stories.Done`, total de historias).
4. Por cada Sprint: "Finalizado" → `SprintQuality(ctx, projectID, id)` y `CalculateMetrics(id)`, errores envueltos
   con `%w`; "Activo" → fila `Active` y `Partial = true`.
5. `Velocity = metrics.SummarizeVelocity(velocidades)` **(T001)**, con `Metrics.Velocity` de los finalizados en orden.
6. `GeneratedAt = now`.

### `texts.go`

| Identificador | Texto o regla |
|---|---|
| `titleReport` | "Reporte del Sprint" |
| `msgPartial` | "Reporte parcial: Sprint en curso" |
| `fmtProject`, `fmtSprint`, `fmtGoal` | "Proyecto: %s", "Sprint %d", "Objetivo: %s" |
| `fmtDates` | "Fechas: %s al %s" (dd/mm/aaaa) |
| `fmtStatus` | "Estado: %s" con "Finalizado" o "En curso" |
| `fmtGenerated` | "Generado: %s" (dd/mm/aaaa hh:mm) |
| `headingStories`, `headingEstimates`, `headingEffort`, `headingMetrics`, `headingDefects` | "Historias", "Estimaciones", "Esfuerzo registrado", "Métricas", "Defectos" |
| columnas de historias | "Historia", "Estado", "Story Points", "Horas estimadas", "Horas reales" |
| columnas de defectos | "Descripción", "Severidad", "Estado", "Historia", "Resuelto en" |
| `msgNotEstimated` | "Sin estimar" |
| `msgReturned` | "%s – volvió al Backlog" (estado final distinto de "Hecho" en un Sprint finalizado) |
| `msgNoStories` | "No hay historias registradas en este Sprint" |
| `fmtProgress` | "%d de %d historias completadas (%s)" con `percentText` |
| `fmtPoints` | "%d Story Points planificados y %d completados" |
| `fmtEstimatedTotal` | "%s horas estimadas en total"; "1 hora estimada en total" |
| `withoutHoursText(n)` | "1 historia sin horas estimadas" / "%d historias sin horas estimadas"; se agrega entre paréntesis solo si n > 0 |
| `fmtRegisteredTotal` | "%s horas registradas en total"; "1 hora registrada en total" |
| `fmtMemberHours` | "%s: %s h" |
| `fmtVelocity` | "Velocidad: %d Story Points" |
| `fmtDeviation` | "Desviación de esfuerzo: %s" con `deviationText` |
| `msgPendingClose` | "Se calculan al cerrar el Sprint" |
| `fmtPendingProgress`, `fmtPendingPoints`, `fmtPendingMetrics` | "Historias completadas: %s", "Story Points planificados y completados: %s", "Velocidad y desviación: %s", con `msgPendingClose` |
| `defectsSummaryText(d, r)` | "No se registraron defectos en este Sprint" si d = r = 0; si no, "<d> y <r> en este Sprint" con "1 defecto detectado"/"%d defectos detectados" y "1 resuelto"/"%d resueltos" |
| `defectsSoFarText(d)` | d = 0: "No se registraron defectos en este Sprint"; "1 defecto detectado hasta el momento"; "%d defectos detectados hasta el momento" |
| `msgNoResolution` | "—" |
| `fmtResolution` | "Sprint %d" |
| `fmtFooter` | "Página %d de %d" |

Textos del reporte del proyecto (mismo archivo):

| Identificador | Texto o regla |
|---|---|
| `titleProjectReport` | "Reporte del proyecto" |
| `msgProjectPartial` | "Reporte parcial: hay un Sprint en curso" |
| `fmtMembers` | "Integrantes: %s" (nombres separados por ", "); sin integrantes: "Integrantes: Sin integrantes" |
| `headingProgress`, `headingSprints` | "Avance", "Sprints" |
| columnas de avance | "Estado", "Historias"; filas "Pendiente", "En Sprint", "En progreso", "Hecho" |
| `fmtProgress` (el mismo del Sprint) | "%d de %d historias completadas (%s)" con `percentText` del porcentaje de HU-12 |
| columnas de Sprints | "Sprint", "Estado", "Historias", "Story Points", "Velocidad", "Desviación", "Defectos" |
| celdas de un finalizado | "Sprint %d", "Finalizado", "%d de %d", "%d de %d", "%d", `deviationText` + " (cálculo parcial)" si `IsPartial`, `sprintDefectsCell(d, r)` |
| `sprintDefectsCell(d, r)` | "%d detectados y %d resueltos"; con 1: "1 detectado", "1 resuelto" |
| celdas de un activo | "Sprint %d", "En curso: se calcula al cerrar el Sprint" y "—" en el resto |
| `msgNoSprints` | "Aún no hay Sprints" |
| `fmtTotalVelocity` | "Velocidad acumulada: %d Story Points" |
| `fmtAverageVelocity` | "Promedio: %s Story Points por Sprint" con `decimalText` ("6.50" → "6,50"); "Promedio: No disponible" sin finalizados |
| `projectDefectsText(open, resolved)` | sin defectos: "No se registraron defectos en este proyecto"; si no, "%d defectos registrados: %s y %s" con "1 defecto registrado", "1 abierto"/"%d abiertos", "1 resuelto"/"%d resueltos" |
| `msgNoOpenDefects` | "No hay defectos abiertos" (hay defectos y ninguno abierto) |
| columnas de abiertos | "Descripción", "Severidad", "Historia", "Detectado en" ("Sprint %d") |

Funciones de formato:

| Función | Entrada → salida |
|---|---|
| `percentText` | "75.00" → "75,00 %"; "No disponible" → igual |
| `deviationText` | "6.25" → "+6,25 %"; "-8.50" → "-8,50 %"; "0.00" → "0,00 %"; "No disponible" → igual |
| `hoursText` | 1700 → "17"; 250 → "2,5"; 25 → "0,25"; 0 → "0" |
| `dateText` | 2026-09-01 → "01/09/2026" |
| `storyStatusText(status, finished)` | ("Hecho", true) → "Hecho"; ("En progreso", true) → "En progreso – volvió al Backlog"; ("En progreso", false) → "En progreso" |
| `decimalText` | "6.50" → "6,50"; "No disponible" → igual |
| `projectDefectsText` | (0, 0) → "No se registraron defectos en este proyecto"; (2, 1) → "3 defectos registrados: 2 abiertos y 1 resuelto"; (1, 0) → "1 defecto registrado: 1 abierto y 0 resueltos"; (0, 2) → "2 defectos registrados: 0 abiertos y 2 resueltos" |
| `sprintDefectsCell` | (0, 0) → "0 detectados y 0 resueltos"; (2, 1) → "2 detectados y 1 resuelto"; (1, 2) → "1 detectado y 2 resueltos" |

### `document.go`

`Render(r SprintReport) ([]byte, error)` arma el `pdf.Document` en el orden de la spec:

1. Título; `msgPartial` en Bold 12 si `Partial`; proyecto, Sprint, objetivo, fechas, estado y generación.
2. "Historias": tabla (o `msgNoStories`); debajo `fmtProgress` (o `fmtPendingProgress`).
3. "Estimaciones": `fmtPoints` (o `fmtPendingPoints`) y `fmtEstimatedTotal` con `withoutHoursText`.
4. "Esfuerzo registrado": `fmtRegisteredTotal` y una línea `fmtMemberHours` por integrante.
5. "Métricas": `fmtVelocity`, `fmtDeviation` y `Metrics.Warning` si `IsPartial`; o `fmtPendingMetrics`.
6. "Defectos": tabla si hay detectados; debajo `defectsSummaryText(Quality.DefectsDetected,
   Quality.DefectsResolved)` o, en curso, `defectsSoFarText(len(Defects))`.
7. `Bytes` con `fmtFooter`.

### `project_document.go`

`RenderProject(r ProjectReport) ([]byte, error)`, en el orden de la spec (research.md §18):

1. `titleProjectReport`; `msgProjectPartial` en Bold 12 si `Partial`; "Proyecto: %s", fechas, "Estado: %s" (HU-12),
   `fmtMembers` y "Generado: %s".
2. "Avance": tabla Estado/Historias con las cuatro filas (con ceros) y `fmtProgress`.
3. "Sprints": tabla con una fila por Sprint (o `msgNoSprints`); si hay Sprints, `fmtTotalVelocity` y
   `fmtAverageVelocity`.
4. "Esfuerzo registrado": `fmtEstimatedTotal` con `withoutHoursText`, `fmtRegisteredTotal` y `fmtMemberHours` por
   integrante (los mismos textos del Sprint).
5. "Defectos": `projectDefectsText`; tabla de abiertos si hay, o `msgNoOpenDefects` si hay defectos y ninguno
   abierto.
6. `Bytes` con `fmtFooter`.

### `postgres_store.go`

`PostgreSQLStore` y `NewPostgreSQLStore(db *sql.DB)`. `BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation:
sql.LevelRepeatableRead})`, `defer tx.Rollback()`:

```sql
-- 1. Encabezado (sin fila → ErrSprintNotFound)
SELECT p.name, s.number, s.goal, s.start_date, s.end_date, s.status
FROM sprints s
JOIN projects p ON p.id = s.project_id
WHERE s.id = $1 AND s.project_id = $2;

-- 2a. Historias de un Sprint 'Finalizado' (T001: nombres de HU-15)
SELECT us.id, us.title, r.final_status, r.story_points,
       ROUND(us.estimated_hours * 100)::bigint,
       COALESCE((SELECT ROUND(SUM(el.hours) * 100)::bigint FROM effort_logs el WHERE el.story_id = us.id), 0)
FROM sprint_story_results r
JOIN user_stories us ON us.id = r.story_id
WHERE r.sprint_id = $1
ORDER BY us.id;

-- 2b. Historias de un Sprint 'Activo'
SELECT us.id, us.title, us.status, us.story_points,
       ROUND(us.estimated_hours * 100)::bigint,
       COALESCE((SELECT ROUND(SUM(el.hours) * 100)::bigint FROM effort_logs el WHERE el.story_id = us.id), 0)
FROM sprint_story_assignments a
JOIN user_stories us ON us.id = a.story_id
WHERE a.sprint_id = $1
ORDER BY us.id;

-- 3. Esfuerzo por integrante; <historias> es la misma fuente del paso 2:
--    (SELECT story_id FROM sprint_story_results WHERE sprint_id = $1) o
--    (SELECT story_id FROM sprint_story_assignments WHERE sprint_id = $1)
SELECT m.display_name, ROUND(SUM(el.hours) * 100)::bigint
FROM effort_logs el
JOIN members m ON m.id = el.member_id
WHERE el.story_id IN <historias>
GROUP BY m.id, m.display_name
ORDER BY m.display_name, m.id;

-- 4. Defectos detectados en el Sprint (T001: nombres de HU-06)
SELECT d.description, d.severity, d.status, us.title, COALESCE(rs.number, 0)
FROM defects d
JOIN user_stories us ON us.id = d.story_id
LEFT JOIN sprints rs ON rs.id = d.resolution_sprint_id
WHERE d.project_id = $2 AND d.detection_sprint_id = $1
ORDER BY d.id;

COMMIT;
```

Cualquier error de lectura se devuelve envuelto (`fmt.Errorf("…: %w", err)`), para que `Fail` distinga la base caída
(503).

`LoadProjectReport`, en otra transacción con las mismas opciones:

```sql
-- 1. Proyecto (sin fila → ErrProjectNotFound)
SELECT id, name, start_date, end_date FROM projects WHERE id = $1;

-- 2. Integrantes
SELECT m.display_name
FROM project_members pm JOIN members m ON m.id = pm.member_id
WHERE pm.project_id = $1
ORDER BY m.display_name, m.id;

-- 3. Historias por estado, horas estimadas y sin estimar
SELECT COUNT(*) FILTER (WHERE status = 'Pendiente'),
       COUNT(*) FILTER (WHERE status = 'En Sprint'),
       COUNT(*) FILTER (WHERE status = 'En progreso'),
       COUNT(*) FILTER (WHERE status = 'Hecho'),
       COALESCE(ROUND(SUM(estimated_hours) * 100)::bigint, 0),
       COUNT(*) FILTER (WHERE estimated_hours IS NULL)
FROM user_stories
WHERE project_id = $1;

-- 4. Sprints
SELECT id, number, status FROM sprints WHERE project_id = $1 ORDER BY number;

-- 5. Esfuerzo por integrante (el total es la suma de las filas)
SELECT m.display_name, ROUND(SUM(el.hours) * 100)::bigint
FROM effort_logs el
JOIN user_stories us ON us.id = el.story_id
JOIN members m ON m.id = el.member_id
WHERE us.project_id = $1
GROUP BY m.id, m.display_name
ORDER BY m.display_name, m.id;

-- 6. Defectos por estado (T001: nombres de HU-06)
SELECT COUNT(*) FILTER (WHERE status = 'Abierto'), COUNT(*) FILTER (WHERE status = 'Resuelto')
FROM defects WHERE project_id = $1;

-- 7. Defectos abiertos
SELECT d.description, d.severity, us.title, ds.number
FROM defects d
JOIN user_stories us ON us.id = d.story_id
JOIN sprints ds ON ds.id = d.detection_sprint_id
WHERE d.project_id = $1 AND d.status = 'Abierto'
ORDER BY d.id;

COMMIT;
```

`RegisteredTotal` es la suma de las filas del paso 5 (cada registro pertenece a una sola historia y a un solo
integrante, así que no hay doble conteo).

## Pantalla (`src/web`)

### `report_catalog.go`

```go
type ReportSprintOption struct {
    ID     int64
    Number int
    Goal   string
    Active bool
}

// ReportCatalog son las lecturas de la pantalla "Reportes" (solo lectura).
type ReportCatalog interface {
    // ReportSprints: el activo primero, después los finalizados por número descendente.
    ReportSprints(ctx context.Context, projectID int64) ([]ReportSprintOption, error)
    // ReportSprint: found = false si no existe en el proyecto.
    ReportSprint(ctx context.Context, projectID, sprintID int64) (ReportSprintOption, bool, error)
}

// ReportGenerator lo implementa *report.Service.
type ReportGenerator interface {
    SprintPDF(ctx context.Context, projectID, sprintID int64) ([]byte, error)
    ProjectPDF(ctx context.Context, projectID int64) ([]byte, error)
}
```

### `postgres_report_catalog.go`

```sql
-- ReportSprints
SELECT id, number, goal, status = 'Activo'
FROM sprints
WHERE project_id = $1
ORDER BY (status = 'Activo') DESC, number DESC;

-- ReportSprint (sin fila → found = false)
SELECT id, number, goal, status = 'Activo'
FROM sprints
WHERE id = $1 AND project_id = $2;
```

Errores envueltos con `%w`, como `PostgreSQLBoardCatalog`.

### `report_view.go`

| Identificador | Texto |
|---|---|
| `TitleReports` | "Reportes" |
| `MsgReportNoSprints` | "Aún no hay Sprints para exportar" |
| `msgReportActiveWarning` | "El Sprint %d sigue en curso: el reporte puede estar incompleto" |
| `LinkReportExport` | "Exportar PDF" |
| `LinkReportGenerateAnyway` | "Generar igual" |
| `LinkReportCancel` | "Cancelar" |
| `ReportStatusActive` / `ReportStatusFinished` | "En curso" / "Finalizado" |
| `msgReportSprint` | "Sprint %d" |
| `LinkReportProject` | "Exportar reporte del proyecto" |
| `msgReportProjectActiveWarning` | "El proyecto tiene un Sprint en curso (Sprint %d): el reporte puede estar incompleto" |

Funciones:

- `reportPDFPath(id int64) string` → "/reportes/sprints/7/pdf".
- `reportConfirmPath(id int64) string` → "/reportes/sprints/7/pdf?confirmar=si".
- `reportFileName(number int) string` → "reporte-sprint-2.pdf".
- `reportProjectPath` = "/reportes/proyecto/pdf"; con confirmación, "/reportes/proyecto/pdf?confirmar=si".
- `reportProjectFileName(id int64) string` → "reporte-proyecto-5.pdf".
- `reportConfirmed(r *http.Request) bool` → solo `r.URL.Query().Get("confirmar") == "si"`.

```go
type reportListData struct {
    Sprints    []reportSprintRow // Label "Sprint 3", Goal, Status, ExportURL
    NoSprints  string            // MsgReportNoSprints si la lista está vacía
    ProjectURL string            // reportProjectPath: el botón está siempre
}

type reportWarningData struct {
    Message     string // msgReportActiveWarning con el número
    GenerateURL string // reportConfirmPath
    CancelURL   string // "/reportes"
}
```

### `report_handlers.go`

`RegisterReports(s *Server, catalog ReportCatalog, generator ReportGenerator) error`:

- si falta algún argumento, devuelve un error sin registrar;
- registra `Section{Name: TitleReports, Path: "/reportes", RequiresProject: true}` con
  `Routes{"GET /reportes": …, "GET /reportes/sprints/{id}/pdf": …, "GET /reportes/proyecto/pdf": …}`.

`GET /reportes`: `ReportSprints`; error → `Fail`; si no, `Page("reports.html")` con 200.

`GET /reportes/sprints/{id}/pdf` (`projectID` de `ActiveContextFrom`):

1. `id := parseID(r.PathValue("id"))`; 0 → `Fail(NotFound(…))`.
2. `ReportSprint`; error → `Fail`; no encontrado → `Fail(NotFound(…))`.
3. Activo y `!reportConfirmed(r)` → `Page("report_warning.html")` con 200.
4. `SprintPDF`; `errors.Is(err, report.ErrSprintNotFound)` → `Fail(NotFound(err))`; otro error → `Fail(err)`.
5. Encabezados `Content-Type: application/pdf`, `Content-Disposition: attachment;
   filename="reporte-sprint-N.pdf"`, `Cache-Control: no-store`, `Content-Length`; `WriteHeader(200)`; `Write`.

Nada se escribe en la respuesta antes del paso 5.

`GET /reportes/proyecto/pdf` (`projectID` de `ActiveContextFrom`):

1. `ReportSprints`; error → `Fail`.
2. Si hay un Sprint con `Active` y `!reportConfirmed(r)` → `Page("report_warning.html")` con 200,
   `msgReportProjectActiveWarning` con su número, "Generar igual" a `/reportes/proyecto/pdf?confirmar=si` y
   "Cancelar" a `/reportes`.
3. `ProjectPDF`; `errors.Is(err, report.ErrProjectNotFound)` → `Fail(NotFound(err))`; otro error → `Fail(err)`.
4. Los mismos encabezados que la descarga de un Sprint, con `filename="reporte-proyecto-N.pdf"`.

### Plantillas

- `templates/reports.html`: el enlace con aspecto de botón "Exportar reporte del proyecto" (siempre) y la tabla con
  "Sprint N", objetivo, estado y el enlace "Exportar PDF" de cada Sprint, o `NoSprints`. Sin formularios ni
  `<script>`.
- `templates/report_warning.html`: el mensaje y dos enlaces, "Generar igual" y "Cancelar"; la usan las dos
  advertencias.

## Respuestas

| Caso | Código | Contenido |
|---|---|---|
| `GET /reportes` con Sprints | 200 | Lista |
| `GET /reportes` sin Sprints | 200 | "Aún no hay Sprints para exportar" |
| PDF de un Sprint finalizado | 200 | `application/pdf`, descarga `reporte-sprint-N.pdf`, `no-store` |
| Sprint en curso sin `confirmar=si` | 200 | Página de advertencia (HTML) |
| Sprint en curso con `confirmar=si` | 200 | PDF parcial |
| PDF del proyecto sin Sprint en curso | 200 | `application/pdf`, descarga `reporte-proyecto-N.pdf`, `no-store` |
| Proyecto con Sprint en curso sin `confirmar=si` | 200 | Página de advertencia (HTML) |
| Proyecto con Sprint en curso con `confirmar=si` | 200 | PDF del proyecto parcial |
| Proyecto borrado mientras se arma el reporte (`ErrProjectNotFound`) | 404 | Página de HU-20a |
| `id` inválido, inexistente o ajeno | 404 | Página de HU-20a |
| Falla de lectura, HU-07, HU-10, coherencia o armado | 500 o 503 | Página de HU-20a, sin bytes del PDF |
| Sin proyecto activo | 303 | `/contexto/proyecto?volver_a=<dirección pedida>` |
| `POST` u otro método | 405 | Página de HU-20a |

## Contratos que se consumen

```go
m, err := metricsSource.CalculateMetrics(sprintID)            // HU-07: Velocity, DeviationPercentage, IsPartial, Warning
q, err := qualitySource.SprintQuality(ctx, projectID, sprintID) // HU-10 (T001): StoriesPlanned, StoriesCompleted,
                                                               // CompletionPercentage, PointsPlanned, PointsCompleted,
                                                               // DefectsDetected, DefectsResolved
v := metrics.SummarizeVelocity(velocities)                     // HU-08 (T001): Total, Average ("6.50")
// HU-12 (T001): estado del proyecto por fechas con la fecha de generación, y porcentaje de avance
// ("57.14"; "0.00" sin historias). Nombres y firmas a confirmar en T001.
```
