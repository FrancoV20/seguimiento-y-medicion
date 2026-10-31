# Tareas: HU-12 – Consulta del Estado de un Proyecto

**Entrada**: `specs/012-project-status/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-01, HU-14, HU-20a y HU-20b integradas en `main` (ya están). Para la verificación manual,
además HU-22 y HU-23.

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 10).
- Cada bloque sigue RED → GREEN. Las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/012-project-status/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- CI (`.github/workflows/ci.yml`): `gofmt -l` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` se corre solo sobre los archivos que se crean; `main.go` se edita a mano, sin cambiar su fin de línea.
- Las pruebas de integración arman sus propios datos (research.md §10) y los borran al final, en este orden:
  asignaciones, historias, Sprints y proyectos. Cada escenario usa su propio proyecto.
- El reloj de las pruebas es 28/10/2026 10:30 (hora local).

## Fase 1: Preparación

- [x] T001 Confirmar con `git grep -n` los nombres que usa HU-12 y ajustar `data-model.md` si alguno cambió:
  - de HU-14: `BoardCatalog.ActiveBoard`, `BoardView` (`Label`, `Goal`, `StartDate`, `EndDate`),
    `NewPostgreSQLBoardCatalog` y `projectIDFrom` en `src/web`;
  - de HU-20a/b: `Server.Register`, `Section.RequiresProject`, `Routes`, `Renderer.Page`, `Renderer.Fail`,
    `NotFound` y `ActiveContextFrom`;
  - de HU-01: las columnas `projects.name`, `start_date` y `end_date`, y que `src/project` sigue sin importar otros
    paquetes del repositorio;
  - que no existan `DateStatus`, `StatusOn` ni `ProgressPercentage` en `src/project`, nombres `ProjectStatus*` en
    `src/web`, ni la ruta `/estado`;
  - si HU-07 exportó una función de redondeo, anotarlo en research.md §3 para reutilizarla.

  Revisar también "Planificación de proyecto.pdf": si contradice la spec, corregirla antes de T002.
- [x] T002 Crear la rama `HU-12` desde `main` actualizado, solo si no hay cambios sin commitear:
  `git switch main; git pull; git switch -c HU-12`.
- [x] T003 Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y `go vet ./...` en verde antes de empezar.
- [x] T004 Coordinar con quien tenga HU-08, HU-09, HU-10, HU-11 y HU-25 el bloque de registro que se suma al final de
  `src/cmd/server/main.go`, y con HU-09 el contrato `StatusOn` / `ProgressPercentage` de `data-model.md`.

## Fase 2: Bloque Reglas (estado y avance)

**Objetivo**: el estado por fechas y el porcentaje de avance como reglas puras del dominio.

**Prueba independiente**: las pruebas de `src/project/status_test.go` pasan sin PostgreSQL.

### Pruebas (RED)

- [x] T005 [E1, E2, E3, E4, E5, E6] Escribir en `src/project/status_test.go`, como pruebas de tabla con los casos de
  `data-model.md`:
  - `TestStatusOn`: «Alfa» "En curso", «Beta» "No iniciado", «Gamma» "Finalizado", «Delta» y «Épsilon» "En curso" a
    las 00:00 y a las 23:59 del 28/10/2026, el día anterior al inicio "No iniciado", el día siguiente al fin
    "Finalizado", y el caso en UTC-3 (inicio 28/10/2026 a medianoche UTC, hoy 27/10/2026 22:00 en UTC-3) "No
    iniciado";
  - `TestProgressPercentage`: 4 y 8 → "50.00"; 3 y 3 → "100.00"; 0 y 0 → "0.00"; 0 y 5 → "0.00"; 2 y 3 → "66.67"; 1 y
    3 → "33.33"; 1 y 8 → "12.50"; 4 y 7 → "57.14"; 1 y 6 → "16.67"; 7 y 11 → "63.64"; 1 y 32 → "3.13".
- [x] T006 Ejecutar T005, confirmar que falla porque el código no existe y guardar la salida en `evidencia-red.txt`,
  sección "Reglas".

### Implementación (GREEN)

