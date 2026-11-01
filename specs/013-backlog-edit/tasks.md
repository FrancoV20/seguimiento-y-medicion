# Tareas: HU-13 – Consulta y Edición del Product Backlog (web)

**Entrada**: `specs/013-backlog-edit/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-02, HU-04, HU-20a, HU-20b y HU-22 integradas en `main`.
**Orden de implementación (decisión del equipo, 27/10)**: HU-13 se implementa **después** de HU-15 (cierre de Sprint) y
HU-17 (horas estimadas), porque edita los mismos archivos de `src/backlog` y de la pantalla del Backlog. Los md se
pueden subir antes; el código, no.

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1, 2a, 2b, 3, 4, 5, 6, 7a, 7b, 7c, 7d, 7e, 8a, 8b, 9, 10a, 10b, 10c,
  11a, 11b, 12a, 12b, 12c, 13a, 13b, 13c, 14 y 15).
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/013-backlog-edit/evidencia-red.txt`.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- CI (`.github/workflows/ci.yml`): el Pull Request tiene que pasar `gofmt -l`, `go vet` y `go test ./... -p 1 -race`.
  Antes de cada commit, correr `gofmt -l` sobre los archivos que se tocaron y `gofmt -w` solo sobre esos; no
  reformatear el resto del repositorio. La base ya está formateada (PR #44).

## Fase 1: Preparación

- [x] T001 Confirmar que HU-15 y HU-17 ya están en `main` (`git log --oneline main` y sus carpetas en `specs/`); si
  no, no empezar. Después crear la rama `HU-13` desde `main` actualizado (`git switch main; git pull; git switch -c
  HU-13`).
- [x] T002 Confirmar en el código los "Datos verificados" de `research.md`:
  - leer `src/backlog`, `src/web/backlog_*.go`, `src/web/postgres_story_catalog.go` y
    `src/web/templates/backlog.html`;
  - comprobar con `git grep -n` que no existen `UpdateStory`, `StoryForEdit`, `StoryEdit`, `ErrStoryNotFound` (en
    `src/backlog`), `ErrStoryInFinishedSprint`, `StoryStatuses`, `validateStoryFields`, `parseStoryFields` ni
    `backlogListURL`;
  - ajustar `plan.md` si algún nombre cambió.
- [x] T003 Coordinar los archivos ajenos que edita HU-13:
  - con quien hizo HU-22: `backlog_handlers.go`, `backlog_form.go`, `backlog_catalog.go`,
    `postgres_story_catalog.go`, `backlog.html`, las tres pruebas del orden y los fakes `hu22FailingBacklogStore`,
    `hu22FailingStoryCatalog` y `hu22BDDFailingBacklogStore`;
  - con quien hizo HU-02: `model.go`, `store.go`, `service.go` y `test_store.go`;
  - con quien tenga HU-15: que su cierre bloquee el Sprint antes que las historias, como dice su `data-model.md`.
- [x] T004 Correr `go test -count=1 -p 1 ./...` y confirmar que pasa en verde antes de empezar (línea base, incluidas
  las pruebas de HU-02 y HU-22).

## Fase 2: Fundamentos (dominio y almacenamiento; bloquea todo lo demás)

### Pruebas (RED)

Todas en `src/backlog/update_service_test.go`, salvo que se indique otra ruta.

- [x] T005 [P] `TestStoryStatuses`: devuelve "Pendiente", "En Sprint", "En progreso" y "Hecho", en ese orden.
- [x] T006 [P] [E10a, E10b, E10c, E11a, E11b] `TestValidateStoryFields`:
  - los errores aparecen en el orden título, descripción, prioridad, criterios y Story Points;
  - recorta título, descripción y criterios;
  - descarta los criterios vacíos y numera los demás desde 1;
  - acepta Story Points en `nil`.
- [x] T007 [P] [E10a, E10b, E10c, E11a, E11b] `TestUpdateStory_Validaciones`, con `InMemoryBacklogStore`:
  - `ProjectID` 0 o `StoryID` 0 → `ErrStoryNotFound`;
  - título vacío, descripción con solo espacios, prioridad "Urgente", criterios vacíos y 4 Story Points → el error
    de HU-02 que corresponde;
  - en ningún caso se llama al almacén (`UpdateCalls == 0`).
- [x] T008 [P] [E7a, E9, E15] `TestUpdateStory_DelegaYPropaga`:
  - un envío válido llega al almacén recortado y con los criterios numerados;
  - con Story Points en `nil`, `Warning == WarnEstimationPending`;
  - `UpdateErr = ErrStoryNotFound` y `UpdateErr = ErrStoryInFinishedSprint` se devuelven sin cambios;
  - `FailOnUpdate` → `ErrPersistence`.
- [x] T009 Preparar las fixtures de integración en `src/backlog/postgres_backlog_store_update_integration_test.go`:
  - proyectos «Alfa» y «Beta»;
  - «Login»: de «Alfa», Media, 5 Story Points, 4 horas estimadas, criterios «Acepta email» y «Rechaza clave
    vacía», "En Sprint" en el Sprint 2 activo;
  - «Pagos»: de «Alfa», "Hecho" en el Sprint 1 'Finalizado';
  - «Cierre»: de «Alfa», "Hecho" en el Sprint 2 activo;
  - «Facturas»: de «Beta»;
  - limpieza en este orden: asignaciones, criterios, historias, Sprints y proyectos.
- [x] T010 [P] [E7a, E7b, E7c, E7d] `TestPostgresUpdateStory_DatosYSinEfectos`: cambian el título, la
  descripción, la prioridad y los Story Points de «Login». No cambian `status`, `estimated_hours`,
  `active_sprint_id`, `created_at` ni la asignación. Es la primera prueba del adaptador.
- [x] T011 [P] [E8a, E8b] `TestPostgresUpdateStory_ReemplazaCriterios`:
  - de dos criterios a tres, reordenados → posiciones 1..3 con el contenido en el orden enviado;
  - de tres a uno → queda solo la posición 1.
- [x] T012 [P] [E9] `TestPostgresUpdateStory_SinStoryPoints`: `story_points` queda `NULL`.
- [x] T013 [P] [E13a, E13c] `TestPostgresUpdateStory_InexistenteYOtroProyecto`:
  - con el identificador 2147483647 → `ErrStoryNotFound`;
  - «Facturas» con el proyecto «Alfa» → `ErrStoryNotFound`, y «Facturas» no cambia.
- [x] T014 [P] [E12b] `TestPostgresUpdateStory_SprintFinalizado`: «Pagos» → `ErrStoryInFinishedSprint`; no cambian
  sus datos ni sus criterios.
- [x] T015 [P] [E7a] `TestPostgresUpdateStory_HechoEnSprintActivo`: «Cierre», "Hecho" en un Sprint activo, se
  guarda (R6).
- [x] T016 [P] [E15] `TestPostgresUpdateStory_FalloNoDejaCambios`: un criterio `"   "` enviado directo al almacén
  hace fallar el `INSERT` por el `CHECK` (paso 5), después del `UPDATE` y del `DELETE`. El título, los Story Points
  y los criterios originales siguen intactos.
- [x] T017 [P] `TestPostgresUpdateStory_SinCambios`: devuelve `nil` y deja los mismos datos.
- [x] T018 [P] `TestPostgresUpdateStory_ConcurrenteUltimaGana`: dos guardados simultáneos de «Login» con criterios
  distintos terminan sin error; el resultado es uno de los dos completo, nunca una mezcla de criterios.
- [x] T019 [P] [E12b] `TestPostgresUpdateStory_EsperaAlCierre`:
  1. Una transacción manual bloquea el Sprint 2 con `FOR UPDATE` y lo marca 'Finalizado'.
  2. `UpdateStory` de «Login» corre en una goroutine.
  3. Se espera a que quede bloqueado, consultando `pg_stat_activity` con `wait_event_type = 'Lock'`.
  4. Se confirma la transacción manual (`COMMIT`).
  5. `UpdateStory` devuelve `ErrStoryInFinishedSprint` y «Login» no cambia.
- [x] T020 [P] `TestPostgresUpdateStory_AsignacionCambiaReintenta`: con el gancho de prueba `beforeStoryLock`, la
  primera vez se asigna «Login» a otro Sprint activo entre el paso 1 y el 3. El guardado repite una vez y termina
  bien.
- [x] T021 [P] `TestPostgresUpdateStory_AsignacionCambiaDosVeces`: el gancho cambia la asignación en los dos
  intentos. Se devuelve un error que no es de dominio y «Login» no cambia.
- [x] T022 Ejecutar T005–T021, confirmar que fallan porque el código no existe y guardar la salida en
  `evidencia-red.txt` (sección "Fundamentos").

### Implementación (GREEN)

- [x] T023 Agregar a `src/backlog/model.go` `StatusInSprint = "En Sprint"`, `StatusInProgress = "En progreso"` y
  `StatusDone = "Hecho"`, más `StoryStatuses()` con los cuatro estados en el orden de T005.
- [x] T024 Crear `UpdateStoryRequest` y `UpdateStoryResult` en `src/backlog/update.go`, y `ErrStoryNotFound` y
  `ErrStoryInFinishedSprint` en `src/backlog/update_errors.go`.
- [x] T025 Extraer `validateStoryFields` de `CreateStory` en `src/backlog/service.go`, sin cambiar su
  comportamiento: las pruebas de HU-02 siguen en verde.
- [x] T026 Implementar `UpdateStory` en `src/backlog/update_service.go`, según los pasos de `data-model.md`.
- [x] T027 Ampliar el puerto y todos sus fakes en el mismo commit:
  - sumar `UpdateStory(ctx context.Context, projectID int64, story UserStory) error` a `BacklogStore`
    (`src/backlog/store.go`);
  - sumar a `src/backlog/test_store.go` `UpdateStory`, `UpdateErr`, `FailOnUpdate` y `UpdateCalls`;
  - sumar un `UpdateStory` que devuelve error a `hu22FailingBacklogStore` (`src/web/backlog_handlers_test.go`) y a
    `hu22BDDFailingBacklogStore` (`test/web_backlog_bdd_test.go`).
- [x] T028 Implementar `PostgreSQLBacklogStore.UpdateStory` en `src/backlog/postgres_backlog_store_update.go`, con
  los pasos de `data-model.md`:
  - Sprint con `FOR SHARE` primero;
  - historia con `FOR NO KEY UPDATE OF us`;
  - relectura de la asignación con un solo reintento;
  - `DELETE` e `INSERT` de los criterios 1..n.

  El campo sin exportar `beforeStoryLock func()` es `nil` en producción.
- [x] T029 Ejecutar T005–T021 en verde, junto con las pruebas existentes de `./src/backlog/...`.

**Punto de control**: el dominio y el almacenamiento de la edición funcionan sin tocar la pantalla.

## Fase 3: Bloque A – Listado y filtro (escenarios 1, 2a, 2b, 3, 4 y 5) 🎯 MVP

**Objetivo**: lista ordenada por prioridad y antigüedad, con filtro por estado y la marca de Sprint finalizado.

**Prueba independiente**:

1. Crear historias de distintas prioridades y verificar el orden.
2. Filtrar por cada estado y por "Todos".
3. Abrir un estado inválido.
4. Abrir un proyecto vacío.

### Pruebas (RED)

- [ ] T030 [P] [E1] Pruebas de integración del catálogo en `src/web/postgres_story_catalog_integration_test.go`:
  - renombrar `TestPostgreSQLStoryCatalogListsOnlyProjectStoriesInDescendingOrder` a
    `TestPostgreSQLStoryCatalogListsOnlyProjectStoriesInPriorityOrder`, con la expectativa de FR-001;
  - agregar `TestPostgreSQLStoryCatalog_OrdenPorPrioridadYAntiguedad`, con dos historias Media creadas en orden y
    empate de `created_at` resuelto por `id`;
  - agregar `TestPostgreSQLStoryCatalog_MarcaSprintFinalizado`, que solo marca la historia asignada a un Sprint
    'Finalizado'.
- [ ] T031 Crear el servidor de prueba `newHU13BacklogApp` en `src/web/backlog_list_handlers_test.go`:
  - reutiliza `newHU22BacklogAppWithDependencies`;
  - fixtures: «Login» (Media, primera), «Pagos» (Alta), «Reportes» (Baja) y «Perfil» (Media, después), con estados
    "Pendiente", "En Sprint" y "Hecho".
- [ ] T032 [P] [E1] `TestHU13_ListadoOrdenado`: la tabla muestra «Pagos», «Login», «Perfil», «Reportes».
- [ ] T033 [P] [E2a, E2b] `TestHU13_FiltroPorEstado`:
  - con `?estado=Pendiente` solo aparecen las "Pendiente", en el orden de FR-001, y la opción queda seleccionada;
  - con `?estado=Todos` y sin parámetro se ve la lista completa.
- [ ] T034 [P] [E3] `TestHU13_FiltroSinResultados`: con `?estado=Hecho` y ninguna historia "Hecho", la respuesta es
  200 con "No hay historias con ese estado", sin página de error.
- [ ] T035 [P] [E4] `TestHU13_FiltroEstadoInvalido`: `?estado=Archivada` y `?estado=pendiente` responden 200 con
  "El estado no es válido" junto al filtro y la lista completa. Con `?estado=Hecho&estado=Pendiente` cuenta el
  primero.
- [ ] T036 [P] [E5] `TestHU13_BacklogVacioConFiltro`: un proyecto sin historias, con y sin `?estado=Hecho`, muestra
  "Aún no hay historias" y el formulario de alta, y nunca "No hay historias con ese estado".
- [ ] T037 [E1] Actualizar dos pruebas de HU-22 a la expectativa de FR-001:
  - `TestBacklogHandler_GetListsActiveProjectStoriesNewestFirst` (`src/web/backlog_handlers_test.go`), renombrada a
    `…InPriorityOrder`;
  - el subtest "…recientes primero" de `TestWebBacklogBDD_BacklogListAndEmptyState`
    (`test/web_backlog_bdd_test.go`), que pasa a "…ordenadas por prioridad".
- [ ] T038 [P] [E1, E2a] Escribir los escenarios BDD "Listado ordenado" y "Filtro por estado" en
  `test/web_backlog_edit_bdd_test.go`.
- [ ] T039 Ejecutar T030–T038, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Listado y
  filtro").

### Implementación (GREEN)

- [ ] T040 Sumar `InFinishedSprint` a `StoryRow` (`src/web/backlog_catalog.go`) y cambiar la consulta de
  `ListStories` en `src/web/postgres_story_catalog.go` por la de `data-model.md`: `JOIN` con Sprints y orden
  `CASE priority`, `created_at`, `id`.
- [ ] T041 Agregar a `src/web/backlog_form.go`:
  - los textos de la tabla "Textos" de `data-model.md`;
  - `backlogListURL(estado)`;
  - `backlogStatusFilter(raw)`, que valida contra `backlog.StoryStatuses()`, trata "Todos" y vacío como la lista
    completa y devuelve el error.
- [ ] T042 En `get` y `pageData` (`src/web/backlog_handlers.go`):
  - leer `estado` con `r.URL.Query().Get`;
  - filtrar las filas;
  - completar `Filter`, `Statuses`, `FilterError` y `NoResults` (solo si hay historias y ninguna coincide);
  - completar `EditURL` e `InFinishedSprint` en cada fila.
- [ ] T043 Editar `src/web/templates/backlog.html`:
  - formulario `GET /backlog` con `<select name="estado">` (Todos y los cuatro estados), el botón "Filtrar" y
    `FilterError` junto al selector;
  - mensaje `NoResults`;
  - columna "Acciones" con el enlace "Editar" (`EditURL`), o con un botón `disabled` y el motivo "Sprint
    finalizado" asociado por `aria-describedby`.
- [ ] T044 Ejecutar T030–T038 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Backlog ya se consulta ordenado y filtrado.

## Fase 4: Bloque B – Edición (escenarios 6, 7a, 7b, 7c, 7d, 7e, 8a, 8b, 9 y 14)

**Objetivo**: abrir el formulario con los datos actuales y guardar los cinco datos, sin tocar el estado ni las
horas.

**Prueba independiente**:

1. Desde la lista filtrada, presionar "Editar".
2. Cambiar cada dato.
3. Agregar y quitar criterios.
4. Guardar y volver a la lista filtrada.

### Pruebas (RED)

- [ ] T045 [P] [E6] Pruebas de integración de `StoryForEdit` en
  `src/web/postgres_story_catalog_edit_integration_test.go`:
  - `TestPostgresStoryForEdit_ConCriteriosEnOrden`;
  - `TestPostgresStoryForEdit_SinCriterios`;
  - `TestPostgresStoryForEdit_Inexistente` y `TestPostgresStoryForEdit_OtroProyecto`, que devuelven `KindNotFound`;
  - `TestPostgresStoryForEdit_SprintFinalizado`.
- [ ] T046 Crear el servidor de prueba `newHU13EditApp` en `src/web/backlog_edit_handlers_test.go`, con el servicio
  y el catálogo reales, y las fixtures de T031 más un Sprint activo y uno finalizado.
- [ ] T047 [P] [E6] `TestHU13_FormularioConDatosActuales`. `GET /backlog/{id}/editar?estado=Pendiente` responde 200
  con:
  - título "Editar historia" y `action="/backlog/{id}/editar"`;
  - título, descripción, prioridad y Story Points `selected`;
  - los criterios en orden;
  - el campo oculto `estado=Pendiente` y el enlace "Cancelar" a `/backlog?estado=Pendiente`;
  - ningún campo `horas_estimadas`.
- [ ] T048 [P] [E7a, E7b, E7c, E7d] `TestHU13_EdicionExitosa`, con un subtest por dato. Cada uno:
  - responde 303 a `/backlog`;
  - muestra "Historia modificada correctamente" una sola vez;
  - la lista muestra el valor nuevo;
  - en la base no cambian el estado ni las horas estimadas.

  En 7c, «Reportes» pasa al grupo Alta, después de «Pagos».
- [ ] T049 [P] [E7e] `TestHU13_VuelveALaListaFiltrada`:
  - el enlace "Editar" de la lista filtrada lleva `?estado=Pendiente`;
  - el `POST` con `estado=Pendiente` redirige a `/backlog?estado=Pendiente` y la historia aparece ahí;
  - con `estado=Archivada`, `estado=` o `estado=https://otro.sitio`, redirige a `/backlog`.
- [ ] T050 [P] [E8a, E8b] `TestHU13_AgregarYQuitarCriterio`:
  - `accion=agregar_criterio` → 200 con un campo más y el título escrito conservado;
  - `accion=quitar_criterio:0` → 200 sin «Acepta email»;
  - en ningún caso cambia la base.
- [ ] T051 [P] [E9] `TestHU13_QuitarStoryPoints`: con `story_points` vacío, la respuesta es 303 con
  "Historia modificada correctamente. Falta completar la estimación de Story Points" como advertencia, y la lista
  muestra "Sin estimar".
- [ ] T052 [P] [E14] `TestHU13_EdicionSinProyecto`:
  1. `GET /backlog/{id}/editar` sin proyecto responde 303 a `/contexto/proyecto?volver_a=%2Fbacklog%2F{id}%2Feditar`.
  2. Al elegir el proyecto, se vuelve a esa dirección.
  3. El formulario se muestra.
- [ ] T053 [P] Pruebas transversales:
  - `TestHU13_IgnoraHorasEstimadas`: `horas_estimadas=abc` y `horas_estimadas=7` no producen error ni cambian las
    horas;
  - `TestHU13_TextoEscapado`: `<script>` en el título y en un criterio se ve como texto en el formulario y en la
    lista;
  - `TestHU13_RecargaSinReenviar`: recargar la lista después del 303 no repite el mensaje.
- [ ] T054 [P] [E7a, E9] Escribir los escenarios BDD "Edición exitosa" y "Quitar los Story Points" en
  `test/web_backlog_edit_bdd_test.go`.
- [ ] T055 Ejecutar T045–T054, confirmar que fallan (la ruta y el catálogo no existen) y guardar la salida en
  `evidencia-red.txt` (sección "Edición").

### Implementación (GREEN)

- [ ] T056 Ampliar el catálogo en el mismo commit:
  - sumar `StoryEdit` y `StoryForEdit` a `StoryCatalog` (`src/web/backlog_catalog.go`);
  - implementar la consulta de `data-model.md` en `src/web/postgres_story_catalog_edit.go` (sin filas →
    `NotFound`; falla de la base → `Unavailable`);
  - sumar un `StoryForEdit` que devuelve `c.err` a `hu22FailingStoryCatalog` (`src/web/backlog_handlers_test.go`).
- [ ] T057 Separar `parseStoryFields(r, withHours bool)` en `src/web/backlog_form.go`. `parseBacklogForm` la llama
  con `true` y las pruebas de HU-22 siguen en verde.
- [ ] T058 Implementar `edit` y `update` en `src/web/backlog_edit_handlers.go`, según el flujo de `plan.md`:
  - `parseID`;
  - `StoryForEdit`;
  - `parseStoryFields(r, false)`, que solo lee los datos;
  - el lugar del control de Sprint finalizado, antes de `applyAction` (lo completa T079);
  - `applyAction` → 200;
  - `UpdateStory`;
  - `SetFlash` de éxito o advertencia, y `RedirectSeeOther(backlogListURL(estado))`.
- [ ] T059 Agregar `"GET /backlog/{id}/editar"` y `"POST /backlog/{id}/editar"` al mapa de `Routes` de
  `RegisterBacklog` (`src/web/backlog_handlers.go`), sin cambiar su firma ni la `Section`.
- [ ] T060 Crear `src/web/templates/backlog_edit.html`:
  - los campos `titulo`, `descripcion`, `prioridad`, `story_points` y `criterio`, con el marcado de errores de
    `backlog.html`;
  - el campo oculto `estado` y el botón oculto "Guardar" primero, como en HU-22;
  - los botones "Quitar", "Agregar criterio" y "Guardar", y el enlace "Cancelar" a `BackURL`.
- [ ] T061 Ejecutar T045–T054 en verde y `go test -count=1 -p 1 ./...` completo.

## Fase 5: Bloque C – Validaciones (escenarios 10a, 10b, 10c, 11a y 11b)

**Objetivo**: rechazar los cambios inválidos con los mensajes de HU-02 y HU-22, sin tocar la historia.

**Prueba independiente**: enviar el formulario con cada dato inválido y comprobar el mensaje, los datos conservados
y la base sin cambios.

### Pruebas (RED)

- [ ] T062 [P] [E10a, E10b] `TestHU13_TituloODescripcionVacios`. El título vacío y la descripción con solo espacios
  responden 422 con:
  - "el título es obligatorio" junto a `titulo`, o "la descripción es obligatoria" junto a `descripcion`;
  - los datos conservados, incluido el campo oculto `estado`;
  - la base sin cambios.
- [ ] T063 [P] [E10c] `TestHU13_SinCriterios`: con todos los criterios vacíos, la respuesta es 422 con "agregá al
  menos un criterio de aceptación" junto a los criterios, y la historia conserva los suyos.
- [ ] T064 [P] [E11a] `TestHU13_PrioridadInvalida`: `prioridad=Urgente` responde 422 con "la prioridad debe ser
  Alta, Media o Baja", y la opción enviada queda seleccionada, como en HU-22.
- [ ] T065 [P] [E11b] `TestHU13_StoryPointsInvalidos`: `story_points=4` y `story_points=abc` responden 422 con "Los
  Story Points no son un valor válido", y el valor enviado queda seleccionado.
- [ ] T066 [P] [E10a] Escribir el escenario BDD "Edición inválida" en `test/web_backlog_edit_bdd_test.go`.
- [ ] T067 Ejecutar T062–T066, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección
  "Validaciones").

