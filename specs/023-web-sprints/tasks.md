# Tasks: Interfaz web de Sprints

**Input**: Design documents from `specs/023-web-sprints/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b (completa) integradas en `main`;para la prueba manual, historias en el Backlog (por HU-22 o con SQL)

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno, sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Cerrar y editar Sprints (HU-15), Sprints anteriores (HU-16), cambio de estado de las historias (HU-14) y mover historias entre Sprints quedan fuera. El único cambio en HU-04 es la validación de quitar (T047).

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza de HU-04, HU-20a y HU-20b.

- [X] T001 Revisar HU-04 y FIX-HU-04 en `main`: ruta del paquete `src/sprint`, firmas del servicio de creación (entrada, resultado con advertencia y errores por historia), si el servicio expone `AssignStories` y `RemoveStory` o solo el store, `ErrActiveSprintExists` y su mensaje con el número del Sprint activo, tipos de error y mensajes exactos (Goal vacío, inconsistencia temporal, historia no disponible, proyecto inexistente), qué escribe al asignar y al quitar (`sprint_story_assignments`, `user_stories.active_sprint_id` y estados `'En Sprint'` y `'Pendiente'`), cómo trata los identificadores repetidos y si el dominio puede consultar horas registradas; anotar el resultado en `specs/023-web-sprints/data-model.md` y ajustar el mapeo de `plan.md`
- [X] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register` con varias rutas y métodos, `RequireContext` aplicado también a `POST` según `Section.RequiresProject`, `ActiveContextFrom`, `SetFlash`, `FormView`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303 y protección contra envíos de otro origen
- [X] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/023-web-sprints/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [X] T004 [P] Confirmar que las migraciones `003` (`effort_logs`: tabla y columna de la historia) y `004` (Sprints) en `main` coinciden con el esquema anotado en `specs/023-web-sprints/data-model.md` y avisar al equipo si cambiaron

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tipos, catálogo de solo lectura y textos compartidos por las historias.

- [X] T005 [P] Definir `SprintView`, `SprintStoryRow`, `StoryOption` y la interfaz `SprintCatalog` con `ActiveSprint` y `ListStoryOptions` en `src/web/sprints_catalog.go`
- [X] T006 [P] Escribir la prueba de integración de `PostgreSQLSprintCatalog` con fixtures propias (dos proyectos; historias pendientes, asignadas, en progreso y hechas; un Sprint activo con asignaciones y una historia con horas registradas; un Sprint finalizado): Sprint activo del proyecto correcto, ningún dato de otro proyecto, `hasEffort`, disponibilidad y motivo de cada historia y sin Sprint activo, en `src/web/postgres_sprint_catalog_integration_test.go`
- [X] T007 Implementar `PostgreSQLSprintCatalog` con `database/sql` y pgx en `src/web/postgres_sprint_catalog.go`
- [X] T008 [P] Definir los nombres de los campos (`objetivo`, `fecha_inicio`, `fecha_fin`, `historia`, `historia_id`), los textos propios en un solo lugar de `src/web/sprints_form.go` (`MsgSprintCreated`, `MsgSprintWithoutStories`, `MsgStoriesAdded`, `MsgStoryRemoved`, `MsgSelectAtLeastOneStory`, `MsgNoActiveSprint`, `MsgMissingStartDate`, `MsgMissingEndDate`, `MsgInvalidDateFormat`) y la interfaz `SprintActions`; el de Sprint activo existente y los de HU-04 se reutilizan

**Checkpoint**: El catálogo PostgreSQL, los tipos y los textos están listos para las historias.

---

## Phase 3: User Story 1 - Crear un Sprint con historias (Priority: P1) 🎯 MVP

**Goal**: Completar el formulario y crear el Sprint con las reglas de HU-04 en el proyecto activo; si ya hay un Sprint activo, mostrarlo y no ofrecer crear otro.

**Independent Test**: Con historias pendientes y sin Sprint activo, crear un Sprint con tres historias; repetir sin historias, con un Sprint activo existente y recargando.

### Tests for User Story 1

