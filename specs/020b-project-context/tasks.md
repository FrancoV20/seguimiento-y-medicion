# Tasks: Formularios seguros y contexto de trabajo

**Input**: Design documents from `specs/020b-project-context/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-20a integrada (US1 a US3)

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Lee las tablas de `001_*.up.sql` (HU-01). Si el equipo agregara alguna, toma el último número de `main` más uno (hoy la próxima es la 008), sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Orden recomendado**: primero US1 y US2 (con eso HU-21 puede empezar) y después US3 a US5 (los necesitan HU-14 y HU-22 a HU-25). Las tareas marcadas `[Extra]` no son criterios de aceptación; se hacen solo si sobra tiempo.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar el esquema que se lee y preparar el entorno.

- [x] T001 Revisar `db/migrations/001_*.up.sql` y el store de HU-01 (tablas `projects`, `members` y `project_members`); anotar en `specs/020b-project-context/data-model.md` los nombres reales de columnas y decidir si se reutiliza el store existente o se crea `ContextStore`
- [x] T002 [P] Verificar que HU-20a expone `Section` (con `RequiresProject` y `RequiresMember`), `Server.Register`, `Server.Use`, `PageData`, `ActiveContext`, `Fail` y `RenderError`
- [x] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y que `001` está aplicada con `migrate -path db/migrations -database $env:DATABASE_URL up`
- [x] T004 Confirmar con `go doc net/http.CrossOriginProtection` la API disponible en Go 1.27.1 y registrar el resultado en `specs/020b-project-context/research.md`
- [ ] T005 [P] Crear el auxiliar de pruebas que registra tres secciones de prueba (sin requisitos, con `RequiresProject` y con `RequiresMember`) y un formulario de prueba en `src/web/testhelpers_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tipos, interfaz y adaptador PostgreSQL compartidos por las historias.

- [ ] T006 [P] Definir `MsgOperationFailed` ("La operación no pudo completarse"), `Flash` y `FormView` en `src/web/flash.go` y `src/web/forms.go`
- [ ] T007 [P] Definir `ResolvedContext`, `ProjectSummary`, `MemberSummary` y los nombres de las cookies de contexto en `src/web/context.go` (`ActiveContext` ya existe desde HU-20a)
- [ ] T008 [P] Definir la interfaz `ContextStore` con `ListProjects`, `ListMembers` y `ResolveContext` en `src/web/context_store.go`
- [ ] T009 [P] Escribir la prueba de integración de `PostgreSQLContextStore` con fixtures de dos proyectos e integrantes, incluyendo "no existe" frente a "no disponible" y la verificación de tiempo de `ResolveContext`, en `src/web/postgres_context_store_integration_test.go`
- [ ] T010 Implementar `PostgreSQLContextStore` con `database/sql` y pgx en `src/web/postgres_context_store.go`, ajustado al esquema de T001
- [ ] T011 [P] Definir los atributos de las cookies de contexto (`ctx_proyecto`, `ctx_integrante`: sesión, `HttpOnly`, `SameSite=Lax`, `Secure` con HTTPS) y su lectura segura en `src/web/context.go`

**Checkpoint**: El modelo, la interfaz y el adaptador PostgreSQL están listos para las historias.

---

## Phase 3: User Story 1 - Mensajes y formularios con comportamiento consistente (Priority: P1) 🎯 MVP

**Goal**: Componentes compartidos de mensajes, errores de campo y Post/Redirect/Get.

**Independent Test**: Un formulario de prueba enviado con datos inválidos, válidos y con falla de guardado, y recargando el resultado.

### Tests for User Story 1

