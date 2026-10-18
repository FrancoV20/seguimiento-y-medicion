# Implementation Plan: Interfaz web de proyectos

**Branch**: `HU-21` | **Date**: 2026-10-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar la pantalla "Gestión de Proyectos" como páginas generadas por el servidor: una sola
ruta `/proyectos` que muestra el formulario de alta y la lista de proyectos. El handler no
duplica reglas de negocio: valida solo el formato del envío (fechas `AAAA-MM-DD`, identificadores
de integrantes, duplicados), delega el alta en el caso de uso de HU-01 y traduce sus errores a
mensajes junto a cada campo. Un adaptador `PostgreSQLProjectCatalog` propio de HU-21 lee la lista
de proyectos y el catálogo de integrantes. Los integrantes se cargan con un script semilla. No
agrega migraciones ni modifica HU-20a ni HU-20b.

> **Supuestos a verificar antes de implementar (T001 y T002)**:
> 1. La ruta del paquete de HU-01 y las firmas de `ProjectService.CreateProject` y `ProjectStore`
>    (el servicio recibe los integrantes como texto y los convierte a enteros).
> 2. Los tipos de error de HU-01 y sus mensajes exactos, la normalización del nombre y el texto
>    del mensaje de éxito.
> 3. HU-20a y HU-20b (US1 y US2) están en `main`: `Server.Register`, `Renderer.Page`, `Fail`,
>    `RenderError`, `SetFlash`, `FormView`, los parciales `alert.html` y `field.html`, y la
>    protección contra envíos de otro origen.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `time`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. El alta usa el `ProjectStore` de HU-01.
`PostgreSQLProjectCatalog` solo lee `projects`, `project_members` y `members` mediante
`database/sql` y pgx.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias del formulario, pruebas de handler
con `net/http/httptest` sobre el `ProjectService` y la base reales, escenarios BDD en
`test/web_projects_bdd_test.go` y pruebas de integración contra PostgreSQL.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Pantalla de una aplicación web SSR sobre el núcleo de dominio en Go.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-006).

**Constraints**: Sin migraciones ni dependencias nuevas; reglas y mensajes de validación de HU-01
sin cambios; ningún detalle interno en las respuestas; los datos del formulario se conservan ante
cualquier error.

**Scale/Scope**: Una pantalla (lista y formulario) y una ruta con dos métodos. No incluye editar
proyectos, estado, selección del proyecto activo, alta de integrantes ni paginación.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Hace consumible el flujo de HU-01 y habilita HU-22 a
  HU-25; no agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. El parseo del formulario, el mapeo de errores, el
  catálogo y los handlers están separados; las reglas de negocio siguen en HU-01.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas de handler y
  un escenario BDD con almacenamiento real; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance acotado y demostrable en la Sprint Review;
  los criterios de usabilidad se miden con el equipo.
- **V. Measurement and Continuous Improvement**: PASS. SC-003 y SC-004 se registran con evidencia
  para la retrospectiva.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `ProjectCatalog` tiene su adaptador
  `PostgreSQLProjectCatalog` con prueba de integración contra PostgreSQL; el alta reutiliza el
  adaptador de HU-01.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/021-web-projects/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML; su comportamiento está en
`spec.md` y las direcciones y campos, abajo.

### Source Code (repository root)

```text
db/
└── seeds/members.sql                   # integrantes de prueba (no es una migración)
src/
└── web/
    ├── projects_catalog.go             # ProjectRow, MemberOption, interfaz ProjectCatalog
    ├── postgres_project_catalog.go     # adaptador PostgreSQL
    ├── projects_form.go                # parseo del envío, textos y mapeo de errores de HU-01
    ├── projects_handlers.go            # GET y POST /proyectos
    ├── templates/projects.html         # formulario y lista
    └── *_test.go
test/
└── web_projects_bdd_test.go
```

**Structure Decision**: Se agregan archivos al paquete `src/web` creado en HU-20a, con un archivo
por responsabilidad. Se edita un único archivo existente, `src/cmd/server/main.go`, para
construir el `ProjectService` de HU-01 y el catálogo y registrar la sección. No se modifica
ningún archivo de HU-20a ni de HU-20b.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/proyectos` | Formulario de alta y lista de proyectos |
| POST | `/proyectos` | Campos `nombre`, `fecha_inicio`, `fecha_fin` e `integrantes` (uno por integrante marcado); crea el proyecto y redirige con 303 a `/proyectos` |

La sección se registra como `Section{Name: "Proyectos", Path: "/proyectos"}`, sin
`RequiresProject` ni `RequiresMember`, para poder entrar sin proyecto activo.

### Flujo del POST

1. Leer el formulario y dejar los valores en `FormView` para conservarlos.
2. Parsear las fechas (`AAAA-MM-DD`) y los identificadores de integrantes, quitando repetidos.
   Un valor inválido genera un error junto a su campo.
3. Si no hubo errores de formato, invocar `ProjectService.CreateProject` (HU-01).
4. Éxito: `SetFlash` de éxito y redirección 303 a `/proyectos`.
5. Error de validación de HU-01: mensaje junto al campo correspondiente y respuesta 422 con el
   formulario y los datos conservados.
6. Cualquier otro error: `MsgOperationFailed` con los datos conservados y respuesta 500. Si la
   lectura de la lista o de los integrantes también falla, se usa `Fail`.

### Mapeo de errores de HU-01 a campos (se confirma en T001)

| Error de HU-01 | Campo |
| --- | --- |
| Nombre obligatorio | `nombre` |
| Fecha de fin anterior a la de inicio | `fecha_fin` |
| Falta una fecha | La fecha faltante |
| Sin integrantes o integrante inexistente | `integrantes` |
| Cualquier otro | Mensaje general `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b (US1 y US2): `SetFlash`, `FormView`, `MsgOperationFailed`, la redirección 303, los
  parciales `alert.html` y `field.html` y la protección contra envíos de otro origen.
- De HU-01: `ProjectService.CreateProject`, `ProjectStore` y sus errores y mensajes.
- No usa `ContextStore`, `project_picker.html` ni `ActiveContext`.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación. `ProjectCatalog` es una
interfaz propia de HU-21 en lugar de ampliar el `ContextStore` de HU-20b, para no modificar esa
historia; el costo es una consulta de lista de proyectos adicional, de lectura.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a la pantalla de alta y lista.
- **II. Quality by Construction**: PASS. Las reglas de negocio siguen en HU-01 y el handler solo
  traduce formato y errores.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas de
  formulario, de handler con almacenamiento real, BDD e integración con PostgreSQL.
- **IV. Agile, Collaborative Execution**: PASS. Las tareas permiten entregar crear y corregir
  primero y la lista después.
- **V. Measurement and Continuous Improvement**: PASS. Los criterios de usabilidad quedan con
  protocolo y evidencia.
- **VI. PostgreSQL Persistence**: PASS. El único almacenamiento nuevo (lecturas de lista e
  integrantes) tiene adaptador e integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.
