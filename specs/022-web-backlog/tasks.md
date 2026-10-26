# Tasks: Interfaz web del Product Backlog

**Input**: Design documents from `specs/022-web-backlog/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-02, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno, sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Orden por prioridad, filtros y edición de historias (HU-13), asignación a Sprints (HU-23) y modificar las horas estimadas después del alta (HU-17) quedan fuera. Las horas estimadas se validan con una sola regla en el dominio (T001 y T018), que HU-17 reutiliza.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza de HU-02, HU-20a y HU-20b.

- [ x ] T001 Revisar HU-02 en `main`: ruta del paquete, firmas del servicio de alta y de `StoryStore`, tipos de error y mensajes exactos de título, descripción, prioridad, criterios, Story Points y horas estimadas, cómo devuelve la advertencia por falta de Story Points, estado inicial `'Pendiente'`, y si el alta de HU-02 ya valida las horas estimadas más allá del `CHECK` de la base (mayor que 0): rechazo de más de dos decimales en lugar de redondeo, coma decimal y máximo de 9999,99. Lo que falte se agrega en el dominio, junto a la regla de HU-02 y no en el handler (T018); mirar también si HU-17 ya está en `main` para reutilizar su regla. Anotar el resultado en `specs/022-web-backlog/data-model.md` y ajustar el mapeo de `plan.md`
- [x ] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register`, `RequireContext` aplicado también a `POST` según `Section.RequiresProject`, `ActiveContextFrom`, `SetFlash`, `FormView`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303 y protección contra envíos de otro origen
- [ x] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/022-web-backlog/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [x ] T004 [P] Confirmar que `db/migrations/002_*.up.sql` en `main` coincide con el esquema anotado en `specs/022-web-backlog/data-model.md` (`user_stories` y `acceptance_criteria`, con prioridad `Alta`/`Media`/`Baja`, Story Points de la serie y `estimated_hours` `NUMERIC(6,2)` opcional), anotar qué migración posterior agregó el estado "En Sprint" y avisar al equipo si algo cambió

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tipos, catálogo de solo lectura y textos compartidos por las historias.

- [x] T005 [P] Definir `StoryRow` (con `id`) y la interfaz `StoryCatalog` con `ListStories(ctx, projectID)` en `src/web/backlog_catalog.go`
- [x] T006 [P] Escribir la prueba de integración de `PostgreSQLStoryCatalog` con fixtures propias de dos proyectos con historias: solo las del proyecto pedido, orden descendente por `id`, Story Points y horas ausentes y proyecto sin historias sin error, en `src/web/postgres_story_catalog_integration_test.go`
- [x] T007 Implementar `PostgreSQLStoryCatalog` con `database/sql` y pgx en `src/web/postgres_story_catalog.go`
- [x] T008 [P] Definir los nombres de los campos del formulario (`titulo`, `descripcion`, `prioridad`, `story_points`, `horas_estimadas`, `criterio`, `accion`) y los textos propios en un solo lugar de `src/web/backlog_form.go`: `MsgStoryCreated` ("Historia creada correctamente"), `MsgStoryCreatedMissingPoints` ("Historia creada correctamente. Falta completar la estimación de Story Points"), `MsgInvalidStoryPoints` ("Los Story Points no son un valor válido") y `MsgHoursTooLarge` ("Las horas estimadas no pueden superar 9999,99"); los de HU-02 se reutilizan si ya existen

**Checkpoint**: El catálogo PostgreSQL, los tipos y los textos están listos para las historias.

---

## Phase 3: User Story 1 - Cargar una historia desde el formulario web (Priority: P1) 🎯 MVP

**Goal**: Completar el formulario, crear la historia con las reglas de HU-02 en el proyecto activo y verla en el Backlog con el mensaje de éxito.

**Independent Test**: Con un proyecto activo, guardar una historia válida y verla con estado "Pendiente"; agregar y quitar criterios, dejar criterios en blanco, guardar sin Story Points y sin horas, ingresar horas con coma y recargar.