- [x] T007 Crear `src/project/status.go` con `DateStatus`, las tres constantes ("No iniciado", "En curso",
  "Finalizado"), `StatusOn` (compara `(año, mes, día)` de `t.Date()`; los dos bordes son "En curso") y
  `ProgressPercentage` (`total <= 0` → "0.00"; si no, `(done*10000*2 + total) / (2*total)` centésimas, con
  `fmt.Sprintf("%d.%02d", …)`). No editar otros archivos de `src/project`.
- [x] T008 Ejecutar T005 en verde junto con las pruebas existentes de `./src/project/...`.

**Punto de control**: HU-09 ya puede programar contra `StatusOn` y `ProgressPercentage`.

## Fase 3: Bloque Catálogo (adaptador PostgreSQL)

**Objetivo**: leer nombre, fechas y cantidades por estado del proyecto en una lectura consistente.

**Prueba independiente**: `TestPostgresProjectStatusCatalog` pasa contra PostgreSQL con datos propios.

### Pruebas (RED)

- [x] T009 Crear los datos de prueba en `src/web/project_status_fixtures_test.go` (los usan T010, T015 y T016), con
  los proyectos del Issue y research.md §10:
  - «Alfa» (01/10/2026 al 30/11/2026): Sprint 1 "Finalizado" (05/10/2026 al 18/10/2026) con 3 historias "Hecho"
    asignadas; Sprint 2 "Activo" ("Pagos online", 19/10/2026 al 01/11/2026) con 1 "Hecho", 1 "En Sprint" y 1 "En
    progreso" asignadas; 2 "Pendiente" sin asignar (2 + 1 + 1 + 4 = 8);
  - «Beta» (01/12/2026 al 31/01/2027), sin historias ni Sprints;
  - «Gamma» (01/08/2026 al 30/09/2026): Sprint 1 "Finalizado" (01/08/2026 al 14/08/2026) con sus 3 historias
    "Hecho";
  - «Delta» (28/10/2026 al 15/12/2026) y «Épsilon» (01/09/2026 al 28/10/2026), sin historias ni Sprints.

  Limpieza en `t.Cleanup`, en el orden del encabezado.
- [x] T010 [E1, E3, E6, E8, E10] Escribir `TestPostgresProjectStatusCatalog` en
  `src/web/postgres_project_status_catalog_integration_test.go`:
  - «Alfa»: nombre, fechas 01/10/2026 y 30/11/2026, cantidades {2, 1, 1, 4} y total 8;
  - «Gamma»: {0, 0, 0, 3}; si se mezclara con «Alfa», el total daría 11;
  - «Beta»: {0, 0, 0, 0};
  - identificador 2147483647 → `KindOf(err) == KindNotFound`.
- [x] T011 Ejecutar T010, confirmar que falla y guardar la salida en `evidencia-red.txt`, sección "Catálogo".

### Implementación (GREEN)

- [x] T012 Crear `src/web/project_status_catalog.go` (`ProjectStatusCounts` con `Total()`, `ProjectStatusSummary`,
  `ProjectStatusCatalog`) y `src/web/postgres_project_status_catalog.go` (`PostgreSQLProjectStatusCatalog`,
  `NewPostgreSQLProjectStatusCatalog`): `BeginTx` con `ReadOnly: true` e `Isolation: sql.LevelRepeatableRead`, las dos
  consultas de `data-model.md`, sin fila de proyecto → `NotFound(...)`, otros errores envueltos con `%w`.
- [x] T013 Ejecutar T010 en verde.

**Punto de control**: el catálogo devuelve los datos propios de la pantalla.

## Fase 4: Bloque Pantalla ("Estado del proyecto") 🎯 MVP visible

**Objetivo**: la sección `GET /estado` con el estado, el avance, las historias por estado y el Sprint en curso.

**Prueba independiente**: con el reloj en 28/10/2026, abrir "Estado del proyecto" con cada proyecto del Issue y
comprobar los valores de "Datos de ejemplo".

### Pruebas (RED)

