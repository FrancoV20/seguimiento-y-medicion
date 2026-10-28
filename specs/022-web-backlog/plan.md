# Implementation Plan: Interfaz web del Product Backlog

**Branch**: `HU-22` | **Date**: 2026-10-26 (documentación final) | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

**Estado final: COMPLETADA.** Se implementaron las tres historias de usuario (US1, US2 y US3) con
el flujo de pruebas TDD en fases RED y GREEN, más escenarios BDD y pruebas de integración con
PostgreSQL. La ruta `/backlog` reúne la creación y la consulta de historias para el proyecto activo.
El alta reutiliza el dominio de HU-02, conserva los datos y presenta errores de validación junto al
campo con HTTP 422; tras un alta correcta aplica Post/Redirect/Get (303) y comunica éxito o
advertencias mediante mensajes flash. La lista muestra identificador, título, prioridad, Story
Points, horas estimadas y estado, con aislamiento por proyecto, estado vacío y el texto "Sin
estimar" para valores ausentes.

No se agregaron migraciones. El catálogo de solo lectura y el store de escritura usan PostgreSQL
mediante `database/sql` y pgx; las reglas de horas y el estado inicial se resuelven en el dominio de
HU-02.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. El alta usa el servicio y el store de HU-02.
`PostgreSQLStoryCatalog` solo lee las historias del proyecto mediante `database/sql` y pgx.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias del formulario, pruebas de handler
con `net/http/httptest` y `net/http/cookiejar` sobre el servicio y la base reales (el proyecto activo
se elige por el flujo real de HU-20b), escenarios BDD en `test/web_backlog_bdd_test.go` y pruebas de
integración contra PostgreSQL.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Pantalla de una aplicación web SSR sobre el núcleo de dominio en Go.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-007).

**Constraints**: Sin migraciones ni dependencias nuevas, sin JavaScript, reglas y mensajes de
validación de HU-02 sin cambios, ningún detalle interno en las respuestas, los datos del formulario
se conservan ante cualquier error, y el proyecto de la historia sale siempre del contexto del
servidor.

**Scale/Scope**: Una pantalla (formulario y lista mínima) y una ruta con dos métodos; 18 escenarios. No incluye
orden por prioridad, filtros ni edición (HU-13), asignación a Sprints (HU-23) ni modificar las horas
estimadas después del alta (HU-17).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Hace consumible el flujo de HU-02 y habilita HU-23 y
  HU-24; no agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. El armado del formulario, el mapeo de errores, el catálogo y
  los handlers están separados; las reglas de negocio siguen en HU-02.
- **III. Test-First and Evidence-Driven Delivery**: PASS. La historia se completó mediante el ciclo
  TDD RED/GREEN y cuenta con pruebas unitarias, de handler, BDD e integración PostgreSQL.
- **IV. Agile, Collaborative Execution**: PASS. Alcance acotado y demostrable en la Sprint Review.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas; la lista mínima deja
  base para HU-13.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `StoryCatalog` tiene su adaptador
  `PostgreSQLStoryCatalog` con prueba de integración contra PostgreSQL; el alta reutiliza el
  adaptador de HU-02.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/022-web-backlog/
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
└── web/
    ├── backlog_catalog.go              # StoryRow e interfaz StoryCatalog
    ├── postgres_story_catalog.go       # adaptador PostgreSQL
    ├── backlog_form.go                 # criterios, parseo del envío, textos y mapeo de errores de HU-02
    ├── backlog_handlers.go             # GET y POST /backlog
    ├── templates/backlog.html          # formulario y lista
    └── *_test.go
test/
└── web_backlog_bdd_test.go
```

**Structure Decision**: Se agregan archivos al paquete `src/web` creado en HU-20a, con un archivo por
responsabilidad. Se edita un único archivo existente, `src/cmd/server/main.go`, para construir el
servicio de HU-02 y el catálogo y registrar la sección. No se modifica ningún archivo de HU-20a,
HU-20b ni HU-21.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/backlog` | Formulario de alta y lista de historias del proyecto activo |
| POST | `/backlog` | Campos `titulo`, `descripcion`, `prioridad`, `story_points`, `horas_estimadas`, `criterio` (uno por campo) y `accion`; según la acción, guarda o vuelve a mostrar el formulario |