### Implementación (GREEN)

- [ ] T068 En `update` (`src/web/backlog_edit_handlers.go`):
  - con errores de formato, responder 422;
  - sin criterios, responder 422 con `ErrAcceptanceCriteriaRequired`, antes del servicio, como HU-22;
  - con errores de HU-02, usar `mapBacklogDomainError` y responder 422;
  - volver a mostrar `backlog_edit.html` con los valores enviados, `InvalidPriority`, `InvalidStoryPoints` y el
    filtro oculto.
- [ ] T069 Ejecutar T062–T066 en verde.

## Fase 6: Bloque D – Reglas, inexistencia y fallas (escenarios 12a, 12b, 12c, 13a, 13b, 13c y 15)

**Objetivo**: bloquear las historias de Sprints finalizados, responder 404 a lo inexistente o ajeno, y 500 sin
cambios cuando falla el guardado.

**Prueba independiente**:

1. Con una historia de un Sprint finalizado, ver "Editar" deshabilitado, abrir su edición y enviarla igual.
2. Enviar identificadores inválidos.
3. Forzar una falla de guardado.

### Pruebas (RED)

- [ ] T070 [P] [E12a] `TestHU13_EditarDeshabilitadoSprintFinalizado`: en la lista, «Pagos» tiene un botón
  "Editar" `disabled` con "Sprint finalizado" y ningún enlace a su edición. Las demás historias tienen enlace.
