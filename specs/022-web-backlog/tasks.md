# Tasks: Interfaz web del Product Backlog

**Input**: Design documents from `specs/022-web-backlog/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-02, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno (hoy la próxima es la 008), sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Orden por prioridad, filtros y edición de historias (HU-13), asignación a Sprints (HU-23) y modificar las horas estimadas después del alta (HU-17) quedan fuera. Las tareas marcadas `[Extra]` no son criterios de aceptación; se hacen solo si sobra tiempo.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza de HU-02, HU-20a y HU-20b.

- [ ] T001 Revisar HU-02: ruta del paquete, firmas del servicio de alta y de `StoryStore`, tipos de error y mensajes exactos de título, descripción, prioridad, criterios, Story Points y horas estimadas, cómo devuelve la advertencia por falta de Story Points, estado inicial `'Pendiente'`, y si acepta y valida las horas estimadas opcionales (mayor que 0, coma decimal, rechazo de más de dos decimales en lugar de redondearlos en silencio); anotar el resultado en `specs/022-web-backlog/data-model.md` y ajustar el mapeo de `plan.md`
- [ ] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register`, `RequireContext` aplicado también a `POST` según `Section.RequiresProject`, `ActiveContextFrom`, `SetFlash`, `FormView`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303 y protección contra envíos de otro origen
- [ ] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/022-web-backlog/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [ ] T004 [P] Confirmar que `db/migrations/002_*.up.sql` en `main` coincide con el esquema anotado en `specs/022-web-backlog/data-model.md` (`user_stories` y `acceptance_criteria`, con prioridad `Alta`/`Media`/`Baja`, Story Points de la serie y `estimated_hours` opcional) y avisar al equipo si cambió

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tipos, catálogo de solo lectura y formulario compartidos por las historias.

- [ ] T005 [P] Definir `StoryRow` y la interfaz `StoryCatalog` con `ListStories(ctx, projectID)` en `src/web/backlog_catalog.go`
- [ ] T006 [P] Escribir la prueba de integración de `PostgreSQLStoryCatalog` con fixtures propias de dos proyectos con historias: solo las del proyecto pedido, orden descendente por `id`, Story Points ausentes y proyecto sin historias sin error, en `src/web/postgres_story_catalog_integration_test.go`
- [ ] T007 Implementar `PostgreSQLStoryCatalog` con `database/sql` y pgx en `src/web/postgres_story_catalog.go`
- [ ] T008 [P] Definir los nombres de los campos del formulario (`titulo`, `descripcion`, `prioridad`, `story_points`, `criterio`, `accion`) y los textos `MsgStoryCreated` y `MsgMissingStoryPoints` (si HU-02 no los define) en `src/web/backlog_form.go`

**Checkpoint**: El catálogo PostgreSQL, los tipos y los textos están listos para las historias.

---

## Phase 3: User Story 1 - Cargar una historia desde el formulario web (Priority: P1) 🎯 MVP

**Goal**: Completar el formulario, crear la historia con las reglas de HU-02 en el proyecto activo y verla en el Backlog con el mensaje de éxito.

**Independent Test**: Con un proyecto activo, guardar una historia válida y verla con estado "Pendiente"; agregar y quitar criterios, dejar criterios en blanco, guardar sin Story Points y recargar.

### Tests for User Story 1

