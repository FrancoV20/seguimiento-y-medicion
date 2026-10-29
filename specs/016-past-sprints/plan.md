# Plan de implementación: HU-16 – Consulta de Sprints Anteriores

**Rama**: `HU-16` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/016-past-sprints/spec.md`

## Resumen

HU-16 agrega la sección "Sprints anteriores", de solo lectura, con dos pantallas:

- `GET /sprints-anteriores`: lista de los Sprints finalizados del proyecto activo, del más reciente al más antiguo;
- `GET /sprints-anteriores/{id}`: detalle de un Sprint, con objetivo, fechas, velocidad y las historias completadas y
  no completadas.

Cómo se arma:

- Un catálogo nuevo, `PastSprintCatalog`, lee los Sprints finalizados y el registro de cierre de HU-15
  (`sprint_story_results`, T001: HU-15) unido a las historias, en una transacción `REPEATABLE READ`.
- La velocidad sale de HU-07 (`CalculateMetrics`) detrás de la interfaz chica `PastSprintMetrics`.
- La pantalla usa el prefijo `PastSprint` en `src/web`.
- No hay migración ni dominio nuevo.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`, `time`, `fmt`) y
`github.com/jackc/pgx/v5/stdlib`, que ya están en el proyecto. Se importa `src/metrics` (HU-07) desde `src/web`. No se
agregan dependencias.
**Almacenamiento**: PostgreSQL, sin migración. HU-16 lee `sprints`, `sprint_story_results` (HU-15) y `user_stories`.
HU-07 lee `sprints`, `sprint_story_assignments`, `user_stories` y `effort_logs`, y escribe `sprint_metrics`.
**Pruebas**:

- unitarias de la vista (tablas);
- integración del catálogo contra PostgreSQL real con datos propios, cargados como los deja HU-15;
- handlers con `httptest`, con el catálogo y HU-07 reales y con dobles que fallan;
- prueba de coherencia: la velocidad mostrada es igual a la suma de los Story Points de las completadas del registro;
- BDD en `test/`.

Se corren con `go test -count=1 -p 1 ./...`. Limpieza de los datos de prueba, en este orden: registro de cierre,
asignaciones, `sprint_metrics`, historias, Sprints y proyecto (research.md §10).
**CI**: `gofmt -l` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. Se formatean solo los
archivos que se crean; `main.go` se edita a mano respetando su fin de línea.
**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript.
**Objetivos de rendimiento**: menos de 500 ms por pantalla con 50 Sprints finalizados y 200 historias en el registro
de un Sprint (SC-004).
**Restricciones**:

- No se editan archivos de HU-07 (`src/metrics`), HU-15, HU-25 ni de otras historias.
- El único archivo compartido que se edita es `src/cmd/server/main.go`.
- Ningún mensaje expone detalles internos.

**Escala**: proyectos del curso.

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Should Have, valor 65. Cada FR sale de un escenario del Issue o de una decisión del equipo; mirar los Sprints pasados sirve para la retrospectiva y la estimación. | PASA |
| II. Calidad por construcción | La velocidad no se recalcula: se usa HU-07. La separación entre completadas y no completadas es una función pura de la vista. Se reutilizan `parseID`, `projectIDFrom`, `Fail`, `NotFound` y `RequireContext`, sin copiarlos. Los textos están en un solo lugar. | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | T001 confirma los nombres de HU-15 y frena si HU-15 no está en `main`. T003 y T029 coordinan `main.go` y el uso del registro de cierre con HU-10, HU-15 y HU-25. | PASA |
| V. Medición | Muestra la velocidad de cada Sprint finalizado y lo que no se completó, base del aprendizaje entre Sprints que pide la constitución. | PASA |
| VI. Persistencia PostgreSQL | El único puerto nuevo con base, `PastSprintCatalog`, tiene adaptador PostgreSQL y prueba de integración. `PastSprintMetrics` la cumple `*metrics.Service`, que ya tiene su adaptador (`PostgreSQLMetricsStore`) probado. | PASA |

## Estructura

### Documentación

```text
specs/016-past-sprints/
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

No hay carpeta `contracts/`. Las rutas están en este plan, y los tipos y las consultas, en
[data-model.md](data-model.md).

### Código

Archivos nuevos:

```text
src/web/past_sprint_catalog.go                           PastSprintSummary, PastSprintStory, PastSprintDetail,
                                                         PastSprintCatalog, PastSprintMetrics
