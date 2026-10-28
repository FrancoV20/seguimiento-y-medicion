# Research: HU-10 – Cálculo de Métricas de Calidad y Avance

## Datos verificados en el código (main `3f85852`, 2026-10-28)

> **Actualización (28/10)**: después de esta verificación entraron a `main` HU-24 (Registro de esfuerzo) y HU-26
> (cookies firmadas). `main.go` ahora carga la clave de las cookies antes de abrir el puerto y registra la sección de
> esfuerzo después de la del Tablero; el bloque de HU-10 va después del último registro. Ninguna de las dos cambia el
> paquete `src/metrics` ni las tablas que lee HU-10.

Verificado con `git grep -n`:

- **`src/metrics` (HU-07)**: archivos `calculation.go`, `closure_validation.go`, `errors.go`, `model.go`,
  `postgres_metrics_store.go`, `service.go`, `store.go`, `test_store.go` y sus pruebas.
  - `roundToTwoDecimals(*big.Rat) *big.Rat` está sin exportar en `calculation.go`. Redondea mitad hacia arriba a
    dos decimales, y `CalculateDeviation` la usa con `FloatString(2)`. HU-10 la reutiliza desde el mismo paquete,
    sin editar el archivo.
  - Textos que usa HU-07: `"No disponible"` cuando no hay base de cálculo; `ErrSprintNotFinished = errors.New("sprint
    is not finished")` en `errors.go`.
  - **No existen** `ErrSprintNotFound`, `QualityStore`, `QualityService`, `SprintQuality` ni nada con el prefijo
    `Quality` en `src/`, `db/` ni `test/`. HU-15 planea su propio `ErrSprintNotFound` en `src/sprint`: está en otro
    paquete, así que no choca.
  - `Service.CalculateMetrics(sprintID)` lee las historias de `sprint_story_assignments`, suma como velocidad los
    Story Points de las "Hecho" y además **guarda** el resultado (`SaveMetrics`, tabla `sprint_metrics`).
  - `ValidateSprintClosure` rechaza las historias "Hecho" sin Story Points o con Story Points inválidos: con el
    cierre de HU-15, ninguna "Hecho" queda sin Story Points.
  - Las interfaces de HU-07 no reciben `context.Context`.
- **HU-06 (defectos)** no está en el código de `main`: no hay `src/defect` ni migración `defects`. Según
  `specs/006-defects/data-model.md` la tabla tiene estas columnas:
  - `id`, `project_id BIGINT`, `story_id`, `description`, `severity`;
  - `status` ('Abierto' o 'Resuelto');
  - `detection_sprint_id BIGINT NOT NULL` y `resolution_sprint_id BIGINT NULL`;
  - un `CHECK` que exige Sprint de resolución si y solo si el estado es "Resuelto".

  Estos nombres se confirman en T001.
- **HU-15 (cierre)** tampoco está en el código de `main`. Según `specs/015-sprint-close/data-model.md`, la tabla
  `sprint_story_results(sprint_id, story_id, final_status, story_points, created_at)` tiene:
  - `PRIMARY KEY (sprint_id, story_id)`;
  - `final_status` entre 'En Sprint', 'En progreso' y 'Hecho';
  - `story_points` que acepta nulos;
  - una fila por historia del Sprint al cerrar, que nunca se actualiza después.

  El cierre deja las "Hecho" asignadas y quita las demás. Estos nombres se confirman en T001.
- **HU-19 (reabrir)**: según `specs/019-defect-resolution`, al reabrir un defecto el estado vuelve a 'Abierto' y
  `resolution_sprint_id` vuelve a `NULL`.
- **Esquema en `main`**:
  - `sprints(id BIGSERIAL, project_id BIGINT, number, goal, start_date, end_date, status IN ('Activo',
    'Finalizado'))`, con `UNIQUE (project_id, number)`;
  - `sprint_metrics` (007).
- **Web (HU-20a/HU-20b)**:
  - `Server.Register(Section{Name, Path, RequiresProject}, Routes)` y `RequireContext`;
  - `ActiveContextFrom`, `Renderer.PageStatus/Page/Fail/RenderError`, `NotFound`, `Unavailable`, `KindOf`;
  - `parseID` (1..`MaxID`), con plantillas embebidas `templates/*.html`;
  - `Server.route` responde 405 con la página de HU-20a si el método no tiene ruta.
- **HU-25** no está en `main`. Su plan usa el prefijo `Metrics` en `src/web` (`MetricsCatalog`, `MetricsSource`,
  `metrics.html`, `/metricas`). HU-10 usa el prefijo `Quality` y no comparte archivos con HU-25. Del aviso de HU-25
  se toma el texto "El Sprint {N} sigue en curso; sus métricas se calculan al cerrarlo".
- **`main.go`** registra cada sección con un bloque (HU-22, HU-21, HU-14 y, desde el 28/10, HU-24). HU-10 suma uno más,
  al final.