- [x] T014 [P] [E1, E6, E7] Escribir `TestProjectStatusView` en `src/web/project_status_view_test.go`:
  `projectStatusPercentText("50.00")` → "50,00 %" y "0.00" → "0,00 %"; `projectStatusDateText` → "01/10/2026"; y
  `buildProjectStatusView` con los datos de «Alfa» (Sprint 2 encontrado) y de «Beta» (sin Sprint): estado, "4 de 8
  historias completadas (50,00 %)" y "0 de 0 historias completadas (0,00 %)", cuatro filas en orden con ceros,
  "Sprint 2 en curso", "Objetivo: Pagos online", "Del 19/10/2026 al 01/11/2026" y "No hay un Sprint en curso".
- [x] T015 [E1, E2, E3, E4, E5, E6, E7, E8] Escribir el servidor de prueba y las pruebas de pantalla en
  `src/web/project_status_handlers_test.go`, con el catálogo y el Tablero reales, el reloj fijo y los datos de T009:
  - `TestHU12_ProyectoEnCurso`: «Alfa» → "Estado: En curso", "Del 01/10/2026 al 30/11/2026", "Sprint 2 en curso",
    "Objetivo: Pagos online", "Del 19/10/2026 al 01/11/2026", Pendiente 2, En Sprint 1, En progreso 1, Hecho 4 y "4
    de 8 historias completadas (50,00 %)";
  - `TestHU12_EstadoPorFechas`: «Beta» "No iniciado", «Gamma» "Finalizado" con "3 de 3 historias completadas
    (100,00 %)", «Delta» y «Épsilon» "En curso";
  - `TestHU12_SinHistorias`: «Beta» → cuatro cantidades en 0 y "0 de 0 historias completadas (0,00 %)", sin "NaN" ni
    error;
  - `TestHU12_SinSprintEnCurso`: «Gamma» → "No hay un Sprint en curso" y el resto igual;
  - `TestHU12_OtroProyecto`: con «Alfa» activo, 8 historias y el Sprint 2 de «Alfa»; no aparece "7 de 11" ni nada de
    «Gamma».

  Todas sin `<script` ni formularios.
- [x] T016 [E9, E10] Escribir las pruebas de contexto, errores y bordes en `src/web/project_status_handlers_test.go`:
  - `TestHU12_SinProyecto`: 303 a `/contexto/proyecto?volver_a=%2Festado`;
  - `TestHU12_ErrorAlConsultar`: con un catálogo que falla y con un `BoardCatalog` que falla, la página de error de
    HU-20a con 500, sin los datos de la pantalla ni texto interno; con `driver.ErrBadConn`, 503; con
    `NotFound`, 404;
  - `TestHU12_PostNoPermitido`: `POST /estado` → 405;
  - `TestHU12_TextosEscapados`: un proyecto propio llamado `<b>Alfa & Co</b>`, con un Sprint activo de objetivo
    `<script>x</script>`, se muestra escapado (`&lt;b&gt;`, `&amp;`, `&lt;script&gt;`) y la página no tiene `<script`;
  - `TestHU12_Rendimiento`: un proyecto con 1000 historias (250 por estado) y un Sprint activo con 50 → "250 de 1000
    historias completadas (25,00 %)" en menos de 500 ms.
- [x] T017 [P] [E1, E2, E3, E9] Escribir los escenarios BDD en `test/web_project_status_bdd_test.go`, con datos
  propios: "Proyecto en curso", "Proyecto no iniciado", "Proyecto finalizado" y "Sin proyecto seleccionado".
- [x] T018 Ejecutar T014–T017, confirmar que fallan (la ruta no existe) y guardar la salida en `evidencia-red.txt`,
  sección "Pantalla".

### Implementación (GREEN)

- [x] T019 Crear `src/web/project_status_view.go` con los textos de la tabla de `data-model.md`,
  `projectStatusDateText`, `projectStatusPercentText`, `projectStatusRow`, `projectStatusView` y
  `buildProjectStatusView` (usa `project.StatusOn` y `project.ProgressPercentage`).
- [x] T020 Crear `RegisterProjectStatus(s, catalog, board, now)` en `src/web/project_status_handlers.go` (reloj `nil` =
  `time.Now`; argumentos `nil` → error sin registrar; `Section{Name: TitleProjectStatus, Path: "/estado",
  RequiresProject: true}`; `Routes{"GET /estado": …}`; los pasos 1 a 3 de `data-model.md`, sin escribir nada antes de
  tener los dos datos) y la plantilla `src/web/templates/project_status.html`, sin `<script>` ni formularios.
