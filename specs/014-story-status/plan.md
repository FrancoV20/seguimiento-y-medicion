# Implementation Plan: Actualización de estado de historias (Tablero)

**Branch**: `HU-14` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar la regla de transiciones de estado de las historias del Sprint activo en el dominio y
presentarla en una pantalla propia, "Tablero" (`/tablero`), como páginas generadas por el servidor. La
regla (`En Sprint → En progreso → Hecho → En progreso`) y el servicio que la aplica viven en `src/sprint`
en archivos nuevos, con un puerto de almacenamiento propio (`StoryStatusStore`) y su adaptador
PostgreSQL, sin modificar archivos de HU-04. El cambio se hace con una única sentencia atómica que
compara el estado esperado y verifica que la historia pertenezca a un Sprint activo. Los Story Points
completados y planificados se calculan al leer, nunca se guardan. La pantalla tiene un catálogo de
lectura propio (`PostgreSQLBoardCatalog`), una sección con `RequiresProject: true` y los handlers solo
arman el envío, llaman al servicio y traducen sus errores. No agrega migraciones y no modifica HU-20a
ni HU-20b.

> **Verificado en T001 y T002 (24/10/2026)**:
> 1. Paquete `src/sprint`; constantes `Pendiente`, `EnSprint`, `InProgress` y `Done` (tipo `StoryStatus`).
>    FIX-HU-04 dejó `ErrActiveSprintExists` y `ErrActiveSprintConflict`, que no se reutilizan.
> 2. HU-07 (`src/metrics`) cuenta como completada la historia con estado `'Hecho'`.
> 3. `effort_logs.story_id`; `user_stories` no exige otras columnas al cambiar `status`.
> 4. HU-20a y HU-20b en `main`: `Server.Register`, `RequireContext`, `ActiveContextFrom`, `SetFlash`,
>    `MsgOperationFailed`, `RedirectSeeOther`, `CrossOriginGuard`, `Renderer.PageStatus`, `RenderError` y
>    `Fail`. El mensaje lo muestra `layout.html` desde `PageData.Flash`.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. `PostgreSQLStoryStatusStore` lee la historia y cambia su
estado; `PostgreSQLBoardCatalog` lee el Sprint activo y sus historias. Ambos usan `database/sql` y
pgx sobre `user_stories`, `sprint_story_assignments`, `sprints` y `effort_logs`.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias de la regla, pruebas del servicio y de
los adaptadores contra PostgreSQL real, pruebas de handler con `net/http/httptest` y
`net/http/cookiejar`, y escenarios BDD en `test/web_board_bdd_test.go`.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Regla de dominio y pantalla de una aplicación web SSR.

**Performance Goals**: El tablero responde en menos de 500 ms bajo uso normal (SC-007).

**Constraints**: Sin migraciones ni dependencias nuevas, sin JavaScript, un solo Sprint activo por
proyecto (FIX-HU-04), cambio atómico, Story Points calculados y no guardados, ningún detalle interno
en las respuestas, y proyecto tomado siempre del contexto del servidor.

**Scale/Scope**: Una pantalla y dos rutas. No incluye asignar o quitar historias (HU-23), cerrar el
Sprint (HU-15) ni identificar a la persona.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Habilita HU-15, HU-07, HU-25 y HU-12; no agrega
  funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. La regla y el servicio están en el dominio, el almacenamiento
  detrás de un puerto y la pantalla solo traduce.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas contra
  PostgreSQL real y un escenario BDD; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Las tres historias de usuario se pueden entregar de forma
  incremental; no hay dependencia de HU-23.
- **V. Measurement and Continuous Improvement**: PASS. Alimenta con datos reales a HU-07 y HU-10.
- **VI. PostgreSQL Persistence**: PASS. `StoryStatusStore` y `BoardCatalog` tienen adaptador PostgreSQL con
  pruebas de integración.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/014-story-status/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML; su comportamiento está en `spec.md` y las
direcciones y campos, abajo.

### Source Code (repository root)

