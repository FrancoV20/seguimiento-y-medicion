# Tasks: Interfaz web de proyectos

**Input**: Design documents from `specs/021-web-projects/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-01, FIX-HU-01, HU-20a y HU-20b (US1 y US2) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. `db/seeds/members.sql` no es una migración y no consume número. Si el equipo agregara alguna, toma el último número de `main` más uno (hoy la próxima es la 008), sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Modificar proyectos (HU-11), el estado (HU-12), la selección del proyecto activo (HU-20b) y el alta de integrantes quedan fuera. Las tareas marcadas `[Extra]` no son criterios de aceptación; se hacen solo si sobra tiempo.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza de HU-01, HU-20a y HU-20b, y preparar los integrantes de prueba.

- [x] T001 Revisar HU-01: ruta del paquete, firmas de `ProjectService.CreateProject` y `ProjectStore`, tipos de error y mensajes exactos de validación, normalización del nombre, si limita el largo del nombre y texto del mensaje de éxito; confirmar que FIX-HU-01 está en `main` (el servicio rechaza fin igual o anterior al inicio y la migración nueva está aplicada) y tomar el texto de su mensaje de fechas; anotar el resultado en `specs/021-web-projects/data-model.md` y ajustar el mapeo de `plan.md`
- [x] T002 Hacer `git merge origin/main` y verificar que FIX-HU-01, HU-20a y HU-20b (US1 y US2) están integradas: `Server.Register`, `Renderer.Page`, `Fail`, `RenderError`, `SetFlash`, `FormView`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303 y protección contra envíos de otro origen
- [x] T003 [P] Crear `db/seeds/members.sql` con los cinco integrantes del equipo, sin duplicar al ejecutarlo más de una vez
- [x] T004 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/021-web-projects/quickstart.md` `DATABASE_URL` (con la nota del puerto 5432 o 5433) y la carga de la semilla

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tipos, catálogo de solo lectura y formulario compartidos por las historias.

- [x] T005 [P] Definir `ProjectRow`, `MemberOption` y la interfaz `ProjectCatalog` con `ListProjects` y `ListMembers` en `src/web/projects_catalog.go`
- [x] T006 [P] Escribir la prueba de integración de `PostgreSQLProjectCatalog` con fixtures propias de dos proyectos con distinta cantidad de integrantes: cantidad correcta, orden descendente por `id`, proyecto sin integrantes en cero, catálogo ordenado por nombre y lista vacía sin error, en `src/web/postgres_project_catalog_integration_test.go`
- [ ] T007 Implementar `PostgreSQLProjectCatalog` con `database/sql` y pgx en `src/web/postgres_project_catalog.go`
- [x] T008 [P] Definir los nombres de los campos del formulario (`nombre`, `fecha_inicio`, `fecha_fin`, `integrantes`) y los textos `MsgNoMembersAvailable` ("No hay integrantes disponibles"), `MsgMissingStartDate` ("Falta la fecha de inicio"), `MsgMissingEndDate` ("Falta la fecha de fin"), `MsgInvalidDateFormat` ("La fecha no tiene un formato válido"), `MsgInvalidMember` ("El integrante seleccionado no es válido") y `MsgProjectCreated` ("Proyecto creado exitosamente", texto real de HU-01) en `src/web/projects_form.go`

**Checkpoint**: El catálogo PostgreSQL, los tipos y los textos están listos para las historias.

---

## Phase 3: User Story 1 - Crear un proyecto desde el formulario web (Priority: P1) 🎯 MVP

**Goal**: Completar el formulario, crear el proyecto con las reglas de HU-01 y verlo en la lista con el mensaje de éxito.

**Independent Test**: Con integrantes cargados, crear un proyecto válido y verlo en la lista; repetir con fechas iguales (se rechaza), recargando y sin integrantes disponibles.

### Tests for User Story 1

