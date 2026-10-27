# Tasks: Interfaz web de registro de esfuerzo

**Input**: Design documents from `specs/024-web-effort/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-03, HU-04, FIX-HU-04, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno, sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Editar o borrar registros, métricas (HU-07, HU-25) y cambio de estado de historias (HU-14) quedan fuera. US3 (historial, P2) se puede dejar fuera sin afectar US1 ni US2. No se modifican archivos de HU-04, HU-14, HU-15, HU-20a ni HU-20b.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza de HU-03, HU-04, HU-20a y HU-20b.

- [X] T001 Revisar HU-03 en `main`: ruta del paquete de esfuerzo y firma del servicio de registro (entrada, tipo de las horas, resultado); errores y mensajes exactos de horas inválidas, actividad vacía, fecha requerida e integrante no especificado; si ya existe una validación que devuelve todas las violaciones; qué historias acepta (solo las del Sprint activo del proyecto, y si incluye "Hecho"); si verifica el proyecto; esquema de `effort_logs` (columnas de historia, integrante, fecha, actividad y horas, tipo de las horas); identificadores ya definidos en el paquete `web` por HU-14, HU-15, HU-17, HU-21, HU-22 y HU-23 y cómo formatean horas HU-17 y HU-22; anotar el resultado en `specs/024-web-effort/data-model.md` y ajustar el mapeo de `plan.md`
- [X] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register` con varias rutas y métodos, `Section.RequiresProject` y `Section.RequiresMember` aplicados también a `POST`, `ActiveContextFrom` con identificador y nombre del integrante, `SetFlash`, `FormView`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303, regreso a la pantalla tras identificarse y protección contra envíos de otro origen — `HU-24` parte de `main` (750b2b7): HU-20a y HU-20b integradas
- [X] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/024-web-effort/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [X] T004 [P] Confirmar que la migración `003` (`effort_logs`) y las de HU-04 en `main` coinciden con el esquema anotado en `specs/024-web-effort/data-model.md` y avisar al equipo si cambiaron

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Catálogo de solo lectura, formulario y adaptador de HU-03, compartidos por las historias.

- [X] T005 [P] Definir `EffortSprint`, `EffortStoryRow`, `EffortEntryRow`, `EffortHistory` y la interfaz `EffortCatalog` con `ActiveSprintStories` y `StoryHistory` en `src/web/effort_catalog.go`
- [X] T006 [P] Escribir la prueba de integración de `PostgreSQLEffortCatalog` con fixtures propias (dos proyectos, cada uno con un Sprint activo; un Sprint finalizado; historias "Pendiente", "En Sprint", "En progreso" y "Hecho"; registros de dos integrantes con distintas fechas): solo historias del Sprint activo del proyecto, horas acumuladas y cero sin registros, historial con integrante, fecha, actividad y horas, orden más reciente primero, total igual a la suma, historia ajena "no encontrada", proyecto sin Sprint activo y el de mayor número con dos activos, en `src/web/postgres_effort_catalog_integration_test.go`
- [X] T007 Implementar `PostgreSQLEffortCatalog` con `database/sql` y pgx y las consultas de `data-model.md` en `src/web/postgres_effort_catalog.go`
- [X] T008 [P] Escribir las pruebas unitarias del armado del formulario: fecha vacía, con formato inválido y válida; historia vacía y no numérica; horas con coma y con punto, negativas, cero, vacías, con letras y en formato científico; actividad con espacios; valores devueltos tal como llegaron; un error por campo al combinar los propios con los de HU-03, en `src/web/effort_form_test.go`
- [X] T009 [P] Escribir la prueba del adaptador sobre el servicio real de HU-03: `Validate` devuelve todas las violaciones juntas con su campo y el mensaje exacto de HU-03, `Register` registra una carga válida y devuelve los errores de HU-03 (integrante faltante, historia fuera del Sprint), en `src/web/effort_actions_test.go`
- [X] T010 Si T001 muestra que HU-03 no ofrece una validación que devuelva todas las violaciones, escribir su prueba y agregar la función en el dominio de HU-03 junto a sus reglas, sin cambiar el comportamiento del registro; si ya existe, escribir solo la prueba de T009 contra ella y avisar a quien mantiene HU-03 — **No hizo falta código en HU-03**: `effort.ValidateInput` ya devuelve todas las violaciones; se agregó solo la prueba (`effort_actions_test.go`)
- [X] T011 Implementar en `src/web/effort_form.go` los nombres de los campos (`historia`, `fecha`, `actividad`, `horas`), los textos propios en un solo lugar (`MsgEffortRecorded`, `MsgEffortSelectStory`, `MsgEffortStoryNotInSprint`, `MsgEffortInvalidDate`, `MsgEffortNoSprint`, `MsgEffortNoEntries`), el armado del envío y la unión de errores con un error por campo
- [X] T012 Implementar en `src/web/effort_actions.go` la interfaz `EffortActions` y su adaptador sobre el servicio de HU-03 (`Validate` y `Register`), usando los errores y mensajes de HU-03 sin copiarlos

