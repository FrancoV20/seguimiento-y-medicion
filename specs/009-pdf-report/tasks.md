# Tareas: HU-09 – Exportación de Reportes en PDF

**Entrada**: `specs/009-pdf-report/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-06, HU-07, HU-08, HU-10, HU-12, HU-15, HU-17, HU-20a y HU-20b integradas en `main`. Para
la verificación manual, además HU-03/HU-24 y HU-23.
**Orden de implementación**: después de HU-08, HU-10 y HU-12 (y por lo tanto de HU-06 y HU-15). Si falta alguna de
las tres en `main`, no se crea la rama (T001).

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 18).
- Cada bloque sigue RED → GREEN. Las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/009-pdf-report/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- CI (`.github/workflows/ci.yml`): `gofmt -l` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` se corre solo sobre los archivos que se crean o se tocan.
- Las pruebas de integración arman sus propios datos (research.md §15) y los borran al final, en este orden:
  defectos, registro de cierre, asignaciones, `sprint_metrics`, `effort_logs`, historias, Sprints, proyectos e
  integrantes. Cada escenario usa sus propios Sprints; ningún Sprint se usa para dos repartos distintos.
- Para buscar textos dentro de un PDF, las pruebas usan `pdf.ExtractText` (texto decodificado y unido, research.md
  §19); `pdf.Literal` se usa solo para verificar los bytes de la codificación.

## Fase 1: Preparación

- [x] T001 Confirmar con `git grep -n` los nombres que usa HU-09, ajustar `data-model.md`, `research.md` y las
  consultas si alguno cambió, y quitar las marcas "(T001)":
  - de HU-10: `NewQualityService`, `NewPostgreSQLQualityStore`, `(*QualityService).SprintQuality` y los campos
    `StoriesPlanned`, `StoriesCompleted`, `CompletionPercentage`, `PointsPlanned`, `PointsCompleted`,
    `DefectsDetected` y `DefectsResolved` de `metrics.SprintQuality`;
  - de HU-08: `metrics.SummarizeVelocity` y `VelocitySummary` (`SprintCount`, `Total`, `Average`);
  - de HU-12: en `src/project`, la regla de estado por fechas ("No iniciado", "En curso", "Finalizado") y la de
    porcentaje de avance; anotar sus nombres y firmas en `data-model.md` y `research.md` §9, si la de estado recibe la
    fecha de referencia, y que dan "En curso" para «Alfa» al 28/10/2026, "57.14" (o su equivalente) para 4 de 7 y
    "0,00 %" para 0 de 0. Si su comportamiento no coincide con los escenarios 15, 16 y 18, avisar al equipo y
    ajustar la spec antes de T002;
  - de HU-06: tabla `defects` con `id`, `project_id`, `story_id`, `description`, `severity` ('Crítica', 'Alta',
    'Media', 'Baja'), `status` ('Abierto', 'Resuelto'), `detection_sprint_id` y `resolution_sprint_id`;
  - de HU-15: tabla `sprint_story_results` (`sprint_id`, `story_id`, `final_status`, `story_points`) y cómo quedan
    `user_stories.status`, `active_sprint_id` y `sprint_story_assignments` al cerrar, para copiarlo en los datos de
    prueba;
  - de HU-17: que `user_stories.estimated_hours` sigue siendo `NUMERIC(6,2)` y opcional;
  - de HU-07: `NewService`, `NewPostgreSQLMetricsStore`, `CalculateMetrics` y los campos `Velocity`,
    `DeviationPercentage`, `IsPartial` y `Warning` de `SprintMetrics`;
  - de HU-03 y HU-20a/b: `effort.Hours`, `parseID`, `NotFound`, `Renderer.Fail`, `ActiveContextFrom` y
    `RequireContext`;
  - si existe un formato de la desviación exportado fuera de `src/web` (HU-25 o HU-08), anotarlo en research.md §12
    para reutilizarlo;
  - que no existan los paquetes `src/pdf` ni `src/report`, ni nombres `Report*`/`report*` en `src/web`, ni la ruta
    `/reportes`.

  **Si HU-08, HU-10 o HU-12 no están en `main`, no seguir y avisar al equipo.** Revisar también "Planificación de
  proyecto.pdf": si contradice la spec, corregirla antes de T002.
- [x] T002 Crear la rama `HU-09` desde `main` actualizado, solo si no hay cambios sin commitear:
  `git switch main; git pull; git switch -c HU-09`.
- [x] T003 Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y `go vet ./...` en verde antes de empezar.
- [x] T004 Coordinar con quien tenga HU-08, HU-10, HU-12 y HU-25 el bloque de registro que se suma al final de
  `src/cmd/server/main.go`.

## Fase 2: Bloque Generador PDF

**Objetivo**: un PDF 1.4 válido con texto, tablas, saltos de página y pie, sin dependencias externas.

**Prueba independiente**: las pruebas de `src/pdf` pasan sin PostgreSQL.

### Pruebas (RED)