- [ ] T009 [P] [US1] Escribir pruebas unitarias del armado del formulario (criterios en blanco ignorados, acción por defecto `guardar`, índice de `quitar_criterio:N` válido, fuera de rango y no numérico, no quitar el último campo, Story Points vacío, horas estimadas vacías o con valor, valores conservados) en `src/web/backlog_form_test.go`
- [ ] T010 [P] [US1] Escribir la prueba de handler de alta exitosa con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: respuesta 303 a `/backlog`, mensaje de éxito una sola vez, historia en la lista con estado "Pendiente" y, con dos proyectos, solo en el activo, en `src/web/backlog_handlers_test.go`
- [ ] T011 [P] [US1] Escribir la prueba de handler de criterios dinámicos: "Agregar criterio" muestra un campo más, conserva lo escrito y no crea nada; "Quitar" elimina ese campo y no permite quitar el último; y Enter dentro de un campo guarda y no agrega ni quita, en `src/web/backlog_handlers_test.go`
- [ ] T012 [P] [US1] Escribir la prueba de handler con criterios en blanco: se ignoran y la historia se guarda solo con los criterios con texto, en `src/web/backlog_handlers_test.go`
- [ ] T013 [P] [US1] Escribir la prueba de handler sin Story Points: la historia se crea, se muestra la advertencia y en la lista figura "Sin estimar", en `src/web/backlog_handlers_test.go`
- [ ] T013a [P] [US1] Escribir la prueba de handler con horas estimadas válidas (por ejemplo `8` y `1,5`): la historia se crea con esas horas y figuran en la lista; sin horas estimadas figura "Sin estimar", en `src/web/backlog_handlers_test.go`
- [ ] T014 [P] [US1] Escribir la prueba de handler de recarga después de guardar: no se crea una segunda historia ni se repite el mensaje, en `src/web/backlog_handlers_test.go`
- [ ] T015 [P] [US1] Escribir los escenarios BDD de alta de historia, criterios dinámicos, criterios en blanco, sin Story Points, horas estimadas y recarga en `test/web_backlog_bdd_test.go`

### Implementation for User Story 1

- [ ] T016 [US1] Implementar el armado del formulario (acciones, criterios, Story Points, horas estimadas, valores conservados) en `src/web/backlog_form.go`
- [ ] T017 [US1] Implementar los handlers `GET /backlog` y `POST /backlog` en `src/web/backlog_handlers.go`: cargan la lista, agregan o quitan criterios sin guardar y, con `guardar`, invocan el servicio de HU-02 con el proyecto de `ActiveContext`, descartando el campo de proyecto del formulario, y usan `SetFlash` (éxito o advertencia) y la redirección 303
- [ ] T018 [US1] Crear `backlog.html` con el formulario (título, descripción, prioridad, Story Points, horas estimadas, campos de criterio), con prioridad y Story Points como selectores, el botón "Guardar" primero y visualmente oculto para la tecla Enter, y los botones "Agregar criterio" y "Quitar", usando los parciales `alert.html` y `field.html`, en `src/web/templates/backlog.html`
- [ ] T019 [US1] Construir el servicio de HU-02 y el `PostgreSQLStoryCatalog` y registrar la sección "Backlog" (`/backlog`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 carga historias desde la web con las reglas de HU-02; HU-23 y HU-24 ya tienen historias con las que trabajar.

---

## Phase 4: User Story 2 - Corregir errores sin perder los datos (Priority: P1)

**Goal**: Mostrar cada error junto a su campo, conservar los datos, no crear historias inválidas y bloquear el guardado sin proyecto activo.

**Independent Test**: Enviar sin criterios, con cada campo inválido y con todos juntos; simular una falla de guardado; enviar sin proyecto activo.

### Tests for User Story 2

- [ ] T020 [P] [US2] Escribir la prueba de handler sin ningún criterio con texto: error junto al campo de criterios, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [ ] T021 [P] [US2] Escribir la prueba de handler con título o descripción vacíos y prioridad inválida: mensajes de HU-02 junto a cada campo, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [ ] T022 [P] [US2] Escribir la prueba de handler con Story Points inválidos: mensaje junto al campo y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [ ] T022a [P] [US2] Escribir la prueba de handler con horas estimadas inválidas (cero, negativas, no numéricas y más de dos decimales, por ejemplo `1,555`): mensaje junto al campo, datos conservados, ninguna historia creada y sin redondeo silencioso, en `src/web/backlog_handlers_test.go`
- [ ] T023 [P] [US2] Escribir la prueba de handler con un `StoryStore` que falla al guardar: "La operación no pudo completarse", datos conservados, sin mensaje de éxito y la historia no aparece en el Backlog, en `src/web/backlog_handlers_test.go`
- [ ] T024 [P] [US2] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): el guardado queda bloqueado, se redirige a la selección de proyecto con el mensaje de HU-20b, no se guarda nada y un campo de proyecto enviado en el formulario se ignora, en `src/web/backlog_handlers_test.go`
- [ ] T025 [P] [US2] Escribir los escenarios BDD de errores de validación, horas estimadas inválidas, falla de guardado y falta de proyecto activo en `test/web_backlog_bdd_test.go`

### Implementation for User Story 2

