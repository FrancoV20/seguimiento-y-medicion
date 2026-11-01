# Tareas: HU-16 – Consulta de Sprints Anteriores

**Entrada**: `specs/016-past-sprints/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-15 integrada en `main` (con la migración de `sprint_story_results`), además de HU-07, HU-20a
y HU-20b (ya están). Para la verificación manual, además HU-21, HU-22, HU-23 y HU-14.

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 9).
- Cada bloque sigue RED → GREEN. Las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/016-past-sprints/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- CI (`.github/workflows/ci.yml`): `gofmt -l` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` se corre solo sobre los archivos que se crean; `main.go` se edita a mano, sin cambiar su fin de línea.
- Las pruebas de integración arman sus propios datos como los deja HU-15 (research.md §10) y los borran al final, en
  este orden:
  1. registro de cierre (`sprint_story_results`, T001: HU-15);
  2. asignaciones;
  3. `sprint_metrics`;
  4. historias;
  5. Sprints;
  6. proyectos.

## Fase 1: Preparación

- [x] T001 Confirmar con `git grep -n` que HU-15 está en `main` y sus nombres. **Si no está, detener todo**: no se
  crea la rama ni se sigue con T002; se avisa al equipo y se espera el merge de HU-15.
  - De HU-15 (T001: HU-15): la migración de `sprint_story_results` en `db/migrations/` y sus columnas `sprint_id`,
    `story_id`, `final_status` (valores 'En Sprint', 'En progreso', 'Hecho') y `story_points`. También, que su cierre
    borra las asignaciones de las no completadas, las pasa a 'Pendiente' con `active_sprint_id = NULL` y conserva las
    de las "Hecho". Si algún nombre o comportamiento cambió, corregir `data-model.md`, `research.md` §10 y las
    consultas antes de T002.
  - De HU-07: `metrics.NewService`, `metrics.NewPostgreSQLMetricsStore`, `(*Service).CalculateMetrics(sprintID
    int64) (SprintMetrics, error)` y `SprintMetrics.Velocity`. Que siga leyendo las "Hecho" asignadas y haciendo el
    `upsert` en `sprint_metrics`.
  - De HU-20a/b y HU-14: `Server.Register`, `Section.RequiresProject`, `Routes`, `parseID`, `projectIDFrom`,
    `Renderer.Page`, `Renderer.Fail`, `NotFound`, `KindOf` y `sprint.Done`.
  - Que no existan la ruta `/sprints-anteriores`, el texto "Sprints anteriores" ni nombres `PastSprint*` en `src/web`.
    Si HU-23, HU-25 u otra historia ya importó `src/metrics` en `main.go`, anotarlo para T015.

  Revisar también "Planificación de proyecto.pdf": si contradice la spec, corregirla antes de T002.
- [x] T002 Crear la rama `HU-16` desde `main` actualizado, solo si no hay cambios sin commitear:
  `git switch main; git pull; git switch -c HU-16`.
- [x] T003 Coordinar el bloque de registro que se suma al final de `src/cmd/server/main.go` (y el import de
  `src/metrics`) con quien tenga HU-08, HU-09, HU-10, HU-12, HU-23 y HU-25. Con HU-10 y HU-25, acordar que el
  registro de cierre de HU-15 se lee igual (`final_status = 'Hecho'` = completada).
- [x] T004 Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y `go vet ./...` en verde antes de empezar.

## Fase 2: Bloque Catálogo (adaptador PostgreSQL)

**Objetivo**: leer los Sprints finalizados del proyecto y el registro de cierre de un Sprint en una lectura
consistente.

**Prueba independiente**: `TestPostgresPastSprintCatalog` pasa contra PostgreSQL con datos propios.

### Pruebas (RED)

