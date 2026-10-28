# Tasks: Registro y trazabilidad de defectos

**Input**: Design documents from `specs/006-defects/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-02, HU-04, FIX-HU-04, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Una migración nueva (`defects`). Toma el último número de `main` más uno, sin importar el número de la historia, con `.up.sql` y `.down.sql` probados en ambos sentidos. Nadie edita una migración ajena.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Resolver y reabrir defectos (HU-19), editar o borrar, registrar quién reporta y las métricas de defectos (HU-10) quedan fuera. US3 (lista, P2) se puede dejar fuera sin afectar US1 ni US2. No se modifican archivos de HU-02, HU-04, HU-15, HU-20a ni HU-20b.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza.

- [x] T001 Revisar en `main` HU-02, HU-04, FIX-HU-04 y HU-15: ruta de los paquetes de historias y Sprints, nombres de tablas y columnas (`sprints.status` con `'Activo'` y `'Finalizado'`, `user_stories.project_id`, `title`, `status`), tipo de los identificadores, que no existan paquetes ni nombres de defectos (`git grep -n -i defect`), cómo cierra HU-15 el Sprint (bloqueo de la fila) y los identificadores del paquete `web` para que los de prefijo `Defect` no choquen; anotar el resultado en `specs/006-defects/data-model.md` y ajustar `plan.md`
- [x] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register` con rutas `GET` y `POST`, `Section.RequiresProject`, `ActiveContextFrom`, `SetFlash`, `FormView`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303 y protección contra envíos de otro origen
- [x] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/006-defects/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [x] T004 [P] Confirmar el último número de migración en `main` y que nadie más haya tomado el siguiente (HU-15 usa `sprint_story_results`); avisar al equipo si cambió

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Migración, dominio, almacenamiento, catálogo y formulario compartidos por las tres historias.

- [x] T005 Escribir la migración `.up.sql` y `.down.sql` de `defects` con el siguiente número de `main` en `db/migrations/`, probarla en ambos sentidos (`up`, `down 1`, `up`) y comprobar que los `CHECK` rechazan severidad inválida, estado inválido, descripción vacía y un estado incoherente con el Sprint de resolución
- [x] T006 [P] Escribir las pruebas del servicio `Register` y de `Validate` en `src/defect/service_test.go`: descripción vacía, solo espacios, de 1000 y de 1001 caracteres (caracteres, no bytes), severidad vacía y fuera de la lista, historia vacía, todas las violaciones juntas y la descripción recortada
- [x] T007 [P] Escribir la prueba de integración de `PostgreSQLStore` con fixtures propias (dos proyectos; un Sprint activo y uno finalizado; historias "Pendiente", "En progreso" y "Hecho" de un Sprint finalizado): registro válido con estado "Abierto", Sprint de detección activo y sin Sprint de resolución; historia sin cambios; sin Sprint activo; historia de otro proyecto; Sprint activo del proyecto A no afecta al B; falla a mitad que revierte todo; y registro y cierre de Sprint simultáneos sin incoherencias, en `src/defect/postgres_store_integration_test.go`
- [x] T008 Definir en `src/defect` `Severity`, `Status`, `RegisterInput`, `Registered`, los errores del dominio y la interfaz `Store` en `model.go`, `errors.go` y `store.go`
- [x] T009 Implementar el servicio con `Validate` completo y `Register` en `src/defect/service.go`
- [x] T010 Implementar `PostgreSQLStore` con `database/sql` y pgx y la transacción de `data-model.md` (Sprint activo con `FOR SHARE`, historia del proyecto, alta) en `src/defect/postgres_store.go`
- [x] T011 [P] Definir `DefectRow`, `DefectStoryOption` y la interfaz `DefectCatalog` con `Defects`, `StoryOptions` y `ActiveSprintNumber` en `src/web/defects_catalog.go`
- [x] T012 [P] Escribir la prueba de integración de `PostgreSQLDefectCatalog` con fixtures propias (dos proyectos con defectos, historias y Sprints): lista del proyecto correcto, más recientes primero, historia y Sprints, "sin Sprint de resolución", ningún dato de otro proyecto, opciones de historia, número del Sprint activo y proyecto sin defectos, en `src/web/postgres_defects_catalog_integration_test.go`
- [x] T013 Implementar `PostgreSQLDefectCatalog` con `database/sql` y pgx y las consultas de `data-model.md` en `src/web/postgres_defects_catalog.go`
- [x] T014 [P] Escribir las pruebas unitarias del armado del formulario: campos ausentes como vacíos, valores devueltos tal como llegaron, `historia` vacía y no numérica, mapeo de cada error del dominio al campo y mensaje y un error por campo, en `src/web/defects_form_test.go`
- [x] T015 Implementar en `src/web/defects_form.go` los nombres de los campos (`descripcion`, `severidad`, `historia`), los textos propios en un solo lugar (`MsgDefectRegistered`, `MsgDefectDescriptionRequired`, `MsgDefectDescriptionTooLong`, `MsgDefectSeverityRequired`, `MsgDefectSeverityInvalid`, `MsgDefectStoryRequired`, `MsgDefectStoryNotInProject`, `MsgDefectNoSprint`, `MsgDefectNone`), el armado del envío y el mapeo de errores

