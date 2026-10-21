# Implementation Plan: Interfaz web de Sprints

**Branch**: `HU-23` | **Date**: 2026-10-20 | **Revised**: 2026-10-21 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar la pantalla "Sprints" como páginas generadas por el servidor: una sola ruta `/sprints`
que muestra el Sprint activo del proyecto con sus historias o, si no hay, el formulario para crear
uno. La sección exige proyecto activo, así que HU-20b redirige a la selección cuando falta. Los
handlers no duplican reglas de negocio: arman el envío (fechas `AAAA-MM-DD`, identificadores de
historias sin repetidos), delegan la creación, la asignación y el quitar en HU-04 con el proyecto del
contexto del servidor y traducen sus errores a mensajes. La regla de un solo Sprint activo la aplica
HU-04 (FIX-HU-04): la pantalla no ofrece el formulario cuando hay un activo y muestra el rechazo si
igual llega un envío. La regla de quitar (solo "En Sprint" y sin horas registradas) se agrega en el
dominio de HU-04 (T047) y la pantalla usa la misma función para deshabilitar el botón. Un adaptador
`PostgreSQLSprintCatalog` propio de HU-23 lee el Sprint activo y las historias del proyecto. No agrega
migraciones y no modifica HU-20a ni HU-20b; en HU-04 solo agrega la validación de quitar.

> **Supuestos a verificar antes de implementar (T001, T002 y T004)**:
> 1. La ruta del paquete `src/sprint` y las firmas del servicio de HU-04: entrada y resultado de
>    `CreateSprint` (advertencia y errores por historia), si el servicio expone `AssignStories` y
>    `RemoveStory` o solo el store, los tipos de error y los mensajes exactos. Además: FIX-HU-04
>    integrada (`ErrActiveSprintExists` y su mensaje) y si el dominio puede consultar horas
>    registradas.
> 2. Qué escribe HU-04 al asignar y al quitar: `sprint_story_assignments`,
>    `user_stories.active_sprint_id` y los estados `'En Sprint'` y `'Pendiente'`; el catálogo lee la
>    fuente que HU-04 mantiene siempre consistente.
> 3. Que `effort_logs` (HU-03) tenga la columna de historia que usa la consulta de horas.
> 4. HU-20a y HU-20b completas en `main`: `Server.Register` con varias rutas y métodos,
>    `RequireContext` también para `POST`, `ActiveContextFrom`, `SetFlash`, `FormView`, parciales
>    `alert.html` y `field.html`.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `time`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. La creación, la asignación y el quitar usan el servicio y
el store de HU-04. `PostgreSQLSprintCatalog` solo lee `sprints`, `sprint_story_assignments`,
`user_stories` y `effort_logs` mediante `database/sql` y pgx.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias del formulario y de la regla de
quitar, pruebas de handler con `net/http/httptest` y `net/http/cookiejar` sobre el servicio de HU-04 y
la base reales (el proyecto activo se elige por el flujo de HU-20b), escenarios BDD en
`test/web_sprints_bdd_test.go` y pruebas de integración contra PostgreSQL.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Pantalla de una aplicación web SSR sobre el núcleo de dominio en Go.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-008).

**Constraints**: Sin migraciones ni dependencias nuevas, sin JavaScript, reglas y mensajes de HU-04
sin cambios (salvo la validación de quitar), un solo Sprint activo por proyecto aplicado por el
dominio (FIX-HU-04), ningún detalle interno en las respuestas, los datos del formulario se conservan
ante cualquier error, y el proyecto y el Sprint salen siempre del contexto del servidor.

**Scale/Scope**: Una pantalla y cuatro rutas. No incluye cerrar ni editar Sprints (HU-15), Sprints
anteriores (HU-16), cambio de estado de historias (HU-14) ni mover historias entre Sprints.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Hace consumible el flujo de HU-04 y habilita HU-14, HU-24 y
  HU-25; no agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. El armado del formulario, el mapeo de errores, el catálogo y
  los handlers están separados; las reglas de Sprint (incluida la de quitar) viven en HU-04.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas de handler y un
  escenario BDD con almacenamiento real; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review.
- **V. Measurement and Continuous Improvement**: PASS. Los criterios de usabilidad quedan con
  protocolo y evidencia para la retrospectiva.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `SprintCatalog` tiene su adaptador
  `PostgreSQLSprintCatalog` con prueba de integración contra PostgreSQL; la escritura reutiliza el
  adaptador de HU-04.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/023-web-sprints/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML; su comportamiento está en `spec.md`
y las direcciones y campos, abajo.

### Source Code (repository root)

```text
src/
├── sprint/                             # HU-04: solo se agrega la validación de quitar (T047)
└── web/
    ├── sprints_catalog.go              # SprintView, SprintStoryRow, StoryOption e interfaz SprintCatalog
    ├── postgres_sprint_catalog.go      # adaptador PostgreSQL
    ├── sprints_form.go                 # armado del envío, textos, interfaz SprintActions y mapeo de errores
    ├── sprints_handlers.go             # GET /sprints y POST de crear, agregar y quitar
    ├── templates/sprints.html          # Sprint activo, formularios y lista de historias
    └── *_test.go
test/
└── web_sprints_bdd_test.go
```

