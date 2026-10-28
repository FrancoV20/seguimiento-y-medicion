# Plan de implementación: HU-12 – Consulta del Estado de un Proyecto

**Rama**: `HU-12` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/012-project-status/spec.md`

## Resumen

HU-12 agrega la sección "Estado del proyecto" (`GET /estado`), de solo lectura, con el estado por fechas, el avance,
las historias por estado y el Sprint en curso del proyecto activo.

- Dos reglas puras nuevas en `src/project/status.go`: `StatusOn` (estado por fechas) y `ProgressPercentage` (avance
  con la regla de redondeo de HU-07). HU-09 las reutiliza.
- Un catálogo nuevo, `ProjectStatusCatalog`, lee nombre, fechas y cantidades por estado en una sola transacción de
  solo lectura.
- El Sprint en curso sale del Tablero de HU-14, sin modificarlo.
- La pantalla usa el prefijo `ProjectStatus` en `src/web`. No hay migración y no se guarda nada.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`, `time`, `fmt`) y
`github.com/jackc/pgx/v5/stdlib`, que ya están en el proyecto. No se agregan dependencias.
**Almacenamiento**: PostgreSQL. Sin migración: solo lectura de `projects` y `user_stories`; el Tablero (HU-14) lee
`sprints` y `sprint_story_assignments`.
**Pruebas**:

- unitarias de `StatusOn` y `ProgressPercentage` (tablas) y de la vista;
- integración del catálogo contra PostgreSQL real con datos propios;
- handlers con `httptest`, con el catálogo y el Tablero reales y un reloj fijo (28/10/2026), y con dobles que fallan;
- BDD en `test/`.

Se corren con `go test -count=1 -p 1 ./...`. Limpieza de los datos de prueba: asignaciones, historias, Sprints y
proyectos (research.md §10).
**CI**: `gofmt -l` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. Se formatean solo los
archivos que se crean; `main.go` se edita a mano respetando su fin de línea.
**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript.
**Objetivos de rendimiento**: menos de 500 ms con 1000 historias y un Sprint activo con 50 (SC-004).
**Restricciones**:

- No se editan archivos de HU-14, HU-21 ni HU-11 (`projects_handlers.go`, `projects_catalog.go`, `projects.html`), ni
  los existentes de `src/project`.
- El único archivo compartido que se edita es `src/cmd/server/main.go`.
- Ningún mensaje expone detalles internos.

**Escala**: proyectos del curso.

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Should Have, valor 75. Cada FR sale de un escenario del Issue o de una decisión del equipo; el estado y el avance son "project health indicators" que pide la constitución. | PASA |
| II. Calidad por construcción | Las reglas son puras y están en el dominio (`src/project`), separadas de la lectura y de la pantalla. Se reutilizan el Tablero (HU-14), `projectIDFrom`, `Fail` y `RequireContext`, sin copiarlos. La regla de redondeo de HU-07 se reimplementa con enteros porque la original no está exportada (research.md §3). Los textos están en un solo lugar. | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | T001 confirma los nombres de HU-01, HU-14 y HU-20a/b; T004 y T028 coordinan `main.go` con las demás historias y avisan a HU-08 y HU-09 del contrato. | PASA |
| V. Medición | Agrega el estado y el avance del proyecto, que la constitución pide entre los indicadores de salud del proyecto. | PASA |
| VI. Persistencia PostgreSQL | El único puerto nuevo, `ProjectStatusCatalog`, tiene adaptador PostgreSQL y prueba de integración. `BoardCatalog` ya tiene el suyo. | PASA |

## Estructura

### Documentación

```text
specs/012-project-status/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
├── checklists/requirements.md
├── evidencia-red.txt        (durante la implementación)
├── evidencia-pruebas.txt    (cierre)
└── cobertura.txt            (cierre)
```

No hay carpeta `contracts/`. La ruta está en este plan y el contrato para HU-09 (`StatusOn` y
`ProgressPercentage`), en [data-model.md](data-model.md).

### Código

Archivos nuevos:

```text
src/project/status.go                                    DateStatus, StatusOn, ProgressPercentage
src/project/status_test.go
src/web/project_status_catalog.go                        ProjectStatusCounts, ProjectStatusSummary, ProjectStatusCatalog
src/web/postgres_project_status_catalog.go               adaptador PostgreSQL (solo lectura)
src/web/postgres_project_status_catalog_integration_test.go
src/web/project_status_view.go                           textos, formatos y buildProjectStatusView
src/web/project_status_view_test.go
src/web/project_status_handlers.go                       RegisterProjectStatus y GET /estado
src/web/project_status_handlers_test.go
src/web/templates/project_status.html
test/web_project_status_bdd_test.go
```