### Tests for User Story 1

- [ ] T009 [P] [US1] Escribir pruebas unitarias del armado del formulario (criterios en blanco ignorados, acción por defecto `guardar`, índice de `quitar_criterio:N` válido, fuera de rango y no numérico, no quitar el último campo, Story Points vacío, horas vacías, con coma, con punto y con valor no numérico, valores conservados) en `src/web/backlog_form_test.go`
- [ ] T010 [P] [US1] Escribir la prueba de handler de alta exitosa con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: respuesta 303 a `/backlog`, "Historia creada correctamente" una sola vez, historia en la lista con estado "Pendiente" y los valores ingresados (con Story Points y horas) y, con dos proyectos, solo en el activo, en `src/web/backlog_handlers_test.go`
- [ ] T011 [P] [US1] Escribir la prueba de handler de criterios dinámicos: "Agregar criterio" muestra un campo más, conserva lo escrito y no crea nada; "Quitar" elimina ese campo; con un solo campo, "Quitar" lo deja y no crea nada; y Enter dentro de un campo guarda y no agrega ni quita, en `src/web/backlog_handlers_test.go`
- [ ] T012 [P] [US1] Escribir la prueba de handler con criterios en blanco: se ignoran y la historia se guarda solo con los criterios con texto, en `src/web/backlog_handlers_test.go`
- [ ] T013 [P] [US1] Escribir la prueba de handler sin Story Points: la historia se crea, se muestra el mensaje combinado de advertencia y en la lista figura "Sin estimar" en Story Points, en `src/web/backlog_handlers_test.go`
- [ ] T014 [P] [US1] Escribir la prueba de handler sin horas estimadas: la historia se crea, se muestra "Historia creada correctamente" sin advertencia y en la lista figura "Sin estimar" en horas, en `src/web/backlog_handlers_test.go`
- [ ] T015 [P] [US1] Escribir la prueba de handler con horas estimadas válidas (`8`, `1,5`, `1.5` y `9999,99`): la historia se crea con esas horas exactas, sin redondeo, y la lista las muestra con coma, en `src/web/backlog_handlers_test.go`
- [ ] T016 [P] [US1] Escribir la prueba de handler de recarga después de guardar: no se crea una segunda historia ni se repite el mensaje, en `src/web/backlog_handlers_test.go`
- [ ] T017 [P] [US1] Escribir los escenarios BDD de alta de historia, agregar criterio, quitar criterio, no quitar el último, criterios en blanco, sin Story Points, sin horas estimadas y horas con coma decimal en `test/web_backlog_bdd_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Implementar el armado del formulario (acciones, criterios, Story Points, horas estimadas con coma o punto y sin separador de miles, valores conservados) en `src/web/backlog_form.go`; la validación de las horas se delega en el dominio (HU-02); si faltan el rechazo de más de dos decimales, la coma decimal o el máximo de 9999,99, se agregan ahí, junto a la regla existente, con su prueba unitaria, y se avisa a quien tome HU-17 para que los reutilice
- [x] T019 [US1] Implementar los handlers `GET /backlog` y `POST /backlog` en `src/web/backlog_handlers.go`: cargan la lista, agregan o quitan criterios sin guardar y, con `guardar`, invocan el servicio de HU-02 con el proyecto de `ActiveContext`, descartando el campo de proyecto del formulario, y usan `SetFlash` (éxito o advertencia) y la redirección 303
- [x] T020 [US1] Crear `backlog.html` con el formulario (título, descripción, prioridad, Story Points, horas estimadas, campos de criterio), con prioridad y Story Points como selectores, el botón "Guardar" primero y visualmente oculto para la tecla Enter, y los botones "Agregar criterio" y "Quitar", usando los parciales `alert.html` y `field.html`, en `src/web/templates/backlog.html`
- [x] T021 [US1] Construir el servicio de HU-02 y el `PostgreSQLStoryCatalog` y registrar la sección "Backlog" (`/backlog`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 carga historias desde la web con las reglas de HU-02; HU-23 y HU-24 ya tienen historias con las que trabajar.

---

## Phase 4: User Story 2 - Corregir errores sin perder los datos (Priority: P1)

**Goal**: Mostrar cada error junto a su campo, conservar los datos, no crear historias inválidas y bloquear el guardado sin proyecto activo.

**Independent Test**: Enviar sin criterios, con cada campo inválido y con todos juntos; simular una falla de guardado; enviar sin proyecto activo.

### Tests for User Story 2

- [x] T022 [P] [US2] Escribir la prueba de handler sin ningún criterio con texto: error de HU-02 junto al campo de criterios, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [x] T023 [P] [US2] Escribir la prueba de handler con título vacío y, por separado, con descripción vacía (también con solo espacios): mensajes de HU-02 junto a cada campo, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [x] T024 [P] [US2] Escribir la prueba de handler con una prioridad fuera de Alta, Media y Baja (envío manipulado): mensaje de HU-02 junto al campo, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [x] T025 [P] [US2] Escribir la prueba de handler con Story Points fuera de la lista (envío manipulado, por ejemplo `4`, `0`, `-1` y `abc`): mensaje junto al campo, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [x] T026 [P] [US2] Escribir la prueba de handler con horas estimadas inválidas (`0`, `-1`, `abc` y `1,555`): "El valor de horas no es válido" junto al campo, datos conservados, ninguna historia creada y sin redondeo silencioso, en `src/web/backlog_handlers_test.go`
- [x] T027 [P] [US2] Escribir la prueba de handler con horas estimadas por encima del máximo (`10000` y `9999,995`): "Las horas estimadas no pueden superar 9999,99" junto al campo, datos conservados y ninguna historia creada, en `src/web/backlog_handlers_test.go`
- [x] T028 [P] [US2] Escribir la prueba de handler con un `StoryStore` que falla al guardar: "La operación no pudo completarse", datos conservados, sin mensaje de éxito y la historia no aparece en el Backlog, en `src/web/backlog_handlers_test.go`
- [x] T029 [P] [US2] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): el guardado queda bloqueado, se redirige a la selección de proyecto con el mensaje de HU-20b, no se guarda nada y un campo de proyecto enviado en el formulario se ignora, en `src/web/backlog_handlers_test.go`
- [x] T030 [P] [US2] Escribir los escenarios BDD de sin criterios, título vacío, descripción vacía, prioridad inválida, Story Points inválidos, horas inválidas, horas por encima del máximo y falla de guardado en `test/web_backlog_bdd_test.go`

### Implementation for User Story 2

- [x] T031 [US2] Implementar el mapeo de los errores de HU-02 a los campos del formulario, con `MsgOperationFailed` para cualquier otro, en `src/web/backlog_form.go`
- [x] T032 [US2] Implementar en `src/web/backlog_handlers.go` la respuesta 422 para errores de validación y 500 para fallas de guardado, conservando los valores en `FormView`
- [x] T033 [US2] Definir en `src/web/backlog_form.go` las opciones del formulario: prioridad (`Alta`, `Media`, `Baja`) y Story Points (1, 2, 3, 5, 8, 13, 21, 34, 55, 89) según la migración `002`, y escribir en `src/web/backlog_form_test.go` una prueba que verifique que HU-02 acepta cada opción y rechaza un valor fuera de la lista
- [x] T034 [US2] Verificar en `src/web/backlog_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 informa cada error junto a su campo, conserva los datos y no deja historias inválidas, falsos éxitos ni guardados sin proyecto.