- [x] T005 [P] Escribir `TestPDF_Texto` en `src/pdf/winansi_test.go` y `src/pdf/widths_test.go`, con las tablas de
  `data-model.md`:
  - `encodeWinAnsi`: "Página", "Año – 5 €", comillas tipográficas y "…", "«Alfa» — ok", tabulación y salto de línea
    como espacio, emoji y "Ω" como "?", UTF-8 inválido como "?";
  - `escapeLiteral` y `Literal`: "Login (v2) C:\x" → `Login \(v2\) C:\\x`;
  - `textWidth(Regular, 10, "Página")` = 31,13 y `textWidth(Bold, 10, "Página")` = 32,79;
  - `wrapText`: texto que entra (una línea), varias palabras (cada línea ≤ ancho, sin perder palabras), palabra más
    ancha que el ancho (cortada por caracteres) y texto vacío (una línea vacía).
- [x] T006 [P] Escribir en `src/pdf/pdftest_test.go` un ayudante que, sin bibliotecas externas, lee la `xref`, verifica
  que cada desplazamiento apunta a `N 0 obj`, que `startxref` apunta a `xref`, que cada `/Length` coincide con su flujo,
  y devuelve la cantidad de objetos `/Type /Page` y el contenido de cada página. Con él, en
  `src/pdf/document_test.go`:
  - `TestPDF_Estructura`: empieza con `%PDF-1.4`, termina con `%%EOF`, `/MediaBox [0 0 595.28 841.89]`, fuentes
    Helvetica y Helvetica-Bold con `/WinAnsiEncoding`, `/Count` igual a `PageCount()`, salida idéntica para el mismo
    documento (determinista), `ErrTableTooWide` y `ErrRowColumnCount`, y `ExtractText` devuelve "Login (v2) – Año"
    escrito así y une con un espacio las líneas de una celda cortada;
  - `TestPDF_SaltoDePagina`: una tabla de 120 filas (una con un texto de 300 caracteres) pasa a varias páginas, el
    encabezado de la tabla está en cada página con filas, cada página tiene su "Página X de Y" con Y igual a la
    cantidad de páginas, y ninguna línea queda por debajo del margen inferior ni más allá del margen derecho.
- [x] T007 Ejecutar T005 y T006, confirmar que fallan porque el código no existe y guardar la salida en
  `evidencia-red.txt`, sección "Generador PDF".

### Implementación (GREEN)

- [x] T008 Crear `src/pdf/winansi.go` (`encodeWinAnsi`, `escapeLiteral`, `Literal`; research.md §3),
  `src/pdf/widths.go` (anchos AFM de Helvetica y Helvetica-Bold en milésimas, `textWidth`, `wrapText`; research.md
  §4) y `src/pdf/extract.go` (`ExtractText`; research.md §19).
- [x] T009 Crear `src/pdf/document.go` (`Font`, `Column`, `Document`, `New`, `Text`, `Space`, `Table`, `PageCount`,
  `Bytes`, errores) y `src/pdf/writer.go` (serialización de research.md §5: encabezado con línea binaria, objetos 1 a
  4, una página y un flujo por página, `xref` de 20 bytes por entrada, `trailer`, `startxref`, `%%EOF`), con las
  medidas de research.md §6.
- [x] T010 Ejecutar T005 y T006 en verde.

**Punto de control**: el generador produce PDF válidos; los dos reportes ya pueden usarlo.

## Fase 3: Bloque Dominio del reporte de un Sprint

**Objetivo**: armar el `SprintReport` con los datos propios, HU-07 y HU-10, y convertirlo en PDF, sin base de datos.

**Prueba independiente**: las pruebas unitarias de `src/report` pasan sin PostgreSQL, con dobles.

### Pruebas (RED)

- [x] T011 [P] [E3, E4, E5, E6, E7, E9] Escribir `TestReportTexts` en `src/report/texts_test.go` con las tablas de
  `data-model.md`: `percentText`, `deviationText` ("+6,25 %", "-8,50 %", "0,00 %", "No disponible"), `hoursText` (17,
  2,5, 0,25 y 0), `dateText`, `storyStatusText`, `withoutHoursText` (0, 1 y 2), totales de horas en singular y plural,
  `defectsSummaryText` (2 y 1 → "2 defectos detectados y 1 resuelto en este Sprint"; 1 y 0; 0 y 1; 0 y 0 → "No se
  registraron defectos en este Sprint") y `defectsSoFarText` (0, 1 y 3).
- [x] T012 [P] [E1, E2, E3, E5, E9, E13] Escribir en `src/report/service_test.go`, con un almacén, un `MetricsSource`
  y un `QualitySource` de prueba locales y un reloj fijo (28/10/2026 10:30):
  - `TestBuildSprintReport`: con los datos del Sprint 2 de «Alfa», el reporte trae los datos leídos, `Metrics` y
    `Quality` de las fuentes, `Partial` falso y `GeneratedAt` del reloj; con el Sprint 3 activo, `Partial` verdadero
    y las fuentes de HU-07 y HU-10 no se llaman;
  - `TestReportService_Errores`: identificadores en 0 → `ErrSprintNotFound` sin llamar al almacén;
    `ErrSprintNotFound` del almacén sin cambios; falla del almacén, de HU-07 y de HU-10 envueltas (`errors.Is` con la
    causa); historias o defectos que no coinciden con HU-10 → `ErrInconsistentReport`; dependencias `nil` → error; en
    todos los casos `SprintPDF` devuelve `nil` como bytes.
- [x] T013 [P] [E1, E2, E3, E4, E5, E6, E7, E9] Escribir `TestRenderSprintReport` en `src/report/document_test.go`,
  buscando con `pdf.ExtractText`:
  - con el reporte del Sprint 2 de «Alfa», el PDF contiene, en este orden, los textos de "Datos de ejemplo"
    ("Proyecto: Alfa", "Sprint 2", "Objetivo: Gestionar perfiles y reportes", "Fechas: 01/09/2026 al 14/09/2026",
    "Estado: Finalizado", "Generado: 28/10/2026 10:30", las 4 historias con "En progreso – volvió al Backlog" y "Sin
    estimar", "3 de 4 historias completadas (75,00 %)", "18 Story Points planificados y 10 completados", "28 horas
    estimadas en total (1 historia sin horas estimadas)", "24 horas registradas en total", "Ana: 17 h", "Beto: 7 h",
    "Velocidad: 10 Story Points", "Desviación de esfuerzo: +6,25 %", "Cálculo parcial: basado en 2 de 3 historias",
    los 2 defectos y "2 defectos detectados y 1 resuelto en este Sprint");
  - con el Sprint 1: "No se registraron defectos en este Sprint" y "0,00 %" sin aviso parcial;
  - con el Sprint 3 en curso: "Reporte parcial: Sprint en curso", "Estado: En curso", tres veces "Se calculan al
    cerrar el Sprint", "6 horas estimadas en total (1 historia sin horas estimadas)", "Ana: 2 h" y "1 defecto
    detectado hasta el momento", y no contiene "Velocidad:";
  - un Sprint sin historias muestra "No hay historias registradas en este Sprint" y "0 de 0 historias completadas (No
    disponible)".
