# Implementation Plan: Cierre de Sprint

**Branch**: `HU-15` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar el cierre del Sprint activo en el dominio y presentarlo en una pantalla propia, "Cierre de
Sprint" (`/cierre`), como páginas generadas por el servidor. El servicio `CloseSprint` vive en `src/sprint`
en archivos nuevos: comprueba que el Sprint sea del proyecto y esté activo, invoca
`metrics.ValidateSprintClosure` sobre los datos cargados de HU-07 y delega en un puerto de almacenamiento
propio (`SprintCloseStore`) con su adaptador PostgreSQL. El
adaptador hace todo en una sola transacción: bloquea el Sprint y sus historias, guarda el registro de
cada historia en `sprint_story_results`, devuelve a "Pendiente" las no completadas quitándoles la
asignación y pasa el Sprint a "Finalizado". La migración `013` crea la tabla del registro. La pantalla
reutiliza `BoardCatalog` de HU-14 para el resumen, tiene una sección con `RequiresProject: true` y los
handlers solo arman el envío, llaman al servicio y traducen sus errores. No modifica HU-03, HU-04,
HU-14, HU-20a ni HU-20b.

> **Resultados del setup (T001–T004, verificados en el árbol actual de `origin/main`)**:
> 1. El paquete de Sprint es `src/sprint`. La validación previa reutilizable de HU-07 es
>    `metrics.ValidateSprintClosure(metrics.SprintData) error`; valida historias `'Hecho'`, rechaza horas
>    reales negativas y Story Points nulos/inválidos, y devuelve motivos identificados por historia. No
>    usar `CalculateMetrics` antes del cierre porque exige estado `'Finalizado'`; la velocidad es la suma
>    de puntos de historias `'Hecho'` y vale cero si no hay ninguna.
> 2. HU-04 asigna filas en `sprint_story_assignments` y actualiza estado/`active_sprint_id`; al quitar,
>    elimina la fila y devuelve la historia a `'Pendiente'` con `active_sprint_id = NULL`. FIX-HU-04
>    evita un segundo Sprint activo por proyecto.
> 3. HU-03 ya rechaza horas si `active_sprint_id` no apunta a un Sprint `'Activo'`. HU-07 suma todas las
>    horas registradas por historia, sin fecha ni atribución al Sprint; las horas históricas de una
>    historia se incluyen mientras su asignación vigente esté en el Sprint consultado.
> 4. HU-14 ofrece `BoardCatalog.ActiveBoard(ctx, projectID) (BoardView, found, error)`; `BoardStory` usa
>    `StoryPoints *int`. En la base, `user_stories.story_points` es `INTEGER` nullable.
> 5. HU-20a/b ya están integradas: `Server.Register` registra varias rutas y aplica `RequireContext`;
>    `ActiveContextFrom`, `SetFlash`, `MsgOperationFailed` y 303 están disponibles. El parcial `"alert"`
>    está definido en `layout.html` (no hay archivo `alert.html`) y `CrossOriginGuard` está instalado.
>    `012_create_defects` ya ocupa el número 012 (HU-06); la migración de HU-15 es `013`.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `database/sql`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose (puerto host `5433` hacia `5432` del contenedor). La migración
`013` crea `sprint_story_results`. El cierre usa
`sprints`, `sprint_story_assignments`, `user_stories` y `sprint_story_results` en una transacción de
`database/sql`.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas del servicio y del adaptador contra PostgreSQL real,
pruebas de handler con `net/http/httptest` y `net/http/cookiejar`, y escenarios BDD en
`test/web_close_bdd_test.go`.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Regla de dominio, migración y pantalla de una aplicación web SSR.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-007).

**Constraints**: Cierre todo o nada, un solo Sprint activo (FIX-HU-04), validación de HU-07 sin copiarla,
ningún detalle interno en las respuestas, y proyecto tomado siempre del contexto del servidor.

**Scale/Scope**: Una pantalla, dos rutas y una tabla. No incluye velocidad ni métricas (HU-07, HU-25),
Sprints anteriores (HU-16) ni reabrir un Sprint.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Habilita HU-07, HU-10, HU-16 y HU-25 y el siguiente Sprint; el
  registro del Sprint evita perder datos que esas historias necesitan.
- **II. Quality by Construction**: PASS. La regla y el servicio están en el dominio, el almacenamiento
  detrás de un puerto y la pantalla solo traduce.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas contra PostgreSQL
  real y un escenario BDD; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review.