---

## Phase 5: User Story 3 - Ver el Backlog del proyecto activo (Priority: P2)

**Goal**: Mostrar las historias del proyecto activo con su identificador, estimación y estado, y un estado vacío con el formulario.

**Independent Test**: Con dos proyectos con historias distintas, verificar columnas y aislamiento; sin historias, verificar el mensaje y el formulario; simular una falla de lectura.

### Tests for User Story 3

- [ ] T035 [P] [US3] Escribir la prueba de handler de la lista: identificador, título, prioridad, Story Points y horas estimadas (cada uno o "Sin estimar", nunca cero), estado, solo las historias del proyecto activo y, como orden provisional, las más recientes primero (la reemplaza HU-13), con verificación de tiempo de respuesta, en `src/web/backlog_handlers_test.go`
- [ ] T036 [P] [US3] Escribir la prueba de handler sin historias: "Aún no hay historias" y el formulario, en `src/web/backlog_handlers_test.go`
- [ ] T037 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no un formulario incompleto, en `src/web/backlog_handlers_test.go`
- [ ] T038 [P] [US3] Escribir los escenarios BDD de lista y Backlog vacío en `test/web_backlog_bdd_test.go`

### Implementation for User Story 3

- [ ] T039 [US3] Agregar a `backlog.html` la tabla de historias (identificador, título, prioridad, Story Points y horas estimadas, cada uno o "Sin estimar", y estado) y el estado vacío "Aún no hay historias", en `src/web/templates/backlog.html`
- [ ] T040 [US3] Usar `Fail` en `src/web/backlog_handlers.go` cuando falla la lectura de las historias