Valores de `accion`: `guardar` (por defecto), `agregar_criterio` y `quitar_criterio:N` (con el
índice del campo). La sección se registra como `Section{Name: "Backlog", Path: "/backlog",
RequiresProject: true}`.

### Flujo del POST

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto con su
   mensaje; el handler no se ejecuta.
2. Leer el formulario y armar la lista de criterios y los valores a conservar.
3. `agregar_criterio`: agregar un campo vacío y volver a mostrar el formulario (200), sin guardar.
   `quitar_criterio:N`: quitar ese campo, salvo que sea el último, y volver a mostrar el formulario
   (200), sin guardar.
4. `guardar`: ignorar criterios en blanco, parsear los Story Points y las horas estimadas (vacíos son
   válidos; coma o punto como decimal, sin separador de miles) e invocar el
   servicio de HU-02 con el proyecto de `ActiveContext`. El campo de proyecto del formulario, si
   llega, se descarta.
5. Éxito: `SetFlash` de éxito ("Historia creada correctamente") o, si faltan Story Points, de
   advertencia con el mensaje combinado; sin horas no hay advertencia. Redirección 303 a `/backlog`.
6. Error de validación de HU-02: mensaje junto al campo correspondiente y respuesta 422 con el
   formulario y los datos conservados.
7. Cualquier otro error: `MsgOperationFailed` con los datos conservados y respuesta 500. Si la lectura
   de la lista también falla, se usa `Fail`.

### Mapeo de errores de HU-02 a campos (se confirma en T001)

| Error | Campo |
| --- | --- |
| Título obligatorio | `titulo` |
| Descripción obligatoria | `descripcion` |
| Prioridad inválida | `prioridad` |
| Sin criterios de aceptación | `criterio` (grupo de criterios) |
| Story Points inválidos | `story_points` |
| Horas estimadas inválidas | `horas_estimadas` |
| Horas estimadas por encima de 9999,99 | `horas_estimadas` |
| Cualquier otro | Mensaje general `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `FormView`, `MsgOperationFailed`, la redirección 303, los parciales
  `alert.html` y `field.html`, la protección contra envíos de otro origen, `ActiveContextFrom` y
  `RequireContext` (aplicado por `Section.RequiresProject`).
- De HU-02: servicio de alta de historias, `StoryStore`, valores válidos, errores y mensajes.
- No usa `ContextStore` ni `project_picker.html`.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación. `StoryCatalog` es una interfaz
propia de HU-22 porque el listado del Backlog (HU-13) no está en el Sprint 2; el costo es una consulta
de lectura que HU-13 podrá extender con orden y filtro.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a la pantalla de alta y lista mínima.
- **II. Quality by Construction**: PASS. Las reglas de negocio siguen en HU-02; el handler traduce
  formato, acciones del formulario y errores.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas de
  formulario, de handler con almacenamiento y contexto reales, BDD e integración con PostgreSQL.
- **IV. Agile, Collaborative Execution**: PASS. Las tareas permiten entregar la carga primero, la
  corrección de errores después y la lista al final.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas.
- **VI. PostgreSQL Persistence**: PASS. El único almacenamiento nuevo (lectura de historias) tiene
  adaptador e integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.

## Resultado de implementación

**HU-22: COMPLETADA.** US1 (alta), US2 (validación y retención de datos) y US3 (listado) están
implementadas. Los casos con errores de validación devuelven HTTP 422 con los valores conservados;
el alta exitosa sigue PRG con HTTP 303. El GET lista las historias del proyecto activo, muestra
"Aún no hay historias" cuando corresponde y representa estimaciones ausentes como "Sin estimar".
Los errores de lectura se delegan al mecanismo genérico `Fail` de HU-20a. El desarrollo siguió el
ciclo TDD RED/GREEN y cuenta con pruebas unitarias, de handler, BDD e integración PostgreSQL.