- [x] T014 Ejecutar T011–T013, confirmar que fallan y guardar la salida en `evidencia-red.txt`, sección "Dominio del
  reporte".

### Implementación (GREEN)

- [x] T015 Crear `src/report/model.go` (`ErrSprintNotFound`, `ErrProjectNotFound`, `ErrInconsistentReport`,
  `SprintInfo`, `Story`, `MemberHours`, `Defect`, `Data`, `SprintReport`), `src/report/store.go` (puerto `Store` con
  `LoadSprintReport`; `LoadProjectReport` se agrega en T043) y `src/report/sources.go` (`MetricsSource`,
  `QualitySource`), como en `data-model.md`.
- [x] T016 Crear `src/report/texts.go` con la tabla de textos del Sprint y las funciones de formato de
  `data-model.md`. Las horas usan `effort.Hours.String()` con coma.
- [x] T017 Crear `src/report/service.go` con `NewService` (reloj `nil` = `time.Now`), `Build` (pasos 1 a 6 de
  `data-model.md`) y `SprintPDF` (`Build` + `Render`).
- [x] T018 Crear `src/report/document.go` con `Render`: las siete partes en el orden de `data-model.md`, con las
  columnas de research.md §6 y el pie `fmtFooter`.
- [x] T019 Ejecutar T011–T013 en verde.

**Punto de control**: con datos en memoria se genera el PDF completo de un Sprint finalizado y de uno en curso.

## Fase 4: Bloque Almacenamiento del reporte de un Sprint

**Objetivo**: leer encabezado, historias, horas, esfuerzo por integrante y defectos en una lectura consistente.

**Prueba independiente**: las pruebas de integración de `src/report` pasan contra PostgreSQL con datos propios.

### Pruebas (RED)

- [ ] T020 Crear los datos de prueba en `src/report/fixtures_test.go` (los usan T021, T022 y T039): integrantes
  «Ana», «Beto» y «Caro»;
  - «Alfa» (18/08/2026 al 30/11/2026, integrantes Ana y Beto) con los Sprints 1, 2 y 3, las historias, los registros
    de esfuerzo y los defectos de "Datos de ejemplo" de la spec (Sprints 1 y 2 cerrados como los deja HU-15: registro
    de cierre, "Hecho" asignadas, «Reportes» en "Pendiente" sin asignación y con `final_status` "En progreso"; Sprint
    3 "Activo" con «Exportar» y «Ayuda» asignadas);
  - «Beta» (01/10/2026 al 31/12/2026, integrante Ana) sin historias ni Sprints;
  - «Gamma» con sus Sprints 1 y 2 finalizados, una historia "Hecho" de 1 Story Point en cada uno, sin horas
    estimadas ni registros de esfuerzo, y un defecto detectado en el 1 y resuelto en el 2, más su Sprint 3
    "Finalizado" sin registro de cierre ni historias;
  - «Delta» con su Sprint 1 finalizado, la historia «D1» "Hecho" de 2 Story Points y 1 hora estimada, un registro de
    1 hora de Caro (que no está en `project_members` de «Delta») y un defecto "Abierto".

  Limpieza en `t.Cleanup`, en el orden del encabezado.
- [ ] T021 [P] [E2, E3, E4, E6, E9, E12] Escribir en `src/report/postgres_store_integration_test.go`:
  - `TestPostgresReportStore_Finalizado`: Sprint 2 de «Alfa» → encabezado; 4 historias en orden de creación con
    estado final, Story Points del registro y horas estimadas (1000, 600, `nil`, 1200) y reales (1100, 600, 400,
    300); integrantes Ana 1700 y Beto 700 en ese orden; los 2 defectos con historia y "Resuelto en" 2 y 0. Sprint 1 de
    «Gamma» → 1 defecto "Resuelto" con Sprint de resolución 2 y ningún integrante; Sprint 2 de «Gamma» → sin
    defectos; Sprint 3 de «Gamma» → sin historias, sin integrantes y sin defectos; Sprint 1 de «Delta» → Caro 100,
    aunque no es integrante del proyecto;
  - `TestPostgresReportStore_EnCurso`: Sprint 3 → «Exportar» "En progreso" y «Ayuda» "En Sprint" sin Story Points ni
    horas, Ana 200 y el defecto "El PDF sale en blanco"; las horas de Ana en «Alta» y en el Sprint 2 no cuentan;
  - `TestPostgresReportStore_OtroProyecto`: el Sprint 1 de «Delta» pedido con «Alfa» y el identificador 2147483647 →
    `ErrSprintNotFound`; nada de «Delta» aparece en los datos de los Sprints de «Alfa».