- **CI** (`.github/workflows/ci.yml`):
  - `gofmt -l .` sin salida, `go vet ./...`, migraciones `up`;
  - `go test ./... -p 1 -race -coverprofile`;
  - las pruebas de integración hacen `TRUNCATE` de tablas compartidas, así que cada prueba arma sus propios datos.
- **Planificación (PDF)**: HU-10 es Should Have, con valor 75 y 5 SP. Depende de HU-14, HU-15 y HU-19. Los Story
  Points planificados son "la suma de todas las historias del Sprint" y los completados, los de las "Hecho".

## Decisiones

### 1. Paquete `src/metrics`, archivos nuevos

- **Decision**: `quality.go` (tipos, cálculo puro y `ErrSprintNotFound`), `quality_store.go` (puerto),
  `quality_service.go` (servicio) y `postgres_quality_store.go` (adaptador), cada uno con sus pruebas. No se edita
  ningún archivo de HU-07.
- **Rationale**: decisión del equipo. Reutiliza `roundToTwoDecimals` y `ErrSprintNotFinished` del mismo paquete.
- **Alternatives considered**: un paquete `src/quality` (descartada: obliga a exportar o copiar el redondeo de
  HU-07).

### 2. Puerto `QualityStore` con `context.Context`

- **Decision**:

  ```go
  type QualityStore interface {
      LoadSprintQuality(ctx context.Context, projectID, sprintID int64) (QualityData, error)
  }
  ```

- **Rationale**:
  - Es un puerto nuevo y no extiende `MetricsStore`, así que no rompe los fakes de HU-07.
  - Lleva `context`, como los puertos más nuevos (HU-03, HU-04, HU-11).
  - Devuelve datos crudos y el servicio aplica las reglas.
- **Alternatives considered**: ampliar `MetricsStore` (descartada: cambia un archivo de HU-07 y su `test_store.go`).

### 3. Cálculo puro separado del almacenamiento

- **Decision**: `calculateSprintQuality(data QualityData) SprintQuality` en `quality.go`, sin base de datos. El
  servicio valida y la llama.
- **Rationale**: los escenarios 1 a 8 se prueban con tablas en milisegundos y la regla queda en un solo lugar.

### 4. Porcentaje como texto con punto en el dominio y coma en la pantalla

- **Decision**: `CompletionPercentage` es `roundToTwoDecimals(completadas/planificadas×100).FloatString(2)` (por
  ejemplo, `"75.00"`), o `"No disponible"` si no hay historias planificadas. La web lo muestra como "75,00 %".
- **Rationale**:
  - Es el mismo contrato que `DeviationPercentage` de HU-07: texto con punto y "No disponible".
  - HU-08 y HU-09 pueden usar el valor sin volver a parsear una coma.
  - Con `big.Rat` el redondeo es exacto: 2/3 da "66.67" y 1/8 da "12.50".
- **Alternatives considered**: devolver `float64` (descartada: errores de redondeo binario); devolver ya "75,00 %"
  (descartada: mezcla presentación con cálculo).

### 5. Lectura en una transacción de solo lectura

- **Decision**: `BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})`:
  1. Sprint del proyecto: sin fila → `ErrSprintNotFound`.
  2. Si no está "Finalizado", se devuelve solo el estado y no se lee nada más.
  3. Filas de `sprint_story_results`.
  4. Los dos conteos de defectos en una sola consulta con `COUNT(*) FILTER`.
- **Rationale**: R10/FR-015. Con `REPEATABLE READ`, una resolución o reapertura simultánea no puede dejar números
  de dos momentos distintos.
- **Alternatives considered**: tres consultas sueltas (descartada: lectura inconsistente).

### 6. Defectos: filtro por proyecto y dos conteos independientes

- **Decision**:

  ```sql
  SELECT COUNT(*) FILTER (WHERE detection_sprint_id = $1),
         COUNT(*) FILTER (WHERE status = 'Resuelto' AND resolution_sprint_id = $1)
  FROM defects
  WHERE project_id = $2 AND (detection_sprint_id = $1 OR resolution_sprint_id = $1);
  ```

- **Rationale**: es la decisión del equipo. Un defecto detectado y resuelto en el mismo Sprint cuenta en los dos
  conteos. Un defecto reabierto tiene `resolution_sprint_id` nulo y deja de contar como resuelto. El filtro por
  `project_id` es una barrera extra aunque el Sprint ya sea del proyecto.

### 7. Sprint inexistente o de otro proyecto: `ErrSprintNotFound` propio

- **Decision**:
  - `ErrSprintNotFound = errors.New("sprint not found")` en `quality.go`, en inglés como los demás errores de
    `metrics`.
  - Un identificador `<= 0` lo devuelve sin consultar la base.
  - Un Sprint activo devuelve `ErrSprintNotFinished` de HU-07, envuelto con el identificador.
- **Rationale**: decisión del equipo. Verificado que no hay otro `ErrSprintNotFound` en `src/metrics`.

### 8. Coherencia con HU-07