- **V. Measurement and Continuous Improvement**: PASS. Congela los datos sobre los que se miden velocidad
  y desviación.
- **VI. PostgreSQL Persistence**: PASS. `SprintCloseStore` tiene adaptador PostgreSQL con pruebas de
  integración y la migración tiene `up` y `down` probados.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/015-sprint-close/
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
db/migrations/
├── 013_sprint_story_results.up.sql
└── 013_sprint_story_results.down.sql
src/
├── sprint/                                   # archivos nuevos; no se modifica HU-04
│   ├── close_sprint.go                       # servicio CloseSprint, errores y puerto SprintCloseStore
│   ├── close_sprint_test.go
│   ├── postgres_sprint_closer.go             # adaptador PostgreSQL (transacción)
│   └── *_integration_test.go
└── web/
    ├── close_form.go                         # armado del envío, textos y mapeo de errores
    ├── close_handlers.go                     # GET /cierre y POST /cierre
    ├── templates/close.html                  # resumen y botón
    └── *_test.go
test/
└── web_close_bdd_test.go
```

**Structure Decision**: Se agregan archivos nuevos al paquete de Sprint (servicio y almacenamiento), una
migración y archivos al paquete `src/web`. Se edita `src/cmd/server/main.go` para construir el servicio
y registrar la sección. HU-03 ya rechaza horas cuando la historia no pertenece a un Sprint activo; T016
solo requiere una prueba que documente ese comportamiento y avisar a su responsable, sin cambiar el
dominio de HU-03.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/cierre` | Resumen del Sprint activo y botón, o "No hay un Sprint activo para cerrar" |
| POST | `/cierre` | Campo `sprint_id`; cierra el Sprint; 303 a `/cierre` |

La sección se registra como `Section{Name: "Cierre de Sprint", Path: "/cierre", RequiresProject: true}`.
El proyecto se obtiene del servidor; el `sprint_id` se valida contra ese proyecto.

### Flujo del POST

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto.
2. Validar que `sprint_id` sea numérico; si no, 422 con "El Sprint no existe".
3. Llamar al servicio con el proyecto de `ActiveContext`: Sprint del proyecto, Sprint activo, validación
   de HU-07 y cierre transaccional.
4. Éxito: `SetFlash` con "Sprint N cerrado correctamente. Historias devueltas al Backlog: Y" y
   redirección 303.
5. Error del dominio: pantalla con el mensaje y 409 (o 422 si el Sprint no existe).
6. Cualquier otro error: `MsgOperationFailed` con la pantalla y 500.

### Mapeo de errores del dominio a mensajes

| Error del dominio | Mensaje |
| --- | --- |
| `ErrSprintNotFound` | "El Sprint no existe" |
| `ErrSprintAlreadyClosed` | "El Sprint ya está cerrado" |
| `ErrCloseBlocked` | Mensaje de HU-07 con la lista de historias a corregir |
| Cualquier otro | `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `MsgOperationFailed`, la redirección 303, el parcial `"alert"` definido en
  `layout.html`, la protección
  contra envíos de otro origen, `ActiveContextFrom` y `RequireContext`.
- De HU-07: la validación previa al cierre y el cálculo de velocidad (solo se invocan).
- De HU-14: `BoardCatalog.ActiveBoard`, `BoardView` y `BoardStory` (solo lectura, para el resumen).
- De HU-04 y FIX-HU-04: el esquema y la regla de un solo Sprint activo.
- De HU-03: la regla de carga de esfuerzo sobre un Sprint finalizado.

## Complexity Tracking

| Decisión | Por qué hace falta | Alternativa descartada |
|---|---|---|
| Migración con `sprint_story_results` | Al devolver historias al Backlog, el Sprint cerrado perdería lo que planificó y HU-10, HU-16 y la Sprint Review quedarían mal | No guardar nada (descartada: no se puede reconstruir después) |

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a cerrar, devolver y congelar.
- **II. Quality by Construction**: PASS. El cierre es una transacción única detrás de un puerto.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas del servicio, del
  adaptador, de handler y BDD.
- **IV. Agile, Collaborative Execution**: PASS. Las tres historias de usuario se entregan de forma
  incremental.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas; las habilita.
- **VI. PostgreSQL Persistence**: PASS. Puerto y migración con pruebas contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.