- [ ] T022 [P] [E2, E5, E6] Escribir `TestReport_CoherenciaConHU07YHU10` en el mismo archivo: con los servicios reales
  de HU-07 y HU-10 y el Sprint 2 de «Alfa», `Build` da velocidad 10 igual a `PointsCompleted`, desviación "6.25",
  aviso "Cálculo parcial: basado en 2 de 3 historias", 4 historias igual a `StoriesPlanned`, "75.00", 18 y 10, y 2
  detectados y 1 resuelto. La prueba borra las filas de `sprint_metrics` que escribe HU-07.
- [ ] T023 Ejecutar T021 y T022, confirmar que fallan y guardar la salida en `evidencia-red.txt`, sección
  "Almacenamiento".

### Implementación (GREEN)

- [ ] T024 Implementar `PostgreSQLStore`, `NewPostgreSQLStore(db *sql.DB)` y `LoadSprintReport` en
  `src/report/postgres_store.go`:
  - `BeginTx` con `ReadOnly: true` e `Isolation: sql.LevelRepeatableRead`, y `defer tx.Rollback()`;
  - las consultas de `data-model.md`: encabezado (sin fila → `ErrSprintNotFound`), historias del registro de cierre
    si está "Finalizado" o de las asignaciones si está "Activo", esfuerzo por integrante sobre las mismas historias
    `ORDER BY m.display_name, m.id`, y defectos con `WHERE d.project_id = $2 AND d.detection_sprint_id = $1`;
  - horas como `ROUND(x * 100)::bigint` en `effort.Hours`;
  - errores envueltos con `%w`.
- [ ] T025 Ejecutar T021 y T022 en verde.

**Punto de control**: `report.Service.SprintPDF` genera el PDF real de un Sprint desde PostgreSQL.

## Fase 5: Bloque Pantalla y descarga del reporte de un Sprint 🎯 MVP visible

**Objetivo**: la sección `GET /reportes` y la descarga `GET /reportes/sprints/{id}/pdf`.

**Prueba independiente**: abrir "Reportes" de «Alfa», descargar los PDF de los Sprints 1, 2 y 3 y comprobar los
valores de "Datos de ejemplo"; repetir con los proyectos de los casos vacíos, ajenos y con una falla.

### Pruebas (RED)

- [ ] T026 Crear los datos de prueba de la web en `src/web/report_fixtures_test.go` (los usan T028, T029 y T040): los
  mismos «Alfa», «Beta», «Gamma» y «Delta» de T020, y «Largo» (integrante Ana) con su Sprint 1 finalizado
  (research.md §15: 120 historias, una con un título de 300 caracteres, y 50 defectos). Escribir
  `TestPostgresReportCatalog` en `src/web/postgres_report_catalog_integration_test.go`:
  - «Alfa»: Sprint 3 (activo), 2 y 1, en ese orden, con objetivo y `Active`;
  - `ReportSprint` del Sprint 1 de «Delta» con «Alfa» → no encontrado; inexistente → no encontrado;
  - «Beta»: lista vacía.
- [ ] T027 [P] [E8, E11, E12] Escribir `TestReportView` en `src/web/report_view_test.go`: `reportPDFPath`,
  `reportConfirmPath`, `reportFileName(2)` = "reporte-sprint-2.pdf", `reportProjectFileName(5)` =
  "reporte-proyecto-5.pdf", `reportConfirmed` ("si" verdadero; "SI", "1", "no", vacío y ausente falsos) y el armado
  de la lista (con `ProjectURL` siempre) y de las dos advertencias.
