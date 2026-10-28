# Investigación: HU-09 – Exportación de Reportes en PDF

## Datos verificados en `main` (`d234b5a`, 28/10, `git grep -n`)

Lo que **está** en el código de `main`:

- **HU-07** (`src/metrics`):
  - `NewService(store MetricsStore) *Service` y `(*Service).CalculateMetrics(sprintID int64) (SprintMetrics, error)`,
    sin `context`;
  - `SprintMetrics` tiene `Velocity int`, `DeviationPercentage string` ("6.25", "-8.50", "0.00" o "No disponible"),
    `IsPartial bool` y `Warning string` ("Cálculo parcial: basado en %d de %d historias");
  - `PostgreSQLMetricsStore.LoadSprintData` lee las historias de `sprint_story_assignments` con
    `user_stories.estimated_hours` y `SUM(effort_logs.hours)` de **todos** los registros de cada historia (`LEFT JOIN
    effort_logs AS el ON el.story_id = us.id`); una historia sin registros queda con horas reales nulas;
  - `calculateSprintMetrics` suma como velocidad los Story Points de las "Hecho" y, para la desviación, usa solo las
    "Hecho" con horas estimadas **y** reales; si usa menos de las que hay, marca `IsPartial` y arma `Warning`;
  - `CalculateMetrics` rechaza un Sprint no finalizado (`ErrSprintNotFinished`), valida el cierre
    (`ValidateSprintClosure`: una "Hecho" sin Story Points da `ClosureValidationError`) y al final guarda con
    `SaveMetrics` en `sprint_metrics`;
  - los errores de lectura se envuelven con `metricsStoreError`, que implementa `Unwrap`: `errors.As` llega al error
    del driver;
  - `roundToTwoDecimals` no está exportada.
- **HU-03 / HU-24**: `effort.Hours` (`src/effort/hours.go`) son centésimas de hora en un `int64`, con `String()` sin
  ceros sobrantes ("17", "2.5", "0.25"); `web.FormatEffortHours` cambia el punto por coma, pero está en `src/web`.
  `effort.ParseHours` rechaza 0 y más de 24 horas: sirve para un registro, no para totales.
- **HU-20a / HU-20b** (`src/web`):
  - `Server.Register(Section, Routes)` protege cada ruta con `RequireContext(section)`; `Routes` admite patrones con
    valores (`"GET /proyectos/{id}"`);
  - `Section{Name, Path, RequiresProject, RequiresMember}`;
  - `ActiveContextFrom(ctx)` (y `projectIDFrom(r)` en `board_handlers.go`, que lo usa);
  - `Renderer.Page`, `PageStatus`, `RenderError` y `Fail` (404 con `KindNotFound`, 503 con `KindUnavailable` o
    `IsDatabaseUnavailable`, 500 en otro caso); `NotFound(err)`;
  - `parseID(s)` devuelve 0 para vacío, no numérico, cero, negativo, con signo, con ceros a la izquierda o mayor que
    `MaxID` (`math.MaxInt32`);
  - `Server.route` responde 405 con la página de HU-20a si el método no tiene ruta;
  - `returnDestination` vuelve, después de elegir proyecto, a la misma dirección pedida con `GET`;
  - `RegisterEffort(s, actions, catalog, now func() time.Time)` ya usa un reloj inyectable (nil = `time.Now`).
- **Esquema**:
  - `projects(id SERIAL, name, start_date, end_date, status DEFAULT 'Activo')` con `end_date > start_date`;
    `members(id SERIAL, display_name)`; `project_members(project_id, member_id)` con borrado en cascada;
  - `sprints(id BIGSERIAL, project_id, number, goal, start_date, end_date, status IN ('Activo', 'Finalizado'))` con
    `UNIQUE (project_id, number)` y `end_date > start_date`;
  - `user_stories(id, project_id, title, story_points IN (1, 2, 3, 5, 8, …), status IN ('Pendiente', 'En Sprint',
    'En progreso', 'Hecho'), estimated_hours NUMERIC(6,2) > 0, active_sprint_id)`; `user_stories.project_id` sin
    cascada;
  - `sprint_story_assignments(sprint_id, story_id)` con `UNIQUE (story_id)`;
  - `effort_logs(story_id, member_id, work_date, activity, hours NUMERIC(5,2) entre 0 excluido y 24)`, **sin**
    cascada en `story_id` ni en `member_id`;
  - `sprint_metrics` (007).