- [ ] T071 [P] [E12b] `TestHU13_EnvioBloqueadoSprintFinalizado`:
  - el `POST` de «Pagos» con un título nuevo responde 409 con "La historia pertenece a un Sprint finalizado; no se
    puede editar"; el formulario muestra el título enviado, y en la base «Pagos» no cambia;
  - `accion=agregar_criterio` sobre «Pagos» también responde 409, no 200;
  - con `InMemoryBacklogStore.UpdateErr = ErrStoryInFinishedSprint` y el catálogo real (el cierre ocurrió después de
    la lectura), también responde 409.
- [ ] T072 [P] [E12c] `TestHU13_AbrirEdicionSprintFinalizado`: `GET /backlog/{id}/editar?estado=Hecho` de «Pagos»
  responde 303 a `/backlog?estado=Hecho`. Al seguirlo, se ve la advertencia una sola vez y no aparece el
  formulario de edición.
- [ ] T073 [P] [E13a, E13b, E13c] `TestHU13_HistoriaInexistente`:
  - `GET` y `POST` con `999999`, `abc`, `0`, `-1`, `007`, `2147483648` y con «Facturas» (de otro proyecto)
    responden con el 404 de HU-20a, sin cambios en la base;
  - `TestHU13_404AntesQue422`: un formulario inválido enviado a una historia inexistente responde 404.