- [ ] T028 [E1, E2, E3, E4, E5, E6, E7, E8, E9, E10, E11, E12] Escribir el servidor de prueba y las pruebas de
  pantalla en `src/web/report_handlers_test.go`, con el catálogo y `report.Service` reales (HU-07, HU-08, HU-10 y
  HU-12 reales, reloj fijo) y los datos de T026, buscando con `pdf.ExtractText`:
  - `TestHU09_ListaDeSprints`: «Alfa» muestra "Sprint 3", "Sprint 2" y "Sprint 1" en ese orden con objetivo, "En
    curso"/"Finalizado" y "Exportar PDF"; «Beta» muestra "Aún no hay Sprints para exportar"; en los dos, "Exportar
    reporte del proyecto" con enlace a `/reportes/proyecto/pdf`; sin `<script` ni formularios;
  - `TestHU09_ExportarFinalizado`: Sprint 2 → 200, `Content-Type: application/pdf`, `Content-Disposition:
    attachment; filename="reporte-sprint-2.pdf"`, `Cache-Control: no-store`, `Content-Length` igual al cuerpo, que
    empieza con `%PDF-1.4`;
  - `TestHU09_ContenidoDelPDF`: el PDF del Sprint 2 contiene los textos de T013;
  - `TestHU09_SinDefectos`: el PDF del Sprint 1 contiene "No se registraron defectos en este Sprint";
  - `TestHU09_ResueltosDetectadosAntes`: Sprint 2 de «Gamma» → "0 defectos detectados y 1 resuelto en este Sprint" y
    ninguna tabla de defectos; Sprint 1 de «Gamma» → el defecto con "Resuelto" y "Sprint 2", y "1 defecto detectado y
    0 resueltos en este Sprint"; en los dos, "0 horas registradas en total" y ningún integrante listado;
  - `TestHU09_SprintSinRegistroDeCierre`: Sprint 3 de «Gamma» (finalizado sin registro de cierre) → "No hay historias
    registradas en este Sprint", "0 de 0 historias completadas (No disponible)" y "0 Story Points planificados y 0
    completados", sin error;
  - `TestHU09_AdvertenciaSprintEnCurso`: Sprint 3 sin `confirmar`, con `confirmar=SI` y con `confirmar=1` → 200 HTML
    con "El Sprint 3 sigue en curso: el reporte puede estar incompleto", el enlace "Generar igual" a
    `/reportes/sprints/<id>/pdf?confirmar=si` y "Cancelar" a `/reportes`; sin `application/pdf` ni `%PDF`; no se
    escribió ninguna fila en `sprint_metrics`;
  - `TestHU09_GenerarIgual`: Sprint 3 con `confirmar=si` → PDF `reporte-sprint-3.pdf` con "Reporte parcial: Sprint en
    curso", las historias en su estado actual y "Se calculan al cerrar el Sprint";
  - `TestHU09_SprintInexistenteOAjeno`: `abc`, `0`, `-1`, `007` y `2147483648` → 404 sin consultar (catálogo de
    prueba que falla si se lo llama); 2147483647 y el Sprint 1 de «Delta» con «Alfa» activo → 404; en todos, página
    HTML de HU-20a y sin `%PDF`.

  Las pruebas borran las filas de `sprint_metrics` que escribe HU-07.
- [ ] T029 [E13] Escribir las pruebas de errores y bordes en `src/web/report_handlers_test.go`, con dobles que
  fallan:
  - `TestHU09_ErrorAlGenerar`: catálogo que falla, `ReportGenerator` que falla, `ErrInconsistentReport` y
    `ClosureValidationError` → 500 con la página de error, sin `application/pdf`, sin `%PDF`, sin
    `Content-Disposition` y sin texto interno; con `driver.ErrBadConn` → 503; `report.ErrSprintNotFound` del
    generador → 404;
  - `TestHU09_SinProyectoYMetodo`: sin proyecto, `/reportes` → 303 a `/contexto/proyecto?volver_a=%2Freportes` y la
    descarga → 303 con su propia dirección; `POST` en las dos rutas → 405; `HEAD` de la descarga del Sprint 2 → 200
    con los mismos encabezados que `GET` y sin cuerpo;
  - `TestHU09_Rendimiento`: el PDF del Sprint 1 de «Largo» por HTTP en menos de 1 s, con "90 de 120 historias
    completadas (75,00 %)", "270 Story Points planificados y 180 completados", "480 horas estimadas en total", "360
    horas registradas en total", "Ana: 360 h", "Desviación de esfuerzo: 0,00 %", "50 defectos detectados y 20
    resueltos en este Sprint", más de una página y "Página X de Y" en todas.
- [ ] T030 [P] [E1, E2, E5, E8, E11] Escribir los escenarios BDD en `test/web_report_bdd_test.go`, con datos propios:
  "Exportación exitosa de un Sprint finalizado", "Historias planificadas y completadas", "Métricas del Sprint",
  "Advertencia al exportar un Sprint en curso" y "Sin Sprints".
- [ ] T031 Ejecutar T026–T030, confirmar que fallan (las rutas y el catálogo no existen) y guardar la salida en
  `evidencia-red.txt`, sección "Pantalla y descarga".

### Implementación (GREEN)

- [ ] T032 Crear `src/web/report_catalog.go` (`ReportSprintOption`, `ReportCatalog`, `ReportGenerator` con
  `SprintPDF`; `ProjectPDF` se agrega en T045) y `src/web/postgres_report_catalog.go` (`PostgreSQLReportCatalog`,
  `NewPostgreSQLReportCatalog`, las dos consultas de `data-model.md`, errores envueltos con `%w`).
- [ ] T033 Crear `src/web/report_view.go` con los textos de la tabla de `data-model.md`, `reportPDFPath`,
  `reportConfirmPath`, `reportFileName`, `reportProjectFileName`, `reportConfirmed`, `reportListData` (con
  `ProjectURL`) y `reportWarningData`.
- [ ] T034 Crear `RegisterReports(s, catalog, generator)` en `src/web/report_handlers.go` (si falta un argumento,
  error sin registrar; `Section{Name: TitleReports, Path: "/reportes", RequiresProject: true}`; `GET /reportes` y la
  descarga del Sprint con los pasos 1 a 5 de `data-model.md`, sin escribir nada antes de tener el PDF completo) y las
  plantillas `src/web/templates/reports.html` (con el botón "Exportar reporte del proyecto") y
  `src/web/templates/report_warning.html`, sin `<script>` ni formularios.
