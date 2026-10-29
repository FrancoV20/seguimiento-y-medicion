# Investigación: HU-08 – Visualización de Dashboard del Proyecto

## Datos verificados en `main` (28/10, `git grep -n`)

Lo que **está** en el código de `main`:

- **HU-07** (`src/metrics`):
  - `NewService(store MetricsStore) *Service` y `(*Service).CalculateMetrics(sprintID int64) (SprintMetrics, error)`,
    sin `context`;
  - `SprintMetrics` tiene `Velocity int`, `DeviationPercentage string` ("20.00", "-8.50", "0.00" o "No disponible"),
    `IsPartial bool` y `Warning string` ("Cálculo parcial: basado en X de Y historias");
  - `CalculateMetrics` lee las historias de `sprint_story_assignments` con `user_stories.estimated_hours` y la suma de
    `effort_logs.hours`, rechaza un Sprint no finalizado (`ErrSprintNotFinished`), valida el cierre
    (`ValidateSprintClosure`: una "Hecho" sin Story Points da `ClosureValidationError`) y al final guarda con
    `SaveMetrics` (`INSERT … ON CONFLICT (sprint_id) DO UPDATE` en `sprint_metrics`);
  - `roundToTwoDecimals(*big.Rat)` (no exportada): mitad hacia arriba a dos decimales;
  - `NewPostgreSQLMetricsStore(db *sql.DB)`.
- **HU-14** (`src/web/board_catalog.go`):
  - `BoardCatalog.ActiveBoard(ctx, projectID) (BoardView, found bool, err error)`;
  - `BoardView` tiene `Number int64`, `Label` ("Sprint N"), `Groups` (En Sprint, En progreso y Hecho, en ese orden),
    `PlannedPoints` y `CompletedPoints` (las sin estimar cuentan 0);
  - `NewPostgreSQLBoardCatalog(database)`; los errores se devuelven envueltos, sin `Unavailable`.
- **HU-20a / HU-20b** (`src/web`): `Server.Register(Section, Routes)`, que protege con `RequireContext`;
  `Section{Name, Path, RequiresProject, RequiresMember}`; `Renderer.Page`, `PageStatus`, `RenderError` y `Fail` (404,
  503 si `IsDatabaseUnavailable`, 500 en otro caso); `ActiveContextFrom`; `projectIDFrom(r)` en
  `board_handlers.go`; el 405 de `Server.route`; plantillas embebidas con `templates/*.html` y `templateFuncs`.
- **Esquema**:
  - `user_stories.status` con `CHECK (status IN ('Pendiente', 'En Sprint', 'En progreso', 'Hecho'))` (migración 004)
    y `estimated_hours NUMERIC(6,2)` (migración 002);
  - `story_points` solo admite 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89;
  - `sprints.status` 'Activo' o 'Finalizado', `UNIQUE (project_id, number)`;
  - `effort_logs.story_id` y `effort_logs.member_id` **sin** borrado en cascada; `hours` entre 0 (excluido) y 24.
- **`main.go`** registra cada sección con un bloque; el último hoy es el de HU-24 (`RegisterEffort`).
- **CI** (`.github/workflows/ci.yml`): `gofmt -l .` sin salida, `go vet ./...`, migraciones y
  `go test ./... -p 1 -race -coverprofile=coverage.out`.
- No existe ningún nombre `Dashboard*`, `dashboard*`, `VelocitySummary` ni `SummarizeVelocity` en `src` ni en `test`.

Lo que **no está** todavía en `main` (se toma de sus documentos y se confirma en T001):

- **HU-06** (`specs/006-defects/data-model.md`): `defects(project_id, story_id, status 'Abierto'|'Resuelto',
  detection_sprint_id NOT NULL, resolution_sprint_id NULL)`, con un `CHECK` que exige `resolution_sprint_id` nulo
  si está 'Abierto' y no nulo si está 'Resuelto'.
- **HU-15** (`specs/015-sprint-close/data-model.md`): `sprint_story_results(sprint_id, story_id, final_status,
  story_points)`; al cerrar, las "Hecho" quedan asignadas y las demás vuelven al Backlog como "Pendiente".
- **HU-10** (`specs/010-quality-metrics/data-model.md`): `metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db))`,
  `SprintQuality(ctx, projectID, sprintID) (metrics.SprintQuality, error)` con `PointsCompleted int`, y en `src/web`
  la interfaz `QualityReader` con ese método.
- **HU-25** (`specs/025-web-metrics/data-model.md`): el formato de la desviación vive en `src/web/metrics_view.go`
  ("+20,00 %", "-8,50 %", "0,00 %"); el nombre exacto de la función no está en sus documentos.

## Decisiones

### 1. Paquetes y prefijo