- [ ] T074 [P] [E15] `TestHU13_ErrorAlGuardar`, con `InMemoryBacklogStore.FailOnUpdate` y el catálogo real. La
  respuesta:
  - es 500 con "La operación no pudo completarse" y los datos enviados;
  - no trae mensaje de éxito;
  - no contiene texto de consultas ni rutas internas.

  La reversión real en la base la prueba T016.
- [ ] T075 [P] `TestHU13_ErrorAlLeer`: con `hu22FailingStoryCatalog`, el `GET` y el `POST` de edición muestran la
  página de error de HU-20a sin texto interno.
- [ ] T076 [P] `TestHU13_Rendimiento`: con 200 historias, 20 de ellas en un Sprint finalizado, la lista sin filtro,
  la lista con `?estado=Pendiente` y el formulario de edición responden en menos de 500 ms.
- [ ] T077 [P] [E12a, E15] Escribir los escenarios BDD "Historia de un Sprint finalizado" y "Error al guardar" en
  `test/web_backlog_edit_bdd_test.go`.
- [ ] T078 Ejecutar T070–T077, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Reglas y
  errores").

### Implementación (GREEN)

- [ ] T079 Completar el mapeo en `src/web/backlog_edit_handlers.go`:
  - `GET` de una historia `InFinishedSprint` → `SetFlash(FlashWarning, MsgStoryInFinishedSprint)` y 303 a
    `backlogListURL(estado)`;
  - `POST` de esa historia (en la lectura o en `ErrStoryInFinishedSprint`) → 409 con el formulario enviado (leído
    antes del control, sin validar) y `Flash`;
  - `ErrStoryNotFound` → `Fail(NotFound(...))`;
  - cualquier otro error → 500 con `MsgOperationFailed`.