- [X] T009 [P] [US1] Escribir pruebas unitarias del armado del formulario (fechas `AAAA-MM-DD` válidas, formato inválido, campo ausente, identificadores repetidos quitados, no numéricos, campo de identificador de Sprint ignorado, valores conservados) en `src/web/sprints_form_test.go`
- [X] T010 [P] [US1] Escribir la prueba de handler de creación exitosa con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: 303 a `/sprints`, "Sprint creado correctamente" una sola vez, "Sprint 1" activo con Goal y fechas, historias en "En Sprint" y, con dos proyectos, solo en el activo, en `src/web/sprints_handlers_test.go`
- [X] T011 [P] [US1] Escribir la prueba de handler sin historias marcadas: el Sprint se crea con "Sprint creado correctamente. Todavía no tiene historias asignadas", en `src/web/sprints_handlers_test.go`
- [X] T012 [P] [US1] Escribir la prueba de handler con un Sprint activo existente: no se ofrece el formulario de creación, se informa "Ya hay un Sprint activo (Sprint N)" y un `POST /sprints` igualmente recibido responde 409 con el mismo mensaje, sin crear nada ni consumir un número de Sprint, en `src/web/sprints_handlers_test.go`
- [X] T013 [P] [US1] Escribir la prueba de handler de recarga después de crear: no se crea un segundo Sprint ni se repite el mensaje, en `src/web/sprints_handlers_test.go`
- [X] T014 [P] [US1] Escribir los escenarios BDD de crear con historias, Sprint sin historias y ya hay un Sprint activo en `test/web_sprints_bdd_test.go`

### Implementation for User Story 1

- [ ] T015 [US1] Implementar el armado del formulario (fechas, identificadores sin repetidos, valores conservados) en `src/web/sprints_form.go`
- [ ] T016 [US1] Implementar los handlers `GET /sprints` y `POST /sprints` en `src/web/sprints_handlers.go`: consultan el Sprint activo para decidir si ofrecen el formulario, invocan la creación de HU-04 con el proyecto de `ActiveContext` (la regla de un solo Sprint activo la aplica el dominio) y usan `SetFlash` (éxito o advertencia) y la redirección 303
- [ ] T017 [US1] Crear `sprints.html` con el formulario de creación (Sprint Goal, fechas y lista de historias con casillas, deshabilitadas con el motivo) y sin campo de identificador, usando los parciales `alert.html` y `field.html`, en `src/web/templates/sprints.html`
- [ ] T018 [US1] Construir el servicio de HU-04 y el `PostgreSQLSprintCatalog` y registrar la sección "Sprints" (`/sprints`, `RequiresProject: true`) con las rutas de la pantalla mediante `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 crea Sprints desde la web con las reglas de HU-04; HU-14, HU-24 y HU-25 ya tienen un Sprint activo con historias.

---

## Phase 4: User Story 2 - Elegir historias y corregir errores (Priority: P1)

**Goal**: Mostrar qué historias se pueden asignar y por qué otras no, mostrar cada error junto a su campo, conservar los datos y bloquear el guardado sin proyecto activo.

**Independent Test**: Lista con historias asignadas y no pendientes; envíos inválidos, con historia no disponible, con falla de guardado y sin proyecto activo.

### Tests for User Story 2

- [ ] T019 [P] [US2] Escribir la prueba de handler de la lista: las historias asignadas a un Sprint y las no pendientes aparecen deshabilitadas con el motivo ("Ya está asignada al Sprint N" o "No está pendiente"), en `src/web/sprints_handlers_test.go`
- [ ] T020 [P] [US2] Escribir la prueba de handler con una historia que dejó de estar disponible mientras se armaba el formulario: no se crea el Sprint, se informa cuál con el mensaje de HU-04, los datos y las marcas se conservan y no se consume un número, en `src/web/sprints_handlers_test.go`
- [ ] T021 [P] [US2] Escribir la prueba de handler con Goal vacío (también con solo espacios) y con fecha de fin anterior y, por separado, igual a la de inicio: mensajes de HU-04 junto a cada campo, datos y marcas conservados y ningún Sprint creado, en `src/web/sprints_handlers_test.go`
- [ ] T022 [P] [US2] Escribir la prueba de handler con cada fecha faltante ("Falta la fecha de inicio", "Falta la fecha de fin") y con una de formato inválido ("La fecha no tiene un formato válido"): mensaje junto al campo y ningún Sprint creado, en `src/web/sprints_handlers_test.go`
- [ ] T023 [P] [US2] Escribir la prueba de handler con un almacenamiento que falla al guardar: "La operación no pudo completarse", datos conservados, sin mensaje de éxito y sin Sprint ni asignaciones parciales, en `src/web/sprints_handlers_test.go`
- [ ] T024 [P] [US2] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): el guardado queda bloqueado, se redirige a la selección de proyecto con el mensaje de HU-20b, no se guarda nada, y un campo de proyecto o de Sprint enviado en el formulario se ignora, en `src/web/sprints_handlers_test.go`
- [ ] T025 [P] [US2] Escribir los escenarios BDD de historia no disponible (en la lista y al guardar), Goal vacío, fecha de fin anterior o igual, fecha faltante, formato de fecha inválido y falla de guardado en `test/web_sprints_bdd_test.go`

### Implementation for User Story 2

- [ ] T026 [US2] Implementar el mapeo de los errores de HU-04 y de `ErrActiveSprintExists` a los campos y mensajes del formulario, con `MsgOperationFailed` para cualquier otro, en `src/web/sprints_form.go`
- [ ] T027 [US2] Implementar en `src/web/sprints_handlers.go` las respuestas 422 para errores de validación, 409 para conflictos y 500 para fallas de guardado, conservando los valores y las marcas en `FormView`
- [ ] T028 [US2] Rechazar en `src/web/sprints_handlers.go` los identificadores de historias no numéricos o de otro proyecto con un mensaje junto a la lista, comprobando que pertenecen a las opciones del proyecto activo, con su prueba en `src/web/sprints_handlers_test.go`
- [ ] T029 [US2] Verificar en `src/web/sprints_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 informa cada error junto a su campo, conserva los datos y no deja Sprints inválidos, historias fuera de lugar ni falsos éxitos.