- **Decision**: la regla de velocidad acumulada y promedio va en `src/metrics/velocity_summary.go` (archivo nuevo);
  todo lo de pantalla va en `src/web` con el prefijo `Dashboard`/`dashboard`.
- **Rationale**: el promedio es una regla del dominio y usa `roundToTwoDecimals` de HU-07, que solo se ve dentro de
  `src/metrics`. El prefijo evita choques con HU-10 (`Quality`) y HU-25 (`Metrics`).
- **Alternatives considered**: calcular el promedio en la vista (mezcla dominio y pantalla y obliga a copiar el
  redondeo); exportar `roundToTwoDecimals` (edita un archivo de HU-07).

### 2. `SummarizeVelocity` recibe solo números

- **Decision**: `SummarizeVelocity(points []int) VelocitySummary`, con `Total`, `SprintCount` y `Average` como texto
  con punto ("10.50") o "No disponible" sin Sprints, igual que `CalculateDeviation` de HU-07.
- **Rationale**: es una función pura, fácil de probar con tabla y sin base; el texto con punto mantiene la
  convención de HU-07 y HU-10, y la coma se pone en la pantalla.
- **Alternatives considered**: devolver `*big.Rat` (la vista tendría que redondear) o `float64` (errores de
  redondeo en casos como 0,125).

### 3. Reutilizar `QualityReader` de HU-10

- **Decision**: el handler recibe un `QualityReader` (de HU-10, mismo paquete `web`), implementado por
  `*metrics.QualityService`. No se define otra interfaz.
- **Rationale**: HU-10 tiene que estar en `main` para empezar (T001), y el tipo ya tiene la forma pedida; redefinirlo
  sería duplicar.
- **Alternatives considered**: una interfaz propia `DashboardQualityReader` (duplica); leer `sprint_story_results`
  directamente (recalcula lo que HU-10 ya hace).

### 4. Interfaz chica para la desviación

- **Decision**: `DashboardDeviationSource` con `CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)`,
  que `*metrics.Service` cumple sin adaptador.
- **Rationale**: permite simular fallas en las pruebas de pantalla sin tocar HU-07.
- **Alternatives considered**: depender de `*metrics.Service` (no se puede simular una falla); un adaptador que
  traduzca tipos (código sin valor agregado).

### 5. Catálogo propio con una lectura consistente

- **Decision**: `DashboardCatalog.DashboardSummary(ctx, projectID)` lee, en una transacción `ReadOnly` con
  `REPEATABLE READ`, las cuatro cantidades de historias (un `COUNT(*) FILTER` por estado), los defectos abiertos y la
  lista de Sprints finalizados (`id`, `number`, por número ascendente).
- **Rationale**: las tres lecturas son del proyecto y salen coherentes entre sí; es el único puerto nuevo de la web
  y tiene su adaptador PostgreSQL (principio VI).
- **Alternatives considered**: usar `QualityCatalog.QualitySprints` de HU-10 para la lista (otra lectura aparte que
  además trae el Sprint activo y viene en orden inverso); contar historias con el Tablero (solo ve el Sprint activo).

### 6. Orden de las lecturas y proyecto vacío

- **Decision**: 1) `DashboardSummary`; 2) `ActiveBoard`; 3) si no hay historias, ni Sprints finalizados, ni Sprint
  activo, se muestra el mensaje único sin pedir nada más; 4) `SprintQuality` por cada Sprint finalizado; 5)
  `CalculateMetrics` del último. Cualquier error corta el armado y va a `Fail`.
- **Rationale**: no hay defectos sin historia (`defects.story_id NOT NULL`), así que "sin historias y sin Sprints"
  ya implica "sin defectos". No se llama a HU-10 ni a HU-07 si no hay Sprints finalizados.
- **Alternatives considered**: pedir todo en paralelo (más complejo, y un error igual descarta todo).

### 7. Gráficos SVG con `html/template`

- **Decision**: barras en un `<svg role="img" aria-label="…">` dentro de la plantilla, con una `<table>` al lado
  con los mismos valores. Las alturas las calcula `dashboardBarHeights(values []int, maxHeight int) []int`:
  `alto = redondeo mitad hacia arriba de valor × maxHeight ÷ máximo`, con mínimo 1 para valores mayores que 0, y
  todos 0 si el máximo es 0. Alto del área de barras: 120.
- **Rationale**: sin JavaScript ni dependencias; `html/template` escapa los textos también dentro del SVG. La tabla
  sirve a lectores de pantalla y a las pruebas. El mínimo 1 evita que un valor real parezca 0.
- **Alternatives considered**: barras con `<div>` y ancho en CSS en línea (no es un gráfico vectorial y depende de
  estilos); una biblioteca de gráficos (dependencia nueva); imágenes generadas (más código y sin accesibilidad).