- [ ] T080 Ejecutar T070–T077 en verde.

## Fase 7: Cierre

- [ ] T081 REFACTOR:
  - revisar la duplicación entre `post` (HU-22) y `update` (HU-13) al armar `PageData` y renderizar, y extraer una
    función local sin cambiar el comportamiento;
  - `gofmt -l` sobre los archivos tocados (nuevos y existentes) sin salida, sin reformatear otros archivos;
  - dejar `go vet ./...` limpio.
- [ ] T082 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde.
- [ ] T083 Guardar `cobertura.txt` con `go test -count=1 -p 1 -cover ./...` y verificar al menos 80% en
  `src/backlog`.
- [ ] T084 Verificar a mano los escenarios con la aplicación en ejecución (`quickstart.md`, verificación manual).
- [ ] T085 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas, y el manual
  breve de usuario con el filtro y la acción "Editar".
- [ ] T086 [P] Avisar qué cambió en sus archivos (lista de T003):
  - a quien hizo HU-22: el orden, las tres pruebas actualizadas y los fakes;
  - a quien hizo HU-02: el puerto y `validateStoryFields`;
  - a quien tenga HU-15: el orden de bloqueos;
  - a quien tenga HU-17: la edición no toca las horas estimadas.
- [ ] T087 Abrir el Pull Request desde `HU-13` citando HU-13, con las evidencias y la lista de archivos ajenos
  editados.
