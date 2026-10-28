# Plan de implementación: HU-09 – Exportación de Reportes en PDF

**Rama**: `HU-09` | **Fecha**: 2026-10-28 (actualizado: reporte del proyecto) | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/009-pdf-report/spec.md`
**Orden de implementación**: después de HU-08, HU-10 y HU-12 (y por lo tanto de HU-06 y HU-15), que tienen que estar
en `main` antes de crear la rama (T001).

## Resumen

HU-09 agrega la sección "Reportes" (`GET /reportes`) con los Sprints del proyecto activo, la descarga del PDF de un
Sprint (`GET /reportes/sprints/{id}/pdf`) y la del PDF del proyecto (`GET /reportes/proyecto/pdf`). Un Sprint en
curso, o un proyecto con un Sprint en curso, pide confirmación y sale marcado como parcial.

- **`src/pdf`** (nuevo): generador de PDF 1.4 sin dependencias externas, con Helvetica y Helvetica-Bold,
  `WinAnsiEncoding`, corte de líneas por ancho, tablas con encabezado repetido, saltos de página, pie "Página X de
  Y" y `ExtractText` para las pruebas.
- **`src/report`** (nuevo): los modelos `SprintReport` y `ProjectReport`, un solo puerto `Store` con su adaptador
  PostgreSQL (una lectura `REPEATABLE READ` por reporte), el servicio que suma HU-07, HU-08, HU-10 y HU-12, y la
  composición de los dos documentos.
- **`src/web`**: catálogo de la pantalla, vista, handlers y dos plantillas, con el prefijo `Report`.
- No hay migración y no se guarda nada propio.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`, `bytes`, `unicode/utf8`) y
`github.com/jackc/pgx/v5/stdlib`, que ya están en el proyecto. No se agregan dependencias (sin bibliotecas de PDF,
research.md §2).
**Almacenamiento**: PostgreSQL. Sin migración: lectura de `projects`, `project_members`, `members`, `sprints`,
`user_stories`, `sprint_story_assignments`, `sprint_story_results` (HU-15), `effort_logs` y `defects` (HU-06); HU-07
reescribe `sprint_metrics` de cada Sprint finalizado con los mismos valores.
**Pruebas**:

- unitarias del generador (`src/pdf`): estructura (`%PDF-1.4`, `%%EOF`, desplazamientos de la `xref` verificados,
  cantidad de páginas), textos en el contenido con acentos codificados en WinAnsi, escape de paréntesis y barras,
  anchos y corte de líneas, salto de página con 120 filas, encabezado repetido y "Página X de Y" en todas,
  `ExtractText`;
- unitarias del reporte (`src/report`): servicio con dobles (incluidas las fallas de cada fuente), textos y formatos,
  y composición de los dos documentos;
- integración contra PostgreSQL real con datos propios (`src/report` y `src/web`), incluida la coherencia con HU-07,
  HU-08, HU-10 y HU-12;
- handlers con `httptest`, con los servicios reales y datos propios, y con dobles que fallan;
- BDD en `test/`.

Se corren con `go test -count=1 -p 1 ./...`. Limpieza de los datos de prueba: defectos, registro de cierre,
asignaciones, `sprint_metrics`, `effort_logs`, historias, Sprints, proyectos e integrantes (research.md §15).
**CI**: `gofmt -l` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. Se formatean solo los
archivos que se crean o se tocan.
**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript.
**Objetivos de rendimiento**: el reporte de un Sprint y el del proyecto, con 120 historias y 50 defectos, en menos de
1 s cada uno (SC-005).
**Restricciones**:

- No se editan archivos de HU-07, HU-08, HU-10 ni HU-12.
- El único archivo compartido que se edita es `src/cmd/server/main.go`.
- Ningún mensaje ni los PDF exponen detalles internos.
- Cada PDF se arma completo en memoria antes de escribir la respuesta.