- [ ] T009 [P] [US1] Escribir pruebas unitarias del parseo del formulario (fechas válidas, formato inválido, campo ausente, identificadores repetidos, valores conservados) en `src/web/projects_form_test.go`
- [ ] T010 [P] [US1] Escribir la prueba de handler de alta exitosa: respuesta 303 a `/proyectos`, mensaje de éxito una sola vez, proyecto en la lista con sus datos y sin cambiar el proyecto activo, en `src/web/projects_handlers_test.go`
- [ ] T011 [P] [US1] Escribir la prueba de handler con la misma fecha de inicio y de fin: se rechaza con el mensaje de fechas de HU-01 (FIX-HU-01) junto a `fecha_fin`, datos conservados y ningún proyecto creado; y con la fecha de fin un día posterior: se crea, en `src/web/projects_handlers_test.go`
- [ ] T012 [P] [US1] Escribir la prueba de handler de recarga después de crear: no se crea un segundo proyecto ni se repite el mensaje, en `src/web/projects_handlers_test.go`
- [ ] T013 [P] [US1] Escribir la prueba de handler sin integrantes cargados: "No hay integrantes disponibles", y un envío directo no crea el proyecto, en `src/web/projects_handlers_test.go`
- [ ] T014 [P] [US1] Escribir los escenarios BDD de alta exitosa, fecha de fin igual a la de inicio (rechazo) y sin integrantes disponibles en `test/web_projects_bdd_test.go`

### Implementation for User Story 1

- [ ] T015 [US1] Implementar el parseo del formulario (recorte del nombre, fechas `AAAA-MM-DD`, identificadores sin repetidos, valores conservados) en `src/web/projects_form.go`
- [ ] T016 [US1] Implementar los handlers `GET /proyectos` y `POST /proyectos` en `src/web/projects_handlers.go`: cargan el catálogo, invocan `ProjectService.CreateProject` y, si tiene éxito, usan `SetFlash` y la redirección 303
- [ ] T017 [US1] Crear `projects.html` con el formulario (nombre, integrantes marcables, fechas) usando los parciales `alert.html` y `field.html`, y el aviso "No hay integrantes disponibles", en `src/web/templates/projects.html`
- [ ] T018 [US1] Construir el `ProjectService` de HU-01 y el `PostgreSQLProjectCatalog` y registrar la sección "Proyectos" (`/proyectos`, sin requisitos de contexto) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 crea proyectos desde la web con las reglas de HU-01; HU-22 a HU-25 ya tienen proyectos con los que trabajar.

---

## Phase 4: User Story 2 - Corregir errores sin perder los datos (Priority: P1)

**Goal**: Mostrar cada error junto a su campo, conservar los datos y no crear proyectos inválidos; informar las fallas de guardado.

**Independent Test**: Enviar cada dato inválido por separado y todos juntos, y simular una falla de guardado; verificar mensajes, datos conservados y que no se crea nada.

### Tests for User Story 2

- [ ] T019 [P] [US2] Escribir la prueba de handler, una por escenario (nombre vacío o solo espacios, fecha de fin anterior a la de inicio, sin integrantes marcados): mensajes de HU-01 junto a cada campo, datos conservados y ningún proyecto creado, en `src/web/projects_handlers_test.go`
- [ ] T020 [P] [US2] Escribir la prueba de handler con la fecha de inicio o de fin faltante: "Falta la fecha de inicio" o "Falta la fecha de fin" junto al campo, en `src/web/projects_handlers_test.go`
- [ ] T021 [P] [US2] Escribir la prueba de handler con una fecha de formato inválido: mensaje junto al campo y ningún proyecto creado, en `src/web/projects_handlers_test.go`
- [ ] T022 [P] [US2] Escribir la prueba de handler con un integrante no numérico o inexistente: mensaje junto al campo de integrantes y ningún proyecto creado, en `src/web/projects_handlers_test.go`
- [ ] T023 [P] [US2] Escribir la prueba de handler con un `ProjectStore` que falla al guardar: "La operación no pudo completarse", datos conservados, sin mensaje de éxito y el proyecto no aparece en la lista, en `src/web/projects_handlers_test.go`
- [ ] T024 [P] [US2] Escribir los escenarios BDD de nombre vacío, fecha de fin anterior, fecha faltante, sin integrantes marcados, fecha con formato inválido, integrante inválido y falla de guardado en `test/web_projects_bdd_test.go`

### Implementation for User Story 2