Archivo existente que se edita: `src/cmd/server/main.go`, para sumar un bloque de registro después del último (hoy el
de HU-24).

No se tocan `src/project/{errors,model,postgres_store,project,service,store,test_store}.go` (HU-01), los archivos de
HU-14 (`src/web/board*.go`, `postgres_board_catalog.go`), los de HU-21/HU-11 (`src/web/projects*.go`,
`templates/projects.html`) ni `db/migrations`.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /estado` | `show` | 200; 404 si el proyecto ya no existe; 303 sin proyecto; 500 o 503 si falla una lectura |
| `POST /estado` (u otro método) | — | 405 de HU-20a |

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. `StatusOn(start, end, today)` compara días del calendario; no lee el reloj.
2. `ProgressPercentage(done, total)` redondea mitad hacia arriba con enteros y devuelve texto con punto ("50.00",
   "0.00").
3. `ProjectStatusCatalog.StatusSummary` lee nombre, fechas y cantidades en una transacción `ReadOnly` +
   `REPEATABLE READ`; sin fila → `NotFound`.
4. El Sprint en curso sale de `BoardCatalog.ActiveBoard` (HU-14).
5. `RegisterProjectStatus(s, catalog, board, now)` con reloj inyectable; cualquier error va a `Fail` antes de escribir.
6. Los textos están en `project_status_view.go`.

## Contratos que se consumen

- **HU-01**: tabla `projects(id, name, start_date, end_date)`.
- **HU-02/HU-04**: `user_stories(project_id, status)` con los cuatro estados.
- **HU-14**: `BoardCatalog.ActiveBoard`, `BoardView` (`Label`, `Goal`, `StartDate`, `EndDate`),
  `NewPostgreSQLBoardCatalog` y `projectIDFrom`.
- **HU-20a / HU-20b**: `Server.Register`, `Routes`, `Section.RequiresProject`, `RequireContext`, `ActiveContextFrom`,
  `Renderer.Page/Fail`, `NotFound`, el menú, `layout.html` y el 405 de `Server.route`.

## Contrato que se ofrece (HU-09 y HU-08)

```go
estado := project.StatusOn(inicio, fin, hoy)            // "No iniciado", "En curso" o "Finalizado"
porcentaje := project.ProgressPercentage(hechas, total)  // "57.14"; "0.00" sin historias
```

HU-08 puede reutilizar `ProjectStatusCatalog` para las historias por estado si HU-12 entra primero (notas del equipo).

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Comparar instantes en lugar de días corre el estado un día según la zona del servidor | `StatusOn` compara `(año, mes, día)`; la prueba de tabla incluye las 00:00, las 23:59 y un caso en UTC-3 (research.md §2). |
| La regla de redondeo se reimplementa y podría diferir de la de HU-07 | Misma regla (mitad hacia arriba) probada con los casos de HU-10 y una mitad exacta (1 de 32 → "3.13"). Si HU-07 exporta su función más adelante, se reemplaza sin cambiar el contrato. |
| HU-09 ya documentó el contrato de HU-12 con nombres "a confirmar" | T028 avisa a HU-09 los nombres `StatusOn` y `ProgressPercentage` para su T001. |
| HU-08 y HU-12 muestran historias por estado con consultas propias | Las dos usan los mismos cuatro estados; si HU-12 entra primero, HU-08 puede reutilizar este catálogo (T028). |
| Conflicto en `main.go` con HU-08, HU-09, HU-10, HU-25 u otras historias | Un bloque independiente al final; al mergear se conservan todos (T004, T028). |
| Fin de línea CRLF en la copia de Windows de `main.go` | Se edita a mano sin `gofmt -w`; `gofmt -l` en el CI corre sobre el archivo con LF. |
| El `TRUNCATE` del CI pisa datos de otras pruebas | Cada prueba arma sus datos y corre con `-p 1`. |
| "Planificación de proyecto.pdf" no está en el repositorio | Se usaron el Issue y las notas del equipo; si la planificación dice otra cosa, prevalece y se corrige la spec antes de T002. |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: `ProjectStatusCatalog` tiene adaptador PostgreSQL y prueba de integración.
- **Principio II**: no se duplicó el Tablero; la única regla reimplementada (redondeo) está justificada.
- **Principio III**: las tareas siguen RED → evidencia → GREEN en cada bloque.

Sigue en **PASA**, sin violaciones que justificar.