- [ ] T026 [US2] Implementar el mapeo de los errores de HU-02 a los campos del formulario, con `MsgOperationFailed` para cualquier otro, en `src/web/backlog_form.go`
- [ ] T027 [US2] Implementar en `src/web/backlog_handlers.go` la respuesta 422 para errores de validación y 500 para fallas de guardado, conservando los valores en `FormView`
- [ ] T028 [US2] Definir en `src/web/backlog_form.go` las opciones del formulario: prioridad (`Alta`, `Media`, `Baja`) y Story Points (1, 2, 3, 5, 8, 13, 21, 34, 55, 89) según la migración `002`, y escribir en `src/web/backlog_form_test.go` una prueba que verifique que HU-02 acepta cada opción y rechaza un valor fuera de la lista
- [ ] T029 [US2] Verificar en `src/web/backlog_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 informa cada error junto a su campo, conserva los datos y no deja historias inválidas, falsos éxitos ni guardados sin proyecto.

---

## Phase 5: User Story 3 - Ver el Backlog del proyecto activo (Priority: P2)

**Goal**: Mostrar las historias del proyecto activo con su estimación y estado, y un estado vacío con el formulario.

**Independent Test**: Con dos proyectos con historias distintas, verificar columnas, orden y aislamiento; sin historias, verificar el mensaje y el formulario; simular una falla de lectura.

### Tests for User Story 3

- [ ] T030 [P] [US3] Escribir la prueba de handler de la lista: título, prioridad, Story Points y horas estimadas (cada uno o "Sin estimar"), estado, más recientes primero y solo del proyecto activo, con verificación de tiempo de respuesta, en `src/web/backlog_handlers_test.go`
- [ ] T031 [P] [US3] Escribir la prueba de handler sin historias: "Aún no hay historias" y el formulario, en `src/web/backlog_handlers_test.go`
- [ ] T032 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no un formulario incompleto, en `src/web/backlog_handlers_test.go`
- [ ] T033 [P] [US3] Escribir los escenarios BDD de lista, Backlog vacío y falla de lectura en `test/web_backlog_bdd_test.go`

### Implementation for User Story 3

- [ ] T034 [US3] Agregar a `backlog.html` la tabla de historias (título, prioridad, Story Points y horas estimadas, cada uno o "Sin estimar", y estado) y el estado vacío "Aún no hay historias", en `src/web/templates/backlog.html`
- [ ] T035 [US3] Usar `Fail` en `src/web/backlog_handlers.go` cuando falla la lectura de las historias

**Checkpoint**: US3 muestra el resultado de la carga y orienta a quien planifica, sin errores internos ante una falla de lectura.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [ ] T036 [P] Documentar en `specs/022-web-backlog/quickstart.md` los pasos para probar a mano con un proyecto activo y la nota de puerto
- [ ] T037 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/backlog_handlers.go` y `src/web/backlog_form.go`
- [ ] T038 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T039 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T040 Guardar en `specs/022-web-backlog/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T041 Guardar en `specs/022-web-backlog/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/022-web-backlog/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`
- [ ] T042 Coordinar con quien tome HU-13 que esta lista y `StoryCatalog` son su punto de partida para el orden por prioridad, el filtro y la edición; con quien tome HU-23 que las historias creadas quedan en estado `'Pendiente'` y se leen del mismo catálogo; y con quien tome HU-17 que la regla de horas estimadas (mayor que 0, hasta dos decimales) sea una sola y la compartan HU-02, HU-17 y este formulario
- [ ] T043 Revisar trazabilidad de FR-001 a FR-019 y SC-001 a SC-007 en `specs/022-web-backlog/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 pueden ejecutarse en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005, T006 y T008 en paralelo; T007 requiere T005 y T006.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T016 y T017; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T017 y T018; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-02 (alta), HU-20a (servidor, layout, errores) y HU-20b completa (mensajes, formularios, seguridad y contexto).
- **US2 (P1)**: Depende del handler de US1, pero se verifica con envíos inválidos de forma independiente.
- **US3 (P2)**: Depende del catálogo de Phase 2; es verificable con datos de prueba.
- **HU-23 y HU-24**: Necesitan historias en el Backlog, que esta historia permite cargar desde la web.

### Parallel Opportunities

