# Tareas: HU-11 – Modificación de Proyecto (web)

**Entrada**: `specs/011-project-update/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-01, FIX-HU-01, HU-03, HU-04, HU-20a, HU-20b y HU-21 integradas en `main`.

Formato: `[ID] [P] [E#] descripción`
- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1, 2a, 2b, 3, 4a, 4b, 5, 6a, 6b, 6c, 7, 8, 9, 10, 11a, 11b, 12a, 12b,
  13a, 13b, 14a, 14b, 15 y 16).
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/011-project-update/evidencia-red.txt`.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- No usar `gofmt -w` sobre archivos existentes (están en CRLF); los archivos nuevos se formatean.

## Fase 1: Preparación

- [X] T001 Crear la rama `HU-11` desde `main` actualizado (`git switch main; git pull; git switch -c HU-11`) — La rama `HU-11` ya existía y está al día con `main`
- [X] T002 Confirmar los "Datos verificados" de `research.md` contra el código: — Ninguno de esos nombres existía; los de `plan.md` se mantienen
  - leer `src/project`, `src/web/projects_*.go`, `src/web/templates/projects.html` y `src/effort`;
  - comprobar con `git grep -n` que no existen `ProjectUpdate`, `UpdateProject`, `ProjectForEdit`,
    `ErrProjectNotFound`, `MemberHasEffortError`, `SprintOutOfRangeError` ni `errMemberNotInProject`;
  - ajustar `plan.md` si algún nombre cambió.
- [ ] T003 Coordinar los archivos ajenos que HU-11 edita:
  - con quien hizo HU-21: `projects_handlers.go`, `projects_catalog.go`, `projects_form.go`, `projects.html` y los
    fakes `failingCreateProjectStore` (`src/web/projects_handlers_test.go`) y `failingBDDProjectStore`
    (`test/web_projects_bdd_test.go`);
  - con quien hizo HU-03: `src/effort/postgres_effort_store.go`, `src/effort/service.go` y `src/effort/errors.go`.
- [ ] T004 `go test -count=1 -p 1 ./...` en verde antes de empezar (línea base, incluidas las pruebas de HU-01, HU-03
  y HU-21)

## Fase 2: Fundamentos (dominio, almacenamiento y HU-03; bloquea todo lo demás)

### Pruebas (RED)

- [X] T005 [P] [E12a] `TestDatesChanged`: mismas fechas → false; inicio distinto → true; fin distinto → true; la
  misma fecha en otra zona horaria → false. En `src/project/update_rules_test.go`.
- [X] T006 [P] [E7, E8] `TestMembershipChanges`: solo altas, solo bajas, ambas y sin cambios; resultados ordenados y
  sin repetidos. En `src/project/update_rules_test.go`.
- [X] T007 [P] [E11a, E11b] `TestSprintsLeftOutside` con tabla, en `src/project/update_rules_test.go`. Casos:
  - acortar el fin → Sprint y `DateFieldEnd`;
  - adelantar el inicio → `DateFieldStart`;
  - ampliar → `nil`;
  - un Sprint que ya estaba fuera → `nil`;
  - un Sprint en el borde exacto (empieza el día de inicio, termina el día de fin) → `nil`;
  - un Sprint que queda fuera por los dos lados → `DateFieldStart`;
  - varios afectados → el de menor número;
  - un Sprint "Finalizado" cuenta;
  - uno que ya estaba fuera y otro que queda fuera → el segundo.
- [X] T008 [P] [E4a, E5, E9, E13b] `TestUpdateProject_Validaciones` con `TestProjectStore`, en
  `src/project/update_service_test.go`:
  - `ID` 0 → `ErrProjectNotFound`;
  - nombre vacío y nombre `"   "` → `ErrEmptyName` (el servicio recorta aunque no pase por la web);
  - nombre `"  Alfa 2  "` → `Update` recibe `"Alfa 2"`;
  - sin integrantes → `ErrMissingMembers`;
  - fin igual al inicio → `ErrInvalidPeriod`;
  - integrante "abc" → `ErrMemberNotFound`;
  - repetidos → `ErrDuplicateMember`;
  - en los casos de error no se llama a `Update` (`len(store.Updates) == 0`).
- [X] T009 [P] [E2a, E16] `TestUpdateProject_DelegaYPropaga`, en `src/project/update_service_test.go`:
  - un envío válido llama a `Update` con los `MemberIDs` convertidos;
  - `UpdateErr` con `ErrProjectNotFound`, `*MemberHasEffortError` o `*SprintOutOfRangeError` se devuelve sin
    cambios (`errors.Is` y `errors.As`);
  - con `ShouldFailUpdate` el error se envuelve y no es ninguno de los de dominio.
- [X] T010 Fixtures de integración en `src/project/postgres_store_update_integration_test.go`:
  - proyectos «Alfa» (2026-03-01 a 2026-06-30) y «Beta»;
  - integrantes propios Ana, Beto y Carla;
  - historias en los dos proyectos, Sprint 2 de «Alfa» del 2026-06-01 al 2026-06-14, horas de Beto en «Beta»;
  - limpieza en orden: `effort_logs`, `sprint_story_assignments`, historias, Sprints, proyectos e integrantes
    propios.
- [X] T011 [P] [E2a, E2b] `TestPostgresUpdate_NombreYFechas`, con las fixtures de T010:
  - cambia el nombre y las fechas;
  - `status`, `next_sprint_number`, los Sprints, las historias y las horas no cambian;
  - es la primera prueba con arreglos: confirma que `[]int32` se pasa como `int[]` (si falla, aplicar la alternativa
    de `data-model.md`).
- [X] T012 [P] [E7] `TestPostgresUpdate_Alta`: Carla pasa a pertenecer a «Alfa»; Ana y Beto siguen.
- [X] T013 [P] [E8] `TestPostgresUpdate_BajaSinHoras`: Beto, con horas solo en «Beta», deja «Alfa»; sus horas en
  «Beta» siguen.
- [X] T014 [P] [E10] `TestPostgresUpdate_BajaConHoras`: Beto con horas en «Alfa».
  - Devuelve `*MemberHasEffortError{MemberName: "Beto"}`.
  - El nombre, las fechas y los integrantes de «Alfa» no cambian, aunque el envío también cambiaba el nombre y
    agregaba a Carla.
- [X] T015 [P] [E10] `TestPostgresUpdate_VariosConHoras_NombraAlPrimero`: Beto y Ana con horas → nombra a "Ana" (orden
  `display_name, id`).
- [X] T016 [P] [E13a] `TestPostgresUpdate_IntegranteInexistente`: el identificador 2147483647 → `ErrMemberNotFound`;
  nada cambia.
- [X] T017 [P] [E14a] `TestPostgresUpdate_ProyectoInexistente` → `ErrProjectNotFound`; no se crea nada.
- [X] T018 [P] [E11a, E11b, E12a] Tres pruebas de Sprints:
  - `TestPostgresUpdate_SprintFuera`: fin 2026-06-10 → `*SprintOutOfRangeError{Number: 2, Field: DateFieldEnd}`;
    nada cambia.
  - `TestPostgresUpdate_SprintYaFuera`: el Sprint 1 empieza el 2026-02-20; ampliar el fin a 2026-07-31 se guarda.
  - `TestPostgresUpdate_SoloNombreConSprints`: se guarda.
- [X] T019 [P] [E16] Sin cambios y falla a mitad:
  - `TestPostgresUpdate_SinCambios`: devuelve `nil` y deja todo igual (R9).
  - `TestPostgresUpdate_FalloNoDejaCambios`: un nombre de 256 caracteres enviado directo al almacén hace fallar el
    `UPDATE` (paso 7) después del `DELETE` (paso 4). La baja de Beto se deshace y nada cambia.
- [X] T020 [P] `TestPostgresUpdate_ConcurrenteUltimaGana`: dos guardados simultáneos de «Alfa» con datos distintos
  terminan sin error, y el resultado es uno de los dos completo, nunca una mezcla.
- [X] T021 [P] [E10] `TestPostgresUpdate_EsperaCargaDeHoras`:
  1. Una transacción manual toma `FOR SHARE` sobre la fila de pertenencia de Beto en «Alfa» e inserta una hora.
  2. `Update` (baja de Beto) corre en una goroutine.
  3. Se espera a que quede bloqueado, consultando `pg_stat_activity` con `wait_event_type = 'Lock'`.
  4. `COMMIT` de la transacción manual.
  5. `Update` devuelve `*MemberHasEffortError` y Beto sigue en «Alfa».
- [X] T022 [P] [E10] Pruebas de HU-03 en `src/effort/postgres_effort_store_membership_integration_test.go`:
  - `TestPostgresSaveEntry_RechazaIntegranteQuitado`: un integrante que no pertenece al proyecto de la historia →
    `errMemberNotInProject`, sin registro.
  - `TestPostgresSaveEntry_EsperaBajaEnCurso`: una transacción manual borra la pertenencia sin confirmar;
    `SaveEntry` en una goroutine queda bloqueado; con `COMMIT` devuelve `errMemberNotInProject` sin registro; con
    `ROLLBACK`, guarda.
- [X] T023 [P] `TestLogEffort_IntegranteQuitadoAlGuardar`: con `TestStore.SaveErr = errMemberNotInProject`, devuelve
  `ValidationErrors` con `FieldMember` y `ErrMemberRequired`, y `StorageError` vacío. En `src/effort/service_test.go`.
- [X] T024 Ejecutar T005–T023, confirmar que fallan porque el código no existe y guardar la salida en
  `evidencia-red.txt` (sección "Fundamentos")

### Implementación (GREEN)

- [X] T025 Crear en `src/project/update.go` los tipos `ProjectUpdateRequest`, `ProjectUpdate` (`MemberIDs []int` "sin
  repetidos"), `SprintPeriod`, `DateField`, `DateFieldStart = "inicio"` y `DateFieldEnd = "fin"`
- [X] T026 Crear en `src/project/update_errors.go` `ErrProjectNotFound`, `MemberHasEffortError{MemberName string}` y
  `SprintOutOfRangeError{Number int; Field DateField}`, con `Error()`
- [X] T027 Sumar el método al puerto y a sus implementaciones de prueba:
  - `Update(ctx context.Context, u ProjectUpdate) error` en `ProjectStore` (`src/project/store.go`);
  - `Update`, `Updates`, `ShouldFailUpdate` y `UpdateErr` en `src/project/test_store.go`;
  - en el mismo commit, `Update` que devuelve `errors.New("simulated update error")` en `failingCreateProjectStore`
    (`src/web/projects_handlers_test.go`) y en `failingBDDProjectStore` (`test/web_projects_bdd_test.go`).
- [X] T028 Implementar `DatesChanged`, `MembershipChanges` y `SprintsLeftOutside` en `src/project/update_rules.go`
  según `data-model.md`. Un Sprint "estaba dentro" si `!s.Start.Before(oldStart) && !s.End.After(oldEnd)`, y "queda
  fuera" si `s.Start.Before(newStart) || s.End.After(newEnd)`. El campo es `DateFieldStart` si
  `s.Start.Before(newStart)`.
- [X] T029 Extraer `parseMemberIDs` de `CreateProject` en `src/project/service.go`, sin cambiar su comportamiento
  (las pruebas de HU-01 siguen en verde). Implementar `UpdateProject` en `src/project/update_service.go` con este
  orden: `ID`, `strings.TrimSpace` del nombre, `parseMemberIDs`, `NewProject` y `store.Update` con el nombre
  recortado.
- [X] T030 Implementar `PostgreSQLProjectStore.Update` en `src/project/postgres_store_update.go` con los ocho pasos de
  `data-model.md`:
  - bloqueo `FOR NO KEY UPDATE`;
  - bajas antes de contar horas;
  - Sprints solo si `DatesChanged`;
  - `ON CONFLICT DO NOTHING` en las altas;
  - el código 23503 en las altas se traduce a `ErrMemberNotFound`.
- [X] T031 Agregar `errMemberNotInProject` en `src/effort/errors.go`. En `SaveEntry`
  (`src/effort/postgres_effort_store.go`), antes del INSERT y dentro de la transacción, agregar la consulta
  `... JOIN project_members pm ... FOR SHARE OF pm`; sin fila, devolver `errMemberNotInProject`.
- [X] T032 En `Service.LogEffort` (`src/effort/service.go`), traducir `errMemberNotInProject` a
  `ValidationErrors: []FieldError{{Field: FieldMember, Err: ErrMemberRequired}}`
- [ ] T033 Ejecutar T005–T023 en verde, junto con las pruebas existentes de `./src/project/...` y `./src/effort/...`

**Punto de control**: el dominio, el almacenamiento y el cierre de la carrera con HU-03 funcionan sin tocar la
pantalla.

## Fase 3: Bloque A – Datos del proyecto (escenarios 1, 2a, 2b, 3, 4a, 4b, 5, 6a, 6b y 6c) 🎯 MVP

**Objetivo**: abrir el formulario desde la lista y guardar nombre y fechas, con sus validaciones.

**Prueba independiente**: desde "Gestión de Proyectos", seguir "Modificar", cambiar el nombre y las fechas y verlos
en la lista y en la estructura común; probar cada error de nombre y de fechas.

### Pruebas (RED)

- [X] T034 [P] [E1] Pruebas de integración del catálogo en
  `src/web/postgres_project_catalog_edit_integration_test.go`:
  - `TestPostgresProjectForEdit_ConIntegrantes`: nombre, fechas e identificadores ordenados;
  - `TestPostgresProjectForEdit_SinIntegrantes`: `MemberIDs` vacío;
  - `TestPostgresProjectForEdit_Inexistente`: `KindOf(err) == KindNotFound`.
- [X] T035 Servidor de prueba `newHU11Server` en `src/web/projects_update_handlers_test.go`:
  - `ProjectService` con `PostgreSQLProjectStore`, `PostgreSQLProjectCatalog`, `cookiejar` y middleware de contexto;
  - fixtures «Alfa» y «Beta», Ana, Beto y Carla, y el Sprint 2.
- [X] T036 [P] [E1] `TestHU11_EnlaceModificarEnLista`: `GET /proyectos` muestra en cada fila un enlace "Modificar" con
  `href="/proyectos/{id}/modificar"`
- [X] T037 [P] [E1] `TestHU11_FormularioConDatosActuales`. `GET /proyectos/{id}/modificar` responde 200 con:
  - el título "Modificar proyecto" y `action="/proyectos/{id}/modificar"`;
  - `value` del nombre;
  - las fechas en `AAAA-MM-DD`;
  - solo Ana y Beto `checked`.
- [X] T038 [P] [E2a, E2b] `TestHU11_ModificacionExitosa` y `TestHU11_SinCambiosEsExito`:
  - responden 303 a `/proyectos`;
  - muestran "Proyecto modificado exitosamente" una sola vez;
  - la lista muestra el nombre nuevo y las fechas en `dd/mm/aaaa`.
- [X] T039 [P] [E3] `TestHU11_NombreEnEstructuraComun`: elegir «Alfa» con el flujo de HU-20b, renombrarlo a «Alfa 2» y
  comprobar que `GET /` muestra «Alfa 2» en la estructura común sin volver a elegirlo
- [X] T040 [P] [E4a, E4b] `TestHU11_NombreVacio`, con `""` y `"   "`:
  - 422 con "project name cannot be empty" junto a `nombre`;
  - las fechas y los integrantes enviados se conservan;
  - la base no cambia.
- [X] T041 [P] [E5] `TestHU11_FechasInvalidas`: fin igual al inicio y fin anterior → 422 con "La fecha de finalización
  debe ser posterior a la fecha de inicio" junto a `fecha_fin`; datos conservados; base sin cambios
- [X] T042 [P] [E6a, E6b, E6c] `TestHU11_FechaFaltanteOFormato`: inicio vacío, fin vacío y "01/03/2026" → 422 con el
  mensaje de HU-21 junto a esa fecha; datos conservados; base sin cambios
- [X] T043 [P] Pruebas transversales:
  - `TestHU11_NombreEscapado`: `<script>` en el nombre se ve como texto en el formulario y en la lista.
  - `TestHU11_RecargaSinReenviar`: después del 303, recargar `/proyectos` no repite el mensaje.
  - `TestHU11_ErrorAlLeer`: con un catálogo que falla en `ProjectForEdit`, se ve la página de error de HU-20a y la
    respuesta no incluye texto de consultas.
- [X] T044 [P] [E2a, E3] Escenarios BDD "Modificación exitosa" y "Nombre visible en la estructura común" en
  `test/web_project_update_bdd_test.go`
- [X] T045 Ejecutar T034–T044, confirmar que fallan (la ruta y el catálogo no existen) y guardar la salida en
  `evidencia-red.txt` (sección "Datos del proyecto")

### Implementación (GREEN)

- [X] T046 Agregar `ProjectEdit` y `ProjectForEdit` a `ProjectCatalog` en `src/web/projects_catalog.go`. Implementar la
  consulta de `data-model.md` en `src/web/postgres_project_catalog_edit.go`: sin filas, `NotFound`; error de la base,
  `Unavailable`.
- [X] T047 Agregar en `src/web/projects_form.go`:
  - `TitleProjectEdit = "Modificar proyecto"`;
  - `MsgProjectUpdated = "Proyecto modificado exitosamente"`;
  - `MsgMemberHasEffort = "No se puede quitar a %s: tiene horas registradas en este proyecto"`;
  - `MsgSprintOutOfRange = "El Sprint %d queda fuera de las nuevas fechas del proyecto"`;
  - `LinkEditProject = "Modificar"`.
- [X] T048 Implementar `edit` y `update` en `src/web/projects_update_handlers.go` según el flujo de `plan.md`:
  - `parseID`: con 0, `Fail(NotFound)`;
  - `ProjectForEdit`, después `parseProjectForm`, después `ListMembers`;
  - errores de formato → 422;
  - `UpdateProject`, con los errores de HU-01 mapeados por `projectFieldError` → 422;
  - éxito: `SetFlash` y `RedirectSeeOther("/proyectos")`.
- [X] T049 Agregar `"GET /proyectos/{id}/modificar"` y `"POST /proyectos/{id}/modificar"` al mapa de `Routes` de
  `RegisterProjects` en `src/web/projects_handlers.go`, sin cambiar su firma ni la `Section`
- [X] T050 Crear `src/web/templates/project_edit.html` con:
  - los campos `nombre`, `integrantes` (casillas con `SelectedMembers`), `fecha_inicio` y `fecha_fin`, con
    `$.Form.Field` y el bloque de error de integrantes de `projects.html`;
  - el botón "Guardar" y el enlace "Cancelar" a `/proyectos`.
- [X] T051 Agregar en `src/web/templates/projects.html` la columna "Acciones" con el enlace `LinkEditProject` en cada
  fila
- [X] T052 Ejecutar T034–T044 en verde y `go test -count=1 -p 1 ./...` completo

**Punto de control**: modificar nombre y fechas funciona de punta a punta.

## Fase 4: Bloque B – Integrantes (escenarios 7, 8, 9, 10, 13a, 13b y 15)

**Objetivo**: agregar y quitar integrantes con las reglas de HU-01 y de horas.

**Prueba independiente**: marcar a Carla, desmarcar a Beto sin horas, intentar quitar a Beto con horas y al último
integrante, y ver el efecto en el selector de identificación.

### Pruebas (RED)

- [X] T053 [P] [E7] `TestHU11_AgregarIntegrante`: al marcar a Carla responde 303.
  - Carla aparece en `/contexto/integrante` con «Alfa» elegido.
  - `effort.NewPostgreSQLEffortStore(db).MemberBelongsToStoryProject` devuelve `true` para Carla y una historia de
    «Alfa».
  - En `src/web/projects_update_handlers_test.go`.
- [X] T054 [P] [E8] `TestHU11_QuitarSinHoras`: al desmarcar a Beto (horas solo en «Beta») responde 303; Beto ya no
  aparece en `/contexto/integrante` de «Alfa» y sus horas en «Beta» siguen
- [X] T055 [P] [E9] `TestHU11_QuitarUltimo`: «Alfa» con solo Ana; sin casillas → 422 con "the member list is required"
  junto a `integrantes`; Ana sigue
- [X] T056 [P] [E10] `TestHU11_QuitarConHoras`: al desmarcar a Beto (con horas en «Alfa») y cambiar el nombre:
  - 409 con "No se puede quitar a Beto: tiene horas registradas en este proyecto" junto a `integrantes`;
  - el formulario conserva lo enviado;
  - en la base, Beto sigue y el nombre no cambió.
- [X] T057 [P] [E13a, E13b] `TestHU11_IntegranteInvalido`: `integrantes=2147483647` y `integrantes=abc` → 422 con "El
  integrante seleccionado no es válido" junto a `integrantes`; base sin cambios
- [X] T058 [P] [E15] `TestHU11_IdentificadoQuitado`:
  1. Identificarse como Beto (sin horas en «Alfa») en «Alfa» con el flujo de HU-20b.
  2. Quitar a Beto con otro cliente.
  3. `GET /` muestra la advertencia `MsgMemberGone` y borra la cookie `ctx_integrante`.
- [X] T059 [P] [E7, E10] Escenarios BDD "Agregar integrante" y "Quitar un integrante con horas registradas" en
  `test/web_project_update_bdd_test.go`. Las horas se cargan con el servicio real de HU-03.
- [X] T060 Ejecutar T053–T059, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección
  "Integrantes")

### Implementación (GREEN)

- [X] T061 En `src/web/projects_update_handlers.go`, mapear `*MemberHasEffortError` a 409 con
  `fmt.Sprintf(MsgMemberHasEffort, e.MemberName)` junto a `integrantes`, y `ErrMemberNotFound` a `MsgInvalidMember`
  (ya lo hace `projectFieldError`)
- [X] T062 Construir `SelectedMembers` con los integrantes enviados en cada rechazo y con los actuales en el `GET`, en
  `src/web/projects_update_handlers.go`
- [X] T063 Ejecutar T053–T059 en verde

## Fase 5: Bloque C – Fechas y Sprints (escenarios 11a, 11b, 12a y 12b)

**Objetivo**: rechazar los cambios de fechas que dejan fuera a un Sprint que estaba dentro, sin bloquear los demás
cambios.

**Prueba independiente**: con el Sprint 2 creado, acortar el fin y ver el rechazo; ampliar y ver que se guarda;
cambiar solo el nombre o los integrantes.

### Pruebas (RED)

- [X] T064 [P] [E11a] `TestHU11_SprintQuedaFuera`. Responde 409 con "El Sprint 2 queda fuera de las nuevas fechas del
  proyecto":
  - con fin 2026-06-10, junto a `fecha_fin`;
  - con inicio 2026-06-05, junto a `fecha_inicio`.

  Los datos enviados se conservan y las fechas no cambian.
- [X] T065 [P] [E11b] `TestHU11_SprintYaFueraNoBloquea`: con el Sprint 1 del 2026-02-20 al 2026-03-05, ampliar el fin
  a 2026-07-31 → 303 y el Sprint 1 no cambia
- [X] T066 [P] [E12a, E12b] `TestHU11_SinCambioDeFechasConSprints`: cambiar solo el nombre y solo los integrantes con
  Sprints creados → 303 en los dos casos
- [X] T067 [P] [E11a] Escenario BDD "Fechas que dejan Sprints fuera" en `test/web_project_update_bdd_test.go`
- [X] T068 Ejecutar T064–T067, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Fechas y
  Sprints")

### Implementación (GREEN)

- [X] T069 En `src/web/projects_update_handlers.go`, mapear `*SprintOutOfRangeError` a 409 con
  `fmt.Sprintf(MsgSprintOutOfRange, e.Number)` junto a `fecha_inicio` si `Field == DateFieldStart` y, si no, junto a
  `fecha_fin`
- [ ] T070 Ejecutar T064–T067 en verde

## Fase 6: Bloque D – Proyecto inexistente y fallas (escenarios 14a, 14b y 16)

**Objetivo**: 404 para identificadores inválidos o inexistentes, y 500 sin cambios ante una falla de guardado.

**Prueba independiente**: abrir y enviar identificadores inválidos y forzar una falla de guardado.

### Pruebas (RED)

- [X] T071 [P] [E14a, E14b] `TestHU11_ProyectoInexistente`: `GET` y `POST` con `999999`, `abc`, `0`, `-1`, `007` y
  `2147483648` → 404 de HU-20a, sin cambios en la base. Incluye `TestHU11_404AntesQue422`: un formulario inválido
  enviado a un proyecto inexistente responde 404.
- [X] T072 [P] [E14a] `TestHU11_ProyectoBorradoAlGuardar`: el catálogo encuentra el proyecto, pero
  `TestProjectStore.UpdateErr = ErrProjectNotFound` → 404
- [X] T073 [P] [E16] `TestHU11_ErrorAlGuardar`: con `TestProjectStore.ShouldFailUpdate` y el catálogo real:
  - 500 con "La operación no pudo completarse" y los datos enviados;
  - sin mensaje de éxito;
  - la respuesta no contiene texto de consultas ni rutas internas;
  - en la base no cambia nada.
- [X] T074 [P] `TestHU11_Rendimiento`: con 50 integrantes en el sistema y 20 Sprints en el proyecto, el `GET` del
  formulario y un `POST` con cambio de fechas responden en menos de 500 ms
- [X] T075 [P] [E16] Escenario BDD "Error al guardar" en `test/web_project_update_bdd_test.go`
- [X] T076 Ejecutar T071–T075, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Errores")

### Implementación (GREEN)

- [X] T077 Completar en `src/web/projects_update_handlers.go` el mapeo de `ErrProjectNotFound` a `Fail(NotFound(...))`
  y de cualquier otro error a 500 con `Flash{FlashError, MsgOperationFailed}` y el formulario
- [X] T078 Ejecutar T071–T075 en verde

## Fase 7: Cierre

- [X] T079 REFACTOR: revisar la duplicación entre el `post` de HU-21 y el `update` de HU-11 (armado de `FormView` y — `update` reutiliza `parseProjectForm`, `projectFormView` y `projectFieldError` de HU-21; solo se agregó `renderEdit` para la plantilla propia. Archivos nuevos formateados y `go vet ./...` limpio
  render) y extraer una función local sin cambiar comportamiento. Formatear solo los archivos nuevos y dejar
  `go vet ./...` limpio.
- [ ] T080 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde
- [ ] T081 Guardar `cobertura.txt` con `go test -count=1 -p 1 -cover ./...` verificando al menos 80% en `src/project`
- [ ] T082 Verificar a mano los escenarios con la aplicación en ejecución (`quickstart.md`, verificación manual)
- [X] T083 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas, y el manual — Manual: sección "Modificar un proyecto" en `docs/manual-usuario.md`
  breve de usuario con la acción "Modificar" de "Gestión de Proyectos"
- [ ] T084 [P] Avisar a quienes hicieron HU-21 y HU-03 qué cambió en sus archivos (lista de T003) y que el guardado de
  horas ahora rechaza a un integrante quitado durante la carga
- [ ] T085 Abrir el Pull Request desde `HU-11` citando HU-11, con las evidencias y la lista de archivos ajenos
  editados
- [ ] T086 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho"

## Dependencias y orden de ejecución

- **Fase 1** primero. **Fase 2** depende de la Fase 1 y bloquea a las demás.
- **Bloque A (Fase 3)** depende de la Fase 2. Es el MVP: crea las rutas, el catálogo y la plantilla.
- **Bloque B (Fase 4)** y **Bloque C (Fase 5)** dependen de T048–T050 (handler, rutas y plantilla). Pueden avanzar
  en paralelo entre sí.
- **Bloque D (Fase 6)** depende de T048. Puede avanzar en paralelo con B y C.
- **Fase 7** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar evidencia → implementación (GREEN).
- T027 cambia el puerto y los dos fakes de HU-21 en el mismo commit; si no, `go test ./...` no compila.
- T031 y T032 (HU-03) dependen de T022 y T023, y no dependen del resto de HU-11: se pueden revisar con el
  responsable de HU-03 por separado.
- Los bloques B, C y D editan el mismo `projects_update_handlers.go`: si trabajan dos personas, coordinar el orden de
  los cambios.

### Oportunidades de paralelismo

- T005–T009 (dominio) y T011–T023, después de T010.
- T025, T026 y T028 (archivos distintos); T031–T032 en paralelo con T029–T030.
- T034 y T036–T044 dentro del Bloque A, después de T035.
- T053–T059, T064–T067 y T071–T075 dentro de su bloque.
- T083 y T084.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T036, T051 |
| FR-002 | T034, T037, T046, T050 |
| FR-003 | T011, T038, T048 |
| FR-004 | T014, T019, T030, T056 |
| FR-005 | T019, T038 |
| FR-006 | T008, T029, T040 |
| FR-007 | T008, T041, T042, T029 |
| FR-008 | T008, T055 |
| FR-009 | T008, T016, T057 |
| FR-010 | T013, T014, T015, T054, T056, T061 |
| FR-011 | T012, T013, T053, T054, T058 |
| FR-012 | T007, T018, T028, T064, T065, T069 |
| FR-013 | T005, T018, T030, T066 |
| FR-014 | T017, T071, T072, T077 |
| FR-015 | T019, T073, T077 |
| FR-016 | T043 |
| FR-017 | T040, T041, T042, T055, T056, T057, T062, T064 |
| FR-018 | T039, T044 |
| FR-019 | T020, T021, T030 |
| FR-020 | T021, T022, T023, T031, T032 |
| FR-021 | T043, T073 |
| FR-022 | T011, T019 |
| FR-023 | T047 |
| SC-001 | T038, T039, T053 |
| SC-002 | T014, T019, T040, T041, T042, T055, T056, T057, T064, T071, T073 |
| SC-003 | T014, T021, T022 |
| SC-004 | T007, T018, T065 |
| SC-005 | T019, T073 |
| SC-006 | T074 |
| SC-007 | T024, T045, T060, T068, T076, T080, T081 |

## Estrategia de implementación

1. Completar las Fases 1 y 2 (dominio, almacenamiento y cierre de la carrera con HU-03).
2. Bloque A: es el MVP. Con él ya se corrigen el nombre y las fechas de un proyecto.
3. Bloques B, C y D, en paralelo si hay manos libres.
4. Cierre, evidencias y PR.