- **`main.go`** registra cada sección con un bloque; el último hoy es el de HU-24 (`RegisterEffort`).
- **CI** (`.github/workflows/ci.yml`): `gofmt -l .` sin salida, `go vet ./...`, migraciones y
  `go test ./... -p 1 -race -coverprofile=coverage.out`, con PostgreSQL 16.
- **`go.mod`**: `go 1.27.1`; única dependencia directa `github.com/jackc/pgx/v5`.
- No existe ningún paquete `src/report` ni `src/pdf`, ni nombres `Report*`/`report*` en `src/web`, ni las rutas
  `/reportes`.

Lo que **no está** todavía en `main` (se toma de sus documentos y se confirma en T001):

- **HU-06** (`specs/006-defects/data-model.md`): `defects(id, project_id, story_id NOT NULL, description, severity IN
  ('Crítica', 'Alta', 'Media', 'Baja'), status IN ('Abierto', 'Resuelto'), detection_sprint_id NOT NULL,
  resolution_sprint_id NULL)`, con un `CHECK` que exige Sprint de resolución si y solo si está "Resuelto".
- **HU-10** (`specs/010-quality-metrics/data-model.md`): `metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db))`
  y `SprintQuality(ctx, projectID, sprintID) (metrics.SprintQuality, error)` con `StoriesPlanned`,
  `StoriesCompleted`, `CompletionPercentage` ("75.00" o "No disponible"), `PointsPlanned`, `PointsCompleted`,
  `DefectsDetected` y `DefectsResolved`; `metrics.ErrSprintNotFound` y `ErrSprintNotFinished` para Sprints ajenos o
  activos.
- **HU-15** (`specs/015-sprint-close/data-model.md`): `sprint_story_results(sprint_id, story_id, final_status IN ('En
  Sprint', 'En progreso', 'Hecho'), story_points)`; al cerrar, las "Hecho" quedan asignadas y las demás vuelven a
  "Pendiente" con `active_sprint_id` nulo y sin asignación.
- **HU-17** (`specs/017-story-estimates/data-model.md`): no crea tablas; edita `user_stories.estimated_hours` y la
  bloquea para historias asignadas a un Sprint finalizado.
- **HU-08** (`specs/008-dashboard/data-model.md`): `metrics.SummarizeVelocity(points []int) metrics.VelocitySummary`
  con `SprintCount`, `Total` y `Average` ("6.50" o "No disponible" sin Sprints), en `src/metrics/velocity_summary.go`.
- **Actualización (28/10)**: HU-12 ya tiene sus md (`specs/012-project-status`). Su contrato es
  `project.StatusOn(start, end, today time.Time) project.DateStatus` y `project.ProgressPercentage(done, total int)
  string`, en `src/project/status.go`; la regla de estado recibe la fecha, así que la prueba con reloj fijo sirve.
- **HU-12** (al escribir esta spec) no tenía documentos en `specs/` ni código en `main` (`git grep -n "HU-12"` solo la nombra en otras
  specs). Lo que se sabe:
  - calcula el estado del proyecto con sus fechas: "No iniciado", "En curso" o "Finalizado"; el primer y el último
    día son "En curso" (`specs/001-fix-project-dates/spec.md`, `specs/021-web-projects/spec.md`);
  - `projects.status` vale 'Activo' por defecto y no se usa: el estado se calcula (`specs/013-backlog-edit`);
  - el avance "X de Y historias completadas (P %)" da "0,00 %" sin historias (escenario 18 del Issue de HU-09).

  Los nombres de sus funciones en `src/project`, si reciben la fecha de referencia y el formato del porcentaje se
  confirman en T001.
- **"Planificación de proyecto.pdf"** no está en el repositorio. Según el equipo, su DoD de la entrega final dice "El
  sistema exporta el informe del proyecto en PDF": por eso HU-09 incluye el reporte del proyecto. El resto no se pudo
  contrastar; si aparece otro conflicto, prevalece la planificación.