**Checkpoint**: El catálogo PostgreSQL, el formulario y el adaptador están listos y probados contra el servicio y la base reales.

---

## Phase 3: User Story 1 - Registrar horas desde la web (Priority: P1) 🎯 MVP

**Goal**: Elegir una historia del Sprint activo y registrar horas a nombre del integrante identificado.

**Independent Test**: Con un Sprint activo, historias asignadas y un integrante identificado, cargar horas y verificar mensaje, registro y acumulado; probar el selector, la fecha por defecto y los casos sin Sprint activo y sin integrante.

### Tests for User Story 1

- [X] T013 [P] [US1] Escribir la prueba de handler de carga exitosa con `cookiejar` y proyecto e integrante elegidos por el flujo de HU-20b: 303 a `/esfuerzo?historia=ID`, "Esfuerzo registrado correctamente" una sola vez, registro con el integrante identificado, fecha, actividad y horas, acumulado actualizado y un campo de integrante enviado ignorado, en `src/web/effort_handlers_test.go`
- [X] T014 [P] [US1] Escribir la prueba de handler de la fecha de hoy por defecto con un reloj fijo y de las horas con coma y con punto, en `src/web/effort_handlers_test.go`
- [X] T015 [P] [US1] Escribir la prueba de handler del selector y la lista: solo historias del Sprint activo del proyecto (sin "Pendiente", sin Sprint finalizado, sin otro proyecto), con estado y horas acumuladas, en `src/web/effort_handlers_test.go`
- [X] T016 [P] [US1] Escribir la prueba de handler sin Sprint activo: `GET` con "No hay un Sprint activo" y sin formulario, y `POST` con 409 y nada registrado, en `src/web/effort_handlers_test.go`
- [X] T017 [P] [US1] Escribir la prueba de handler sin integrante identificado: redirección de HU-20b, regreso a `/esfuerzo` tras identificarse y nada registrado; un `POST` directo sin integrante lo rechaza HU-03 sin registrar, en `src/web/effort_handlers_test.go`
- [X] T018 [P] [US1] Escribir los escenarios BDD de carga exitosa, historias disponibles, sin Sprint activo y sin integrante identificado en `test/web_effort_bdd_test.go`

### Implementation for User Story 1

