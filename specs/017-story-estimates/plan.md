# Implementation Plan: Horas estimadas por historia (Estimaciones)

**Branch**: `HU-17` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar el registro y la modificación de las horas estimadas de una historia con una pantalla
propia, "Estimaciones" (`/estimaciones`), como páginas generadas por el servidor. El servicio
`UpdateEstimatedHours` vive en el paquete de historias de HU-02 (`src/backlog`), en archivos nuevos, y
reutiliza la regla de horas que HU-22 deja en el dominio; tiene un puerto de almacenamiento propio (`EstimateStore`) con su
adaptador PostgreSQL, sin modificar archivos de HU-02. El guardado es una única sentencia que solo toca
`user_stories.estimated_hours` y excluye las historias de un Sprint finalizado. Un catálogo de lectura
propio (`PostgreSQLEstimateCatalog`) lista las historias con sus horas estimadas y las horas reales de
HU-03. La sección exige proyecto activo (`RequiresProject`) y los handlers solo arman el envío, llaman al
servicio y traducen sus errores. No agrega migraciones y no modifica HU-20a ni HU-20b.

> **Supuestos verificados (T001, T002 y T004, 2026-10-28, `main` en `d12067f`)**:
> 1. El paquete de historias es `src/backlog`, no `src/story`. La regla numérica de horas es
>    `validEstimatedHours(*float64)` con `maxEstimatedHours = 9999.99`, sin exportar, y un único error,
>    `ErrInvalidEstimatedHours`. La interpretación del texto (coma o punto) está en la web
>    (`backlog_form.go`, con `strconv.ParseFloat`) y no en el dominio. "Sin estimar" es un texto literal
>    de la web, no una constante del dominio. Ver "Ajustes por T001" más abajo.
> 2. HU-07 (`src/metrics/postgres_metrics_store.go`) lee `user_stories.estimated_hours`; el `UPDATE` de
>    una sola columna no choca con otras columnas obligatorias.
> 3. `effort_logs` usa `story_id` y `hours NUMERIC(5,2)`.
> 4. HU-20a y HU-20b están completas en `main`: `Server.Register(section, routes)` con varias rutas y
>    métodos aplica `RequireContext` a cada ruta (incluido `POST`) según `Section.RequiresProject`;
>    `ActiveContextFrom`, `SetFlash`, `MsgOperationFailed` (`forms.go`), `RedirectSeeOther` (303) y
>    `CrossOriginGuard` registrado en `main.go`. Los parciales `alert` y `field` se definen en
>    `templates/layout.html` (no hay archivos `alert.html` ni `field.html` separados).
> 5. `git merge origin/main`: la rama `HU-17` ya estaba al día ("Already up to date").

## Ajustes por T001

- **Rutas**: donde `tasks.md` dice `src/story/...`, se usa `src/backlog/...`. Para no chocar con los
  archivos de HU-22 (`estimated_hours_test.go` y `postgres_estimated_hours_integration_test.go`), los
  archivos nuevos son `src/backlog/update_estimated_hours.go`, `update_estimated_hours_test.go`,
  `postgres_estimate_store.go` y `postgres_estimate_store_integration_test.go`.
- **Interpretación del texto**: como el dominio de HU-22 solo valida números, HU-17 agrega en un archivo
  nuevo de `src/backlog` `ParseEstimatedHours(text)`, estricto (sin signo, exponente ni separador de
  miles; más de dos decimales se rechaza antes que el máximo), que confirma el valor con
  `validEstimatedHours`. Detalle en `data-model.md` ("Decisión para HU-17").
- **Error por encima del máximo**: se agrega `ErrEstimatedHoursTooLarge`, reconocido también como
  `ErrInvalidEstimatedHours` con `errors.Is`, para que la web elija el mensaje sin volver a comparar el
  número. HU-22 no cambia.
- **Mensajes**: se reutilizan `MsgInvalidEstimatedHours` y `MsgHoursTooLarge` de `backlog_form.go`. Los
  nuevos (`MsgHoursRegistered`, `MsgHoursUpdated`, `MsgStoryNotFound`, `MsgHoursLockedSprint` y la
  constante de "Sin estimar") se definen en `estimates_form.go`; antes de declararlos se comprueba con
  `go vet ./...` que no choquen con otros nombres del paquete `web`.
- **Coordinación**: avisar a quien mantiene HU-22 que el texto de horas del alta acepta `1e3` y `1.000`
  (como 1), y que `ParseEstimatedHours` podría reutilizarse allí.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. `PostgreSQLEstimateStore` lee y actualiza
`user_stories.estimated_hours`; `PostgreSQLEstimateCatalog` lee `user_stories`, `effort_logs`,
`sprint_story_assignments` y `sprints`. Ambos con `database/sql` y pgx.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas del servicio y de los adaptadores contra
PostgreSQL real, pruebas de handler con `net/http/httptest` y `net/http/cookiejar`, y escenarios BDD en
`test/web_estimates_bdd_test.go`.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Servicio de dominio y pantalla de una aplicación web SSR.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-007).