## Decisiones

### 1. Paquetes nuevos: `src/pdf`, `src/report` y prefijo `Report` en `src/web`

- **Decision**:
  - `src/pdf`: generador de PDF genérico, sin conocer el dominio.
  - `src/report`: modelos `SprintReport` y `ProjectReport`, puerto `Store` (uno solo, para los dos reportes) con su
    adaptador PostgreSQL, servicio y composición de los dos documentos.
  - `src/web`: catálogo de la pantalla, vista, handlers y plantillas, todo con el prefijo `Report`/`report`.
- **Rationale**: decisión del equipo. El generador queda reutilizable (lo usan los dos reportes) y probado aparte; el reporte no depende de la web; el prefijo evita chocar con HU-08 (`Dashboard`),
  HU-10 (`Quality`) y HU-25 (`Metrics`).
- **Alternatives considered**: todo en `src/web` (mezcla HTTP con armado de documentos); el reporte dentro de
  `src/metrics` (obliga a tocar el paquete de HU-07 y HU-10 y mezcla cálculo con presentación).

### 2. PDF sin dependencias externas

- **Decision**: generador propio en `src/pdf` con la biblioteca estándar: PDF 1.4, A4 (595,28 × 841,89 pt),
  fuentes estándar Type1 Helvetica y Helvetica-Bold sin incrustar, `WinAnsiEncoding`, contenido sin comprimir y
  tabla `xref` con desplazamientos exactos.
- **Rationale**: la constitución pide que cualquier cambio de tecnología se documente como actualización de la
  gobernanza antes de adoptarse. Un generador propio con la biblioteca estándar no agrega tecnología y cumple la
  constitución v1.1.0 sin enmendarla. Lo que hace falta (texto, tablas, saltos de página) es acotado.
- **Alternatives considered**: `github.com/go-pdf/fpdf` (descartada: es una dependencia nueva; usarla exige primero
  una enmienda de la constitución, y el alcance no lo justifica); generar HTML y "imprimir a PDF" con un navegador
  sin interfaz (dependencia externa y proceso aparte); comprimir el contenido con `compress/zlib` (posible con la
  biblioteca estándar, pero impide verificar los textos en las pruebas sin descomprimir; se deja para más adelante si
  el tamaño molesta).

### 3. Codificación WinAnsi y escape