**Structure Decision**: Se agregan archivos al paquete `src/web` creado en HU-20a, con un archivo por
responsabilidad. Se edita `src/cmd/server/main.go` para construir el servicio de HU-04 y el catálogo
y registrar la sección. Se modifica HU-04 (`src/sprint`) solo para agregar la validación de quitar
(T047); no se modifica ningún archivo de HU-20a ni HU-20b.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/sprints` | Sprint activo con sus historias, o el formulario de creación si no hay |
| POST | `/sprints` | Campos `objetivo`, `fecha_inicio`, `fecha_fin` y `historia` (una por historia marcada); crea el Sprint y asigna; 303 a `/sprints` |
| POST | `/sprints/historias` | Campo `historia` (una por historia marcada); agrega al Sprint activo; 303 a `/sprints` |
| POST | `/sprints/historias/quitar` | Campo `historia_id`; quita esa historia del Sprint activo; 303 a `/sprints` |

La sección se registra como `Section{Name: "Sprints", Path: "/sprints", RequiresProject: true}` con
las cuatro rutas. El proyecto y el Sprint activo se obtienen del servidor; ningún campo de
identificador de Sprint ni de proyecto se lee del formulario.

### Disponibilidad de historias

| Condición de la historia | Se puede marcar | Motivo mostrado |
| --- | --- | --- |
| Estado `'Pendiente'` y sin asignación | Sí | |
| Con asignación a un Sprint | No | "Ya está asignada al Sprint N" |
| Sin asignación y estado distinto de `'Pendiente'` | No | "No está pendiente" |

### Regla para quitar

Se puede quitar una historia del Sprint activo solo si su estado es `'En Sprint'` y no tiene horas
registradas. La regla vive en el dominio (`src/sprint`: función pura `CanRemoveStory` y validación en
`RemoveStory`, T047). La pantalla usa la misma función para deshabilitar el botón con el motivo ("Ya
está en progreso", "Ya está hecha" o "Tiene horas registradas") y el handler solo traduce el error.

### Flujo del POST de creación

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto; el handler no
   se ejecuta.
2. Parsear fechas (`AAAA-MM-DD`) e identificadores de historias (sin repetidos, numéricos). Un valor
   inválido genera un error junto a su campo.
3. Invocar `CreateSprint` de HU-04 con el proyecto de `ActiveContext`. Si devuelve
   `ErrActiveSprintExists`, responder 409 con "Ya hay un Sprint activo (Sprint N)".
4. Éxito: `SetFlash` de éxito (o de advertencia si no hay historias) y redirección 303 a `/sprints`.
5. Error de validación de HU-04: mensaje junto al campo y respuesta 422 con el formulario y los datos.
6. Historia no disponible: respuesta 409 con las historias rechazadas y los datos conservados.
7. Cualquier otro error: `MsgOperationFailed` con los datos conservados y respuesta 500.

### Mapeo de errores de HU-04 a campos (se confirma en T001)

| Error de HU-04 | Campo |
| --- | --- |
| Sprint Goal obligatorio | `objetivo` |
| Fecha faltante | La fecha faltante |
| Inconsistencia temporal (fin anterior o igual) | `fecha_fin` |
| Historia no disponible o en conflicto | `historia` (lista de historias) |
| Ya hay un Sprint activo (`ErrActiveSprintExists`) | Mensaje general "Ya hay un Sprint activo (Sprint N)" |
| Historia no removible (estado u horas) | Mensaje general con el motivo |
| Proyecto inexistente | Se trata como falta de proyecto activo |
| Cualquier otro | Mensaje general `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `FormView`, `MsgOperationFailed`, la redirección 303, los parciales
  `alert.html` y `field.html`, la protección contra envíos de otro origen, `ActiveContextFrom` y
  `RequireContext` (aplicado por `Section.RequiresProject`).
- De HU-04 y FIX-HU-04: servicio de creación (`CreateSprint`), `AssignStories` y `RemoveStory`,
  `ErrActiveSprintExists` y `CanRemoveStory`, sus errores y mensajes. HU-23 declara una interfaz
  mínima `SprintActions` que esas piezas satisfacen.
- No usa `ContextStore`, `project_picker.html` ni el catálogo de HU-21 o HU-22.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación. `SprintCatalog` es una interfaz
propia de HU-23 porque `SprintStore` de HU-04 no tiene lecturas; el costo es una consulta de lectura
que HU-16 y HU-24 podrán reutilizar.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a crear, ver, agregar y quitar.
- **II. Quality by Construction**: PASS. Las reglas de Sprint siguen en HU-04; el handler traduce
  formato, acciones y errores, y la regla de quitar es una función pura y testeable.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas de formulario,
  de handler con almacenamiento y contexto reales, BDD e integración con PostgreSQL.
- **IV. Agile, Collaborative Execution**: PASS. Las tareas separan las cinco historias de usuario y
  se pueden entregar de forma incremental (US1 a US3 como MVP).
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas.
- **VI. PostgreSQL Persistence**: PASS. El único almacenamiento nuevo (lecturas) tiene adaptador e
  integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.