- [ ] T025 [US2] Implementar el mapeo de los errores de HU-01 a los campos del formulario, con `MsgOperationFailed` para cualquier otro, en `src/web/projects_form.go`
- [ ] T026 [US2] Implementar en `src/web/projects_handlers.go` la respuesta 422 para errores de validación y 500 para fallas de guardado, conservando los valores en `FormView`
- [ ] T027 [US2] [Extra] Rechazar un nombre de más de 255 caracteres con un mensaje junto al campo, si HU-01 no lo valida (según T001), en `src/web/projects_form.go` y su prueba en `src/web/projects_form_test.go`
- [ ] T028 [US2] Verificar en `src/web/projects_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 informa cada error junto a su campo, conserva los datos y no deja proyectos inválidos ni falsos éxitos.

---

## Phase 5: User Story 3 - Ver la lista de proyectos (Priority: P2)

**Goal**: Mostrar los proyectos con sus fechas y cantidad de integrantes, y un estado vacío con el formulario.

**Independent Test**: Con dos proyectos de distintas fechas e integrantes, verificar columnas y orden; sin proyectos, verificar el mensaje y el formulario.

### Tests for User Story 3

- [ ] T029 [P] [US3] Escribir la prueba de handler de la lista: nombre, fechas, cantidad de integrantes y más recientes primero, con verificación de tiempo de respuesta, en `src/web/projects_handlers_test.go`
- [ ] T030 [P] [US3] Escribir la prueba de handler sin proyectos: "Aún no hay proyectos" y el formulario, en `src/web/projects_handlers_test.go`
- [ ] T031 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no un formulario incompleto, en `src/web/projects_handlers_test.go`
- [ ] T032 [P] [US3] Escribir los escenarios BDD de lista y lista vacía en `test/web_projects_bdd_test.go`

### Implementation for User Story 3

- [ ] T033 [US3] Agregar a `projects.html` la tabla de proyectos (nombre, fechas `dd/mm/aaaa`, cantidad de integrantes) y el estado vacío "Aún no hay proyectos", en `src/web/templates/projects.html`
- [ ] T034 [US3] Usar `Fail` en `src/web/projects_handlers.go` cuando falla la lectura de proyectos o integrantes

**Checkpoint**: US3 muestra el resultado del alta y orienta al equipo, sin errores internos ante una falla de lectura.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, usabilidad, coordinación y trazabilidad.

- [ ] T035 [P] Documentar en `specs/021-web-projects/quickstart.md` la semilla, la nota de puerto y el protocolo de la prueba de usabilidad
- [ ] T036 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/projects_handlers.go` y `src/web/projects_form.go`
- [ ] T037 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T038 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T039 Guardar en `specs/021-web-projects/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T040 Guardar en `specs/021-web-projects/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/021-web-projects/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`
- [ ] T041 Hacer la prueba de usabilidad de SC-003 y SC-004 en la Sprint Review y registrar los resultados en `specs/021-web-projects/evidencia-usabilidad.txt`
- [ ] T042 Coordinar con el responsable de HU-20b que la sección "Proyectos" (`/proyectos`) queda registrada para que su aviso "acceso a crear el primero" apunte a esta pantalla, y con quien tome HU-11 que esta lista es el punto de partida de la edición
- [ ] T043 Revisar trazabilidad de FR-001 a FR-017 y SC-001 a SC-007 en `specs/021-web-projects/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T044 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, y verificar que el pipeline de GitHub Actions queda en verde
- [ ] T045 [P] Actualizar el manual breve de usuario con la pantalla "Gestión de Proyectos" (ver `quickstart.md`)
- [ ] T046 Verificar a mano con la aplicación en ejecución los 12 escenarios del Issue y la lista de `quickstart.md`
- [ ] T047 Abrir el Pull Request desde `HU-21`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 pueden ejecutarse en paralelo.
- **Foundational (Phase 2)**: Depende de T001; T005, T006 y T008 en paralelo; T007 requiere T005 y T006.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002 (incluida FIX-HU-01 en `main`); es el MVP.
- **User Story 2 (Phase 4)**: Depende de T015 y T016; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T016 y T017; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen; T041 requiere la pantalla terminada.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-01 (alta), HU-20a (servidor, layout, errores) y HU-20b US1 y US2 (mensajes, formularios y seguridad).
- **US2 (P1)**: Depende del handler de US1, pero se verifica con envíos inválidos de forma independiente.
- **US3 (P2)**: Depende del catálogo de Phase 2; es verificable con datos de prueba.
- **HU-22 a HU-25**: Necesitan al menos un proyecto, que esta historia permite crear desde la web.

