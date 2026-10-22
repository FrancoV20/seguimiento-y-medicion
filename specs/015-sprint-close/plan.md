# Implementation Plan: Cierre de Sprint

**Branch**: `HU-15` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar el cierre del Sprint activo en el dominio y presentarlo en una pantalla propia, "Cierre de
Sprint" (`/cierre`), como páginas generadas por el servidor. El servicio `CloseSprint` vive en `src/sprint`
en archivos nuevos: comprueba que el Sprint sea del proyecto y esté activo, invoca la validación de HU-07
y delega en un puerto de almacenamiento propio (`SprintCloseStore`) con su adaptador PostgreSQL. El
adaptador hace todo en una sola transacción: bloquea el Sprint y sus historias, guarda el registro de
cada historia en `sprint_story_results`, devuelve a "Pendiente" las no completadas quitándoles la
asignación y pasa el Sprint a "Finalizado". Una migración nueva crea la tabla del registro. La pantalla
reutiliza `BoardCatalog` de HU-14 para el resumen, tiene una sección con `RequiresProject: true` y los
handlers solo arman el envío, llaman al servicio y traducen sus errores. No modifica HU-04, HU-14, HU-20a
ni HU-20b; en HU-03 solo agrega la regla de carga posterior si no existiera.

> **Supuestos a verificar antes de implementar (T001, T002 y T004)**:
> 1. La ruta del paquete de Sprint, la validación previa al cierre de HU-07 (firma, datos que devuelve y
>    mensaje) y su cálculo de velocidad, y cómo HU-07 prueba el caso de datos inválidos.
> 2. Qué hace HU-04 al asignar y al quitar (`sprint_story_assignments`, `user_stories.active_sprint_id`,
>    estados) y qué dejó FIX-HU-04 (errores, constantes).
> 3. Si HU-03 ya rechaza cargar horas en una historia que no esté en el Sprint activo, y cómo cuenta HU-07
>    las horas de una historia que estuvo en dos Sprints.
> 4. `BoardCatalog` de HU-14 en `main` (`ActiveBoard`, `BoardView`, `BoardStory`), el último número de
>    migración en `main` y el tipo de `user_stories.story_points`.
> 5. HU-20a y HU-20b completas en `main`: `Server.Register`, `RequireContext` para `POST`,
>    `ActiveContextFrom`, `SetFlash`, `MsgOperationFailed`, parciales `alert.html`, redirección 303 y
>    protección contra envíos de otro origen.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `database/sql`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. Una migración nueva crea `sprint_story_results`. El cierre usa
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
├── NNN_create_sprint_story_results.up.sql    # NNN = último de main + 1
└── NNN_create_sprint_story_results.down.sql
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
y registrar la sección. Si HU-03 no rechaza cargar horas en un Sprint finalizado, se agrega esa regla ahí
(T016).

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
- De HU-20b: `SetFlash`, `MsgOperationFailed`, la redirección 303, el parcial `alert.html`, la protección
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