```text
src/
├── sprint/                                   # archivos nuevos; no se modifica HU-04
│   ├── story_status.go                       # estados, transiciones, errores, StoryStatusStore, servicio
│   ├── story_status_test.go
│   ├── postgres_story_status_store.go        # adaptador PostgreSQL
│   └── *_integration_test.go
└── web/
    ├── board_catalog.go                      # BoardView, BoardStory, interfaz BoardCatalog
    ├── postgres_board_catalog.go             # adaptador PostgreSQL
    ├── board_form.go                         # acciones, textos, mapeo de errores
    ├── board_handlers.go                     # GET /tablero y POST /tablero/estado
    ├── templates/board.html                  # grupos por estado, Story Points y acciones
    └── *_test.go
test/
└── web_board_bdd_test.go
```

**Structure Decision**: Se agregan archivos nuevos al paquete de Sprint (regla y almacenamiento) y al
paquete `src/web` (pantalla), uno por responsabilidad. Se edita `src/cmd/server/main.go` para construir
el servicio y el catálogo y registrar la sección. La ubicación exacta del paquete de Sprint se confirma
en T001.

### Direcciones y campos

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/tablero` | Sprint activo con sus historias por estado, o "No hay un Sprint activo" |
| POST | `/tablero/estado` | Campos `historia_id`, `accion` y `estado_actual` (opcional); cambia el estado; 303 a `/tablero` |

La sección se registra como `Section{Name: "Tablero", Path: "/tablero", RequiresProject: true}` con las
dos rutas. El proyecto se obtiene del servidor; ningún campo de proyecto ni de Sprint se lee del
formulario.

### Reglas de transición

| Acción | Origen | Destino |
| --- | --- | --- |
| `comenzar` | En Sprint | En progreso |
| `completar` | En progreso | Hecho |
| `reabrir` | Hecho | En progreso |

Cualquier otro par se rechaza. La pantalla muestra solo la acción válida de cada estado, tomada de la
misma función del dominio (`NextAction`).

### Flujo del POST

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto.
2. Validar `accion` (una de las tres) y `historia_id` numérico; si no, 422.
3. Llamar al servicio con el proyecto de `ActiveContext`: historia del proyecto, Sprint activo, estado
   ya aplicado, estado esperado, transición permitida, origen de la acción, cambio atómico,
   advertencia de horas.
4. Éxito: `SetFlash` de éxito (o de éxito con advertencia en un solo mensaje) y redirección 303.
5. Estado ya aplicado: `SetFlash` de advertencia y 303.
6. Error del dominio: tablero con el mensaje y 409 (o 422 si la historia no existe).
7. Cualquier otro error: `MsgOperationFailed` con el tablero y 500.

### Mapeo de errores del dominio a mensajes

| Error del dominio | Mensaje |
| --- | --- |
| `ErrStoryNotFound` | "La historia no existe" |
| `ErrNotInActiveSprint` | "La historia no está asignada a un Sprint activo" |
| `ErrAlreadyInStatus` | "La historia ya está en {estado}" |
| `ErrStatusChanged` | "El estado de la historia cambió; revise el estado actual y vuelva a intentarlo" |
| `ErrTransitionNotAllowed` | "No se puede pasar de {origen} a {destino}" |
| Cualquier otro | `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `MsgOperationFailed`, la redirección 303, la plantilla `alert` de `layout.html`, la
  protección contra envíos de otro origen, `ActiveContextFrom` y `RequireContext`.
- De HU-04 y FIX-HU-04: el esquema (`sprints`, `sprint_story_assignments`, estados) y la regla de un solo
  Sprint activo; no se llama a su servicio.
- De HU-03: la tabla `effort_logs` (solo lectura, para la advertencia).
- No usa `SprintCatalog` de HU-23 ni el catálogo de HU-22.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación. `BoardCatalog` repite parte de la
lectura del Sprint activo de HU-23 porque ambas historias se hacen en paralelo; se unifican después si
conviene.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a las tres transiciones y su pantalla.
- **II. Quality by Construction**: PASS. La regla es una función pura y el cambio es una sola sentencia
  atómica.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas de la regla, del
  servicio, de los adaptadores, de handler y BDD.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas.
- **VI. PostgreSQL Persistence**: PASS. Ambos puertos tienen adaptador con integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.