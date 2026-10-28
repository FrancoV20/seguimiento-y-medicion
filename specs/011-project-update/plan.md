# Plan de implementación: HU-11 – Modificación de Proyecto (web)

**Rama**: `HU-11` | **Fecha**: 2026-10-27 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/011-project-update/spec.md`

## Resumen

HU-11 agrega a "Gestión de Proyectos" un enlace "Modificar" y la pantalla `/proyectos/{id}/modificar`, que guarda
el nombre, las fechas y los integrantes de un proyecto en una sola transacción.

- **Reutiliza**:
  - de HU-01, la regla de fechas, nombre e integrantes (`NewProject`);
  - de HU-21, el parseo del formulario y los mensajes (`parseProjectForm`, `projectFieldError`);
  - de HU-20a y HU-20b, la base web (404, flash, 303, escape, contexto).
- **Agrega**:
  - el método `Update` al puerto `ProjectStore`, con su adaptador PostgreSQL;
  - la lectura `ProjectForEdit` al catálogo de HU-21;
  - dos reglas puras: el integrante con horas no se quita (R5) y los Sprints que el cambio deja fuera (R6).
- **Cambio chico en HU-03**: el guardado de horas vuelve a comprobar y bloquear la pertenencia dentro de su
  transacción. Así se cierra la carrera entre quitar un integrante y que cargue horas.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`) y `github.com/jackc/pgx/v5/stdlib`,
ya presentes; sin dependencias nuevas.
**Almacenamiento**: PostgreSQL. Sin migración nueva.
**Pruebas**:

- unitarias del dominio con `TestProjectStore`;
- integración con PostgreSQL (`*_integration_test.go`);
- handlers con `httptest` y `cookiejar`;
- BDD en `test/`.

**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor.
**Objetivos de rendimiento**: el formulario y el guardado responden en menos de 500 ms con 50 integrantes y 20
Sprints (SC-006).
**Restricciones**:

- Los archivos existentes están en CRLF: no usar `gofmt -w` sobre ellos.
- Sin JavaScript.
- Sin token en los formularios (hay `CrossOriginGuard` global).
- Ningún mensaje expone detalles internos.

**Escala**: proyectos del curso (decenas de integrantes y Sprints).

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | La historia viene del backlog (Should Have, valor 70) y cada FR sale de un escenario del Issue o de Clarifications. | PASA |
| II. Calidad por construcción | Una sola regla de fechas (`NewProject`). Conversión de integrantes extraída y compartida. Reglas puras separadas de la transacción. Textos en un solo lugar. | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces. Cada bloque RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | Coordinación explícita con HU-21 y HU-03 (archivos ajenos) en T003 y T084. | PASA |
| V. Medición | No agrega métricas. Mantiene la integridad de las horas (HU-03) y de los Sprints (HU-04), que alimentan HU-07. | PASA |
| VI. Persistencia PostgreSQL | `ProjectStore.Update` con `PostgreSQLProjectStore` y pruebas de integración. `ProjectCatalog.ProjectForEdit` con `PostgreSQLProjectCatalog` y pruebas de integración. El cambio de HU-03 tiene su prueba de integración. | PASA |

## Estructura

### Documentación

```text
specs/011-project-update/
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

No hay carpeta `contracts/`: las rutas están en este plan y el modelo de vista en `data-model.md`, como en HU-05a.

### Código

Archivos nuevos:

```text
src/project/update.go                                   tipos de la modificación
src/project/update_errors.go                            ErrProjectNotFound, MemberHasEffortError, SprintOutOfRangeError
src/project/update_rules.go                             DatesChanged, MembershipChanges, SprintsLeftOutside
src/project/update_service.go                           ProjectService.UpdateProject
src/project/postgres_store_update.go                    PostgreSQLProjectStore.Update
src/project/update_rules_test.go
src/project/update_service_test.go
src/project/postgres_store_update_integration_test.go
src/effort/postgres_effort_store_membership_integration_test.go
src/web/projects_update_handlers.go                     edit (GET) y update (POST)
src/web/postgres_project_catalog_edit.go                ProjectForEdit
src/web/postgres_project_catalog_edit_integration_test.go
src/web/projects_update_handlers_test.go
src/web/templates/project_edit.html
test/web_project_update_bdd_test.go
```

Archivos existentes que se editan:

```text
src/project/store.go                    + Update en ProjectStore (HU-01)
src/project/test_store.go               + Update, Updates, ShouldFailUpdate, UpdateErr (HU-01)
src/project/service.go                  parseMemberIDs extraída de CreateProject (REFACTOR, HU-01)
src/web/projects_handlers.go            + dos rutas en el mapa de RegisterProjects (HU-21)
src/web/projects_catalog.go             + ProjectEdit y ProjectForEdit en ProjectCatalog (HU-21)
src/web/projects_form.go                + textos de HU-11 (HU-21)
src/web/templates/projects.html         + columna "Acciones" con "Modificar" (HU-21)
src/web/projects_handlers_test.go       + Update en failingCreateProjectStore (HU-21)
test/web_projects_bdd_test.go           + Update en failingBDDProjectStore (HU-21)
src/effort/postgres_effort_store.go     + comprobación con bloqueo en SaveEntry (HU-03)
src/effort/errors.go                    + errMemberNotInProject (HU-03)
src/effort/service.go                   errMemberNotInProject → ErrMemberRequired (HU-03)
src/effort/service_test.go              + prueba del mapeo (HU-03)
```

**No se tocan**: `src/cmd/server/main.go` (ya construye `ProjectService` con `PostgreSQLProjectStore` y llama a
`RegisterProjects`), `db/migrations`, `src/backlog` ni `src/sprint`.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /proyectos` (HU-21) | `get` | 200 (lista con "Modificar") |
| `GET /proyectos/{id}/modificar` | `edit` | 200 formulario; 404; 500/503 si falla la lectura |
| `POST /proyectos/{id}/modificar` | `update` | 303 a `/proyectos`; 422; 409; 404; 500 |