- [ ] T088 Pedir la revisión y aprobación de otro integrante, hacer el merge con "Create a merge commit" y pasar la
  tarjeta a "Hecho".

## Dependencias y orden de ejecución

- La **Fase 1** va primero. La **Fase 2** depende de la Fase 1 y bloquea a las demás.
- El **Bloque A (Fase 3)** depende de la Fase 2 solo por `StoryStatuses` (T023). Es el MVP: lista ordenada y
  filtrada.
- El **Bloque B (Fase 4)** depende de la Fase 2 completa y de T041–T043 (textos, `backlogListURL` y la columna
  "Acciones").
- Los **Bloques C (Fase 5)** y **D (Fase 6)** dependen de T058–T060 (handler, rutas y plantilla). Pueden avanzar
  en paralelo entre sí.
- La **Fase 7** depende de los bloques que se entreguen.
- Dentro de cada bloque el orden es: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- T027 y T056 cambian un puerto y sus fakes de HU-22 en el mismo commit; si no, `go test ./...` no compila.
- T030 y T037 cambian expectativas de HU-22. Se hacen en RED: con el orden viejo todavía en el código, tienen que
  fallar.
- Los bloques B, C y D editan el mismo `backlog_edit_handlers.go`. Si trabajan dos personas, tienen que coordinar el
  orden de los cambios.