---

## Phase 5: User Story 3 - Ver el Sprint activo (Priority: P1)

**Goal**: Mostrar el Sprint activo con su identificador, Goal, fechas e historias con su estado, y un estado sin Sprint con el formulario.

**Independent Test**: Con un Sprint activo con historias en distintos estados, verificar los datos; sin Sprint activo, verificar el mensaje y el formulario; simular una falla de lectura.

### Tests for User Story 3

- [ ] T030 [P] [US3] Escribir la prueba de handler de la vista: identificador, Goal, fechas `dd/mm/aaaa` e historias con su estado, solo del Sprint activo del proyecto activo, con verificación de tiempo de respuesta, en `src/web/sprints_handlers_test.go`
- [ ] T031 [P] [US3] Escribir la prueba de handler sin Sprint activo: "No hay un Sprint activo" y el formulario de creación, en `src/web/sprints_handlers_test.go`
- [ ] T032 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no una pantalla incompleta, en `src/web/sprints_handlers_test.go`
- [ ] T033 [P] [US3] Escribir los escenarios BDD de vista del Sprint activo y sin Sprint activo en `test/web_sprints_bdd_test.go`

### Implementation for User Story 3

- [ ] T034 [US3] Agregar a `sprints.html` el bloque del Sprint activo (identificador, Goal, fechas y tabla de historias con su estado) y el estado "No hay un Sprint activo", en `src/web/templates/sprints.html`
- [ ] T035 [US3] Usar `Fail` en `src/web/sprints_handlers.go` cuando falla la lectura del Sprint o de las historias

**Checkpoint**: US3 muestra el resultado de la planificación y deja el punto de partida para agregar y quitar historias.

---

## Phase 6: User Story 4 - Agregar historias al Sprint activo (Priority: P2)

**Goal**: Completar la planificación de un Sprint ya creado agregando historias disponibles, todas o ninguna.

**Independent Test**: Con un Sprint activo y historias pendientes, agregar algunas; repetir sin marcar nada y con una historia que dejó de estar disponible.

### Tests for User Story 4

