# Implementation Plan: Formularios seguros y contexto de trabajo

**Branch**: `020b-project-context` | **Date**: 2026-10-17 | **Updated**: 2026-10-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar sobre la base web de HU-20a tres piezas: (1) componentes compartidos de mensajes y
formularios con el patrón Post/Redirect/Get y un mensaje de una sola vista en cookie; (2)
protección de contenido (escape) y de envíos con `http.CrossOriginProtection`; y (3) el contexto
de trabajo: selección del proyecto activo e identificación del integrante mediante cookies de
sesión (sin tabla ni migración), validadas contra PostgreSQL en cada solicitud. Un middleware
resuelve el contexto, lo expone a las plantillas y a los handlers, limpia selecciones inválidas
con una advertencia y protege las secciones que declaran `RequiresProject` o `RequiresMember`,
redirigiendo a la selección con un destino de regreso seguro. Un adaptador
`PostgreSQLContextStore` lee proyectos e integrantes.

> **Supuestos a verificar antes de implementar** (ajustar si difieren):
> 1. HU-20a está integrada: `Section`, `Server.Register`, `Server.Use`, `PageData`,
>    `ActiveContext`, `Renderer.Page`, `Fail`, `RenderError` y el layout.
> 2. El esquema real de `db/migrations/001_*.up.sql`: las tablas son `projects`, `members` y
>    `project_members`; los nombres de columnas se confirman en T001 y las consultas de este plan
>    se ajustan a ellos.
> 3. Los paquetes siguen en `src/web`.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `net/url`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose; `PostgreSQLContextStore` solo lee proyectos e
integrantes mediante `database/sql` y pgx. La selección y el mensaje de una sola vista viven en
cookies del navegador.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias, pruebas de handler con
`httptest` y `net/http/cookiejar`, escenarios BDD en `test/web_forms_bdd_test.go` y
`test/context_bdd_test.go`, y pruebas de integración contra PostgreSQL. `-p 1` evita que los
paquetes de integración se pisen al compartir una base.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Componentes compartidos y capa de contexto dentro de la aplicación web SSR.

**Performance Goals**: Validación del contexto en menos de 50 ms por solicitud, con una sola
consulta.

**Constraints**: Sin migraciones nuevas, sin dependencias nuevas, cookies de sesión (sin
`Expires`), validación contra la base en cada solicitud, destinos de regreso restringidos a
secciones propias, mensajes en español.

**Scale/Scope**: Un proyecto y un integrante activos por navegador; no incluye creación de
proyectos (HU-21), alta o baja de integrantes (HU-11), roles ni autenticación.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Habilita HU-21 a HU-25 y HU-14; no agrega
  funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. Mensajes, formularios, seguridad, resolución de contexto,
  protección de secciones y saneamiento del destino de regreso son piezas pequeñas y testeables.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada regla tiene pruebas unitarias o de
  handler y un escenario BDD con el almacenamiento real.
- **IV. Agile, Collaborative Execution**: PASS. Las dos personas del equipo pueden trabajar en
  paralelo: US1 y US2 (formularios y seguridad) y US3 a US5 (contexto).
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas; la decisión sobre
  roles queda como deuda técnica para la retrospectiva.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `ContextStore` tiene su adaptador
  `PostgreSQLContextStore` con pruebas de integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; las cookies son una
  decisión de arquitectura interna registrada en `research.md`.

## Project Structure

### Documentation (this feature)

```text
specs/020b-project-context/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML. Las direcciones y campos se
describen abajo.

### Source Code (repository root)

```text
src/
└── web/
    ├── flash.go                        # mensaje de una sola vista (cookie)
    ├── forms.go                        # FormView, MsgOperationFailed, redirección 303
    ├── security.go                     # CrossOriginProtection y (extra) cabeceras y límite de cuerpo
    ├── context.go                      # ResolvedContext, cookies de contexto
    ├── context_store.go                # interfaz ContextStore
    ├── postgres_context_store.go       # adaptador PostgreSQL
    ├── context_middleware.go           # resolución, limpieza y RequireContext
    ├── context_handlers.go             # selección de proyecto e integrante
    ├── return_to.go                    # saneamiento del destino de regreso
    ├── templates/
    │   ├── select_project.html
    │   ├── select_member.html
    │   └── partials/{alert.html,field.html,project_picker.html}
    └── *_test.go