**Escala**: proyectos del curso (un reporte típico entra en una o dos páginas).

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Should Have, valor 70: la DoD de la entrega final pide el informe del proyecto en PDF. Cada FR sale de un escenario del Issue o de una decisión del equipo; "reporting" está en el alcance que pide la constitución. | PASA |
| II. Calidad por construcción | Generador, reporte y web en paquetes separados con interfaces chicas. Se reutilizan `CalculateMetrics` (HU-07), `SummarizeVelocity` (HU-08), `SprintQuality` (HU-10), las reglas de estado y avance de HU-12, `effort.Hours` (HU-03), `parseID`, `Fail` y `RequireContext` (HU-20a/b), sin copiarlos. Un solo puerto para los dos reportes. Los textos están en un solo lugar por capa. El formato de la desviación se escribe en `src/report` porque el de HU-25 está en `src/web` (research.md §12). | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | T001 confirma los nombres de HU-06, HU-07, HU-08, HU-10, HU-12, HU-15 y HU-17 y frena si HU-08, HU-10 o HU-12 no están. T004 y T050 coordinan `main.go` y avisan del generador de PDF. | PASA |
| V. Medición | Los PDF dejan como evidencia velocidad, desviación, avance y defectos de cada Sprint y del proyecto ("project health indicators"), para la Sprint Review y la entrega final. | PASA |
| VI. Persistencia PostgreSQL | Los dos puertos nuevos, `report.Store` (con sus dos lecturas) y `web.ReportCatalog`, tienen adaptador PostgreSQL y prueba de integración. `MetricsSource`, `QualitySource` y `ReportGenerator` no son puertos de almacenamiento: son la forma de servicios que ya existen. | PASA |
| Restricción de tecnología | Sin dependencias nuevas: el PDF se genera con la biblioteca estándar. No hace falta enmendar la constitución; `go-pdf/fpdf` se descartó por eso (research.md §2). | PASA |

## Estructura

### Documentación

```text
specs/009-pdf-report/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
├── checklists/requirements.md
├── evidencia-red.txt                (durante la implementación)
├── evidencia-pruebas.txt            (cierre)
├── cobertura.txt                    (cierre)
├── ejemplo-reporte-sprint.pdf       (cierre: PDF del Sprint 2 de «Alfa», verificación manual)
└── ejemplo-reporte-proyecto.pdf     (cierre: PDF del proyecto «Alfa», verificación manual)
```

No hay carpeta `contracts/`: las interfaces externas son las tres rutas (tabla de este plan) y el formato del PDF;
los tipos están en [data-model.md](data-model.md).

### Código

Archivos nuevos:

```text
src/pdf/document.go                                  Document, Font, Column, Text, Space, Table, Bytes
src/pdf/winansi.go                                   encodeWinAnsi, escapeLiteral, Literal
src/pdf/widths.go                                    anchos AFM, textWidth, wrapText
src/pdf/writer.go                                    objetos, xref, trailer
src/pdf/extract.go                                   ExtractText
src/pdf/document_test.go
src/pdf/winansi_test.go
src/pdf/widths_test.go
src/pdf/pdftest_test.go                              ayudante de pruebas: xref, páginas y flujos
src/report/model.go                                  Data, Story, MemberHours, Defect, SprintReport, errores
src/report/project_model.go                          ProjectData, ProjectSprintRow, ProjectReport
src/report/store.go                                  puerto Store (LoadSprintReport, LoadProjectReport)
src/report/sources.go                                MetricsSource, QualitySource
src/report/service.go                                Service, NewService, Build, SprintPDF
src/report/project_service.go                        BuildProject, ProjectPDF
src/report/texts.go                                  textos y formatos de los dos PDF
src/report/document.go                               Render
src/report/project_document.go                       RenderProject
src/report/postgres_store.go                         adaptador PostgreSQL (solo lectura, dos métodos)
src/report/service_test.go
src/report/project_service_test.go
src/report/texts_test.go
src/report/document_test.go
src/report/project_document_test.go
src/report/fixtures_test.go                          datos de prueba de PostgreSQL
src/report/postgres_store_integration_test.go
src/web/report_catalog.go                            ReportSprintOption, ReportCatalog, ReportGenerator
src/web/postgres_report_catalog.go
src/web/report_view.go                               textos y armado de la pantalla
src/web/report_handlers.go                           RegisterReports, GET /reportes y las dos descargas
src/web/templates/reports.html
src/web/templates/report_warning.html
src/web/report_fixtures_test.go
src/web/postgres_report_catalog_integration_test.go
src/web/report_view_test.go
src/web/report_handlers_test.go
src/web/report_project_handlers_test.go
test/web_report_bdd_test.go
```

Archivo existente que se edita: `src/cmd/server/main.go`, para sumar un bloque de registro después del último.