- T003 y T004; T005, T006 y T008.
- T009 a T015 pueden escribirse en paralelo antes de implementar US1.
- T020 a T025 pueden escribirse en paralelo con la implementación de US1.
- T030 a T033 pueden escribirse en paralelo con US2.
- T036 y T037 pueden ejecutarse en paralelo después de la implementación.
- Mientras HU-20a y HU-20b no estén integradas, se pueden adelantar T001, T004 a T008.

---

## Parallel Example: User Story 1

```text
T009: Pruebas del armado del formulario en src/web/backlog_form_test.go
T010: Alta exitosa en src/web/backlog_handlers_test.go
T011: Criterios dinámicos y tecla Enter en src/web/backlog_handlers_test.go
T012: Criterios en blanco en src/web/backlog_handlers_test.go
T013: Sin Story Points en src/web/backlog_handlers_test.go
T013a: Horas estimadas válidas en src/web/backlog_handlers_test.go
T014: Recarga sin duplicar en src/web/backlog_handlers_test.go
T015: Escenarios BDD en test/web_backlog_bdd_test.go
```

## Parallel Example: User Story 2

```text
T020: Sin criterios en src/web/backlog_handlers_test.go
T021: Título, descripción y prioridad inválidos en src/web/backlog_handlers_test.go
T022: Story Points inválidos en src/web/backlog_handlers_test.go
T022a: Horas estimadas inválidas en src/web/backlog_handlers_test.go
T023: Falla de guardado en src/web/backlog_handlers_test.go
T024: Sin proyecto activo en src/web/backlog_handlers_test.go
T025: Escenarios BDD en test/web_backlog_bdd_test.go
```

## Parallel Example: User Story 3

```text
T030: Lista con datos, orden y aislamiento en src/web/backlog_handlers_test.go
T031: Backlog vacío en src/web/backlog_handlers_test.go
T032: Falla de lectura en src/web/backlog_handlers_test.go
T033: Escenarios BDD en test/web_backlog_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 y User Story 2)

1. Completar Setup y Foundational; mientras HU-20a y HU-20b no estén integradas, adelantar T001 y T004 a T008.
2. Guardar `evidencia-red.txt` con las pruebas de US1 y US2 fallando.
3. Implementar US1 y US2; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
4. Verificar a mano la carga completa con un proyecto activo antes de avanzar a US3.

### Incremental Delivery

1. Entregar US1 como MVP funcional, porque habilita historias para HU-23 y HU-24.
2. Agregar US2 para corregir errores sin perder datos y bloquear el guardado sin proyecto.
3. Agregar US3 para la lista y su estado vacío.
4. Ejecutar la validación completa del quickstart antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos e historias; no dependen de la semilla de HU-21 ni de HU-21 integrada.
- Los nombres, valores válidos y mensajes reales de HU-02 deben verificarse en su código (T001 y T004) antes de escribir T020 a T022a y T026.
- El proyecto activo en las pruebas se elige con el flujo real de HU-20b, no se inyecta a mano.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures y las pruebas de "proyecto eliminado" borran primero las historias (los criterios caen en cascada) y después el proyecto.
- `estimated_hours` es `NUMERIC(6,2)`: la base redondea sin error los valores con más de dos decimales, por eso la validación se prueba antes de guardar (T022a).
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base local con `docker compose down -v` y volver a migrar.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T010, T019 |
| FR-002 | T010, T013a, T018 |
| FR-003 | T010, T015 |
| FR-004 | T009, T011, T015 |
| FR-005 | T009, T012 |
| FR-006 | T020, T025 |
| FR-007 | T013 |
| FR-008 | T021, T025, T028 |
| FR-009 | T022, T028 |
| FR-010 | T023, T025 |
| FR-011 | T024, T025 |
| FR-012 | T010, T024 |
| FR-013 | T006, T013a, T030 |
| FR-014 | T031 |
| FR-015 | T032 |
| FR-016 | T014 |
| FR-017 | T029, T037 |
| FR-018 | T013a, T015 |
| FR-019 | T022a, T025 |
| SC-001 | T010, T030 |
| SC-002 | T020, T021, T022, T022a |
| SC-003 | T013 |
| SC-004 | T006, T010 |
| SC-005 | T023 |
| SC-006 | T024 |
| SC-007 | Verificación de tiempo en T030 |