src/web/postgres_past_sprint_catalog.go                  adaptador PostgreSQL (solo lectura)
src/web/past_sprint_fixtures_test.go                     datos de «Alfa», «Beta» y «Gamma» como los deja HU-15
src/web/postgres_past_sprint_catalog_integration_test.go
src/web/past_sprint_view.go                              textos, formatos, buildPastSprintListView/DetailView
src/web/past_sprint_view_test.go
src/web/past_sprint_handlers.go                          RegisterPastSprints, lista y detalle
src/web/past_sprint_handlers_test.go
src/web/templates/past_sprints.html
src/web/templates/past_sprint.html
test/web_past_sprints_bdd_test.go
```

Archivo existente que se edita: `src/cmd/server/main.go`. Se suma el import de `src/metrics` y un bloque de registro
después del último (hoy el de HU-24).

No se tocan `src/metrics/*` (HU-07), los archivos de HU-15 (`src/sprint`, su migración y su pantalla), los de HU-25
ni `db/migrations`.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /sprints-anteriores` | `list` | 200; 303 sin proyecto; 500 o 503 si falla la lectura |
| `GET /sprints-anteriores/{id}` | `detail` | 200; 404 si el `{id}` es inválido, no existe, es de otro proyecto o el Sprint está en curso; 303 sin proyecto; 500 o 503 si falla el catálogo o HU-07 |
| `POST`, `PUT`, `DELETE`… en las dos | — | 405 de HU-20a |

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. `PastSprintCatalog.FinishedSprints` lee los Sprints 'Finalizado' del proyecto por número descendente.
2. `PastSprintCatalog.SprintDetail` filtra por `id`, proyecto y 'Finalizado' en la misma consulta (sin fila →
   `NotFound`) y lee el registro de cierre con el título, en `ReadOnly` + `REPEATABLE READ`.
3. `PastSprintMetrics` = `CalculateMetrics(sprintID)` de HU-07; en `main.go`,
   `metrics.NewService(metrics.NewPostgreSQLMetricsStore(database))`.
4. Orden del detalle: `parseID` → catálogo → HU-07 → vista. El 404 se decide antes de llamar a HU-07, y nada se
   escribe antes de la vista.
5. `buildPastSprintDetailView` separa por `FinalStatus == string(sprint.Done)`. Sin filas, muestra solo el mensaje
   "No hay registro…" y la velocidad.
6. Los textos están en `past_sprint_view.go`.

## Contratos que se consumen

- **HU-07**: `metrics.NewService`, `metrics.NewPostgreSQLMetricsStore`, `(*Service).CalculateMetrics(sprintID int64)
  (SprintMetrics, error)` y `SprintMetrics.Velocity`. `CalculateMetrics` reescribe los mismos valores en
  `sprint_metrics`.
- **HU-15** (T001: HU-15): tabla `sprint_story_results(sprint_id, story_id, final_status, story_points)` con
  `final_status` en 'En Sprint', 'En progreso' o 'Hecho'. El cierre borra las asignaciones de las no completadas y
  conserva las de las "Hecho".
- **HU-04**: `sprints(id, project_id, number, goal, start_date, end_date, status)`.
- **HU-14**: `projectIDFrom` y `sprint.Done`.
- **HU-20a / HU-20b**: `Server.Register`, `Routes`, `Section.RequiresProject`, `RequireContext`, `parseID`,
  `Renderer.Page/Fail`, `NotFound`, el menú, `layout.html` y el 405 de `Server.route`.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| HU-15 no está en `main`; sus nombres (`sprint_story_results` y sus columnas) pueden cambiar | T001 confirma los nombres y detiene todo si HU-15 no está en `main`; si cambiaron, se corrigen `data-model.md`, las consultas y las fixtures antes de T002. |
| La velocidad de HU-07 (historias "Hecho" asignadas, con sus Story Points actuales) puede no coincidir con el registro si alguien cambia datos después del cierre | La pantalla muestra la de HU-07 (decisión del equipo). `TestHU16_VelocidadCoherente` comprueba la igualdad con datos como los deja HU-15 y avisa si HU-15, HU-13 o HU-17 cambian ese supuesto. |
| Un GET escribe en `sprint_metrics` (efecto de HU-07) | Documentado (decisión del equipo, como en HU-25). Es un `upsert` de los mismos valores; las pruebas borran esas filas en la limpieza. |
| Un Sprint "Finalizado" cargado a mano con una historia "Hecho" sin Story Points hace fallar HU-07 → 500 | Caso límite documentado en la spec; HU-15 no deja cerrar un Sprint así. |
| `CalculateMetrics` no recibe `context`: no se cancela si el usuario corta el pedido | Es la firma de HU-07, que no se edita. Son dos consultas y un `upsert`, cubiertos por SC-004. |
| Conflicto en `main.go` (bloque e import) con HU-08, HU-09, HU-10, HU-12, HU-15, HU-23 o HU-25 | Un bloque independiente al final; si otra historia ya importó `src/metrics`, no se repite. Al mergear se conservan todos (T003, T029). |
| Fin de línea CRLF en la copia de Windows de `main.go` | Se edita a mano sin `gofmt -w`; `gofmt -l` en el CI corre sobre el archivo con LF. |
| El `TRUNCATE` del CI pisa datos de otras pruebas | Cada prueba arma sus datos y corre con `-p 1`. |
| "Planificación de proyecto.pdf" no está en el repositorio | Se usaron el Issue, las notas del equipo y los datos de HU-09; si la planificación dice otra cosa, prevalece y se corrige la spec antes de T002. |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: `PastSprintCatalog` tiene adaptador PostgreSQL y prueba de integración. HU-07 ya tiene el suyo.
- **Principio II**: no se recalcula la velocidad ni se duplica el cierre de HU-15. La regla nueva (separar
  completadas y no completadas) es pura y está probada.
- **Principio III**: las tareas siguen RED → evidencia → GREEN en cada bloque.

Sigue en **PASA**, sin violaciones que justificar.
