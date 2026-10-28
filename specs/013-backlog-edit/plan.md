# Plan de implementación: HU-13 – Consulta y Edición del Product Backlog (web)

**Rama**: `HU-13` | **Fecha**: 2026-10-27 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/013-backlog-edit/spec.md`
**Orden de implementación**: después de HU-15 y HU-17, que tienen que estar en `main` antes de crear la rama (T001).

## Resumen

HU-13 cambia la lista del Backlog de HU-22:

- la ordena por prioridad y antigüedad;
- le agrega un filtro por estado;
- suma un enlace "Editar" por historia, deshabilitado si la historia pertenece a un Sprint finalizado.

La pantalla nueva `/backlog/{id}/editar` cambia título, descripción, prioridad, Story Points y criterios en una
sola transacción.

- **Reutiliza**:
  - de HU-02, la validación (`validateStoryFields`, extraída de `CreateStory`);
  - de HU-22, el parseo, las acciones de criterios y los mensajes (`parseStoryFields`, `applyAction`,
    `mapBacklogDomainError`);
  - de HU-20a y HU-20b, la base web.
- **Agrega**:
  - el método `UpdateStory` al puerto `BacklogStore`, con su adaptador PostgreSQL;
  - la lectura `StoryForEdit` al catálogo de HU-22;
  - el dato `InFinishedSprint` en la lista.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`) y
`github.com/jackc/pgx/v5/stdlib`, ya presentes; sin dependencias nuevas.
**Almacenamiento**: PostgreSQL, sin migración nueva.
**Pruebas**:

- unitarias del dominio con `InMemoryBacklogStore`;
- integración con PostgreSQL (`*_integration_test.go`);
- handlers con `httptest` y `cookiejar`;
- BDD en `test/`.

**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor.
**Objetivos de rendimiento**: la lista, con y sin filtro, y el formulario responden en menos de 500 ms con 200
historias (SC-005).
**Restricciones**:

- CI: `gofmt -l`, `go vet` y `go test -race` tienen que pasar; se formatean solo los archivos que se tocan (la base
  ya está formateada desde el PR #44).
- Sin JavaScript.
- Sin token en los formularios (hay `CrossOriginGuard` global).
- Ningún mensaje expone detalles internos.

**Escala**: proyectos del curso (decenas a cientos de historias).

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Must Have, valor 85; cada FR sale de un escenario del Issue o de Clarifications. | PASA |
| II. Calidad por construcción | Una sola validación (`validateStoryFields`), un solo parseo (`parseStoryFields`), una sola lista de estados (`StoryStatuses`) y los textos en un solo lugar. | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces; cada bloque hace RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | Coordinación con HU-22, HU-02 y HU-15 (archivos y orden de bloqueos) en T003 y T086. | PASA |
| V. Medición | La regla de Sprint finalizado protege los Story Points que usa la velocidad (HU-07 y HU-15). | PASA |
| VI. Persistencia PostgreSQL | `BacklogStore.UpdateStory` con `PostgreSQLBacklogStore` y pruebas de integración; `StoryCatalog.StoryForEdit` y el nuevo `ListStories` con `PostgreSQLStoryCatalog` y pruebas de integración. | PASA |

## Estructura

### Documentación

```text
specs/013-backlog-edit/
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

No hay carpeta `contracts/`: las rutas están en este plan y el modelo de vista en `data-model.md`, como en HU-11.

### Código

Archivos nuevos:

```text
src/backlog/update.go                                   UpdateStoryRequest, UpdateStoryResult
src/backlog/update_errors.go                            ErrStoryNotFound, ErrStoryInFinishedSprint
src/backlog/update_service.go                           BacklogService.UpdateStory
src/backlog/postgres_backlog_store_update.go            PostgreSQLBacklogStore.UpdateStory
src/backlog/update_service_test.go
src/backlog/postgres_backlog_store_update_integration_test.go
src/web/backlog_edit_handlers.go                        edit (GET) y update (POST)
src/web/postgres_story_catalog_edit.go                  StoryForEdit
src/web/postgres_story_catalog_edit_integration_test.go
src/web/backlog_list_handlers_test.go                   orden y filtro
src/web/backlog_edit_handlers_test.go
src/web/templates/backlog_edit.html
test/web_backlog_edit_bdd_test.go
```

Archivos existentes que se editan:

```text
src/backlog/model.go                                    + StatusInSprint, StatusInProgress, StatusDone, StoryStatuses (HU-02)
src/backlog/store.go                                    + UpdateStory en BacklogStore (HU-02)
src/backlog/service.go                                  validateStoryFields extraída de CreateStory (HU-02)
src/backlog/test_store.go                               + UpdateStory, UpdateErr, FailOnUpdate, UpdateCalls (HU-02)
src/web/backlog_handlers.go                             + 2 rutas; filtro en get; EditURL e InFinishedSprint en pageData (HU-22)
src/web/backlog_form.go                                 parseStoryFields(r, withHours); textos de HU-13; backlogListURL (HU-22)
src/web/backlog_catalog.go                              + InFinishedSprint, StoryEdit, StoryForEdit (HU-22)
src/web/postgres_story_catalog.go                       nuevo ORDER BY y JOIN con Sprints (HU-22)
src/web/templates/backlog.html                          filtro, columna "Acciones", mensajes (HU-22)
src/web/postgres_story_catalog_integration_test.go      orden nuevo (HU-22)
src/web/backlog_handlers_test.go                        orden nuevo; StoryForEdit en hu22FailingStoryCatalog; UpdateStory en hu22FailingBacklogStore (HU-22)
test/web_backlog_bdd_test.go                            orden nuevo; UpdateStory en hu22BDDFailingBacklogStore (HU-22)
```

**No se tocan**: `src/cmd/server/main.go` (ya construye `BacklogService` con `PostgreSQLBacklogStore` y
`PostgreSQLStoryCatalog`), `db/migrations` ni `src/sprint`.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /backlog[?estado=…]` (HU-22) | `get` | 200 (lista ordenada, filtrada, con "Editar") |
| `POST /backlog` (HU-22) | `post` | sin cambios |
| `GET /backlog/{id}/editar[?estado=…]` | `edit` | 200 formulario; 303 si el Sprint está finalizado; 404; 500 o 503 si falla la lectura |
| `POST /backlog/{id}/editar` | `update` | 303; 200 al agregar o quitar un criterio; 422; 409; 404; 500 |

Las rutas nuevas van en el mismo `Routes` de `RegisterBacklog`, con la `Section` de HU-22 (`RequiresProject:
true`).

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. `BacklogStore` suma `UpdateStory(ctx, projectID, story)` y `BacklogService` suma `UpdateStory(ctx, req)`. Ni
   `main.go` ni la firma de `RegisterBacklog` cambian.
2. `validateStoryFields` es la única validación, tanto para el alta como para la edición.
3. Los estados viven en el dominio (`StoryStatuses`). El filtro se aplica en la web sobre la lista ya ordenada.
4. Orden: `CASE priority`, después `created_at` y después `id`.
5. Transacción: Sprint `FOR SHARE`, después historia `FOR NO KEY UPDATE`, con relectura de la asignación. Los
   criterios se reemplazan completos.
6. El 404 se decide antes de leer el formulario. Con el Sprint finalizado: 409 al enviar, con los datos enviados (el
   formulario se lee antes, sin validar), y 303 con advertencia al abrir.
7. La edición conserva el filtro; `backlogListURL` valida el estado antes de armar la dirección.
8. La edición ignora `horas_estimadas` (`parseStoryFields(r, false)`).
9. El estado inválido se informa junto al filtro, con respuesta 200.
10. Mapeo de respuestas: tabla en [data-model.md](data-model.md#mapeo-de-respuestas).

## Flujo del `POST /backlog/{id}/editar`

1. `id := parseID(r.PathValue("id"))`; si da 0, `Fail(NotFound)`.
2. `active := ActiveContextFrom(ctx)`. `RequireContext` ya garantiza el proyecto.
3. `catalog.StoryForEdit(ctx, active.ProjectID, id)`: `NotFound` → 404; otro error → `Fail`.
4. `ParseForm`, `parseStoryFields(r, false)` y el filtro oculto `estado`. Solo se leen los datos, sin validarlos ni
   aplicar la acción, para poder mostrarlos en cualquier respuesta que siga.
5. Si `InFinishedSprint`, 409 con el formulario enviado y `MsgStoryInFinishedSprint`. Va antes de la acción, así que
   "Agregar criterio" o "Quitar" sobre una historia de un Sprint finalizado también responden 409.
6. `applyAction()`: si la acción es agregar o quitar, 200 con el formulario.
7. Errores de formato o falta de criterios: 422.
8. `creator.UpdateStory(ctx, UpdateStoryRequest{...})`: los errores se mapean según la tabla.
9. Éxito: `SetFlash` (éxito o advertencia) y `RedirectSeeOther(backlogListURL(estado))`.

## Contratos que se consumen

- **HU-20a**: `Renderer.PageStatus`, `Renderer.Fail`, `NotFound`, `Unavailable`, `PageData`, `FormView`,
  `NewFormView`, `FormFromRequest`, `AddError`.
- **HU-20b**: `ActiveContextFrom`, `RequireContext`, `SetFlash` (`FlashSuccess`, `FlashWarning`, `FlashError`),
  `RedirectSeeOther`, `parseID`, `MaxID`, `MsgOperationFailed`, `CrossOriginGuard`.
- **HU-22**: `RegisterBacklog`, `parseBacklogForm`, `applyAction`, `mapBacklogDomainError`, `StoryCatalog`,
  `StoryRow`, `backlogPriorities`, `backlogStoryPointValues`, `invalidBacklogPriority`,
  `invalidBacklogStoryPoints`, los nombres de los campos y los textos.
- **HU-02**: `BacklogService`, `BacklogStore`, `ValidateStoryPoints`, los errores y `WarnEstimationPending`.
- **HU-04/HU-15**: el orden de bloqueos (Sprint antes que historias) y el estado 'Finalizado'.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Ampliar `BacklogStore` y `StoryCatalog` rompe la compilación de los fakes de HU-22 | T003 los identifica. T027 y T056 suman los métodos a los tres fakes en el mismo commit que los puertos. |
| El orden nuevo rompe tres pruebas de HU-22 | T030 y T037 las actualizan, en RED, con el orden de FR-001. No se borran: se cambia la expectativa. |
| Bloqueos cruzados con el cierre de Sprint (HU-15) o la asignación (HU-23) | Se usa su mismo orden: Sprint primero (`FOR SHARE`), después la historia. Lo prueba T019 con una transacción manual. |
| La asignación cambia entre la lectura y el bloqueo de la historia | Relectura en el paso 3 y un solo reintento (`data-model.md`). Lo prueban T020 y T021. |
| HU-15 no está en `main` | Las pruebas preparan Sprints 'Finalizado' en la base. Se coordina con quien tenga HU-15 que su cierre no cambie el orden de bloqueos (T003). |
| `parseID` limita los identificadores a `MaxInt32` y `user_stories.id` es `BIGSERIAL` | Una historia con identificador mayor sería inalcanzable. Es aceptable con el volumen del curso y queda anotado como límite conocido. |

## Verificación posterior al diseño

Vuelvo a revisar la tabla de la constitución con el diseño cerrado:

- **Principio VI**: los dos puertos ampliados tienen adaptador PostgreSQL y prueba de integración.
- **Principio II**: no se duplicó ninguna regla ni parseo.
- **Principio III**: las tareas siguen RED, evidencia y GREEN en cada bloque.

Sigue en **PASA**. No hay violaciones que justificar.