### Oportunidades de paralelismo

- T005–T008, y T010–T021 después de T009.
- T023, T024 y T025 (archivos distintos).
- Dentro del Bloque A: T030 y T032–T038, después de T031.
- T045 y T047–T054, después de T046.
- T062–T066 y T070–T077, cada grupo en su bloque.
- T085 y T086.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T030, T032, T037, T040 |
| FR-002 | T033, T042, T043 |
| FR-003 | T033, T041 |
| FR-004 | T034, T042 |
| FR-005 | T035, T041, T042 |
| FR-006 | T036, T043 |
| FR-007 | T030, T043, T070 |
| FR-008 | T045, T047, T056, T060 |
| FR-009 | T050, T058 |
| FR-010 | T010, T048, T058 |
| FR-011 | T008, T012, T051 |
| FR-012 | T010, T053 |
| FR-013 | T006, T007, T025, T062, T063, T064, T065, T068 |
| FR-014 | T014, T019, T071, T072, T079 |
| FR-015 | T015 |
| FR-016 | T013, T045, T073, T079 |
| FR-017 | T052 |
| FR-018 | T016, T074, T079 |
| FR-019 | T075 |
| FR-020 | T062, T063, T064, T065, T068 |
| FR-021 | T011, T018, T020, T021, T028 |
| FR-022 | T053, T074 |
| FR-023 | T041 |
| FR-024 | T047, T049, T072 |
| SC-001 | T030, T032, T033 |
| SC-002 | T010, T048, T051 |
| SC-003 | T013, T014, T016, T050, T062, T063, T064, T065, T071, T073, T074 |
| SC-004 | T014, T019 |
| SC-005 | T076 |
| SC-006 | T022, T039, T055, T067, T078, T082, T083 |

## Estrategia de implementación

1. Completar las Fases 1 y 2 (dominio y almacenamiento de la edición).
2. Bloque A, que es el MVP: con él, el Backlog ya se consulta ordenado y filtrado.
3. Bloque B: editar los datos.
4. Bloques C y D, en paralelo si hay manos libres.
5. Cierre, evidencias y PR.