- [ ] T036 [P] [US4] Escribir la prueba de handler de agregar historias disponibles: quedan "En Sprint", aparecen en el Sprint y "Historias agregadas correctamente" se muestra una sola vez, en `src/web/sprints_handlers_test.go`
- [ ] T037 [P] [US4] Escribir la prueba de handler sin historias marcadas: "Elija al menos una historia" y sin cambios, en `src/web/sprints_handlers_test.go`
- [ ] T038 [P] [US4] Escribir la prueba de handler con una historia no disponible o de otro proyecto entre las marcadas: no se asigna ninguna, se informa cuál y la selección se conserva, y un identificador de Sprint enviado se ignora, en `src/web/sprints_handlers_test.go`
- [ ] T039 [P] [US4] Escribir los escenarios BDD de agregar historias y agregar sin marcar ninguna en `test/web_sprints_bdd_test.go`

### Implementation for User Story 4

- [ ] T040 [US4] Implementar `POST /sprints/historias` en `src/web/sprints_handlers.go`: toma el Sprint activo del servidor, quita identificadores repetidos, valida que pertenezcan a las opciones del proyecto, invoca `AssignStories` de HU-04 y usa `SetFlash` y la redirección 303
- [ ] T041 [US4] Agregar el formulario "Agregar historias" con la lista de historias disponibles al bloque del Sprint activo en `src/web/templates/sprints.html`
- [ ] T042 [US4] Mapear los errores de asignación de HU-04 a mensajes con las historias rechazadas en `src/web/sprints_form.go`

**Checkpoint**: US4 evita que un Sprint creado vacío o incompleto quede sin salida desde la web.

---

## Phase 7: User Story 5 - Quitar historias del Sprint activo (Priority: P2)

**Goal**: Corregir un error de planificación quitando una historia que no se empezó, sin dejar horas huérfanas, con la regla en el dominio.

**Independent Test**: Quitar una historia "En Sprint" sin horas; intentar quitar una historia en progreso, hecha o con horas, desde la pantalla y con un envío directo.

### Tests for User Story 5

- [ ] T043 [P] [US5] Escribir en `src/sprint` las pruebas de la regla de quitar: función pura `CanRemoveStory(status, hasEffort)` con su motivo ("Ya está en progreso", "Ya está hecha", "Tiene horas registradas"), y `RemoveStory` que rechaza con error tipado una historia en progreso, hecha, con horas, de otro Sprint o inexistente y acepta una "En Sprint" sin horas devolviéndola a "Pendiente" (incluye la prueba de integración contra PostgreSQL de la consulta de horas si T047 agrega una al store)
- [ ] T044 [P] [US5] Escribir la prueba de handler de quitar una historia "En Sprint" sin horas: "Historia quitada del Sprint", sale del Sprint, vuelve a "Pendiente" y vuelve a estar disponible en la lista, en `src/web/sprints_handlers_test.go`
- [ ] T045 [P] [US5] Escribir la prueba de handler con una historia en progreso, hecha o con horas registradas: "Quitar" deshabilitado con el motivo en la vista y un envío directo rechazado con el motivo y sin cambios; una historia de otro proyecto o Sprint rechazada, en `src/web/sprints_handlers_test.go`
- [ ] T046 [P] [US5] Escribir los escenarios BDD de quitar una historia, historia que no se puede quitar e intento de quitarla en `test/web_sprints_bdd_test.go`

### Implementation for User Story 5

- [ ] T047 [US5] Implementar en `src/sprint` la validación de `RemoveStory` (solo `'En Sprint'` y sin horas registradas, con errores tipados y la función pura `CanRemoveStory` exportada); si el dominio no puede consultar `effort_logs`, agregar la lectura al store de Sprints con su adaptador PostgreSQL y su prueba de integración; avisar a quien mantiene HU-04
- [ ] T048 [US5] Implementar `POST /sprints/historias/quitar` en `src/web/sprints_handlers.go`: verifica que la historia pertenece al Sprint activo del proyecto activo, invoca `RemoveStory` (que aplica la regla del dominio) y usa `SetFlash` y la redirección 303; un rechazo responde 409 con el motivo
- [ ] T049 [US5] Mostrar en `src/web/templates/sprints.html` un botón "Quitar" por historia, deshabilitado con el motivo cuando `CanRemoveStory` lo indique, y llenar `SprintStoryRow.canRemove` en `src/web/postgres_sprint_catalog.go` con esa misma función