- [x] T005 Crear los datos de prueba en `src/web/past_sprint_fixtures_test.go`; los usan T006, T011, T018 y T019. Son
  los de la tabla "Datos de prueba" de `data-model.md`, insertados directamente como los deja HU-15:
  - «Alfa» (18/08/2026 al 30/11/2026), con las historias en este orden: «Alta» 3, «Login» 5, «Tablero» 3, «Perfil» 2,
    «Reportes» 8, «Exportar» 5 y «Ayuda» sin estimar.
    - Sprint 1 "Dar de alta usuarios" (18/08/2026 al 31/08/2026), 'Finalizado': registro «Alta» 'Hecho' 3; «Alta»
      asignada.
    - Sprint 2 "Gestionar perfiles y reportes" (01/09/2026 al 14/09/2026), 'Finalizado': registro «Login» 'Hecho' 5,
      «Tablero» 'Hecho' 3, «Perfil» 'Hecho' 2 y «Reportes» 'En progreso' 8. Las tres "Hecho" quedan asignadas;
      «Reportes» queda 'Pendiente', sin asignación ni `active_sprint_id`.
    - Sprint 3 "Exportar e informar" (15/09/2026 al 28/09/2026), 'Activo', con «Exportar» 'En progreso' y «Ayuda» 'En
      Sprint' asignadas.
  - «Beta» (01/10/2026 al 31/12/2026), sin Sprints ni historias.
  - «Gamma» (01/09/2026 al 31/12/2026), con las historias «Migración» 5, «Respaldo» sin estimar y «Auditoría» 2.
    - Sprint 1 "Migrar datos" (01/09/2026 al 14/09/2026), 'Finalizado': registro «Migración» 'En progreso' 5 y
      «Respaldo» 'En Sprint' `NULL`; las dos 'Pendiente', sin asignar.
    - Sprint 2 "Auditar accesos" (15/09/2026 al 28/09/2026), 'Finalizado', sin filas en el registro, con «Auditoría»
      'Hecho' 2 asignada.

  Limpieza en `t.Cleanup`, en el orden del encabezado.
- [x] T006 [E1, E2, E3, E4, E5, E6, E8] Escribir `TestPostgresPastSprintCatalog` en
  `src/web/postgres_past_sprint_catalog_integration_test.go`:
  - `FinishedSprints` de «Alfa» → números [2, 1] con objetivo y fechas; el 3 no está;
  - `FinishedSprints` de «Beta» → vacío, sin error; los de «Gamma» no aparecen con «Alfa»;
  - `SprintDetail` del Sprint 2 → «Login» 5 'Hecho', «Tablero» 3 'Hecho', «Perfil» 2 'Hecho' y «Reportes» 8 'En
    progreso', en ese orden;
  - del Sprint 1 → solo «Alta» 3 'Hecho';
  - del Sprint 1 de «Gamma» → «Migración» 5 y «Respaldo» `nil`;
  - del Sprint 2 de «Gamma» → `Stories` vacío, sin error;
  - el Sprint 3 (en curso), el Sprint 1 de «Gamma» pedido con «Alfa» y el identificador 2147483647 →
    `KindOf(err) == KindNotFound`.
- [x] T007 Ejecutar T006, confirmar que falla porque el código no existe y guardar la salida en `evidencia-red.txt`,
  sección "Catálogo".

### Implementación (GREEN)

- [x] T008 Crear los dos archivos del catálogo:
  - `src/web/past_sprint_catalog.go`, con `PastSprintSummary`, `PastSprintStory`, `PastSprintDetail`,
    `PastSprintCatalog` y `PastSprintMetrics` (`CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)`);
  - `src/web/postgres_past_sprint_catalog.go` (`PostgreSQLPastSprintCatalog`, `NewPostgreSQLPastSprintCatalog`).
    `FinishedSprints` con `ORDER BY number DESC`. `SprintDetail` con `BeginTx` (`ReadOnly: true`, `Isolation:
    sql.LevelRepeatableRead`) y las dos consultas de `data-model.md`: el Sprint filtrado por `id`, `project_id` y
    `status = 'Finalizado'` (sin fila → `NotFound(...)`) y el registro con `ORDER BY r.story_id`. Los otros errores
    se envuelven con `%w`.
- [x] T009 Ejecutar T006 en verde.

**Punto de control**: el catálogo devuelve los datos propios de las dos pantallas.

## Fase 3: Bloque Lista ("Sprints anteriores")

**Objetivo**: la sección `GET /sprints-anteriores` con los Sprints finalizados del proyecto activo.

**Prueba independiente**: abrir "Sprints anteriores" con «Alfa» y con «Beta» y comprobar los valores de "Datos de
ejemplo".

### Pruebas (RED)