- [x] T021 Agregar en `src/cmd/server/main.go`, después del último bloque de registro (hoy el de HU-23,
  `RegisterSprints`) y antes de `srv.Serve`, el bloque que llama a `web.RegisterProjectStatus(srv, web.NewPostgreSQLProjectStatusCatalog(database),
  web.NewPostgreSQLBoardCatalog(database), nil)`. Si falla, cerrar `ln` y `database`, como los otros bloques.
- [x] T022 Ejecutar T014–T017 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Scrum Master ve el estado y el avance del proyecto activo.

## Fase 5: Cierre

- [x] T023 REFACTOR:
  - revisar duplicación en la vista y el handler y extraer una función local sin cambiar el comportamiento;
  - comprobar que ningún archivo de HU-01, HU-11, HU-14 ni HU-21 cambió y que no hay migraciones: `git diff --stat
    main` lista solo archivos nuevos y `main.go`.
- [x] T024 Correr `gofmt -l` sobre los archivos creados (sin salida), `go vet ./...` limpio y
  `go test ./... -p 1 -race` en verde, como en el CI. (-race no se pudo correr en Windows: sin cgo; lo verifica el CI)
- [x] T025 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde, y `cobertura.txt`
  con `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en `src/project`.
- [x] T026 Verificar a mano los escenarios con la aplicación en ejecución (`quickstart.md`, verificación manual).
- [x] T027 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas, y el manual
  breve de usuario con la sección "Estado del proyecto".
- [x] T028 [P] Avisar:
  - a HU-09, los nombres finales `project.StatusOn` y `project.ProgressPercentage` para su T001;
  - a HU-08, que puede reutilizar `ProjectStatusCatalog` para las historias por estado;
  - a HU-11, HU-14 y HU-21, que no se tocaron sus archivos;
  - al equipo, que si hay conflicto en `main.go` al mergear se conservan todos los bloques.
- [ ] T029 Abrir el Pull Request desde `HU-12` citando HU-12, con las evidencias y la confirmación de T001.
- [ ] T030 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- La **Fase 1** va primero.
- El **Bloque Reglas (Fase 2)** depende de la Fase 1.
- El **Bloque Catálogo (Fase 3)** depende de la Fase 1; puede ir en paralelo con la Fase 2.
- El **Bloque Pantalla (Fase 4)** depende de T007 (la vista usa las reglas) y de T012 (los tipos y el adaptador).
- La **Fase 5** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- `main.go` es el único archivo compartido (T021). Si hay conflicto al mergear, se conservan todos los bloques.

### Oportunidades de paralelismo

- Fase 2 y Fase 3 completas.
- T014 y T017 en paralelo con T015; T016 comparte archivo con T015 y va después.
- T027 y T028.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T016, T020 |
| FR-002 | T010, T014, T015, T019 |
| FR-003 | T005, T007, T015, T019 |
| FR-004 | T005, T007, T028 |
| FR-005 | T005, T007, T014, T015 |
| FR-006 | T005, T007, T028 |
| FR-007 | T010, T012, T014, T015, T020 |
| FR-008 | T014, T015, T019 |
| FR-009 | T014, T015, T019 |
| FR-010 | T009, T010, T015 |
| FR-011 | T012, T016, T020 |
| FR-012 | T010, T012, T016 |
| FR-013 | T016, T017 |
| FR-014 | T012, T023 |
| FR-015 | T016, T019 |
| SC-001 | T005, T010, T015 |
| SC-002 | T010, T015 |
| SC-003 | T016 |
| SC-004 | T016 |
| SC-005 | T006, T011, T018, T025 |

## Estrategia de implementación

1. Fase 1: confirmar nombres (T001) y crear la rama.
2. Bloques Reglas y Catálogo, en paralelo. Al terminar Reglas, HU-09 ya puede usar el contrato.
3. Bloque Pantalla: la sección "Estado del proyecto". Es el MVP visible para la Sprint Review.
4. Cierre, evidencias y PR.