**Checkpoint**: La migración, el dominio, el catálogo y el formulario están listos y probados contra PostgreSQL.

---

## Phase 3: User Story 1 - Registrar un defecto (Priority: P1) 🎯 MVP

**Goal**: Registrar un defecto con su historia y el Sprint activo desde la pantalla.

**Independent Test**: Con un Sprint activo y una historia, registrar un defecto y verificar mensaje, estado, Sprint de detección y que la historia no cambia; repetir con una historia "Hecho" de un Sprint anterior.

### Tests for User Story 1

- [x] T016 [P] [US1] Escribir la prueba de handler de registro exitoso con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: 303 a `/defectos`, "Defecto registrado correctamente. Sprint de detección: Sprint N" una sola vez, defecto "Abierto" con el Sprint activo, un campo de Sprint o de proyecto enviado ignorado y, con dos proyectos, solo registra en el activo, en `src/web/defects_handlers_test.go`
- [x] T017 [P] [US1] Escribir la prueba de handler de una historia "Hecho" de un Sprint finalizado y de una historia sin Sprint: el defecto queda "Abierto" con el Sprint activo y la historia no cambia de estado ni de asignación, en `src/web/defects_handlers_test.go`
- [x] T018 [P] [US1] Escribir la prueba de handler de las cuatro severidades (Crítica, Alta, Media y Baja) guardadas tal cual, en `src/web/defects_handlers_test.go`
- [x] T019 [P] [US1] Escribir los escenarios BDD de vinculación de defecto a historia e historia de un Sprint anterior en `test/web_defects_bdd_test.go`

### Implementation for User Story 1