**Constraints**: Sin migraciones ni dependencias nuevas, sin JavaScript, una sola regla de horas
(HU-22), una sola fuente del dato (`estimated_hours`), bloqueo atómico de Sprints finalizados, ningún
detalle interno en las respuestas, y proyecto tomado siempre del contexto del servidor.

**Scale/Scope**: Una pantalla y dos rutas. No incluye el alta con horas (HU-22), editar otros datos de la
historia (HU-13) ni registrar horas reales (HU-03 y HU-24).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Habilita la desviación de HU-07 y la pantalla de HU-25; no
  agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. La regla y el servicio están en el dominio, el almacenamiento
  detrás de un puerto y la pantalla solo traduce.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas contra PostgreSQL
  real y un escenario BDD; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review.
- **V. Measurement and Continuous Improvement**: PASS. Alimenta el cálculo de desviación de HU-07.
- **VI. PostgreSQL Persistence**: PASS. `EstimateStore` y `EstimateCatalog` tienen adaptador PostgreSQL con
  pruebas de integración.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/017-story-estimates/
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
├── backlog/                                  # paquete de HU-02 y HU-22 (confirmado en T001); solo archivos nuevos
│   ├── update_estimated_hours.go             # ParseEstimatedHours, servicio UpdateEstimatedHours, errores y puerto EstimateStore
│   ├── update_estimated_hours_test.go
│   ├── postgres_estimate_store.go            # adaptador PostgreSQL
│   └── postgres_estimate_store_integration_test.go
└── web/
    ├── estimates_catalog.go                  # EstimateRow e interfaz EstimateCatalog
    ├── postgres_estimate_catalog.go          # adaptador PostgreSQL
    ├── estimates_form.go                     # armado del envío, textos y mapeo de errores
    ├── estimates_handlers.go                 # GET /estimaciones y POST /estimaciones
    ├── templates/estimates.html              # lista con un campo por fila
    └── *_test.go
test/
└── web_estimates_bdd_test.go
```

**Structure Decision**: Se agregan archivos nuevos al paquete de historias (servicio y almacenamiento) y
al paquete `src/web` (pantalla), uno por responsabilidad. Se edita `src/cmd/server/main.go` para construir
el servicio y el catálogo y registrar la sección. No se modifica ningún archivo de HU-02, HU-22, HU-20a
ni HU-20b.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/estimaciones` | Lista de historias del proyecto activo con horas estimadas y reales |
| POST | `/estimaciones` | Campos `historia_id` y `horas_estimadas`; guarda la estimación; 303 a `/estimaciones` |

La sección se registra como `Section{Name: "Estimaciones", Path: "/estimaciones", RequiresProject: true}`
con las dos rutas. El proyecto se obtiene del servidor; ningún campo de proyecto se lee del formulario.

### Flujo del POST

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto.
2. Validar que `historia_id` sea numérico; si no, 422 con "La historia no existe".
3. Llamar al servicio con el proyecto de `ActiveContext` y el texto de `horas_estimadas`: la regla de
   HU-22 valida y normaliza el valor, luego se lee la historia y se guarda de forma atómica.
4. Éxito: `SetFlash` con "Horas estimadas registradas" (si no tenía) o "Horas estimadas actualizadas" y
   redirección 303.
5. Valor inválido o por encima del máximo: lista con el mensaje junto al campo de esa fila, lo escrito
   conservado y 422.
6. Historia de un Sprint finalizado: lista con el mensaje y 409.
7. Cualquier otro error: `MsgOperationFailed` con la lista y 500.

### Mapeo de errores del dominio a mensajes

| Error del dominio | Mensaje |
| --- | --- |
| Horas inválidas (regla de HU-22) | "El valor de horas no es válido" |
| Horas por encima del máximo (regla de HU-22) | "Las horas estimadas no pueden superar 9999,99" |
| `ErrStoryNotFound` | "La historia no existe" |
| `ErrStoryInFinishedSprint` | "La historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar" |
| Cualquier otro | `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `MsgOperationFailed`, la redirección 303 (`RedirectSeeOther`), los parciales
  `alert` y `field` (definidos en `templates/layout.html`), la protección contra envíos de otro origen, `ActiveContextFrom` y `RequireContext`.
- De HU-02 y HU-22 (`src/backlog`): el esquema de `user_stories`, `validEstimatedHours`,
  `ErrInvalidEstimatedHours` y los mensajes `MsgInvalidEstimatedHours` y `MsgHoursTooLarge` de la web.
- De HU-03: la tabla `effort_logs` (solo lectura, para las horas reales).
- No usa el catálogo de historias de HU-22 ni el tablero de HU-14.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación. `EstimateCatalog` repite parte de la
lectura de historias de HU-22 porque necesita las horas reales y el bloqueo por Sprint; se unifican
después si conviene.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a registrar, modificar y ver las horas.
- **II. Quality by Construction**: PASS. Una sola regla de horas y una sola sentencia de guardado atómica.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas del servicio, de
  los adaptadores, de handler y BDD.
- **IV. Agile, Collaborative Execution**: PASS. Las tres historias de usuario se entregan de forma
  incremental.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas.
- **VI. PostgreSQL Persistence**: PASS. Ambos puertos tienen adaptador con integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.