- [X] T019 [US1] Implementar el handler `GET /esfuerzo` en `src/web/effort_handlers.go`: lee las historias del Sprint activo del proyecto, arma la vista con la fecha de hoy y la historia elegida y usa `Fail` si falla la lectura
- [X] T020 [US1] Implementar el camino exitoso de `POST /esfuerzo` en `src/web/effort_handlers.go`: arma el envío, llama a HU-03 con el proyecto y el integrante del contexto, usa `SetFlash` y redirige con 303 a `/esfuerzo?historia=ID`
- [X] T021 [US1] Crear `effort.html` con la lista de historias y sus horas acumuladas (con enlace por historia), el formulario (historia, fecha, actividad, horas) y el estado "No hay un Sprint activo", usando los parciales `alert.html` y `field.html`, en `src/web/templates/effort.html`
- [X] T022 [US1] Construir el catálogo y el adaptador de HU-03 y registrar la sección "Registro de esfuerzo" (`/esfuerzo`, `RequiresProject: true`, `RequiresMember: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 registra esfuerzo desde la web; HU-07 y HU-25 ya tienen horas reales cargadas desde el navegador.

---

## Phase 4: User Story 2 - Errores y rechazos (Priority: P1)

**Goal**: Mostrar todos los errores juntos con los mensajes de HU-03 y rechazar cargas inválidas sin perder los datos.

**Independent Test**: Envíos con varios campos inválidos, con historias ajenas al Sprint activo y con una falla de guardado.

### Tests for User Story 2

- [X] T023 [P] [US2] Escribir la prueba de handler de errores juntos: historia vacía, fecha vacía, actividad vacía y horas en cero a la vez, con los cuatro mensajes (los de HU-03 tomados de sus constantes) junto a su campo, datos conservados, 422 y nada registrado, en `src/web/effort_handlers_test.go`
- [X] T024 [P] [US2] Escribir la prueba de handler de horas inválidas (negativas, cero, mayores de 24, con letras, vacías) y de fecha con formato inválido, cada una con su mensaje junto al campo, en `src/web/effort_handlers_test.go`
- [X] T025 [P] [US2] Escribir la prueba de handler de historia no disponible (de otro proyecto, "Pendiente", de un Sprint finalizado, inexistente, no numérica): error junto a `historia`, datos conservados y nada registrado, en `src/web/effort_handlers_test.go`
- [X] T026 [P] [US2] Escribir la prueba de handler con un almacenamiento que falla al guardar: "La operación no pudo completarse", 500, datos conservados, sin éxito y nada registrado, en `src/web/effort_handlers_test.go`
- [X] T027 [P] [US2] Escribir la prueba de handler del Sprint cerrado entre abrir y enviar: HU-03 rechaza y el error aparece junto a `historia` sin registrar, en `src/web/effort_handlers_test.go`
- [X] T028 [P] [US2] Escribir los escenarios BDD de errores juntos, historia no disponible y falla de almacenamiento en `test/web_effort_bdd_test.go`

### Implementation for User Story 2

- [X] T029 [US2] Implementar en `src/web/effort_form.go` y `src/web/effort_actions.go` el mapeo de los errores de HU-03 a campos y mensajes, con `MsgOperationFailed` para cualquier otro
- [X] T030 [US2] Implementar en `src/web/effort_handlers.go` las respuestas 422, 409 y 500 con la pantalla, el formulario con los datos conservados y los errores junto a cada campo
- [X] T031 [US2] Verificar en `src/web/effort_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 evita cargas inválidas, muestra todos los errores y mantiene los datos.

---

## Phase 5: User Story 3 - Ver el historial de una historia (Priority: P2)

**Goal**: Mostrar quién cargó qué horas en una historia y el total.

**Independent Test**: Historia con registros de dos integrantes, historia sin registros y `historia` inválida en la dirección.

### Tests for User Story 3

- [X] T032 [P] [US3] Escribir la prueba de handler del historial: integrante, fecha, actividad y horas, más recientes primero, total igual a la suma y coincidente con la lista, solo del proyecto activo, con verificación de tiempo de respuesta, en `src/web/effort_handlers_test.go`
- [X] T033 [P] [US3] Escribir la prueba de handler de la historia sin registros: "Todavía no hay horas registradas en esta historia" y total en cero, en `src/web/effort_handlers_test.go`
- [X] T034 [P] [US3] Escribir la prueba de handler de un `historia` no numérico, inexistente o de otro proyecto en la dirección: se ignora y se muestra la pantalla sin historial, en `src/web/effort_handlers_test.go`
- [X] T035 [P] [US3] Escribir la prueba de handler de una actividad con `<script>`: se muestra como texto en el historial, en `src/web/effort_handlers_test.go`
- [X] T036 [P] [US3] Escribir los escenarios BDD de historial de la historia e historia sin registros en `test/web_effort_bdd_test.go`

### Implementation for User Story 3

- [X] T037 [US3] Completar `effort.html` con el historial (integrante, fecha, actividad, horas y total) y el mensaje de historia sin registros en `src/web/templates/effort.html`
- [X] T038 [US3] Usar `Fail` en `src/web/effort_handlers.go` cuando falla la lectura del historial y escribir su prueba con un catálogo que falla

**Checkpoint**: US3 muestra el historial y el acumulado de cada historia.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [X] T039 [P] Documentar en `specs/024-web-effort/quickstart.md` los pasos para probar a mano con un Sprint con historias y un integrante identificado, y la nota de puerto
- [X] T040 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/effort_handlers.go` y `src/web/effort_form.go`
- [X] T041 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local — `docker compose up -d postgres` en la notebook; hizo falta aplicar la migración `008` (FIX-HU-01) que faltaba en la base local
- [X] T042 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md` — Todos los paquetes en `ok` (`evidencia-pruebas.txt`)
- [X] T043 [P] Escribir la prueba de handler de recarga después de guardar: no se registra una segunda carga ni se repite el mensaje, en `src/web/effort_handlers_test.go`
- [X] T044 [P] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): redirección de HU-20b y no se registra nada, en `src/web/effort_handlers_test.go`
- [X] T045 [P] Escribir la prueba de handler con un catálogo que falla al leer las historias: página de error de HU-20a y no una pantalla incompleta, en `src/web/effort_handlers_test.go`
- [X] T046 Guardar en `specs/024-web-effort/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando — Generada desde el commit RED (`web` y `test` no compilan sin la implementación)
- [X] T047 Guardar en `specs/024-web-effort/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/024-web-effort/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en los paquetes de reglas y cálculos — `web` 91,5%, `effort` 93,3% (`cobertura.txt`)
- [X] T048 Coordinar con HU-07 y HU-25 que las horas se siguen leyendo de `effort_logs` sin cambios; con HU-23 que quitar una historia con horas registradas sigue bloqueado; con HU-14 que completar una historia sin horas ya se puede evitar cargando horas desde la web; con HU-15 que las historias devueltas dejan de aparecer en el selector; y con quien mantiene HU-03 el cambio de T010 si hizo falta — Ver `coordinacion.md` (incluye el mensaje para el equipo)
- [X] T049 Revisar trazabilidad de FR-001 a FR-016 y SC-001 a SC-006 en `specs/024-web-effort/tasks.md` y la tabla de trazabilidad de `spec.md`
- [X] T050 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, verificar que el pipeline de GitHub Actions queda en verde y actualizar el manual breve de usuario con la pantalla "Registro de esfuerzo" — `go vet` limpio; `gofmt -l` sin diferencias en los archivos de HU-24 (23 archivos de HU-20b ya traían diferencias y esta historia no los modifica); el repositorio no tiene pipeline de GitHub Actions; sección "Registro de esfuerzo" agregada a `docs/manual-usuario.md` al integrar `main` (HU-21)
- [ ] T051 Verificar a mano con la aplicación en ejecución los 9 escenarios del Issue y la lista de `quickstart.md` — Lo verifica Alejandro en el navegador
- [ ] T052 Abrir el Pull Request desde `HU-24`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T001; T005, T006, T008 y T009 en paralelo; T010 depende de T001; T007 después de T005 y T006; T011 después de T008; T012 después de T009 y T010.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T020; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T019 y T021; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen; T043 a T045 se escriben junto a las pruebas de las historias.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-03, HU-04, FIX-HU-04, HU-20a y HU-20b.
- **US2 (P1)**: Depende del handler de US1; se verifica con envíos inválidos de forma independiente.
- **US3 (P2)**: Depende de la lista de US1; se puede dejar fuera sin afectar US1 ni US2.