- [ ] T012 [P] [US1] Escribir pruebas de `flash`: se muestra una sola vez, se borra al leerse y se renderiza escapado en `src/web/flash_test.go`
- [ ] T013 [P] [US1] Escribir pruebas de errores de campo con valores conservados y del texto único `MsgOperationFailed` sin mensaje de éxito en `src/web/forms_test.go`
- [ ] T014 [P] [US1] Escribir la prueba de Post/Redirect/Get: respuesta 303 y recarga sin duplicar el registro en `src/web/forms_test.go`
- [ ] T015 [P] [US1] Escribir la prueba de formato de mensajes: cada tipo tiene su clase y una etiqueta de texto además del color, igual en todas las páginas de prueba, en `src/web/alert_test.go`
- [ ] T016 [P] [US1] Escribir los escenarios BDD de mensajes y formularios en `test/web_forms_bdd_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Implementar `SetFlash` y la lectura con borrado en `src/web/flash.go`
- [ ] T018 [US1] Implementar `FormView` y los partials `alert.html` y `field.html` en `src/web/forms.go` y `src/web/templates/partials/`
- [ ] T019 [US1] Implementar el auxiliar de redirección 303 después de un envío válido en `src/web/forms.go`
- [ ] T020 [US1] Agregar `Flash` y `Form` a `PageData`, mostrar las alertas en `layout.html` y agregar al final de `app.css` los estilos de mensajes (éxito, error, advertencia) y de campos con error

**Checkpoint**: US1 deja los componentes que HU-21 a HU-25 reutilizan; con US1 y US2 HU-21 ya puede comenzar.

---

## Phase 4: User Story 2 - Mostrar contenido de forma segura (Priority: P1)

**Goal**: Contenido siempre escapado y envíos de otro origen rechazados.

**Independent Test**: Mostrar texto con `<script>` y enviar formularios desde el mismo origen y desde otro.

### Tests for User Story 2

- [ ] T021 [P] [US2] Escribir pruebas de escape: texto con HTML y scripts se muestra como texto en `src/web/escape_test.go`
- [ ] T022 [P] [US2] Escribir pruebas de protección: `Sec-Fetch-Site: cross-site` y `Origin` ajeno rechazados con 403 y mensaje claro, sin ejecutar la acción; mismo origen aceptado, en `src/web/security_test.go`
- [ ] T023 [P] [US2] Escribir los escenarios BDD de seguridad en `test/web_forms_bdd_test.go`
- [ ] T027 [P] [US2] [Extra] Escribir pruebas de cuerpo mayor a 1 MiB rechazado con 413 y de las cabeceras de seguridad en `src/web/security_test.go`

### Implementation for User Story 2

- [ ] T024 [US2] Integrar `http.CrossOriginProtection` con respuesta 403 usando `RenderError` en `src/web/security.go`
- [ ] T025 [US2] Registrar la protección con `Server.Use` en `src/cmd/server/main.go`
- [ ] T026 [US2] Revisar que ninguna plantilla use `template.HTML` con datos de usuarios en `src/web/templates/`
- [ ] T028 [US2] [Extra] Implementar cabeceras de seguridad (`nosniff`, `Referrer-Policy`, `Content-Security-Policy`) y el límite de cuerpo en `src/web/security.go`

**Checkpoint**: US2 protege el contenido y los formularios para todas las historias posteriores.

---

## Phase 5: User Story 3 - Seleccionar el proyecto activo (Priority: P2)

**Goal**: Elegir, mantener, cambiar y limpiar el proyecto activo de forma segura.

**Independent Test**: Con dos proyectos, seleccionar uno, navegar, cambiar de proyecto y repetir con identificadores inválidos, un proyecto eliminado y la base de datos caída.

### Tests for User Story 3

- [ ] T029 [P] [US3] Escribir pruebas de lectura de cookies (no numérica, negativa, muy grande, vacía) tratadas como ausencia en `src/web/context_test.go`
- [ ] T030 [P] [US3] Escribir pruebas de handler con `cookiejar`: seleccionar proyecto y ver su nombre en el layout, mantenerlo entre secciones y que no se autoselecciona con un solo proyecto, en `src/web/context_handlers_test.go`
- [ ] T031 [P] [US3] Escribir pruebas de handler con `cookiejar`: cambiar de proyecto limpia el integrante, volver a elegir el mismo lo conserva y una cookie de sesión nueva vuelve a pedir el proyecto, en `src/web/context_handlers_test.go`
- [ ] T032 [P] [US3] Escribir pruebas de handler: identificador inexistente o inválido conserva la selección anterior con mensaje claro; proyecto eliminado limpia la selección con advertencia, en `src/web/context_handlers_test.go`
- [ ] T033 [P] [US3] Escribir pruebas de handler: sin proyectos, "Aún no hay proyectos" y acceso a crear el primero solo si la sección Proyectos está registrada, en `src/web/context_handlers_test.go`
- [ ] T034 [P] [US3] Escribir la prueba con la base de datos caída: no se limpia la selección y se responde 503, en `src/web/context_middleware_test.go`
- [ ] T035 [P] [US3] Escribir los escenarios BDD de selección de proyecto en `test/context_bdd_test.go`

### Implementation for User Story 3

- [ ] T036 [US3] Implementar el middleware de resolución de contexto (lee cookies, valida con `ResolveContext`, limpia lo inválido con `SetFlash`, completa el `ActiveContext`) en `src/web/context_middleware.go`
- [ ] T037 [US3] Implementar `GET` y `POST /contexto/proyecto` (303, borra el integrante solo si el proyecto cambia) en `src/web/context_handlers.go`
- [ ] T038 [US3] Crear `select_project.html` y el parcial `project_picker.html` con la lista y el estado vacío en `src/web/templates/`
- [ ] T039 [US3] Registrar las rutas del contexto y el middleware con `Server.Register` y `Server.Use` en `src/cmd/server/main.go`

**Checkpoint**: US3 permite elegir y mantener el proyecto activo; desbloquea a HU-22, HU-23 y HU-25.

---

## Phase 6: User Story 4 - Identificarme como integrante del proyecto (Priority: P2)

**Goal**: Elegir el integrante entre los del proyecto y rechazar identificaciones ajenas o caducas.

**Independent Test**: Con un proyecto de varios integrantes, identificarse, actuar sin identificación, y repetir con un integrante de otro proyecto y con uno quitado.

### Tests for User Story 4

- [ ] T040 [P] [US4] Escribir pruebas de handler: el selector muestra solo integrantes del proyecto activo y el nombre queda visible en el layout, en `src/web/context_handlers_test.go`
- [ ] T041 [P] [US4] Escribir pruebas de handler: acción que requiere integrante sin identificarse redirige a identificarse y vuelve, en `src/web/context_handlers_test.go`
- [ ] T042 [P] [US4] Escribir pruebas de handler: integrante de otro proyecto rechazado con identificación limpiada; integrante quitado limpiado con advertencia, en `src/web/context_middleware_test.go`
- [ ] T043 [P] [US4] Escribir los escenarios BDD de identificación del integrante en `test/context_bdd_test.go`

### Implementation for User Story 4

- [ ] T044 [US4] Implementar `GET` y `POST /contexto/integrante` en `src/web/context_handlers.go`, validando que el integrante pertenezca al proyecto activo
- [ ] T045 [US4] Crear `select_member.html` con el estado vacío en `src/web/templates/`
- [ ] T046 [US4] Implementar en `src/web/context_middleware.go` el rechazo de acciones con integrante ajeno y la limpieza de identificación
- [ ] T047 [US4] Mostrar el integrante activo y el enlace "Cambiar" en `src/web/templates/layout.html`

**Checkpoint**: US4 identifica al integrante y deja listo el contexto para HU-14 y HU-24.

---

## Phase 7: User Story 5 - Acceder a secciones que requieren contexto (Priority: P2)

**Goal**: Guiar hacia la selección y volver al destino pedido, sin redirecciones inseguras.

**Independent Test**: Entrar a una sección protegida sin contexto, elegirlo y comprobar el regreso; probar destinos de regreso inválidos. Se usan las secciones de prueba de T005.

### Tests for User Story 5

- [ ] T048 [P] [US5] Escribir pruebas por tabla del saneamiento de `volver_a` (`/`, `/backlog`, `//otro.com`, `https://x`, `\\x`, caracteres de control, sección no registrada) en `src/web/return_to_test.go`
- [ ] T049 [P] [US5] Escribir pruebas de handler: sección protegida sin proyecto redirige con mensaje y regresa tras elegir; sección sin requisitos accesible, en `src/web/context_middleware_test.go`
- [ ] T050 [P] [US5] Escribir los escenarios BDD de secciones protegidas en `test/context_bdd_test.go`