- [ ] T035 Agregar al final de los bloques de registro de `src/cmd/server/main.go` (hoy después del de HU-24, o del
  último que esté en `main`) el bloque de research.md §17, que llama a `web.RegisterReports` con
  `web.NewPostgreSQLReportCatalog(database)` y `report.NewService(...)`. Si falla, cerrar `ln` y `database`, como los
  otros bloques.
- [ ] T036 Ejecutar T026–T030 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Scrum Master descarga el PDF de cualquier Sprint del proyecto activo.

## Fase 6: Bloque Reporte del proyecto

**Objetivo**: `GET /reportes/proyecto/pdf` con las seis partes del reporte del proyecto, reutilizando el generador,
el puerto, el servicio y la advertencia.

**Prueba independiente**: exportar el reporte de «Alfa» (con advertencia), de «Gamma» (sin Sprint en curso) y de
«Beta» (vacío) y comprobar las cuentas de "Datos de ejemplo".

### Pruebas (RED)

- [ ] T037 [P] [E15, E16, E17, E18] Escribir en `src/report/texts_test.go` `TestProjectReportTexts` (`decimalText`,
  `projectDefectsText` y `sprintDefectsCell` con las tablas de `data-model.md`, y "Integrantes: Sin integrantes") y en
  `src/report/project_document_test.go` `TestRenderProjectReport`, buscando con `pdf.ExtractText`:
  - con el reporte de «Alfa»: en este orden, "Reporte parcial: hay un Sprint en curso", "Proyecto: Alfa", "Fechas:
    18/08/2026 al 30/11/2026", "Estado: En curso", "Integrantes: Ana, Beto", "Generado: 28/10/2026 10:30",
    "Pendiente 1", "En Sprint 1", "En progreso 1", "Hecho 4", "4 de 7 historias completadas (57,14 %)", las filas
    "Sprint 1 Finalizado 1 de 1 3 de 3 3 0,00 % 0 detectados y 0 resueltos", "Sprint 2 Finalizado 3 de 4 10 de 18 10
    +6,25 % (cálculo parcial) 2 detectados y 1 resuelto" y "Sprint 3 En curso: se calcula al cerrar el Sprint",
    "Velocidad acumulada: 13 Story Points", "Promedio: 6,50 Story Points por Sprint", "38 horas estimadas en total (2
    historias sin horas estimadas)", "30 horas registradas en total", "Ana: 23 h", "Beto: 7 h", "3 defectos
    registrados: 2 abiertos y 1 resuelto" y los 2 abiertos con su Sprint de detección;
  - con «Beta»: sin marca parcial, "Pendiente 0", "0 de 0 historias completadas (0,00 %)", "Aún no hay Sprints", sin
    "Velocidad acumulada", "0 horas estimadas en total", "0 horas registradas en total" y "No se registraron defectos
    en este proyecto";
  - con un proyecto con solo un Sprint en curso: "Velocidad acumulada: 0 Story Points" y "Promedio: No disponible";
  - con defectos todos resueltos: "No hay defectos abiertos".
- [ ] T038 [P] [E13, E15, E16, E18] Escribir en `src/report/project_service_test.go`, con el almacén de prueba de T012
  extendido con `LoadProjectReport`, las fuentes de prueba de T012, las reglas reales de HU-08 y HU-12 y el reloj
  fijo:
  - `TestBuildProjectReport`: con los datos de «Alfa», `Status` de HU-12 ("En curso"), `Progress` de HU-12 para 4 de
    7, filas de los Sprints 1 y 2 con `Quality` y `Metrics` de las fuentes, la del 3 `Active`, `Partial` verdadero y
    `Velocity` = `SummarizeVelocity([3, 10])` (13 y "6.50"); con «Beta», `Partial` falso, sin filas, y las fuentes no
    se llaman;
  - `TestProjectReportService_Errores`: `projectID` 0 → `ErrProjectNotFound` sin llamar al almacén;
    `ErrProjectNotFound` del almacén sin cambios; falla del almacén, de HU-07 y de HU-10 envueltas; en todos los casos
    `ProjectPDF` devuelve `nil` como bytes.
- [ ] T039 [E16, E17, E18] Escribir en `src/report/postgres_store_integration_test.go`, con los datos de T020:
  - `TestPostgresReportStore_Proyecto`: «Alfa» → nombre y fechas, integrantes Ana y Beto, historias 1, 1, 1 y 4,
    Sprints 1, 2 y 3 con su estado, `EstimatedTotal` 3800 y `WithoutEstimate` 2, integrantes Ana 2300 y Beto 700
    (suma 3000), 2 abiertos y 1 resuelto, y los 2 abiertos con historia y Sprint de detección 2 y 3; «Beta» → todo en
    cero y listas vacías, con su integrante Ana; nada de «Delta» (ni Caro, ni «D1», ni su defecto) en «Alfa»;
    identificador 2147483647 → `ErrProjectNotFound`;
  - `TestProjectReport_Coherencia`: con los servicios reales, cada fila de «Alfa» coincide con `SprintQuality` y
    `CalculateMetrics` del mismo Sprint, `Velocity` con `SummarizeVelocity` de esas velocidades, y `Status` y
    `Progress` con las reglas de HU-12. Borra las filas de `sprint_metrics`.