test/
├── web_forms_bdd_test.go
└── context_bdd_test.go
```

**Structure Decision**: Se agregan archivos al paquete `src/web` creado en HU-20a, con un archivo
por responsabilidad. Se editan unos pocos archivos de HU-20a: `render.go` (agrega `Flash` y `Form`
a `PageData` y lee el `ActiveContext`), `layout.html` (alertas, proyecto e integrante activos),
`app.css` (estilos de mensajes y formularios) y `main.go` (registra los middleware con
`Server.Use`). El parcial `project_picker.html` permite que HU-21 reutilice la lista
seleccionable de proyectos en su pantalla.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/contexto/proyecto` | Lista de proyectos para elegir (o "Aún no hay proyectos" y acceso a crear el primero) |
| POST | `/contexto/proyecto` | Campos `proyecto_id` y `volver_a`; fija el proyecto activo (303) |
| GET | `/contexto/integrante` | Integrantes del proyecto activo |
| POST | `/contexto/integrante` | Campos `integrante_id` y `volver_a`; fija el integrante (303) |

Cookies: `ctx_proyecto` y `ctx_integrante` (de sesión) y `flash` (corta duración), con `HttpOnly`,
`SameSite=Lax`, `Path=/` y `Secure` cuando la solicitud llega por HTTPS. Las de contexto no llevan
`Expires`.

### Contratos internos para HU-21 a HU-25

- `SetFlash(w, kind, text)`: mensaje de una sola vista (éxito, error, advertencia).
- `FormView{Values, Errors}` y partials `alert.html` y `field.html`: errores por campo y valores
  conservados.
- `MsgOperationFailed`: constante con "La operación no pudo completarse".
- Redirección 303 después de un envío válido (auxiliar en `forms.go`).
- `ActiveContext` completado en cada solicitud por el middleware de contexto, y
  `RequireContext(section)` aplicado según `Section.RequiresProject` y `Section.RequiresMember`.
- `project_picker.html`: lista seleccionable de proyectos reutilizable.

### Contrato con HU-20a

- Usa `Section.RequiresProject` y `Section.RequiresMember` para decidir qué proteger.
- Completa el `ActiveContext` con `WithActiveContext` y `Renderer.Page` lo lee de la solicitud.
- Usa `Fail` para indisponibilidad (503) o no encontrado, y `RenderError` para el 403.
- Registra sus middleware con `Server.Use` y sus rutas con `Server.Register`.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a mensajes, formularios,
  seguridad y contexto; no crea ni edita proyectos o integrantes.
- **II. Quality by Construction**: PASS. Flash, formularios, seguridad, middleware, handlers,
  almacenamiento y saneamiento están separados.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas
  unitarias, de handler con cookies, BDD y de integración con PostgreSQL.
- **IV. Agile, Collaborative Execution**: PASS. Las tareas permiten entregar US1 y US2 primero
  (HU-21 puede empezar) y luego US3 a US5.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas.
- **VI. PostgreSQL Persistence**: PASS. El almacenamiento tiene adaptador concreto e integración
  contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.

## Desviaciones respecto al plan original (registradas al implementar)

- Los parciales `alert`, `field`, `project_picker` y `member_picker` se definen con `{{define}}` en `templates/layout.html` y `select_member.html`, no en `templates/partials/`: `render.go` solo embebe `templates/*.html`.
- `SetFlash` recibe también la solicitud: `SetFlash(w, r, kind, text)`, para decidir el atributo `Secure`.
- `RequireContext` se aplica dentro de `Server.Register` (`server.go`), no en `sections.go`: ahí se unen la sección y sus rutas.
- Las advertencias que detecta el middleware (proyecto o integrante que dejó de ser válido) viajan en el contexto de la solicitud (`withPendingFlash`) para verse en la misma respuesta y no en la siguiente.
- Las pruebas de US4 y US5 están en `context_member_test.go` y `context_guard_test.go`, no en los archivos indicados en `tasks.md`.
- T027 y T028 (extras) no se implementaron.