**Checkpoint**: US5 permite corregir la planificación sin recrear el Sprint ni dejar horas huérfanas.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, usabilidad, coordinación y trazabilidad.

- [ ] T050 [P] Documentar en `specs/023-web-sprints/quickstart.md` los pasos para probar a mano con un proyecto activo con historias, el cierre manual de un Sprint de prueba y la nota de puerto
- [ ] T051 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/sprints_handlers.go` y `src/web/sprints_form.go`
- [ ] T052 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T053 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T054 Guardar en `specs/023-web-sprints/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T055 Guardar en `specs/023-web-sprints/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/023-web-sprints/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`
- [ ] T056 Hacer la prueba de usabilidad de SC-009 y SC-010 en la Sprint Review y registrar los resultados en `specs/023-web-sprints/evidencia-usabilidad.txt`
- [ ] T057 Coordinar con quien tome HU-14 que las historias asignadas quedan en `'En Sprint'` y la transición debe partir de ese estado y no de `'Pendiente'`; con quien tome HU-15 que el cierre debe eliminar la asignación de las historias no terminadas (la relación es única por historia) y que habilita crear el siguiente Sprint; con quien tome HU-24 que las historias del Sprint activo salen de `SprintCatalog`; y con quien mantiene HU-04 la validación de quitar (T047)
- [ ] T058 Revisar trazabilidad de FR-001 a FR-021 y SC-001 a SC-010 en `specs/023-web-sprints/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T059 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, y verificar que el pipeline de GitHub Actions queda en verde
- [ ] T060 [P] Actualizar el manual breve de usuario con la pantalla "Sprints" (ver `quickstart.md`)
- [ ] T061 Verificar a mano con la aplicación en ejecución los 17 escenarios del Issue y la lista de `quickstart.md`
- [ ] T062 Abrir el Pull Request desde `HU-23`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 pueden ejecutarse en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005, T006 y T008 en paralelo; T007 requiere T005 y T006.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T015 y T016; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T016 y T017; puede avanzar en paralelo con US2.
- **User Story 4 (Phase 6)**: Depende de T034 (bloque del Sprint activo) y de T016.
- **User Story 5 (Phase 7)**: Depende de T034; T047 (dominio) antes de T048.
- **Polish (Phase 8)**: Depende de las historias que se entreguen; T056 requiere la pantalla terminada.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-04 y FIX-HU-04 (creación y un solo Sprint activo), HU-20a (servidor, layout, errores) y HU-20b completa (mensajes, formularios, seguridad y contexto).
- **US2 (P1)**: Depende del handler de US1, pero se verifica con envíos inválidos de forma independiente.
- **US3 (P1)**: Depende del catálogo de Phase 2; es verificable con datos de prueba.
- **US4 (P2)**: Depende de la vista del Sprint activo de US3.
- **US5 (P2)**: Depende de la vista del Sprint activo de US3 y de HU-03 (horas registradas).
- **HU-14, HU-24 y HU-25**: Necesitan un Sprint activo con historias, que esta historia permite crear desde la web.

### Parallel Opportunities

- T003 y T004; T005, T006 y T008.
- T009 a T014 pueden escribirse en paralelo antes de implementar US1.
- T019 a T025 pueden escribirse en paralelo con la implementación de US1.
- T030 a T033 pueden escribirse en paralelo con US2.
- T036 a T039 y T043 a T046 pueden escribirse en paralelo con US3.
- T050, T051, T059 y T060 pueden ejecutarse en paralelo después de la implementación.
- Mientras HU-20a y HU-20b no estén integradas, se pueden adelantar T001 y T004 a T008.

---

## Parallel Example: User Story 1

```text
T009: Pruebas del armado del formulario en src/web/sprints_form_test.go
T010: Creación exitosa en src/web/sprints_handlers_test.go
T011: Sin historias en src/web/sprints_handlers_test.go
T012: Ya hay un Sprint activo en src/web/sprints_handlers_test.go
T013: Recarga sin duplicar en src/web/sprints_handlers_test.go
T014: Escenarios BDD en test/web_sprints_bdd_test.go
```

## Parallel Example: User Story 2

```text
T019: Historias deshabilitadas con motivo en src/web/sprints_handlers_test.go
T020: Historia tomada mientras tanto en src/web/sprints_handlers_test.go
T021: Goal vacío y fechas inconsistentes en src/web/sprints_handlers_test.go
T022: Fecha faltante o formato inválido en src/web/sprints_handlers_test.go
T023: Falla de guardado en src/web/sprints_handlers_test.go
T024: Sin proyecto activo en src/web/sprints_handlers_test.go
T025: Escenarios BDD en test/web_sprints_bdd_test.go
```

## Parallel Example: User Story 3

```text
T030: Vista del Sprint activo en src/web/sprints_handlers_test.go
T031: Sin Sprint activo en src/web/sprints_handlers_test.go
T032: Falla de lectura en src/web/sprints_handlers_test.go
T033: Escenarios BDD en test/web_sprints_bdd_test.go
```

## Parallel Example: User Story 4

```text
T036: Agregar historias disponibles en src/web/sprints_handlers_test.go
T037: Sin selección en src/web/sprints_handlers_test.go
T038: Historia no disponible al agregar en src/web/sprints_handlers_test.go
T039: Escenarios BDD en test/web_sprints_bdd_test.go
```

## Parallel Example: User Story 5

```text
T043: Regla de quitar en src/sprint
T044: Quitar exitoso en src/web/sprints_handlers_test.go
T045: Historia no removible en src/web/sprints_handlers_test.go
T046: Escenarios BDD en test/web_sprints_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1, User Story 2 y User Story 3)