- [ ] T040 [E11, E13, E14, E15, E16, E17, E18] Escribir en `src/web/report_project_handlers_test.go`, con el servidor
  de prueba y los datos de T026:
  - `TestHU09_ProyectoAdvertencia`: «Alfa» sin `confirmar` y con `confirmar=SI` → 200 HTML con "El proyecto tiene un
    Sprint en curso (Sprint 3): el reporte puede estar incompleto", "Generar igual" a
    `/reportes/proyecto/pdf?confirmar=si` y "Cancelar" a `/reportes`; sin `%PDF` y sin filas nuevas en
    `sprint_metrics`;
  - `TestHU09_ProyectoContenido`: «Alfa» con `confirmar=si` → `filename="reporte-proyecto-<id de Alfa>.pdf"`, los
    encabezados de T028 y los textos de T037;
  - `TestHU09_ProyectoSinSprints`: «Beta» → descarga sin advertencia con los textos del escenario 18;
  - `TestHU09_ProyectoSinSprintActivo`: «Gamma» → descarga sin advertencia, sin marca parcial, con las filas de sus
    Sprints 1, 2 y 3 (la del 3, "0 de 0"), "Velocidad acumulada: 2 Story Points", "Promedio: 0,67 Story Points por
    Sprint" y "1 defecto registrado: 0 abiertos y 1 resuelto";
  - `TestHU09_ProyectoErrores`: catálogo que falla, `ProjectPDF` que falla → 500 sin `%PDF`; `driver.ErrBadConn` →
    503; `report.ErrProjectNotFound` → 404; sin proyecto → 303 a `/contexto/proyecto?volver_a=%2Freportes%2Fproyecto%2Fpdf`;
    `POST` → 405.

  Sumar a `TestHU09_Rendimiento` (T029) el PDF del proyecto «Largo» en menos de 1 s, con "90 de 120 historias
  completadas (75,00 %)", "Sprint 1 Finalizado 90 de 120 180 de 270 180 0,00 % 50 detectados y 20 resueltos",
  "Velocidad acumulada: 180 Story Points", "Promedio: 180,00 Story Points por Sprint", "480 horas estimadas en
  total", "360 horas registradas en total", "50 defectos registrados: 30 abiertos y 20 resueltos" y "Página X de Y"
  en todas las páginas.
- [ ] T041 [E14, E16, E18] Agregar a `test/web_report_bdd_test.go` (después de T030) los escenarios "Advertencia al
  exportar un proyecto con un Sprint en curso", "Avance y Sprints del proyecto" y "Proyecto sin Sprints ni
  historias", con datos propios.
- [ ] T042 Ejecutar T037–T041, confirmar que fallan y guardar la salida en `evidencia-red.txt`, sección "Reporte del
  proyecto".

### Implementación (GREEN)

- [ ] T043 Crear `src/report/project_model.go` (`StoryCounts`, `ProjectSprint`, `OpenDefect`, `ProjectData`,
  `ProjectSprintRow`, `ProjectReport`), agregar `LoadProjectReport` al puerto en `src/report/store.go`, sumar a
  `src/report/texts.go` los textos del proyecto de `data-model.md` y crear
  `src/report/project_document.go` con `RenderProject` (columnas de research.md §18).
- [ ] T044 Crear `src/report/project_service.go` con `BuildProject` (pasos 1 a 6 de `data-model.md`, con las reglas de
  HU-12 y `metrics.SummarizeVelocity` de HU-08) y `ProjectPDF`, e implementar `LoadProjectReport` en
  `src/report/postgres_store.go` con las siete consultas de `data-model.md` en una transacción `ReadOnly` +
  `REPEATABLE READ` (sin fila de proyecto → `ErrProjectNotFound`).
- [ ] T045 En la web: agregar `ProjectPDF` a `ReportGenerator` en `src/web/report_catalog.go` y a los dobles de T029;
  sumar a `src/web/report_handlers.go` la ruta `GET /reportes/proyecto/pdf` con los pasos 1 a 4 de `data-model.md`
  (advertencia con `msgReportProjectActiveWarning`, misma plantilla `report_warning.html`, encabezados con
  `reportProjectFileName`). `main.go` no cambia: `*report.Service` ya cumple la interfaz completa.
- [ ] T046 Ejecutar T037–T041 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Scrum Master descarga el informe del proyecto en PDF (DoD de la entrega final).

## Fase 7: Cierre

- [ ] T047 REFACTOR y CI:
  - revisar duplicación entre los dos reportes (textos, tablas, manejo de la advertencia y de los encabezados) y
    extraer funciones locales sin cambiar el comportamiento;
  - comprobar que ningún archivo de HU-03, HU-07, HU-08, HU-10, HU-12 ni HU-25 cambió y que no hay migraciones ni
    dependencias nuevas: `git diff --stat main` lista solo archivos nuevos y `main.go`, y `go.mod` no cambió;
  - `gofmt -l` sobre los archivos creados y `main.go` sin salida, `go vet ./...` limpio y `go test ./... -p 1 -race`
    en verde.