### Implementation for User Story 5

- [ ] T051 [US5] Implementar el saneamiento del destino de regreso en `src/web/return_to.go`
- [ ] T052 [US5] Implementar `RequireContext(section)` que redirige a `/contexto/proyecto` o `/contexto/integrante` con `volver_a` en `src/web/context_middleware.go`
- [ ] T053 [US5] Aplicar `RequireContext` a las secciones según `RequiresProject` y `RequiresMember` al registrarlas, en `src/web/sections.go`

**Checkpoint**: US5 protege las secciones y vuelve al destino pedido de forma segura.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación y trazabilidad.

- [ ] T054 [P] Documentar en `specs/020b-project-context/quickstart.md` los datos de prueba reales una vez confirmado el esquema de `001`
- [ ] T055 [P] Registrar la deuda técnica "roles (Scrum Master y Product Builder) sin distinguir" en `docs/` o en el backlog para la retrospectiva
- [ ] T056 [P] Revisar que errores y mensajes de formularios y del contexto no expongan detalles internos en `src/web/context_handlers.go` y `src/web/forms.go`
- [ ] T057 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T058 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T059 Avisar al equipo que HU-21 puede comenzar al cerrar US1 y US2, y que HU-14 y HU-22 a HU-25 pueden usar el contexto al cerrar US3 a US5
- [ ] T060 Guardar la salida de `go test -count=1 -p 1 -v ./...` en `specs/020b-project-context/evidencia-pruebas.txt` y la cobertura de `go test -count=1 -p 1 -cover ./...` para el informe de métricas y cobertura
- [ ] T061 Revisar trazabilidad de FR-001 a FR-024 y SC-001 a SC-012 en `specs/020b-project-context/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 define lo que leen T009 y T010; T002, T003 y T005 en paralelo; T004 independiente.
- **Foundational (Phase 2)**: Depende de Setup; T006, T007, T008, T009 y T011 en paralelo; T010 requiere T001, T008 y T009.
- **User Story 1 (Phase 3)**: Depende de T006 y de HU-20a (US1 a US3); es el MVP y desbloquea HU-21.
- **User Story 2 (Phase 4)**: Depende de T004 y de la página 403 vía `RenderError` de HU-20a; puede avanzar en paralelo con US1.
- **User Story 3 (Phase 5)**: Depende de toda la Phase 2 y de T017 (`SetFlash`); no depende de US2.
- **User Story 4 (Phase 6)**: Depende de T036 y T037; la validación de pertenencia usa `ResolveContext`.
- **User Story 5 (Phase 7)**: Depende de T036; puede avanzar en paralelo con US4.
- **Polish (Phase 8)**: Depende de las historias que se entreguen.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-20a (layout, `PageData`, `RenderError`); se verifica con un formulario de prueba.
- **US2 (P1)**: Depende de HU-20a (`Server.Use`, página de error); se verifica con un formulario de prueba.
- **US3 (P2)**: Depende de HU-20a, de los datos de HU-01 y de US1 (`SetFlash` para las advertencias); se prueba con el almacenamiento PostgreSQL real.
- **US4 (P2)**: Depende del middleware y de la selección de proyecto de US3 (T036 y T037).
- **US5 (P2)**: Depende del middleware (T036) y del registro de secciones de HU-20a; es independiente de US4.
- **HU-21**: Depende de US1 y US2.
- **HU-14, HU-22, HU-23, HU-24 y HU-25**: Dependen de US3 a US5 además de US1 y US2.

### Cross-feature Dependencies

- **HU-20a**: debe estar integrada (`Section`, `Server.Register`, `Server.Use`, `ActiveContext`, `Fail`, `RenderError`, layout).
- **HU-01**: aporta los proyectos e integrantes que se leen.
- **HU-21**: puede reutilizar `project_picker.html` en su lista de proyectos y los componentes de US1.
- **HU-14, HU-22, HU-23, HU-24 y HU-25**: consumen `ActiveContext`, las secciones protegidas y los componentes de mensajes y formularios.
- **HU-11**: no hay acoplamiento; quitar un integrante se refleja porque se valida en cada solicitud.

### Parallel Opportunities

- T002, T003 y T005; T006, T007, T008, T009 y T011.
- Las pruebas de cada historia son [P] entre sí.
- US1 y US2 pueden desarrollarse en paralelo; US4 y US5 pueden desarrollarse en paralelo después de US3.
- Reparto sugerido: una persona con US1 y US2 (formularios, mensajes y seguridad) y otra con US3 a US5 (contexto).

## Parallel Example: User Story 1

```text
T012: Pruebas de mensajes de una sola vista en src/web/flash_test.go
T013: Pruebas de errores de campo en src/web/forms_test.go
T014: Prueba de Post/Redirect/Get en src/web/forms_test.go
T015: Formato de mensajes en src/web/alert_test.go
T016: Escenarios BDD en test/web_forms_bdd_test.go
```

## Parallel Example: User Story 2

```text
T021: Pruebas de escape en src/web/escape_test.go
T022: Pruebas de origen en src/web/security_test.go
T023: Escenarios BDD en test/web_forms_bdd_test.go
```

## Parallel Example: User Story 3

```text
T029: Pruebas de lectura de cookies en src/web/context_test.go
T030: Selección y permanencia en src/web/context_handlers_test.go
T031: Cambio de proyecto y sesión nueva en src/web/context_handlers_test.go
T032: Identificadores inválidos y proyecto eliminado en src/web/context_handlers_test.go
T033: Sin proyectos en src/web/context_handlers_test.go
T034: Base de datos caída en src/web/context_middleware_test.go
T035: Escenarios BDD en test/context_bdd_test.go
```

## Parallel Example: User Story 4

```text
T040: Selector de integrantes en src/web/context_handlers_test.go
T041: Acción sin identificar en src/web/context_handlers_test.go
T042: Integrante ajeno o quitado en src/web/context_middleware_test.go
T043: Escenarios BDD en test/context_bdd_test.go
```

## Parallel Example: User Story 5

```text
T048: Saneamiento de volver_a en src/web/return_to_test.go
T049: Secciones protegidas en src/web/context_middleware_test.go
T050: Escenarios BDD en test/context_bdd_test.go
```

## Implementation Strategy

### MVP First (US1 y US2)

1. Completar Setup y Foundational.
2. Escribir y observar fallar las pruebas de US1 y US2.
3. Implementar US1 y US2 y ejecutar `docker compose up -d postgres` seguido de `go test -count=1 -p 1 ./...`.
4. Avisar al equipo que HU-21 ya puede comenzar.

### Incremental Delivery

1. Entregar US1 y US2 como base de formularios para HU-21.
2. Agregar US3 para elegir el proyecto activo.
3. Agregar US4 para identificar al integrante.
4. Agregar US5 para proteger las secciones y regresar al destino pedido.
5. Ejecutar la validación completa de `quickstart.md` antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Los nombres de columnas de la migración 001 deben verificarse en el archivo real antes de escribir T009 y T010.
- Las pruebas de handler usan `net/http/cookiejar` y el `PostgreSQLContextStore` real; no se reemplazan con un store en memoria.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T015, T016 |
| FR-002 | T013, T016 |
| FR-003 | T013, T016 |
| FR-004 | T012, T014, T016 |
| FR-005 | T021, T023 |
| FR-006 | T022, T023 |
| FR-007 | T030, T035 |
| FR-008 | T009, T030, T035 |
| FR-009 | T030, T031 |
| FR-010 | T033, T035 |
| FR-011 | T030 |
| FR-012 | T031 |
| FR-013 | T032 |
| FR-014 | T032 |
| FR-015 | T009, T040, T043 |
| FR-016 | T041, T043 |
| FR-017 | T042, T043 |
| FR-018 | T042, T043 |
| FR-019 | T049, T050 |
| FR-020 | T048, T050 |
| FR-021 | T049, T050 |
| FR-022 | T029, T032 |
| FR-023 | T009, T032, T042 |
| FR-024 | T034 |
| SC-001 | T015 |
| SC-002 | T013 |
| SC-003 | T014 |
| SC-004 | T021, T023 |
| SC-005 | T022, T023 |
| SC-006 | T009, T030 |
| SC-007 | T041, T049 |
| SC-008 | T029, T032 |
| SC-009 | T042 |
| SC-010 | T048 |
| SC-011 | No automatizable: se verifica en el chequeo manual de `quickstart.md` |
| SC-012 | Verificación de tiempo de `ResolveContext` en T009 |