**Checkpoint**: US3 muestra el resultado de la carga y orienta a quien planifica, sin errores internos ante una falla de lectura.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [ ] T041 [P] Documentar en `specs/022-web-backlog/quickstart.md` los pasos para probar a mano con un proyecto activo y la nota de puerto
- [ ] T042 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/backlog_handlers.go` y `src/web/backlog_form.go`
- [ ] T043 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T044 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T045 Guardar en `specs/022-web-backlog/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T046 Guardar en `specs/022-web-backlog/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/022-web-backlog/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`
- [ ] T047 Coordinar con quien tome HU-13 que esta lista y `StoryCatalog` son su punto de partida (orden por prioridad que reemplaza al provisional y a la prueba de orden de T035, filtro, edición y su escenario de Backlog vacío que referencia el de esta historia); con quien tome HU-23 que las historias creadas quedan en estado `'Pendiente'`, se leen del mismo catálogo y la lista muestra su identificador; y con quien tome HU-17 que la regla de horas (mayor que 0, hasta dos decimales, coma o punto, máximo 9999,99), su mensaje y el texto "Sin estimar" ya existen en el dominio y los reutilice en lugar de escribir otros
- [ ] T048 Revisar trazabilidad de FR-001 a FR-023 y SC-001 a SC-008 en `specs/022-web-backlog/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T049 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, y verificar que el pipeline de GitHub Actions queda en verde
- [ ] T050 [P] Actualizar el manual breve de usuario con la pantalla "Backlog" (ver `quickstart.md`)
- [ ] T051 Verificar a mano con la aplicación en ejecución los 18 escenarios del Issue y la lista de `quickstart.md`
- [ ] T052 Abrir el Pull Request desde `HU-22`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 pueden ejecutarse en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005, T006 y T008 en paralelo; T007 requiere T005 y T006.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T018 y T019; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T019 y T020; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-02 (alta, con las horas estimadas), HU-20a (servidor, layout, errores) y HU-20b completa (mensajes, formularios, seguridad y contexto).
- **US2 (P1)**: Depende del handler de US1, pero se verifica con envíos inválidos de forma independiente.
- **US3 (P2)**: Depende del catálogo de Phase 2; es verificable con datos de prueba.
- **HU-23 y HU-24**: Necesitan historias en el Backlog, que esta historia permite cargar desde la web.

### Parallel Opportunities

- T003 y T004; T005, T006 y T008.
- T009 a T017 pueden escribirse en paralelo antes de implementar US1.
- T022 a T030 pueden escribirse en paralelo con la implementación de US1.
- T035 a T038 pueden escribirse en paralelo con US2.
- T041, T042, T049 y T050 pueden ejecutarse en paralelo después de la implementación.
- Mientras HU-20a y HU-20b no estén integradas, se pueden adelantar T001, T004 a T008.

---

## Parallel Example: User Story 1

```text
T009: Pruebas del armado del formulario en src/web/backlog_form_test.go
T010: Alta exitosa en src/web/backlog_handlers_test.go
T011: Criterios dinámicos y tecla Enter en src/web/backlog_handlers_test.go
T012: Criterios en blanco en src/web/backlog_handlers_test.go
T013: Sin Story Points en src/web/backlog_handlers_test.go
T014: Sin horas estimadas en src/web/backlog_handlers_test.go
T015: Horas válidas (coma, punto y máximo) en src/web/backlog_handlers_test.go
T016: Recarga sin duplicar en src/web/backlog_handlers_test.go
T017: Escenarios BDD en test/web_backlog_bdd_test.go
```

## Parallel Example: User Story 2

```text
T022: Sin criterios en src/web/backlog_handlers_test.go
T023: Título y descripción vacíos en src/web/backlog_handlers_test.go
T024: Prioridad inválida en src/web/backlog_handlers_test.go
T025: Story Points inválidos en src/web/backlog_handlers_test.go
T026: Horas inválidas en src/web/backlog_handlers_test.go
T027: Horas por encima del máximo en src/web/backlog_handlers_test.go
T028: Falla de guardado en src/web/backlog_handlers_test.go
T029: Sin proyecto activo en src/web/backlog_handlers_test.go
T030: Escenarios BDD en test/web_backlog_bdd_test.go
```

## Parallel Example: User Story 3

```text
T035: Lista con datos y aislamiento en src/web/backlog_handlers_test.go
T036: Backlog vacío en src/web/backlog_handlers_test.go
T037: Falla de lectura en src/web/backlog_handlers_test.go
T038: Escenarios BDD en test/web_backlog_bdd_test.go
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
- Los nombres, valores válidos y mensajes reales de HU-02 deben verificarse en su código (T001 y T004) antes de escribir T022 a T027 y T031.
- El proyecto activo en las pruebas se elige con el flujo real de HU-20b, no se inyecta a mano.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures y las pruebas de "proyecto eliminado" borran primero las historias (los criterios caen en cascada) y después el proyecto.
- `estimated_hours` es `NUMERIC(6,2)`: la base redondea sin error los valores con más de dos decimales y falla con un error genérico por encima de 9999,99, por eso las dos reglas se prueban antes de guardar (T026 y T027).
- La prueba de orden de T035 es provisional: HU-13 la reemplaza cuando ordene por prioridad.
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base local con `docker compose down -v` y volver a migrar.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T010, T021 |
| FR-002 | T010, T020 |
| FR-003 | T010, T017 |
| FR-004 | T009, T011, T017 |
| FR-005 | T009, T011, T017 |
| FR-006 | T009, T012 |
| FR-007 | T022, T030 |
| FR-008 | T013 |
| FR-009 | T014 |
| FR-010 | T023, T030 |
| FR-011 | T024, T033 |
| FR-012 | T025, T033 |
| FR-013 | T026 |
| FR-014 | T027 |
| FR-015 | T009, T015 |
| FR-016 | T028, T030 |
| FR-017 | T029 |
| FR-018 | T010, T029 |
| FR-019 | T006, T035 |
| FR-020 | T036 |
| FR-021 | T037 |
| FR-022 | T016 |
| FR-023 | T034, T042 |
| SC-001 | T010, T035 |
| SC-002 | T022, T023, T024, T025, T026, T027 |
| SC-003 | T013, T014 |
| SC-004 | T006, T010 |
| SC-005 | T028 |
| SC-006 | T029 |
| SC-007 | Verificación de tiempo en T035 |
| SC-008 | T015 |