Las dos rutas nuevas se agregan al mismo `Routes` de `RegisterProjects`, porque `Register` no acepta una segunda
sección con la misma dirección. La sección sigue siendo `Section{Name: "Proyectos", Path: "/proyectos"}`, sin
exigir contexto.

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. `ProjectStore` suma `Update(ctx, ProjectUpdate)`; `ProjectService` suma `UpdateProject(ctx, req)`. Ni
   `main.go` ni la firma de `RegisterProjects` cambian.
2. `UpdateProject` valida con `NewProject` (la regla única de fechas de FIX-HU-01).
3. Las reglas R5, R6 y R7 son funciones puras que el adaptador llama dentro de la transacción.
4. La transacción `READ COMMITTED` bloquea el proyecto con `FOR NO KEY UPDATE` y quita las bajas antes de contar
   sus horas (ver [data-model.md](data-model.md)).
5. HU-03: `SaveEntry` comprueba la pertenencia con `FOR SHARE OF pm` dentro de su transacción, y el servicio
   traduce el rechazo al error que ya existe para un integrante que no pertenece.
6. R6 revisa solo los Sprints que estaban dentro y quedan fuera. El mensaje va junto a la fecha de inicio si el
   Sprint empieza antes del nuevo inicio y, si no, junto a la fecha de fin.
7. El 404 se decide antes de validar el formulario.
8. Los textos nuevos están en `projects_form.go`, y los mensajes de HU-01 salen de sus errores, como en HU-21.
9. La plantilla de edición es propia (`project_edit.html`), y la lista suma la columna "Acciones".
10. Mapeo de errores: tabla en [data-model.md](data-model.md#mapeo-de-errores-en-el-handler).

## Flujo del `POST /proyectos/{id}/modificar`

1. `id := parseID(r.PathValue("id"))`. Si es 0: `Fail(NotFound)`.
2. `catalog.ProjectForEdit(ctx, id)`. `NotFound` → 404; otro error → `Fail`.
3. `r.ParseForm()` y `parseProjectForm(r.PostForm)`, la misma de HU-21.
4. `catalog.ListMembers(ctx)` para volver a mostrar el formulario.
5. Con errores de formato: 422 con todos juntos.
6. `service.UpdateProject(ctx, ProjectUpdateRequest{ID, Name, Members, StartDate, EndDate})`.
7. Error: mapear según la tabla y volver a mostrar `project_edit.html` con los datos enviados (`PageStatus`).
8. Éxito: `SetFlash(FlashSuccess, MsgProjectUpdated)` y `RedirectSeeOther("/proyectos")`.

## Contratos que se consumen

- **HU-20a**: `Renderer.PageStatus`, `Renderer.Fail`, `NotFound`, `Unavailable`, `PageData`, `FormView`,
  `NewFormView`, `AddError`.
- **HU-20b**: `SetFlash`, `RedirectSeeOther`, `parseID`, `MaxID`, `MsgOperationFailed`, `CrossOriginGuard`, y el
  middleware de contexto (nombre del proyecto y limpieza de la identificación en cada solicitud).
- **HU-21**: `RegisterProjects`, `parseProjectForm`, `projectFieldError`, `ProjectCatalog`, `MemberOption`,
  `memberOptionKey` y los nombres de los campos.
- **HU-01**: `NewProject` y sus errores.
- **HU-03**: `PostgreSQLEffortStore.SaveEntry` y `Service.LogEffort` (se modifican).

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Ampliar `ProjectStore` rompe la compilación de los fakes de HU-21 | T003 los identifica; T027 suma `Update` a los dos fakes en el mismo commit que el puerto. |
| Modificar el guardado de horas de HU-03 cambia su comportamiento | El cambio solo agrega un rechazo que antes no ocurría. Lo cubren las pruebas de HU-03 existentes (T004, línea base) y las nuevas (T021, T022, T023). Se avisa a su responsable (T003). |
| pgx no convierte `[]int32` a `int[]` con `database/sql` | La primera prueba de integración (T011) lo verifica; la alternativa es un bucle de sentencias con un solo identificador (`data-model.md`). |
| Las pruebas de concurrencia son frágiles por los tiempos | Usan transacciones manuales que retienen el bloqueo y esperan a que la otra operación quede bloqueada consultando `pg_stat_activity` (`wait_event_type = 'Lock'`), sin pausas fijas. |
| Los mensajes de HU-01 están en inglés | Se reutilizan tal cual (decisión 5 del equipo). Si HU-01 los traduce, HU-11 no cambia. Queda anotado para el equipo. |

## Verificación posterior al diseño

Vuelvo a revisar la tabla de la constitución con el diseño cerrado:

- **Principio VI**: los dos puertos nuevos o ampliados tienen adaptador PostgreSQL y prueba de integración.
- **Principio II**: no se duplicó ninguna regla.
- **Principio III**: las tareas están organizadas en RED, evidencia y GREEN por bloque.

Sigue en **PASA**. No hay violaciones que justificar.