- [x] T020 [US1] Implementar el handler `GET /defectos` en `src/web/defects_handlers.go`: lee los defectos, las historias y el Sprint activo del proyecto del contexto y usa `Fail` si falla la lectura
- [x] T021 [US1] Implementar el camino exitoso de `POST /defectos` en `src/web/defects_handlers.go`: arma el envío, llama al servicio con el proyecto del contexto, usa `SetFlash` con el número del Sprint y redirige con 303 a `/defectos`
- [x] T022 [US1] Crear `defects.html` con el formulario (descripción, severidad, historia) y la lista, usando los parciales `alert.html` y `field.html`, en `src/web/templates/defects.html`
- [x] T023 [US1] Construir el servicio, el almacenamiento y el catálogo y registrar la sección "Defectos" (`/defectos`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 registra defectos desde la web; HU-19, HU-10, HU-08 y HU-09 ya tienen datos de defectos.

---

## Phase 4: User Story 2 - Errores y rechazos (Priority: P1)

**Goal**: Mostrar todos los errores juntos y no guardar nada inválido.

**Independent Test**: Envíos con campos faltantes, con historia ajena, sin Sprint activo y con una falla de guardado.

### Tests for User Story 2

- [x] T024 [P] [US2] Escribir la prueba de handler de errores juntos: descripción vacía, severidad vacía y historia vacía a la vez, con los tres mensajes junto a su campo, datos conservados, 422 y nada guardado, en `src/web/defects_handlers_test.go`
- [x] T025 [P] [US2] Escribir la prueba de handler de descripción solo con espacios y de 1001 caracteres (y de 1000 aceptada), cada una con su mensaje junto al campo, en `src/web/defects_handlers_test.go`
- [x] T026 [P] [US2] Escribir la prueba de handler de historia de otro proyecto, inexistente y no numérica, y de severidad fuera de la lista: error junto al campo, datos conservados y nada guardado, en `src/web/defects_handlers_test.go`
- [x] T027 [P] [US2] Escribir la prueba de handler sin Sprint activo (sin Sprints y con todos finalizados): `GET` con "No hay un Sprint activo; no se puede registrar un defecto" y sin formulario, `POST` con 409 y nada guardado, en `src/web/defects_handlers_test.go`
- [x] T028 [P] [US2] Escribir la prueba de handler con un almacenamiento que falla al guardar: "La operación no pudo completarse", 500, datos conservados, sin éxito y nada guardado, en `src/web/defects_handlers_test.go`
- [x] T029 [P] [US2] Escribir los escenarios BDD de omisión de datos obligatorios, descripción obligatoria, sin Sprint activo, historia de otro proyecto y error al guardar en `test/web_defects_bdd_test.go`

### Implementation for User Story 2

- [ ] T030 [US2] Implementar en `src/web/defects_handlers.go` las respuestas 422, 409 y 500 con la pantalla, el formulario con los datos conservados y los errores junto a cada campo, ocultando el formulario sin Sprint activo
- [ ] T031 [US2] Verificar en `src/web/defects_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 evita defectos inválidos y mantiene los datos.

---

## Phase 5: User Story 3 - Ver los defectos del proyecto (Priority: P2)

**Goal**: Mostrar la lista de defectos del proyecto.

**Independent Test**: Defectos de dos proyectos y un proyecto sin defectos.

### Tests for User Story 3

- [ ] T032 [P] [US3] Escribir la prueba de handler de la lista: identificador, descripción, severidad, estado, historia, Sprint de detección y Sprint de resolución ("—"), más recientes primero, solo del proyecto activo, con verificación de tiempo de respuesta, en `src/web/defects_handlers_test.go`
- [ ] T033 [P] [US3] Escribir la prueba de handler de la lista vacía: "Aún no hay defectos" y el formulario, en `src/web/defects_handlers_test.go`
- [ ] T034 [P] [US3] Escribir la prueba de handler de una descripción y un título de historia con `<script>`: se muestran como texto, en `src/web/defects_handlers_test.go`
- [ ] T035 [P] [US3] Escribir los escenarios BDD de lista de defectos y lista vacía en `test/web_defects_bdd_test.go`
- [ ] T036 [US3] Completar `defects.html` con las columnas de la lista, el "—" del Sprint de resolución y el mensaje de lista vacía en `src/web/templates/defects.html`
- [ ] T037 [US3] Usar `Fail` en `src/web/defects_handlers.go` cuando falla la lectura de la lista o de las historias y escribir su prueba con un catálogo que falla

**Checkpoint**: US3 muestra los defectos y su trazabilidad.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [ ] T038 [P] Documentar en `specs/006-defects/quickstart.md` los pasos para probar a mano con un Sprint activo y la nota de puerto
- [ ] T039 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/defects_handlers.go` y `src/web/defects_form.go`
- [ ] T040 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T041 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [x] T042 [P] Escribir la prueba de handler de recarga después de registrar: no se crea un segundo defecto ni se repite el mensaje, en `src/web/defects_handlers_test.go`
- [x] T043 [P] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): redirección de HU-20b y no se guarda nada, en `src/web/defects_handlers_test.go`
- [x] T044 [P] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no una pantalla incompleta, en `src/web/defects_handlers_test.go`
- [x] T045 Guardar en `specs/006-defects/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T046 Guardar en `specs/006-defects/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/006-defects/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en `src/defect`
- [ ] T047 Coordinar con quien tome HU-19 que extiende `defects.html` y `defects_handlers.go` (acciones de resolver y reabrir) y completa `resolution_sprint_id` y el estado; con quien tome HU-10 que cuenta defectos por `detection_sprint_id` y `resolution_sprint_id`; con HU-08 y HU-09 que leen la misma tabla; con HU-15 que su cierre bloquea la fila del Sprint con `FOR UPDATE`; y el número de migración con quien tenga la de HU-15
- [ ] T048 Revisar trazabilidad de FR-001 a FR-016 y SC-001 a SC-006 en `specs/006-defects/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T049 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, verificar que el pipeline de GitHub Actions queda en verde y actualizar el manual breve de usuario con la pantalla "Defectos"
- [ ] T050 Verificar a mano con la aplicación en ejecución los 9 escenarios del Issue y la lista de `quickstart.md`
- [ ] T051 Abrir el Pull Request desde `HU-06`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005 junto a T006, T007, T011, T012 y T014; T008 antes de T009 y T010; T013 después de T011 y T012; T015 después de T014.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T021; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T020 y T022; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen; T042 a T044 se escriben junto a las pruebas de las historias.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-02, HU-04, FIX-HU-04, HU-20a y HU-20b.
- **US2 (P1)**: Depende del handler de US1; se verifica con envíos inválidos de forma independiente.
- **US3 (P2)**: Depende del handler y la plantilla de US1; se puede dejar fuera sin afectar US1 ni US2.
- **HU-19, HU-10, HU-08 y HU-09**: Necesitan la tabla `defects` y la pantalla de esta historia.

### Parallel Opportunities

- T003 y T004; T006, T007, T011, T012 y T014.
- T016 a T019, T024 a T029 y T032 a T035 pueden escribirse en paralelo antes de implementar.
- T038, T039, T042 a T044 y T049 pueden ejecutarse en paralelo después de la implementación.

## Implementation Strategy

### MVP First (User Story 1 y User Story 2)

1. Completar Setup y Foundational; guardar `evidencia-red.txt` (T045) con las pruebas fallando.
2. Implementar US1 y US2; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
3. Verificar a mano un registro válido y uno inválido antes de abrir el Pull Request.

### Incremental Delivery

1. Entregar US1 como MVP funcional: registra defectos con su Sprint de detección.
2. Agregar US2 para bloquear datos inválidos y la falta de Sprint activo.
3. Agregar US3 (P2) para ver la lista.
4. Ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, historias y Sprints; no dependen de HU-21, HU-22 ni HU-23.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures borran primero las historias (los defectos, los Sprints y sus asignaciones caen en cascada) y después el proyecto.
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base con `docker compose down -v` y volver a migrar. Si dos ramas crean el mismo número de migración, la segunda en integrarse renombra la suya.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T023, T043 |
| FR-002 | T016, T018 |
| FR-003 | T006, T007, T016, T042 |
| FR-004 | T007, T017 |
| FR-005 | T006, T014, T024 |
| FR-006 | T006, T025 |
| FR-007 | T007, T027 |
| FR-008 | T007, T026 |
| FR-009 | T012, T032 |
| FR-010 | T033 |
| FR-011 | T007, T028 |
| FR-012 | T037, T044 |
| FR-013 | T031, T039 |
| FR-014 | T034 |
| FR-015 | T016, T043 |
| FR-016 | T007 |
| SC-001 | T007, T016 |
| SC-002 | T024, T025, T026 |
| SC-003 | T007, T026, T027 |
| SC-004 | T007 |
| SC-005 | T012, T032 |
| SC-006 | Verificación de tiempo en T032 |