1. Completar Setup y Foundational; mientras HU-20a y HU-20b no estén integradas, adelantar T001 y T004 a T008.
2. Guardar `evidencia-red.txt` con las pruebas de US1 a US3 fallando.
3. Implementar US1 a US3; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
4. Verificar a mano la creación completa con un proyecto activo antes de avanzar a US4.

### Incremental Delivery

1. Entregar US1 como MVP funcional, porque habilita un Sprint activo para HU-14, HU-24 y HU-25.
2. Agregar US2 para corregir errores y bloquear historias no disponibles.
3. Agregar US3 para ver el Sprint activo.
4. Agregar US4 para completar un Sprint ya creado.
5. Agregar US5 (quitar historias) con su regla en el dominio.
6. Hacer la prueba de usabilidad en la Sprint Review y ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, historias, Sprints y horas; no dependen de HU-21 ni de HU-22 integradas.
- Los nombres, firmas y mensajes reales de HU-04 y FIX-HU-04 deben verificarse en su código (T001) antes de escribir T020, T021 y T026.
- La regla de un solo Sprint activo no está en esta historia: la aplica HU-04 (FIX-HU-04). Para dejar un Sprint finalizado en las pruebas, cambiar su estado directamente en la base (HU-15 todavía no existe).
- `user_stories.project_id` no tiene borrado en cascada: las fixtures y las pruebas de "proyecto eliminado" borran primero las historias (los Sprints y sus asignaciones caen en cascada) y después el proyecto.
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base local con `docker compose down -v` y volver a migrar.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T010, T018 |
| FR-002 | T010, T017 |
| FR-003 | T010, T014 |
| FR-004 | T011 |
| FR-005 | T012, T014 |
| FR-006 | T006, T019 |
| FR-007 | T020, T025 |
| FR-008 | T021, T025 |
| FR-009 | T009, T022, T025 |
| FR-010 | T023, T025 |
| FR-011 | T024 |
| FR-012 | T010, T024, T038 |
| FR-013 | T030 |
| FR-014 | T031 |
| FR-015 | T032 |
| FR-016 | T013 |
| FR-017 | T036, T039 |
| FR-018 | T037, T039 |
| FR-019 | T043, T044, T046 |
| FR-020 | T043, T045, T046 |
| FR-021 | T029, T051 |
| SC-001 | T010, T030 |
| SC-002 | T021, T022 |
| SC-003 | T012 |
| SC-004 | T019, T020, T038 |
| SC-005 | T023 |
| SC-006 | T043, T045 |
| SC-007 | T024 |
| SC-008 | Verificación de tiempo en T030 |
| SC-009 | T056 (se verifica manualmente en la Sprint Review) |
| SC-010 | T056 (se verifica manualmente en la Sprint Review) |