- **Decision**: `encodeWinAnsi(s string) []byte` recorre el texto como runas:
  - U+0020–U+007E y U+00A0–U+00FF van directo (Latin-1 coincide con WinAnsi en ese rango): acentos, "ñ", "«»", "°";
  - nueve extras de WinAnsi: "€" 0x80, "…" 0x85, "‘" 0x91, "’" 0x92, "“" 0x93, "”" 0x94, "–" 0x96, "—" 0x97;
  - tabulación, retorno y salto de línea se convierten en espacio;
  - cualquier otra runa (controles, emojis, otros alfabetos, UTF-8 inválido) se reemplaza por "?".

  Después se escapan `(`, `)` y `\` con una barra invertida. Cada línea se escribe como cadena literal `(…) Tj`.
- **Rationale**: las fuentes estándar con `WinAnsiEncoding` muestran bien el español sin incrustar fuentes. El
  reemplazo por "?" hace visible lo que no se puede mostrar sin romper el archivo.
- **Alternatives considered**: cadenas hexadecimales (`<…>`), que no necesitan escape pero son ilegibles en las
  pruebas; incrustar una fuente TrueType con Unicode (mucho más código y una fuente que distribuir).

### 4. Anchos de Helvetica y corte de líneas

- **Decision**: `src/pdf/widths.go` tiene dos tablas de 256 anchos (en milésimas del tamaño de letra), una por fuente,
  copiadas de las métricas AFM estándar de Helvetica y Helvetica-Bold e indexadas por el byte WinAnsi.
  `textWidth(font, size, s)` suma los anchos del texto ya codificado. `wrapText` corta por palabras para que cada
  línea quepa en el ancho dado; una palabra más ancha que el ancho se corta por caracteres. Un texto vacío ocupa una
  línea vacía.
- **Rationale**: sin anchos reales, un título largo se saldría de la columna o de la página (pedido del equipo). Las
  métricas AFM son las que usan los lectores para dibujar las fuentes estándar.
- **Alternatives considered**: ancho promedio fijo por carácter (corta mal y deja huecos); truncar con "…" (pierde
  información en un documento que se archiva).

### 5. Diseño en dos pasadas: páginas primero, bytes después

- **Decision**: `Document` acumula operaciones de dibujo por página mientras se agregan títulos, párrafos y tablas;
  cuando una línea o una fila no entra, abre una página nueva y, si está dentro de una tabla, repite su encabezado.
  `Bytes(footer)` recién al final conoce el total de páginas, agrega el pie de cada una y serializa:
  1. `%PDF-1.4` y una línea de comentario con cuatro bytes mayores que 127 (indica contenido binario);
  2. objeto 1 catálogo, 2 árbol de páginas (`/Kids`, `/Count`), 3 Helvetica y 4 Helvetica-Bold (`/Type /Font
     /Subtype /Type1 /Encoding /WinAnsiEncoding`);
  3. por cada página, un objeto `/Page` (`/MediaBox [0 0 595.28 841.89]`, recursos `/F1` y `/F2`) y su flujo de
     contenido con `/Length` exacto;
  4. `xref` con una entrada de 20 bytes por objeto (`%010d 00000 n \n`), `trailer << /Size N /Root 1 0 R >>`,
     `startxref` y `%%EOF`.

  La salida es determinista: los mismos datos dan los mismos bytes (no se escribe fecha de creación en el PDF; la
  fecha de generación va en el texto).
- **Rationale**: "Página X de Y" exige conocer Y antes de escribir. Contar los bytes al serializar da los
  desplazamientos exactos de la `xref`. La salida determinista permite pruebas exactas.
- **Alternatives considered**: escribir directo en el `ResponseWriter` (no se puede numerar "de Y" ni garantizar que
  no salgan bytes ante una falla); objetos de formulario para el pie (más complejo sin necesidad).

### 6. Medidas del documento

- **Decision**: márgenes de 50 pt a los lados y arriba, y 60 pt abajo; ancho útil 495 pt. Título en Helvetica-Bold
  16, subtítulos en Helvetica-Bold 12, texto en Helvetica 10 y tablas en Helvetica 9 (encabezado en Bold 9), con
  interlineado de 1,25 veces el tamaño. Pie "Página X de Y" en Helvetica 9, centrado a 30 pt del borde inferior.
  Columnas (suman 495):
  - historias: Historia 165, Estado 135, Story Points 55, Horas estimadas 70, Horas reales 70;
  - defectos: Descripción 170, Severidad 60, Estado 60, Historia 120, Resuelto en 85.
- **Rationale**: entra una fila por línea en el caso normal y deja lugar al estado más largo ("En progreso – volvió
  al Backlog"). Una tabla cuyas columnas suman más que el ancho útil es un error de armado (`ErrTableTooWide`).

### 7. Puerto `report.Store` con una sola lectura

- **Decision**:

  ```go
  type Store interface {
      LoadSprintReport(ctx context.Context, projectID, sprintID int64) (Data, error)
      LoadProjectReport(ctx context.Context, projectID int64) (ProjectData, error)
  }
  ```

  `PostgreSQLStore` implementa cada método en una transacción `ReadOnly` con `REPEATABLE READ`:
  - `LoadSprintReport`: encabezado (Sprint del proyecto con el nombre del proyecto; sin fila → `ErrSprintNotFound`),
    historias (del registro de cierre si está "Finalizado", de las asignaciones si está "Activo"), esfuerzo por
    integrante y defectos detectados;
  - `LoadProjectReport`: proyecto (sin fila → `ErrProjectNotFound`), integrantes, historias por estado, Sprints
    (`id`, `number`, `status` por número), horas estimadas y sin estimar, horas registradas y por integrante, conteo
    de defectos por estado y lista de abiertos.
- **Rationale**: decisión del equipo (un solo puerto para los dos reportes); principio VI (puerto con adaptador
  PostgreSQL y prueba de integración). Las partes propias de cada reporte salen coherentes entre sí.
- **Alternatives considered**: reutilizar el catálogo de HU-17 o el de HU-06 (otras lecturas aparte, de otro paquete,
  y no traen el registro de cierre); varias lecturas sueltas (datos de dos momentos distintos).

### 8. Horas como centésimas exactas

- **Decision**: las consultas devuelven `ROUND(x * 100)::bigint` para `estimated_hours` y para las sumas de
  `effort_logs.hours`; en Go se guardan como `effort.Hours`. Los totales se suman con `Add`. El texto se arma con
  `Hours.String()` cambiando el punto por coma ("17", "2,5").
- **Rationale**: reutiliza el tipo de HU-03, que ya suma sin errores de punto flotante y tiene el formato sin ceros
  sobrantes. No hace falta interpretar texto decimal (`effort.ParseHours` rechaza totales mayores que 24).
- **Alternatives considered**: `*big.Rat` como HU-07 (más conversión sin ganancia); `float64` (errores de redondeo).

### 9. Fuentes de métricas con interfaces chicas

- **Decision**: en `src/report`:

  ```go
  type MetricsSource interface { // *metrics.Service (HU-07)
      CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
  }
  type QualitySource interface { // *metrics.QualityService (HU-10, T001)
      SprintQuality(ctx context.Context, projectID, sprintID int64) (metrics.SprintQuality, error)
  }
  ```

- **Rationale**: los dos servicios las cumplen sin adaptador; las pruebas simulan fallas con dobles. No se edita
  ningún archivo de HU-07 ni de HU-10.
- **Reglas de HU-08 y HU-12**: `metrics.SummarizeVelocity` (HU-08) y las funciones de estado y avance de HU-12 en
  `src/project` son funciones puras: el servicio las llama directo, sin interfaces. Si T001 encuentra que la de estado
  usa la hora del sistema sin recibir una fecha, se pide a HU-12 una variante que la reciba o, si no llega a tiempo,
  la prueba compara contra lo que devuelva HU-12 en el mismo momento en lugar de un texto fijo (riesgo en plan.md).
- **Alternatives considered**: reutilizar `web.QualityReader` de HU-10 (está en `src/web`, que no puede importar
  `src/report` sin ciclo al revés); depender de los tipos concretos (no se pueden simular fallas).

### 10. Orden del armado y Sprint en curso

- **Decision**: `Service.Build(ctx, projectID, sprintID)`:
  1. identificadores `<= 0` → `ErrSprintNotFound` sin consultar;
  2. `LoadSprintReport`; `ErrSprintNotFound` se devuelve tal cual, otro error se envuelve con `%w`;
  3. si el Sprint está "Finalizado": `CalculateMetrics` (HU-07) y `SprintQuality` (HU-10); cualquier error se
     envuelve y corta;
  4. si está "Activo": no se llama a HU-07 ni a HU-10 y el reporte queda `Partial` (respuesta de
     `/speckit-clarify`: avance, Story Points y métricas dicen "Se calculan al cerrar el Sprint");
  5. `GeneratedAt = now()`.

  `SprintPDF` llama a `Build` y después a `Render`.
- **Rationale**: HU-07 y HU-10 rechazan Sprints activos; no llamarlos evita convertir un caso normal en error. Las
  dos llamadas dependen del estado leído en el paso 2.
- **Alternatives considered**: llamar siempre y tratar `ErrSprintNotFinished` como "en curso" (usa un error para el
  flujo normal y HU-07 escribiría antes de rechazar si cambiara su orden interno).

### 11. Control de coherencia con HU-10

- **Decision**: en un Sprint finalizado, si `len(Data.Stories) != q.StoriesPlanned` o `len(Data.Defects) !=
  q.DefectsDetected`, `Build` devuelve `ErrInconsistentReport` (500).
- **Rationale**: la lectura propia y la de HU-10 son transacciones distintas. En un Sprint cerrado esos dos números
  no deberían cambiar (el registro de cierre no se modifica y no se registran defectos en un Sprint cerrado); si
  difieren, hubo un cambio de datos en el medio o un error, y es mejor no entregar un PDF cuyos números no cierran
  (FR-017). Los resueltos no se comparan: pueden incluir defectos detectados antes.
- **Alternatives considered**: reintentar una vez (más código para un caso raro); no comparar (riesgo de un PDF
  incoherente sin aviso).

### 12. Textos y formatos en un solo lugar

- **Decision**: `src/report/texts.go` reúne las constantes y funciones de texto del PDF:
  - fechas `02/01/2006` y fecha y hora `02/01/2006 15:04`;
  - estado del Sprint: "Finalizado" o "En curso";
  - estado de la historia: el del registro; si no es "Hecho" en un Sprint finalizado, "<estado> – volvió al
    Backlog";
  - `percentText("75.00")` → "75,00 %"; "No disponible" igual;
  - `deviationText`: "6.25" → "+6,25 %", "-8.50" → "-8,50 %", "0.00" → "0,00 %", "No disponible" igual;
  - `hoursText(effort.Hours)` → "17", "2,5";
  - singular y plural de horas ("1 hora", "N horas"), de historias sin horas y de defectos.

  En `src/web/report_view.go` van los textos de la pantalla (lista, advertencia y enlaces).
- **Rationale**: FR-021. El formato de la desviación de HU-25 y HU-08 vive en `src/web`, que `src/report` no puede
  importar; se escribe uno propio de pocas líneas con la misma regla (si T001 encuentra uno exportado fuera de
  `src/web`, se reutiliza y se anota acá).
- **Alternatives considered**: mover el formato de HU-25 a un paquete común (edita archivos de otra historia).

### 13. Pantalla y descarga

- **Decision**:
  - `ReportCatalog` (en `src/web`) con `ReportSprints(ctx, projectID)` (el activo primero, después los finalizados
    por número descendente) y `ReportSprint(ctx, projectID, sprintID)` (un Sprint del proyecto o "no encontrado");
    adaptador `PostgreSQLReportCatalog`; errores envueltos con `%w`, como `PostgreSQLBoardCatalog`.
  - `ReportGenerator` con `SprintPDF(ctx, projectID, sprintID) ([]byte, error)`, que cumple `*report.Service`.
  - `GET /reportes/sprints/{id}/pdf`: `parseID(r.PathValue("id"))`; 0 → `Fail(NotFound(…))` sin consultar.
    `ReportSprint`; no encontrado → 404. Activo sin `confirmar=si` → `report_warning.html` con 200. Si no,
    `SprintPDF`; `report.ErrSprintNotFound` (Sprint borrado en el medio) → 404; otro error → `Fail`. Con los bytes en
    mano: `Content-Type: application/pdf`, `Content-Disposition: attachment; filename="reporte-sprint-N.pdf"`,
    `Cache-Control: no-store`, `Content-Length`, 200 y el cuerpo.
- **Rationale**: decisiones del equipo. La advertencia necesita el estado antes de generar, sin pagar la lectura
  completa; la lista y la búsqueda son lecturas de pantalla, como en HU-10, HU-14 y HU-22. El nombre del archivo solo
  lleva dígitos, así que no hace falta escaparlo.
- **Alternatives considered**: `PathID` de HU-20a (acepta ceros a la izquierda y valores mayores que `MaxID`; el
  equipo pidió `parseID`); un formulario `POST` para "Generar igual" (no hace falta: la descarga no cambia datos).

  Reporte del proyecto: `GET /reportes/proyecto/pdf`, con el proyecto de `ActiveContextFrom` (no hay `id` en la
  dirección). Se pide `ReportSprints` al catálogo; si hay un Sprint activo y no viene `confirmar=si`, se muestra la
  misma plantilla de advertencia con el texto del proyecto y `/reportes/proyecto/pdf?confirmar=si`. Si no,
  `ProjectPDF`; `report.ErrProjectNotFound` (proyecto borrado en el medio) → 404; otro error → `Fail`. Encabezados
  iguales, con `filename="reporte-proyecto-N.pdf"`.
- **Rationale (proyecto)**: decisión del equipo. Reutilizar `ReportSprints` evita otra consulta para saber si hay un
  Sprint activo y su número; el proyecto sale del contexto, así que no hay identificador que validar.

### 14. Sin proyecto y `HEAD`

- **Decision**: se acepta el comportamiento de HU-20b: sin proyecto, 303 a elegir proyecto con regreso a la misma
  dirección (`/reportes` o la de la descarga). Si después se elige otro proyecto, la descarga responde 404. `HEAD`
  sobre la descarga genera el PDF y responde los mismos encabezados sin cuerpo (el servidor descarta el cuerpo).
- **Rationale**: no se cambia HU-20b; los dos casos son raros y no exponen datos ajenos.

### 15. Datos de prueba propios

- **Decision**: cada paquete con pruebas de integración (`src/report`, `src/web`, `test`) arma sus datos con un
  ayudante local:
  - «Alfa» (18/08/2026 al 30/11/2026, integrantes Ana y Beto) con los Sprints 1, 2 y 3 y las historias, registros y
    defectos de "Datos de ejemplo" de la spec, armados como los deja el cierre de HU-15 (registro de cierre, "Hecho"
    asignadas, las demás en "Pendiente" sin asignación);
  - «Beta» (01/10/2026 al 31/12/2026, integrante Ana) sin historias ni Sprints (escenarios 11 y 18);
  - «Delta» con su Sprint 1 finalizado, una historia «D1» "Hecho" de 2 Story Points con 1 hora estimada, un registro
    de 1 hora de «Caro» (que no es integrante de «Delta») y un defecto "Abierto": da el Sprint de otro proyecto del
    escenario 12, el integrante que ya no está en el proyecto y datos que no tienen que aparecer en «Alfa»;
  - «Gamma» con los Sprints 1 y 2 finalizados, una historia "Hecho" de 1 Story Point en cada uno, sin registros de
    esfuerzo, y un defecto detectado en el 1 y resuelto en el 2 (en el 2: "0 defectos detectados y 1 resuelto en este
    Sprint"; en el 1: "1 defecto detectado y 0 resueltos en este Sprint"), más su Sprint 3 finalizado sin registro de
    cierre ni historias ("0 de 0 historias completadas (No disponible)");
  - «Largo» con su Sprint 1 finalizado: 120 historias (90 "Hecho" de 2 Story Points con 4 horas estimadas y un
    registro de 4 horas de Ana; 30 que volvieron al Backlog, de 3 Story Points y 4 horas estimadas, sin registros),
    una de ellas con un título de 300 caracteres, y 50 defectos detectados en ese Sprint, 20 resueltos en él.
    Cuentas: "90 de 120 historias completadas (75,00 %)", 270 Story Points planificados (90 × 2 + 30 × 3) y 180
    completados, 480 horas estimadas (120 × 4), 360 registradas (Ana 360 h), velocidad 180, desviación (360 − 360) ÷
    360 = "0,00 %" sin aviso, "50 defectos detectados y 20 resueltos en este Sprint".

  Integrantes «Ana», «Beto» y «Caro». Limpieza en `t.Cleanup`, en este orden: defectos, registro de cierre,
  asignaciones, `sprint_metrics`, `effort_logs`, historias, Sprints, proyectos (con `project_members` en cascada) e
  integrantes.
- **Rationale**: pedido del equipo; `effort_logs` y `user_stories.project_id` no borran en cascada. Cada escenario
  tiene sus propios Sprints: ningún Sprint se usa para dos repartos distintos. Corre con `-p 1` porque el CI
  comparte la base.
- **Alternatives considered**: `TRUNCATE` (pisa datos de otras pruebas).

### 16. Rendimiento

- **Decision**: `TestHU09_Rendimiento` pide por HTTP el PDF del Sprint 1 de «Largo» y el PDF del proyecto «Largo»
  (catálogo, lectura, HU-07, HU-10, HU-08, HU-12, armado), y exige menos de 1 s para cada uno. Los dos tienen varias
  páginas y "Página X de Y" en todas.
- **Rationale**: SC-005. El reporte de un Sprint hace unas ocho consultas; el del proyecto, una lectura más HU-07 y
  HU-10 por cada Sprint finalizado (N + 1, como HU-08), aceptable en proyectos del curso.

### 17. Registro en `main.go`

- **Decision**: un bloque nuevo al final de los registros (hoy después del de HU-24, o del último que esté en `main`)
  que construye `report.NewService(report.NewPostgreSQLStore(database), metrics.NewService(
  metrics.NewPostgreSQLMetricsStore(database)), metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(database)),
  nil)` y llama a `web.RegisterReports(srv, web.NewPostgreSQLReportCatalog(database), service)`. Si falla, cierra
  `ln` y `database`, como los demás.
- **Rationale**: único archivo compartido que se edita (decisión del equipo). Los servicios de HU-07 y HU-10 no
  guardan estado: otra instancia no cambia nada.

### 18. Reporte del proyecto: armado y medidas

- **Decision**: `Service.BuildProject(ctx, projectID)`:
  1. `projectID <= 0` → `ErrProjectNotFound` sin consultar;
  2. `LoadProjectReport` (`ErrProjectNotFound` tal cual; otro error envuelto);
  3. estado con la regla de HU-12 (fechas y `now()`) y avance con la regla de HU-12 (historias "Hecho" y total);
  4. por cada Sprint, en orden de número: si está "Finalizado", `SprintQuality` (HU-10) y `CalculateMetrics` (HU-07);
     si está "Activo", la fila queda "En curso" y el reporte, `Partial`;
  5. `metrics.SummarizeVelocity` (HU-08) con las velocidades de HU-07 de los finalizados;
  6. `GeneratedAt = now()`.

  `RenderProject` arma, en este orden: título "Reporte del proyecto" y la marca parcial; encabezado; "Avance" (tabla
  Estado/Historias con las cuatro filas y la línea "X de Y"); "Sprints" (tabla Sprint 45, Estado 70, Historias 50,
  Story Points 55, Velocidad 50, Desviación 110, Defectos 115 = 495; la fila activa lleva "En curso: se calcula al
  cerrar el Sprint" en Estado y "—" en el resto) con la acumulada y el promedio debajo; "Esfuerzo registrado";
  "Defectos" (resumen y tabla Descripción 200, Severidad 60, Historia 140, Detectado en 95 = 495).
- **Rationale**: decisiones del equipo. Las columnas entran en el ancho útil con los textos de los datos de ejemplo
  ("+6,25 % (cálculo parcial)" mide unos 99 pt y "2 detectados y 1 resuelto", unos 101 pt en Helvetica 9).
- **Alternatives considered**: una línea de texto por Sprint (más de 100 caracteres: se corta y se lee peor que una
  tabla); llamar a HU-08 `SummarizeVelocity` con los completados de HU-10 en vez de la velocidad de HU-07 (coinciden,
  pero la fila mostraría un número y la suma usaría otro).

### 19. Texto del PDF en las pruebas

- **Decision**: además de `Literal`, `src/pdf` exporta `ExtractText(b []byte) (string, error)`: recorre los flujos de
  contenido, decodifica cada cadena `(...) Tj` (quita el escape y pasa de WinAnsi a UTF-8) y las une con un espacio,
  en el orden en que se dibujan. Las pruebas de `src/report`, `src/web` y `test/` buscan los textos esperados en esa
  cadena; `Literal` queda para verificar los bytes de la codificación.
- **Rationale**: una celda angosta corta su texto en varias líneas ("En curso: se calcula al cerrar el Sprint"), y
  cada línea es una cadena distinta en el PDF. Unidas con un espacio, el texto vuelve a leerse completo y las filas
  de una tabla quedan como "Sprint 1 Finalizado 1 de 1 3 de 3 3 0,00 % 0 detectados y 0 resueltos".
- **Alternatives considered**: buscar cada cadena cruda con `Literal` (falla cuando el texto se corta); anchos de
  columna que eviten todo corte (imposible con textos de usuario).

## Preguntas abiertas

Ninguna de decisión. Los nombres de HU-06, HU-08, HU-10, HU-12, HU-15 y HU-17 están pendientes de confirmación en
T001; HU-12 no tiene documentos y su contrato se toma del comportamiento descrito en la spec.