### Parallel Opportunities

- T003 y T004; T005, T006 y T008.
- T009 a T014 pueden escribirse en paralelo antes de implementar US1.
- T019 a T024 pueden escribirse en paralelo con la implementación de US1.
- T029 a T032 pueden escribirse en paralelo con US2.
- T035 y T036 pueden ejecutarse en paralelo después de la implementación.
- Mientras FIX-HU-01, HU-20a y HU-20b no estén integradas, se pueden adelantar T003 a T008.

---

## Parallel Example: User Story 1

```text
T009: Pruebas del parseo del formulario en src/web/projects_form_test.go
T010: Alta exitosa en src/web/projects_handlers_test.go
T011: Fecha de fin igual a la de inicio (rechazo) en src/web/projects_handlers_test.go
T012: Recarga sin duplicar en src/web/projects_handlers_test.go
T013: Sin integrantes disponibles en src/web/projects_handlers_test.go
T014: Escenarios BDD en test/web_projects_bdd_test.go
```

## Parallel Example: User Story 2

```text
T019: Errores de validación de HU-01 en src/web/projects_handlers_test.go
T020: Fecha faltante en src/web/projects_handlers_test.go
T021: Formato de fecha inválido en src/web/projects_handlers_test.go
T022: Integrante inválido en src/web/projects_handlers_test.go
T023: Falla de guardado en src/web/projects_handlers_test.go
T024: Escenarios BDD en test/web_projects_bdd_test.go
```

## Parallel Example: User Story 3

```text
T029: Lista con datos y orden en src/web/projects_handlers_test.go
T030: Lista vacía en src/web/projects_handlers_test.go
T031: Falla de lectura en src/web/projects_handlers_test.go
T032: Escenarios BDD en test/web_projects_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 y User Story 2)

1. Completar Setup y Foundational; mientras HU-20a y HU-20b no estén integradas, adelantar T003 a T008.
2. Guardar `evidencia-red.txt` con las pruebas de US1 y US2 fallando.
3. Implementar US1 y US2; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
4. Verificar a mano el alta completa antes de avanzar a US3.

### Incremental Delivery

1. Entregar US1 como MVP funcional, porque habilita proyectos para HU-22 a HU-25.
2. Agregar US2 para corregir errores sin perder datos.
3. Agregar US3 para la lista y su estado vacío.
4. Hacer la prueba de usabilidad en la Sprint Review y ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios integrantes y proyectos; no dependen de `db/seeds/members.sql`.
- Los nombres y mensajes reales de HU-01 deben verificarse en su código (T001) antes de escribir T019 y T025.
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base local con `docker compose down -v` y volver a migrar (la semilla se vuelve a ejecutar).

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T010, T018 |
| FR-002 | T010, T017 |
| FR-003 | T006, T013 |
| FR-004 | T013 |
| FR-005 | T010, T014 |
| FR-006 | T019, T024 |
| FR-007 | T020, T024 |
| FR-008 | T009, T021, T024 |
| FR-009 | T022, T024 |
| FR-010 | T011, T014 |
| FR-011 | T006, T029 |
| FR-012 | T030 |
| FR-013 | T023 |
| FR-014 | T012 |
| FR-015 | T010 |
| FR-016 | T031 |
| FR-017 | T028, T036 |
| SC-001 | T010, T029 |
| SC-002 | T019, T020, T021, T022 |
| SC-003 | T041 (se verifica manualmente en la Sprint Review) |
| SC-004 | T041 (se verifica manualmente en la Sprint Review) |
| SC-005 | T023 |
| SC-006 | Verificación de tiempo en T029 |
| SC-007 | T012 |