### Parallel Opportunities

- T003 y T004; T005, T006, T008 y T009.
- T013 a T018, T023 a T028 y T032 a T036 pueden escribirse en paralelo antes de implementar.
- T039, T040, T043 a T045 y T050 pueden ejecutarse en paralelo después de la implementación.

## Implementation Strategy

### MVP First (User Story 1 y User Story 2)

1. Completar Setup y Foundational; guardar `evidencia-red.txt` (T046) con las pruebas fallando.
2. Implementar US1 y US2; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
3. Verificar a mano una carga válida y una con errores antes de abrir el Pull Request.

### Incremental Delivery

1. Entregar US1 como MVP funcional: registra horas desde la web.
2. Agregar US2 para mostrar todos los errores juntos y rechazar historias ajenas.
3. Agregar US3 (P2) para ver el historial y el total.
4. Ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, integrantes, historias, Sprints y registros; no dependen de HU-21, HU-22 ni HU-23.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures borran primero las historias (los Sprints, sus asignaciones y los registros caen en cascada) y después el proyecto.
- Los mensajes de HU-03 se toman de sus constantes en las pruebas, no se reescriben, para que sigan siendo "los exactos".

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T017, T022 |
| FR-002 | T013, T015, T019 |
| FR-003 | T014 |
| FR-004 | T006, T015 |
| FR-005 | T013, T043 |
| FR-006 | T008, T009, T023, T024 |
| FR-007 | T025, T027 |
| FR-008 | T013, T017 |
| FR-009 | T006, T032 |
| FR-010 | T033 |
| FR-011 | T016 |
| FR-012 | T026 |
| FR-013 | T038, T045 |
| FR-014 | T031, T040 |
| FR-015 | T035 |
| FR-016 | T006, T044 |
| SC-001 | T013 |
| SC-002 | T023, T024 |
| SC-003 | T006, T015, T025 |
| SC-004 | T006, T032 |
| SC-005 | T026 |
| SC-006 | Verificación de tiempo en T032 |