- [ ] T048 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde, y `cobertura.txt`
  con `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en `src/pdf` y en `src/report`.
- [ ] T049 Verificación manual (`quickstart.md`): con la aplicación en ejecución, armar «Alfa» y «Beta», descargar los
  PDF de los Sprints 1, 2 y 3 y los del proyecto («Alfa» con advertencia y «Beta» sin ella), y abrir cada uno en dos
  lectores distintos (Edge y Adobe Acrobat Reader): sin avisos de archivo dañado, acentos y "ñ" correctos, columnas
  sin textos superpuestos, saltos de página y "Página X de Y" correctos. Probar también las dos advertencias,
  "Cancelar", el 404 y el 503 con la base detenida. Guardar `specs/009-pdf-report/ejemplo-reporte-sprint.pdf` (Sprint
  2) y `specs/009-pdf-report/ejemplo-reporte-proyecto.pdf` («Alfa») y anotar en `evidencia-pruebas.txt` las versiones
  de los dos lectores.
- [ ] T050 Cierre de la historia:
  - actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas y el manual breve de
    usuario con la sección "Reportes" (los dos reportes);
  - avisar a HU-08, HU-10, HU-12 y HU-25 que no se tocaron sus archivos, al equipo que `src/pdf` queda disponible y
    que, si hay conflicto en `main.go` al mergear, se conservan todos los bloques;
  - abrir el Pull Request desde `HU-09` citando HU-09 y la DoD de la entrega final, con las evidencias, los dos PDF de
    ejemplo y la confirmación de T001; merge con "Create a merge commit" después de la aprobación de otro integrante,
    y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- La **Fase 1** va primero. T001 bloquea todo: sin HU-08, HU-10 y HU-12 en `main` no se crea la rama.
- El **Bloque Generador PDF (Fase 2)** depende solo de la Fase 1.
- El **Bloque Dominio del reporte de un Sprint (Fase 3)** depende de T008–T009 (usa `src/pdf`); T011 y T015–T016 se
  pueden adelantar.
- El **Bloque Almacenamiento (Fase 4)** depende de T015 (tipos y puerto) y T017 (T022 usa `Build`).
- El **Bloque Pantalla y descarga (Fase 5)** depende de las Fases 2 a 4.
- El **Bloque Reporte del proyecto (Fase 6)** depende de la Fase 5: reutiliza el generador, el puerto, el servicio, los
  datos de prueba (T020, T026), el servidor de prueba y la plantilla de advertencia. T037 y T038 se pueden adelantar
  apenas termina la Fase 4.
- La **Fase 7** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- `main.go` es el único archivo compartido (T035). Si hay conflicto al mergear, se conservan todos los bloques.

### Oportunidades de paralelismo

- T005 y T006; T008 y T009 después de T007 (archivos distintos; T009 usa T008).
- T011, T012 y T013.
- T021 y T022, después de T020.
- T027 y T030 en paralelo con T028; T029 comparte archivo con T028 y va después.
- T032 y T033.
- T037 y T038; T040 en paralelo con T039 (paquetes distintos); T041 comparte archivo con T030 y va después.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T029, T034, T040, T045 |
| FR-002 | T026, T028, T032, T034 |
| FR-003 | T026, T028, T033 |
| FR-004 | T021, T026, T028, T034 |
| FR-005 | T027, T028, T034 |
| FR-006 | T027, T028, T033, T034 |
| FR-007 | T027, T028, T034 |
| FR-008 | T013, T018, T028 |
| FR-009 | T011, T013, T018, T021, T028 |
| FR-010 | T011, T013, T016, T018, T028 |
| FR-011 | T013, T021, T024 |
| FR-012 | T012, T013, T017, T018, T022 |
| FR-013 | T011, T013, T021, T028 |
| FR-014 | T006, T009, T029, T040 |
| FR-015 | T012, T017, T022, T024 |
| FR-016 | T021, T026, T028, T039 |
| FR-017 | T012, T017, T029, T034, T038, T040 |
| FR-018 | T005, T006, T008, T009, T049 |
| FR-019 | T028, T040, T047 |
| FR-020 | T029, T040 |
| FR-021 | T016, T029, T033, T043 |
| FR-022 | T027, T040, T045 |
| FR-023 | T027, T028, T033, T034 |
| FR-024 | T027, T037, T038, T040, T045 |
| FR-025 | T037, T038, T043 |
| FR-026 | T037, T038, T039, T043 |
| FR-027 | T037, T038, T039, T043, T044 |
| FR-028 | T037, T039, T043, T044 |
| FR-029 | T037, T039, T043, T044 |
| FR-030 | T038, T039, T044 |
| SC-001 | T013, T021, T028, T037, T039, T040 |
| SC-002 | T022, T039 |
| SC-003 | T021, T026, T028, T039 |
| SC-004 | T012, T029, T038, T040 |
| SC-005 | T029, T040 |
| SC-006 | T049 |
| SC-007 | T007, T014, T023, T031, T042, T048 |

## Estrategia de implementación

1. Fase 1: confirmar HU-08, HU-10 y HU-12 (y con ellas HU-06 y HU-15) en `main` (T001) y crear la rama.
2. Bloque Generador PDF: el PDF válido, probado sin base.
3. Bloque Dominio del reporte de un Sprint: el reporte completo en memoria, finalizado y en curso.
4. Bloque Almacenamiento: los datos reales de PostgreSQL y la coherencia con HU-07 y HU-10.
5. Bloque Pantalla y descarga: la sección "Reportes" con el reporte de un Sprint. Es el primer incremento visible.
6. Bloque Reporte del proyecto: el informe del proyecto que pide la DoD de la entrega final. Reutiliza todo lo
   anterior; si el Sprint se queda sin tiempo, es lo que se termina primero después de la Fase 5.
7. Cierre: evidencias, verificación manual en dos lectores, los dos PDF de ejemplo y PR.