### 8. Formato de la desviación

- **Decision**: si HU-25 ya está en `main` con una función de formato de la desviación, se reutiliza; si no, se
  crea `dashboardDeviationText` en `dashboard_view.go`: positivo con "+", cero y negativo sin agregar signo, punto
  por coma y " %"; "No disponible" tal cual. T001 decide y lo anota acá.
- **Rationale**: evita dos formatos distintos para el mismo número (principio II), sin bloquear HU-08 si HU-25 no
  llegó.
- **Alternatives considered**: editar HU-25 para exportar su función (prohibido por el plan).

### 9. Aviso de cálculo parcial

- **Decision**: si `SprintMetrics.IsPartial`, se muestra `SprintMetrics.Warning` tal cual debajo de la desviación
  (respuesta de `/speckit-clarify`). Incluye el caso "basado en 0 de 1 historias" con "No disponible".
- **Rationale**: es el mismo aviso que muestra HU-25 con el mismo cálculo; el texto ya viene de HU-07.
- **Alternatives considered**: ocultarlo cuando la desviación es "No disponible" (contradice "tal cual").

### 10. La escritura de HU-07 en `sprint_metrics`

- **Decision**: se acepta que `CalculateMetrics` vuelva a escribir `sprint_metrics` del último Sprint en cada
  apertura, con los mismos valores, como ya pasa en HU-25. HU-08 no escribe nada propio.
- **Rationale**: es la forma de reutilizar el cálculo de HU-07 sin editarlo; el resultado de un Sprint finalizado no
  cambia mientras no cambien sus datos.
- **Alternatives considered**: leer `sprint_metrics` directamente (puede no existir la fila o estar
  desactualizada); recalcular la desviación en la web (duplica HU-07).

### 11. Errores

- **Decision**: los adaptadores devuelven los errores envueltos con `%w` (como `PostgreSQLBoardCatalog`), y el handler
  llama a `Fail`, que elige 503 con `IsDatabaseUnavailable` o 500. El Sprint de la desviación sale de la lista del
  propio proyecto, así que `ErrSprintNotFinished` o un Sprint ajeno no pueden aparecer; si aparecieran, también van
  a `Fail`.
- **Rationale**: misma convención que HU-14; ningún detalle técnico llega a la página.
- **Alternatives considered**: devolver `Unavailable` para todo error de base (daría 503 también ante errores de
  consulta).

### 12. Prueba de coherencia con HU-07 y HU-10

- **Decision**: `TestHU08_CoherenciaConHU07` comprueba que, en los Sprints 1 y 2 de «Alfa», el valor de cada barra
  es igual a `SprintQuality(...).PointsCompleted` y a `CalculateMetrics(id).Velocity`. La prueba borra las filas de
  `sprint_metrics` que deja HU-07.
- **Rationale**: SC-002; detecta si los datos de prueba no reflejan un cierre real.

### 13. Datos de prueba propios

- **Decision**: cada prueba de integración crea sus proyectos («Alfa», «Beta» y los de los escenarios vacíos), sus
  Sprints, historias, registro de cierre, asignaciones, esfuerzo, integrante y defectos, y los borra al final en
  este orden: defectos, registro de cierre, asignaciones, `sprint_metrics`, esfuerzo, historias, Sprints, proyectos
  e integrante.
- **Rationale**: el orden que pidió el equipo, con dos agregados obligatorios: `effort_logs` y `members` no borran
  en cascada, y el escenario 7 necesita horas reales. Corre con `-p 1` porque el CI comparte la base.
- **Alternatives considered**: `TRUNCATE` (pisa datos de otras pruebas).

### 14. Bloque en `main.go`

- **Decision**: un bloque nuevo al final de los registros (hoy después del de HU-24, o del de HU-10 si ya está),
  que construye el catálogo, el Tablero, `metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(database))` y
  `metrics.NewService(metrics.NewPostgreSQLMetricsStore(database))`, y llama a `web.RegisterDashboard`. Si falla,
  cierra `ln` y `database`, como los demás.
- **Rationale**: no edita el bloque de HU-10 (el servicio de calidad no guarda estado, así que una segunda instancia
  no cambia nada).

### 15. Rendimiento

- **Decision**: se acepta una llamada a `SprintQuality` por Sprint finalizado (con 20 Sprints, unas 60 consultas) más
  una a `CalculateMetrics`, el catálogo y el Tablero. `TestHU08_Rendimiento` lo mide con 20 Sprints finalizados, 200
  historias y 100 defectos: menos de 500 ms.
- **Rationale**: reutilizar HU-10 vale más que ahorrar consultas en proyectos del curso.
- **Alternatives considered**: una consulta que sume todos los Sprints de una vez (recalcula HU-10 por fuera).