- [x] T010 [P] [E1, E6] Escribir la parte de lista de `TestPastSprintView` en `src/web/past_sprint_view_test.go`:
  - `pastSprintDateText` → "01/09/2026";
  - `buildPastSprintListView` con los Sprints 2 y 1 de «Alfa» → "Sprint 2", "Gestionar perfiles y reportes", "Del
    01/09/2026 al 14/09/2026", enlace `/sprints-anteriores/{id}`, y después "Sprint 1";
  - con una lista vacía → `Empty` = "Aún no hay Sprints anteriores".
- [x] T011 [E1, E2, E6] Escribir el servidor de prueba y las pruebas de la lista en
  `src/web/past_sprint_handlers_test.go`, con el catálogo y HU-07 reales y los datos de T005:
  - `TestHU16_Listado`: con «Alfa» activo, 200; "Sprint 2" aparece antes que "Sprint 1", cada uno con su objetivo y
    sus fechas; "Sprint 3" y "Exportar e informar" no aparecen; nada de «Gamma»;
  - `TestHU16_SinSprintsAnteriores`: con «Beta», "Aún no hay Sprints anteriores";
  - `TestHU16_SinProyecto` (lista): 303 a `/contexto/proyecto?volver_a=%2Fsprints-anteriores`.
- [x] T012 [P] [E1, E6] Escribir los escenarios BDD "Listado de Sprints finalizados" y "Sin Sprints finalizados" en
  `test/web_past_sprints_bdd_test.go`, con datos propios.
- [x] T013 Ejecutar T010–T012, confirmar que fallan (la ruta no existe) y guardar la salida en `evidencia-red.txt`,
  sección "Lista".

### Implementación (GREEN)

- [x] T014 Crear los archivos de la lista:
  - `src/web/past_sprint_view.go`, con todos los textos de la tabla de `data-model.md`, `pastSprintDateText`,
    `pastSprintItem`, `pastSprintListView` y `buildPastSprintListView`;
  - `src/web/past_sprint_handlers.go`, con `RegisterPastSprints(s, catalog, calc)`: argumentos `nil` → error sin
    registrar; `Section{Name: TitlePastSprints, Path: "/sprints-anteriores", RequiresProject: true}`; por ahora,
    `Routes{"GET /sprints-anteriores": …}` con los pasos de la lista de `data-model.md`;
  - la plantilla `src/web/templates/past_sprints.html`, sin formularios ni `<script>`.
- [x] T015 Agregar en `src/cmd/server/main.go` el import de `src/metrics` (si no está) y, después del último bloque de
  registro (hoy el de HU-24, `RegisterEffort`), el bloque que llama a `web.RegisterPastSprints(srv,
  web.NewPostgreSQLPastSprintCatalog(database), metrics.NewService(metrics.NewPostgreSQLMetricsStore(database)))`. Si
  falla, cerrar `ln` y `database`, como los otros bloques.
- [x] T016 Ejecutar T010–T012 en verde.

**Punto de control**: el Scrum Master ve la lista de Sprints anteriores del proyecto activo.

## Fase 4: Bloque Detalle (`/sprints-anteriores/{id}`) 🎯 MVP visible

**Objetivo**: el detalle de un Sprint finalizado con objetivo, fechas, velocidad de HU-07 y las dos tablas.

**Prueba independiente**: abrir el detalle de cada Sprint de "Datos de ejemplo" y comprobar tablas, mensajes y
velocidad.

### Pruebas (RED)

- [x] T017 [P] [E3, E4, E5] Escribir la parte de detalle de `TestPastSprintView` en `src/web/past_sprint_view_test.go`:
  - `pastSprintPointsText(nil)` → "Sin estimar";
  - `buildPastSprintDetailView` con los cuatro casos de la tabla de `data-model.md` (Sprint 2, Sprint 1, «Gamma» 1 y
    «Gamma» 2 sin registro): filas en orden, "volvió al Backlog" solo en las no completadas, "Velocidad: 10 Story
    Points", "Todas las historias se completaron", "Ninguna historia se completó", y "No hay registro de las
    historias de este Sprint" sin tablas ni los otros dos mensajes.