- **Decision**: una prueba de integración prepara un Sprint cerrado como lo deja HU-15 y compara `PointsCompleted`
  con `NewService(NewPostgreSQLMetricsStore(db)).CalculateMetrics(id).Velocity`. El Sprint cerrado tiene:
  - registro de cierre con 3, 5, 8 y 2 Story Points;
  - las de 3 y 5 en "Hecho" y asignadas;
  - las otras devueltas.
- **Rationale**: R6/FR-010. HU-07 suma las "Hecho" asignadas con sus Story Points actuales, y HU-10 suma las "Hecho"
  del registro; coinciden si los Story Points no cambian después del cierre. `CalculateMetrics` escribe en
  `sprint_metrics`, y la prueba lo limpia.
- **Alternatives considered**: que HU-10 llame a HU-07 para sacar los completados (descartada: HU-07 guarda
  resultados y falla con historias sin Story Points; HU-10 tiene que ser de solo lectura).

### 9. Pantalla con catálogo propio y servicio del dominio

- **Decision**:
  - `src/web/quality_catalog.go` define `QualityCatalog` (`QualitySprints(ctx, projectID) (QualitySprintList,
    error)`: Sprints finalizados del proyecto por `number DESC`, más el número del Sprint activo si existe) y
    `QualityReader` (`SprintQuality(ctx, projectID, sprintID) (metrics.SprintQuality, error)`), que implementa
    `*metrics.QualityService`.
  - `postgres_quality_catalog.go` es el adaptador.
  - `quality_view.go` tiene los textos, el formato y la elección del Sprint; `quality_handlers.go`, el handler y
    `RegisterQuality`; `templates/quality.html`, la pantalla.
- **Rationale**:
  - La lista es lectura de pantalla, igual que en HU-14, HU-22 y HU-25.
  - Los números salen del contrato que van a usar HU-08 y HU-09.
  - Una interfaz chica en la web permite simular la falla del escenario 12.
  - El prefijo `Quality` evita chocar con HU-25.

### 10. Elección del Sprint

- **Decision**:
  - `parseID(r.URL.Query().Get("sprint"))`: si el resultado está en la lista de finalizados, se usa; si no, se usa
    el primero de la lista (el más reciente).
  - La lista son enlaces `<a href="/calidad?sprint=ID">Sprint N</a>`, con `aria-current="page"` en el elegido.
- **Rationale**: es la regla de HU-25 y funciona sin JavaScript. Como el valor de la dirección solo se acepta si está
  en la lista, nunca se calcula un Sprint ajeno.

### 11. Sin números parciales

- **Decision**: cualquier error del catálogo o del cálculo, incluso `ErrSprintNotFound` o `ErrSprintNotFinished` por
  una carrera, va a `Renderer.Fail` y no se renderiza nada más.
- **Rationale**: escenario 12 y FR-015. `Fail` distingue el 503 (base caída) del 500.

### 12. Textos

- **Decision**: en `src/web/quality_view.go`:
  - `TitleQuality`: "Calidad y avance";
  - `MsgQualityNoSprints`: "Aún no hay Sprints finalizados";
  - `MsgQualityActiveHint`: "El Sprint %d sigue en curso; sus métricas se calculan al cerrarlo";
  - `MsgQualityNotAvailable`: "No disponible";
  - `qualityWithoutPointsWarning(n)`, en singular ("1 historia sin Story Points no se suma a los planificados") o en
    plural ("N historias sin Story Points no se suman a los planificados");
  - `qualityPercentText` ("75.00" → "75,00 %").
- **Rationale**: FR-019, un solo lugar.

### 13. Registro en `main.go`

- **Decision**: un bloque nuevo, después del último bloque de registro (hoy HU-24), que hace `metrics.NewQualityService(
  metrics.NewPostgreSQLQualityStore(database))`, `web.NewPostgreSQLQualityCatalog(database)` y
  `web.RegisterQuality(srv, service, catalog)`, con el mismo manejo de error que los demás.
- **Rationale**: notas del equipo, punto 8. Si hay conflicto al mergear, se conservan todos los bloques.

### 14. Pruebas con datos propios

- **Decision**: cada prueba de integración crea sus proyectos, Sprints, historias, filas de `sprint_story_results` y
  defectos, y limpia en este orden: defectos, registro de cierre, asignaciones, historias, Sprints y proyectos (las
  historias primero, porque `user_stories.project_id` no tiene cascada).
- **Rationale**: el CI hace `TRUNCATE` y corre con `-p 1`. Los datos de HU-15 y HU-06 se insertan directo, porque las
  pruebas no dependen de sus flujos (notas del equipo, punto 6).

### 15. Rendimiento (SC-005)

- **Decision**: una prueba de handler con 20 Sprints finalizados, 50 historias por Sprint y 100 defectos mide `GET
  /calidad` con un umbral de 500 ms.
- **Rationale**: el volumen supera al de un proyecto del curso. Las consultas usan los índices de `defects`
  (detección y resolución) y la clave de `sprint_story_results`.

## Preguntas abiertas

Ninguna. Los nombres de HU-06 y HU-15 están pendientes de confirmación en T001, no de decisión.