No se tocan los archivos de HU-07 (`src/metrics/{calculation,closure_validation,errors,model,service,store,
test_store,postgres_metrics_store}.go`), de HU-08 (`src/metrics/velocity_summary.go`, `src/web/dashboard*.go`), de
HU-10 (`src/metrics/quality*.go`, `src/web/quality*.go`), de HU-12 (`src/project`), de HU-03 (`src/effort`), de
HU-25 si ya está en `main`, ni `db/migrations`.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /reportes` | `list` | 200 (lista o "Aún no hay Sprints para exportar", con el botón del proyecto); 303 sin proyecto; 500 o 503 si falla la lectura |
| `GET /reportes/sprints/{id}/pdf[?confirmar=si]` | `downloadSprint` | 200 PDF (finalizado, o en curso confirmado); 200 advertencia (en curso sin confirmar); 404 inválido, inexistente o ajeno; 303 sin proyecto; 500 o 503 ante cualquier falla |
| `GET /reportes/proyecto/pdf[?confirmar=si]` | `downloadProject` | 200 PDF (sin Sprint en curso, o confirmado); 200 advertencia (con Sprint en curso sin confirmar); 303 sin proyecto; 500 o 503 ante cualquier falla |
| `POST` (u otro método) en las tres rutas | — | 405 de HU-20a |

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. Tres capas: `src/pdf` (genérico), `src/report` (dominio, puerto y armado de los dos reportes) y `src/web`
   (pantalla y descargas).
2. Generador propio con la biblioteca estándar; `go-pdf/fpdf` se descartó porque exigiría enmendar la constitución.
3. WinAnsi con Latin-1 directo más nueve extras; lo demás, "?". Escape de `(`, `)` y `\`.
4. Anchos AFM de Helvetica y Helvetica-Bold para cortar por palabras; palabra larga cortada por caracteres.
5. Dos pasadas: se arman las páginas y recién al final se escriben el pie "Página X de Y", los objetos y la `xref`.
   Salida determinista. `ExtractText` devuelve el texto unido para las pruebas.
6. Un solo puerto `report.Store`: `LoadSprintReport` y `LoadProjectReport`, cada uno en una transacción `ReadOnly` +
   `REPEATABLE READ`; las horas llegan como centésimas (`effort.Hours`).
7. Reporte de un Sprint finalizado: HU-07 y HU-10 con interfaces chicas y control de coherencia con HU-10. En curso:
   no se llaman y se muestra "Se calculan al cerrar el Sprint".
8. Reporte del proyecto: estado y avance con HU-12, HU-07 y HU-10 por cada Sprint finalizado, acumulada y promedio
   con `SummarizeVelocity` de HU-08 sobre las velocidades de HU-07 (research.md §18).
9. Web: `ReportCatalog` para la lista, para ubicar el Sprint y para saber si el proyecto tiene un Sprint activo;
   `ReportGenerator` (lo cumple `*report.Service`) para los dos PDF. `parseID` → 404 sin consultar. Los encabezados y
   el cuerpo se escriben solo con el PDF completo.
10. Textos de los PDF en `src/report/texts.go`; de la pantalla, en `src/web/report_view.go`.

## Contratos que se consumen

- **HU-07**: `NewService`, `NewPostgreSQLMetricsStore`, `CalculateMetrics` (`Velocity`, `DeviationPercentage`,
  `IsPartial`, `Warning`).
- **HU-08** **(T001)**: `metrics.SummarizeVelocity` y `VelocitySummary` (`Total`, `Average`).
- **HU-10** **(T001)**: `NewQualityService`, `NewPostgreSQLQualityStore`, `SprintQuality` (`StoriesPlanned`,
  `StoriesCompleted`, `CompletionPercentage`, `PointsPlanned`, `PointsCompleted`, `DefectsDetected`,
  `DefectsResolved`).
- **HU-12** **(T001)**: en `src/project`, la regla de estado por fechas (con la fecha de referencia) y la de
  porcentaje de avance; nombres y firmas a confirmar (no tiene documentos en el repositorio).
- **HU-06** **(T001)**: `defects(id, project_id, story_id, description, severity, status, detection_sprint_id,
  resolution_sprint_id)`.
- **HU-15** **(T001)**: `sprint_story_results(sprint_id, story_id, final_status, story_points)` y el estado en que el
  cierre deja historias y asignaciones (para los datos de prueba).
- **HU-17** **(T001)**: `user_stories.estimated_hours` sin cambios de nombre ni de tipo.
- **HU-03**: `effort.Hours` y su `String()`.
- **HU-20a / HU-20b**: `Server.Register`, `Routes`, `Section.RequiresProject`, `RequireContext`, `ActiveContextFrom`,
  `Renderer.Page/PageStatus/Fail`, `NotFound`, `parseID`, el menú, `layout.html` y el 405 de `Server.route`.

## Contrato que se ofrece

```go
doc := pdf.New()
doc.Text(pdf.Bold, 16, "Título")
err := doc.Table([]pdf.Column{{Title: "Col", Width: 495}}, [][]string{{"valor"}})
b, err := doc.Bytes(func(page, total int) string { return fmt.Sprintf("Página %d de %d", page, total) })
text, err := pdf.ExtractText(b) // para pruebas
```

`src/pdf` no conoce el dominio: lo puede reutilizar cualquier otro reporte.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| HU-08, HU-10 o HU-12 (y con ellas HU-06 y HU-15) no están en `main`, o sus nombres cambian | T001 los confirma con `git grep -n` antes de crear la rama, ajusta `data-model.md` y frena si falta alguna de las tres. |
| HU-12 no tiene documentos: su regla de avance o de estado puede no coincidir con lo que fija el Issue ("0,00 %" sin historias, "En curso" en los bordes) | T001 compara su comportamiento con los escenarios 15, 16 y 18; si difiere, se avisa al equipo y se ajusta la spec antes de T002 (prevalece la planificación). |
| La regla de estado de HU-12 usa la hora del sistema y no acepta la fecha de generación | T001 lo verifica; si es así, se pide a HU-12 una variante con fecha o la prueba compara con lo que devuelve HU-12 en el mismo momento (research.md §9). |
| Un PDF hecho a mano que algún lector no abre (desplazamientos de la `xref`, `/Length`, codificación) | Pruebas que recorren la `xref` y comparan cada desplazamiento con `N 0 obj`, verifican `/Length` y `/Count` (T006); verificación manual en dos lectores (T049). |
| Anchos de fuente mal copiados: texto que se pisa o se sale de la columna | Prueba de `textWidth` con valores AFM conocidos y prueba de corte (T005); revisión visual en T049. |
| Caracteres fuera de WinAnsi en títulos o descripciones | Se reemplazan por "?" (caso límite con prueba); queda documentado. |
| `CalculateMetrics` reescribe `sprint_metrics` en cada exportación (en el reporte del proyecto, una vez por Sprint finalizado) | Escribe los mismos valores, como en HU-25 y HU-08 (decisión aceptada). Las pruebas borran esas filas. |
| HU-07 rechaza un Sprint (historia "Hecho" sin Story Points cargada a mano) y no se puede exportar ese Sprint ni el proyecto | Es el comportamiento pedido (cualquier falla → error); con el cierre de HU-15 no se llega a ese estado. |
| La lectura propia y la de HU-10 son transacciones distintas | Control de coherencia (`ErrInconsistentReport`) en el reporte de un Sprint (research.md §11). En el del proyecto no hay totales que comparar entre fuentes. |
| El reporte del proyecto llama a HU-07 y HU-10 una vez por Sprint finalizado (N + 1) | Medido en `TestHU09_Rendimiento`; aceptable en proyectos del curso, como en HU-08. |
| Las horas reales de una historia replanificada suman registros de otros Sprints, y las horas estimadas de una historia devuelta pueden editarse después del cierre | Es el criterio de HU-07 y HU-17; queda en los casos límite de la spec. Un "reporte congelado" requeriría guardar el PDF (fuera de alcance). |
| Un reporte muy grande ocupa memoria al armarse completo | 120 historias y 50 defectos generan pocas decenas de KB; se mide en `TestHU09_Rendimiento`. |
| Conflicto en `main.go` con HU-08, HU-10, HU-25 u otras historias | Un bloque independiente al final; al mergear se conservan todos (T004, T050). |
| El `TRUNCATE` del CI o pruebas en paralelo pisan datos | Cada prueba arma sus datos, los borra al final y corre con `-p 1`. |
| "Planificación de proyecto.pdf" no está en el repositorio | Se usaron el Issue, la DoD que citó el equipo y las notas; si la planificación dice otra cosa, prevalece y se corrige la spec antes de T002. |
| El tamaño creció de 5 a 8 SP con el reporte del proyecto | El bloque del proyecto reutiliza el generador, el puerto y la advertencia; está al final y se puede entregar después del de Sprint (estrategia en tasks.md). |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: `report.Store` (dos lecturas) y `web.ReportCatalog` tienen adaptador PostgreSQL y prueba de
  integración.
- **Principio II**: no se duplicaron los cálculos de HU-07, HU-08, HU-10 ni HU-12; el único formato propio
  (desviación) está justificado en research.md §12.
- **Principio III**: las tareas siguen RED → evidencia → GREEN en cada bloque.
- **Tecnología**: sin dependencias nuevas.

Sigue en **PASA**, sin violaciones que justificar.