- [x] T018 [E3, E4, E5, E8] Escribir las pruebas del detalle en `src/web/past_sprint_handlers_test.go`, con el
  catálogo y HU-07 reales:
  - `TestHU16_Detalle`: Sprint 2 → "Sprint 2", "Objetivo: Gestionar perfiles y reportes", "Del 01/09/2026 al
    14/09/2026", "Velocidad: 10 Story Points"; Completadas «Login» 5, «Tablero» 3 y «Perfil» 2; No completadas
    «Reportes» 8, "En progreso", "volvió al Backlog";
  - `TestHU16_TodasCompletadas`: Sprint 1 → «Alta» 3, "Velocidad: 3 Story Points" y "Todas las historias se
    completaron", sin tabla de no completadas;
  - `TestHU16_NingunaCompletada`: Sprint 1 de «Gamma» (con «Gamma» activo) → "Velocidad: 0 Story Points", "Ninguna
    historia se completó", «Migración» 5 y «Respaldo» "Sin estimar", las dos con "volvió al Backlog";
  - `TestHU16_SinRegistro`: Sprint 2 de «Gamma» → "No hay registro de las historias de este Sprint" y "Velocidad: 2
    Story Points";
  - `TestHU16_VelocidadCoherente`: para los Sprints 1 y 2 de «Alfa» y el 1 de «Gamma», la velocidad mostrada es
    igual a la suma de `story_points` de las filas 'Hecho' de `sprint_story_results` (T001: HU-15), leída con SQL en
    la prueba (3, 10 y 0);
  - `TestHU16_DetalleNoEncontrado`: con «Alfa» activo, 404 de HU-20a para el Sprint 3 (en curso), el Sprint 1 de
    «Gamma», 2147483647, "abc", "0", "-1" y "01". Con un `PastSprintMetrics` que cuenta llamadas, cero llamadas en
    todos. Con un `PastSprintCatalog` que envuelve al real y cuenta llamadas, cero llamadas a `SprintDetail` para
    "abc", "0", "-1" y "01" (FR-010: sin consultar la base).
- [x] T019 [E7, E9] Escribir las pruebas de solo lectura, errores y bordes en `src/web/past_sprint_handlers_test.go`:
  - `TestHU16_SoloLectura`: el detalle del Sprint 2 no tiene `<form`, `<button` ni `<script`. `POST`, `PUT` y
    `DELETE` a `/sprints-anteriores/{id}` y `POST` a `/sprints-anteriores` (del mismo origen) → 405 con la página de
    HU-20a. El registro y las historias no cambian;
  - `TestHU16_ErrorAlConsultar`: con un catálogo que falla en la lista y en el detalle, y con un
    `PastSprintMetrics` que falla, la página de error de HU-20a con 500, sin objetivo, historias, velocidad ni texto
    interno. Con `driver.ErrBadConn`, 503;
  - `TestHU16_SinProyecto` (detalle): 303 a `/contexto/proyecto?volver_a=%2Fsprints-anteriores%2F{id}`;
  - `TestHU16_TextosEscapados`: un proyecto propio con un Sprint finalizado de objetivo `<script>x</script>` y una
    historia `<b>Login & Co</b>` de 3 Story Points, "Hecho" y asignada, con su fila 'Hecho' 3 en el registro, se
    muestra escapado (`&lt;script&gt;`, `&lt;b&gt;`, `&amp;`), en la lista y en el detalle ("Velocidad: 3 Story
    Points");
  - `TestHU16_Rendimiento`: un proyecto con 50 Sprints finalizados y, en uno, 200 filas en el registro (150 "Hecho"
    de 1 Story Point asignadas y 50 devueltas) → la lista y el detalle ("Velocidad: 150 Story Points") en menos de
    500 ms cada uno.
- [x] T020 [P] [E3, E7] Escribir los escenarios BDD "Detalle de un Sprint" y "Solo lectura" en
  `test/web_past_sprints_bdd_test.go`.
- [x] T021 Ejecutar T017–T020, confirmar que fallan (la ruta del detalle no existe) y guardar la salida en
  `evidencia-red.txt`, sección "Detalle".

### Implementación (GREEN)

- [x] T022 Completar los archivos del detalle:
  - `src/web/past_sprint_view.go`: `pastSprintPointsText`, `pastSprintRow`, `pastSprintDetailView` y
    `buildPastSprintDetailView`, que separa por `FinalStatus == string(sprint.Done)` y conserva el orden;
  - `src/web/past_sprint_handlers.go`: la ruta `"GET /sprints-anteriores/{id}"` con los pasos 1 a 4 de
    `data-model.md` (`parseID` → catálogo → `CalculateMetrics` → vista), sin escribir nada antes del paso 4;
  - la plantilla `src/web/templates/past_sprint.html`, sin formularios, botones ni `<script>`.
- [x] T023 Ejecutar T017–T020 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Scrum Master revisa cualquier Sprint anterior con su velocidad y sus historias.

## Fase 5: Cierre

- [ ] T024 REFACTOR:
  - revisar duplicación entre la lista y el detalle (fechas, etiqueta) y extraer una función local sin cambiar el
    comportamiento;
  - comprobar que ningún archivo de HU-07, HU-15, HU-25 ni de otras historias cambió y que no hay migraciones:
    `git diff --stat main` lista solo archivos nuevos y `main.go`.
- [ ] T025 Correr `gofmt -l` sobre los archivos creados (sin salida), `go vet ./...` limpio y
  `go test ./... -p 1 -race` en verde, como en el CI.
- [ ] T026 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde y `cobertura.txt`
  con `go test -count=1 -p 1 -cover ./...`. Anotar la cobertura de los archivos `past_sprint*` de `src/web`
  (`go tool cover -func`).
- [ ] T027 Verificar a mano los escenarios con la aplicación en ejecución (`quickstart.md`, verificación manual).
- [ ] T028 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas, y el manual
  breve de usuario con la sección "Sprints anteriores".
- [ ] T029 [P] Avisar:
  - a HU-10 y HU-25, cómo se lee el registro de cierre (completada = 'Hecho');
  - a HU-15, que HU-16 depende de los nombres de `sprint_story_results`;
  - a HU-07, que HU-16 usa `CalculateMetrics` y su `Velocity`;
  - al equipo, que si hay conflicto en `main.go` al mergear se conservan todos los bloques y un solo import de
    `src/metrics`.
- [ ] T030 Abrir el Pull Request desde `HU-16` citando HU-16, con las evidencias y la confirmación de T001. Después de
  la revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- La **Fase 1** va primero. T001 detiene todo si HU-15 no está en `main`.
- El **Bloque Catálogo (Fase 2)** depende de la Fase 1.
- El **Bloque Lista (Fase 3)** depende de T008 (los tipos y el adaptador).
- El **Bloque Detalle (Fase 4)** depende de T014 (`RegisterPastSprints`, los textos y la vista base).
- La **Fase 5** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- `main.go` es el único archivo compartido (T015). Si hay conflicto al mergear, se conservan todos los bloques.

### Oportunidades de paralelismo

- T010 y T012 en paralelo con T011.
- T017 y T020 en paralelo con T018; T019 comparte archivo con T018 y va después.
- T028 y T029.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T011, T014, T019, T022 |
| FR-002 | T006, T008, T010, T011, T014 |
| FR-003 | T006, T008, T011 |
| FR-004 | T010, T011, T014 |
| FR-005 | T017, T018, T022 |
| FR-006 | T006, T008, T017, T018, T022 |
| FR-007 | T017, T018, T022 |
| FR-008 | T006, T017, T018, T022 |
| FR-009 | T015, T018, T022 |
| FR-010 | T006, T008, T018, T022 |
| FR-011 | T008, T019, T022 |
| FR-012 | T005, T006, T011, T018 |
| FR-013 | T011, T019 |
| FR-014 | T019, T024 |
| FR-015 | T014, T019 |
| SC-001 | T006, T011, T018 |
| SC-002 | T018 |
| SC-003 | T006, T018, T019 |
| SC-004 | T019 |
| SC-005 | T007, T013, T021, T026 |

## Estrategia de implementación

1. Fase 1: confirmar HU-15 y los nombres (T001); sin HU-15 no se sigue. Crear la rama.
2. Bloque Catálogo: el adaptador con los datos como los deja HU-15.
3. Bloque Lista: la sección visible con los Sprints finalizados.
4. Bloque Detalle: el detalle con velocidad y tablas. Es el MVP visible para la Sprint Review.
5. Cierre, evidencias y PR.
