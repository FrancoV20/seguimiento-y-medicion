# Tareas: HU-08 – Visualización de Dashboard del Proyecto

**Entrada**: `specs/008-dashboard/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-06, HU-07, HU-10, HU-14, HU-15, HU-20a y HU-20b integradas en `main`.
**Orden de implementación**: después de HU-10 (y por lo tanto de HU-06 y HU-15). Sin HU-10 en `main` no se crea la
rama (T001).

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 12).
- Cada bloque sigue RED → GREEN. Las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/008-dashboard/evidencia-red.txt`.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- CI (`.github/workflows/ci.yml`): `gofmt -l` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` se corre solo sobre los archivos que se crean o se tocan.
- Las pruebas de integración arman sus propios datos y los borran al final, en este orden: defectos, registro de
  cierre, asignaciones, `sprint_metrics`, esfuerzo, historias, Sprints, proyectos e integrante.

## Fase 1: Preparación

- [x] T001 Confirmar con `git grep -n` los nombres que usa HU-08, ajustar `data-model.md`, `research.md` y las
  consultas si alguno cambió, y quitar las marcas "(T001)":
  - de HU-10: `NewQualityService`, `NewPostgreSQLQualityStore`, `(*QualityService).SprintQuality`,
    `SprintQuality.PointsCompleted` en `src/metrics` y la interfaz `QualityReader` en `src/web`;
  - de HU-06: tabla `defects` con `project_id` y `status` ('Abierto' y 'Resuelto');
  - de HU-15: tabla `sprint_story_results` (`sprint_id`, `story_id`, `final_status`, `story_points`) y cómo quedan
    `user_stories.status`, `active_sprint_id` y `sprint_story_assignments` al cerrar, para copiarlo en los datos de
    prueba;
  - de HU-07: `NewService`, `NewPostgreSQLMetricsStore`, `CalculateMetrics`, `SprintMetrics.DeviationPercentage`,
    `IsPartial`, `Warning` y `roundToTwoDecimals`;
  - de HU-14: `BoardCatalog.ActiveBoard`, `BoardView` (`Number`, `Groups`, `PlannedPoints`, `CompletedPoints`),
    `NewPostgreSQLBoardCatalog` y `projectIDFrom`;
  - de HU-25: si está en `main`, el nombre de su función de formato de la desviación (si existe, se usa en lugar de
    `dashboardDeviationText`; anotarlo en `research.md` §8);
  - que no existan `VelocitySummary` ni `SummarizeVelocity` en `src/metrics`, ni nombres `Dashboard*`/`dashboard*` en
    `src/web`.

  **Si HU-10 no está en `main`, no seguir y avisar al equipo.**
- [x] T002 Crear la rama `HU-08` desde `main` actualizado, solo si no hay cambios sin commitear:
  `git switch main; git pull; git switch -c HU-08`.
- [x] T003 Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y `go vet ./...` en verde antes de empezar.
- [x] T004 Coordinar con quien tenga HU-09, HU-10 y HU-25 el bloque de registro que se suma al final de
  `src/cmd/server/main.go`.

## Fase 2: Bloque Dominio de velocidad

**Objetivo**: la velocidad acumulada y el promedio como regla del dominio, sin base de datos.

**Prueba independiente**: `TestSummarizeVelocity` pasa sin PostgreSQL.

### Pruebas (RED)

- [x] T005 [P] [E5, E6] Escribir `TestSummarizeVelocity` como prueba de tabla en
  `src/metrics/velocity_summary_test.go`, con los casos de `data-model.md`:
  - `[]` → 0 y "No disponible";
  - `[5]` → 5 y "5.00";
  - `[8, 13]` → 21 y "10.50";
  - `[21, 21, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5]` → 92 y "7.67";
  - `[1, 0, 0, 0, 0, 0, 0, 0]` → 1 y "0.13";
  - `[2, 0, 0]` → 2 y "0.67";
  - `[0, 0]` → 0 y "0.00";
  - `SprintCount` igual a la cantidad de valores.
- [x] T006 Ejecutar T005, confirmar que falla porque el código no existe y guardar la salida en `evidencia-red.txt`,
  sección "Dominio de velocidad".

### Implementación (GREEN)

- [x] T007 Crear `src/metrics/velocity_summary.go` con `VelocitySummary` (`SprintCount`, `Total`, `Average`) y
  `SummarizeVelocity`: "No disponible" sin Sprints; si no,
  `roundToTwoDecimals(big.NewRat(int64(Total), int64(SprintCount))).FloatString(2)`. No editar otros archivos de
  `src/metrics`.
- [x] T008 Ejecutar T005 en verde junto con las pruebas existentes de `./src/metrics/...`. Las de HU-07 y HU-10 no
  cambian.

**Punto de control**: el promedio con el redondeo de HU-07 está listo para la pantalla.

## Fase 3: Bloque Catálogo (adaptador PostgreSQL)

**Objetivo**: leer historias por estado, defectos abiertos y Sprints finalizados del proyecto en una lectura
consistente.

**Prueba independiente**: `TestPostgresDashboardCatalog` pasa contra PostgreSQL con datos propios.

### Pruebas (RED)

- [x] T009 Crear los datos de prueba compartidos en `src/web/dashboard_fixtures_test.go` (los usan T010 y T021), con
  un integrante «Ana» para el esfuerzo. Cada proyecto tiene sus propios Sprints; ningún Sprint se usa para dos
  repartos:
  - «Alfa», como la tabla "Datos de ejemplo" de la spec:
    - Sprint 1 "Finalizado": A (8, "Hecho") y P1 (3, volvió a "Pendiente"); registro de cierre A "Hecho" 8 y P1
      "En progreso" 3; A queda asignada;
    - Sprint 2 "Finalizado": B (13, "Hecho", 10 horas estimadas, esfuerzo de 8 y 4 horas) y P2 (2, volvió a
      "Pendiente"); registro B "Hecho" 13 y P2 "En Sprint" 2; B queda asignada;
    - Sprint 3 "Activo": C (1, "En Sprint"), D (3) y E (5, "En progreso"), F (2, "Hecho"), las cuatro asignadas;
    - defectos: 3 "Abierto" (dos detectados en el Sprint 3 y uno en el 2) y 2 "Resuelto" (uno detectado en el 1 y
      resuelto en el 2, otro detectado y resuelto en el 2);
    - cuentas: 2, 1, 2 y 3 historias; Sprint 3 con 11 planificados y 2 completados; velocidades 8 y 13; acumulada 21;
      promedio "10,50"; desviación "+20,00 %"; 3 defectos abiertos;
  - «Beta»: R (3, "Pendiente") y Q (5, "Hecho") en su Sprint 1 "Finalizado" (registro Q "Hecho" 5, asignada), y un
    defecto "Abierto" detectado en ese Sprint. Si se mezclara con «Alfa», cambiarían las cantidades (3 Pendiente, 4
    Hecho), la velocidad y los defectos (4);
  - «Vacío»: sin historias, Sprints ni defectos (escenario 1);
  - «Sin Sprint»: 2 historias "Pendiente", sin Sprints (escenario 4);
  - «Solo activo»: 1 "Pendiente" y su Sprint 1 "Activo" con 1 historia "En Sprint" de 3 (escenario 6);
  - «Sin horas»: su Sprint 1 "Finalizado" con 1 historia "Hecho" de 5 sin horas estimadas (escenario 8: velocidad 5,
    "No disponible", aviso "basado en 0 de 1 historias");
  - «Parcial»: su Sprint 1 "Finalizado" con G (3, "Hecho", 10 horas estimadas, esfuerzo de 8 y 4) y H (2, "Hecho",
    sin horas estimadas; sí tiene Story Points, para pasar la validación de cierre de HU-07): velocidad 5, "+20,00 %"
    y "Cálculo parcial: basado en 1 de 2 historias";
  - «Doce»: Sprints 1 a 12 "Finalizado", cada uno con una historia "Hecho" propia: 21 en los Sprints 1 y 2, y 5 en
    los Sprints 3 a 12 (acumulada 92, promedio "7,67");
  - «Sin historias»: su Sprint 1 "Finalizado" sin historias (barras en 0, velocidad 0, promedio "0,00", "No
    disponible").

  Limpieza en `t.Cleanup`, en el orden del encabezado.
- [x] T010 [P] [E1, E2, E9, E10] Escribir `TestPostgresDashboardCatalog` en
  `src/web/postgres_dashboard_catalog_integration_test.go`:
  - «Alfa»: `Stories` = {2, 1, 2, 3}, `OpenDefects` = 3, `Finished` = Sprints 1 y 2 en ese orden;
  - «Beta»: {1, 0, 0, 1}, 1 defecto, solo su Sprint 1;
  - «Vacío»: todo en cero y `Finished` vacío;
  - «Doce»: 12 finalizados, del 1 al 12;
  - «Solo activo»: el Sprint activo no figura en `Finished`.
- [x] T011 Ejecutar T010, confirmar que falla y guardar la salida en `evidencia-red.txt`, sección "Catálogo".

### Implementación (GREEN)

- [x] T012 Crear `src/web/dashboard_catalog.go` con `DashboardStoryCounts`, `DashboardSprint`, `DashboardSummary`,
  y `DashboardCatalog`, como en `data-model.md`. No redefinir `QualityReader`, `BoardCatalog` ni `MetricsSource`
  (HU-25).
- [x] T013 Crear `PostgreSQLDashboardCatalog` y `NewPostgreSQLDashboardCatalog(db *sql.DB)` en
  `src/web/postgres_dashboard_catalog.go`:
  - `BeginTx` con `ReadOnly: true` e `Isolation: sql.LevelRepeatableRead`, y `defer tx.Rollback()`;
  - las tres consultas de `data-model.md` (cuatro `COUNT(*) FILTER` sobre `user_stories`, defectos "Abierto" y
    Sprints "Finalizado" `ORDER BY number`), todas con `WHERE project_id = $1`;
  - errores envueltos con `%w`, sin `Unavailable` (como `PostgreSQLBoardCatalog`).
- [x] T014 Ejecutar T010 en verde.

**Punto de control**: el catálogo devuelve los datos propios del Dashboard de un proyecto.

## Fase 4: Bloque Vista y gráficos

**Objetivo**: alturas de barras, formatos y armado de los cinco bloques, sin base de datos.

**Prueba independiente**: las pruebas unitarias de `src/web/dashboard_chart_test.go` y `dashboard_view_test.go`
pasan sin PostgreSQL.

### Pruebas (RED)

- [x] T015 [P] [E2, E5, E11] Escribir en `src/web/dashboard_chart_test.go`:
  - `TestDashboardBarHeights` (tabla, alto 120): `[2, 1, 2, 3]` → `[80, 40, 80, 120]`; `[8, 13]` → `[74, 120]`;
    `[0, 0, 0, 0]` → todo 0; `[1, 1000]` → `[1, 120]`; `[]` → `[]`;
  - `TestNewDashboardChart`: posiciones `X` e `Y` de cada barra, `Width` y `Height`, y el `AriaLabel` "Historias por
    estado: Pendiente 2, En Sprint 1, En progreso 2, Hecho 3".
- [x] T016 [P] [E1, E4, E5, E6, E7, E8, E9] Escribir `TestDashboardView` en `src/web/dashboard_view_test.go`:
  - la desviación con `metricsDeviationText` de HU-25 (T001), dentro de `buildDashboardView`: "20.00" →
    "+20,00 %", "No disponible" sin cambios;
  - `dashboardDecimalText("10.50")` → "10,50";
  - `dashboardOpenDefectsText`: 0 → "No hay defectos abiertos", 1 → "1 defecto abierto", 3 → "3 defectos
    abiertos";
  - `buildDashboardView`:
    - sin historias, sin finalizados y sin Sprint activo → `NoData`;
    - un Sprint sin historias → no es `NoData`;
    - sin Sprint activo → `HasActiveSprint` falso;
    - sin finalizados → `HasFinished` falso;
    - con 12 finalizados → 10 barras (Sprints 3 a 12), `LastTenNotice` y acumulada de los 12;
    - la desviación es la del último finalizado;
    - `PartialWarning` solo si `IsPartial`.
- [x] T017 Ejecutar T015 y T016, confirmar que fallan y guardar la salida en `evidencia-red.txt`, sección "Vista y
  gráficos".

### Implementación (GREEN)

- [x] T018 Crear `src/web/dashboard_chart.go` con las constantes, `dashboardBarHeights`, `dashboardBar`,
  `dashboardChart` y `newDashboardChart`, con las reglas de `data-model.md` (máximo 0 → todo 0 sin dividir; mitad
  hacia arriba; un valor mayor que 0 mide al menos 1).
- [x] T019 Crear `src/web/dashboard_view.go` con los textos de la tabla de `data-model.md`, `dashboardDecimalText`,
  `dashboardOpenDefectsText`, `dashboardView`,
  `dashboardVelocityRow` y `buildDashboardView` (usa `metrics.SummarizeVelocity`).
- [x] T020 Ejecutar T015 y T016 en verde.

**Punto de control**: con datos en memoria se arman los cinco bloques y los dos gráficos.

## Fase 5: Bloque Pantalla ("Dashboard") 🎯 MVP visible

**Objetivo**: la sección `GET /dashboard` con los cinco bloques.

**Prueba independiente**: abrir el Dashboard de «Alfa» y comprobar los valores de "Datos de ejemplo"; repetir con
los proyectos de los escenarios vacíos y con una falla.

### Pruebas (RED)

- [x] T021 [E1, E2, E3, E4, E5, E6, E7, E8, E9, E10, E11] Escribir el servidor de prueba y las pruebas de pantalla en
  `src/web/dashboard_handlers_test.go`, con el catálogo, el Tablero, `QualityService` y `NewHU07MetricsSource` reales y
  los datos de T009:
  - `TestHU08_SinDatos` («Vacío»): solo "Aún no hay datos suficientes para calcular métricas", sin `<svg`;
  - `TestHU08_HistoriasPorEstado` («Alfa»): tabla 2, 1, 2 y 3, y barras de alto 80, 40, 80 y 120;
  - `TestHU08_SprintEnCurso`: "Sprint 3 en curso", 1, 2 y 1, "11 Story Points planificados y 2 completados";
  - `TestHU08_SinSprintEnCurso` («Sin Sprint»): "No hay un Sprint en curso" y la tabla de historias con 2
    Pendiente;
  - `TestHU08_Velocidad`: barras "Sprint 1" (8, alto 74) y "Sprint 2" (13, alto 120), "Velocidad acumulada: 21
    Story Points" y "Promedio: 10,50 Story Points por Sprint";
  - `TestHU08_UltimosDiezSprints` («Doce»): barras de los Sprints 3 a 12 (las etiquetas se comparan completas, para
    que "Sprint 1" no coincida con "Sprint 12"), acumulada 92, promedio "7,67" y el aviso de los últimos 10;
  - `TestHU08_SinSprintsFinalizados` («Solo activo»): el mensaje en velocidad y desviación, las historias y "Sprint
    1 en curso";
  - `TestHU08_Desviacion`: "Desviación de esfuerzo del Sprint 2: +20,00 %", sin aviso parcial;
  - `TestHU08_DesviacionParcial` («Parcial»): "+20,00 %" y "Cálculo parcial: basado en 1 de 2 historias";
  - `TestHU08_DesviacionNoDisponible` («Sin horas»): "No disponible", "Velocidad acumulada: 5 Story Points" y
    "Cálculo parcial: basado en 0 de 1 historias";
  - `TestHU08_DefectosAbiertos`: "3 defectos abiertos";
  - `TestHU08_OtroProyecto`: con «Beta» activo, 1 Pendiente, 1 Hecho, una barra de 5 y "1 defecto abierto"; con
    «Alfa» activo, nada de «Beta»;
  - `TestHU08_SprintSinHistorias` («Sin historias»): cuatro barras en 0, tabla en ceros, velocidad 0, "0,00" y "No
    disponible";
  - `TestHU08_GraficosSinJavaScript`: sin `<script`, dos `<svg role="img"` con `aria-label` y dos tablas con los
    mismos valores que las barras;
  - `TestHU08_CoherenciaConHU07`: en los Sprints 1 y 2 de «Alfa», cada barra es igual a
    `SprintQuality(...).PointsCompleted` y a `CalculateMetrics(id).Velocity`.

  Además se comprueba que abrir el Dashboard no escribe filas en `sprint_metrics`.
- [x] T022 [E12] Escribir las pruebas de errores y bordes en `src/web/dashboard_handlers_test.go`, con dobles
  que fallan:
  - `TestHU08_ErrorAlLeer`: con el catálogo, el Tablero, el `QualityReader` y la fuente de desviación fallando (uno
    por subtest, y uno con `ClosureValidationError`), se ve la página de error de HU-20a con 500, sin `<svg`, sin
    números ni texto interno; con `driver.ErrBadConn`, 503;
  - `TestHU08_SinProyecto`: 303 a `/contexto/proyecto?volver_a=%2Fdashboard`;
  - `TestHU08_PostNoPermitido`: 405;
  - `TestHU08_Rendimiento`: 20 Sprints finalizados, cada uno con 10 historias (5 "Hecho" de 2 y 5 que volvieron a
    "Pendiente"), y 100 defectos (60 "Abierto" y 40 "Resuelto"), en menos de 500 ms; muestra 10 barras (Sprints 11 a
    20), "Velocidad acumulada: 200 Story Points", "Promedio: 10,00 Story Points por Sprint" y "60 defectos
    abiertos".
- [x] T023 [P] [E1, E2, E5, E7] Escribir los escenarios BDD en `test/web_dashboard_bdd_test.go`: "Proyecto sin datos
  suficientes", "Historias por estado", "Velocidad del equipo" y "Desviación de esfuerzo del último Sprint".
- [x] T024 Ejecutar T021–T023, confirmar que fallan (la ruta no existe) y guardar la salida en `evidencia-red.txt`,
  sección "Pantalla".

### Implementación (GREEN)

- [x] T025 Crear `RegisterDashboard(s, catalog, board, quality, deviation MetricsSource)` en `src/web/dashboard_handlers.go`:
  - si falta algún argumento, devolver un error sin registrar;
  - `Section{Name: TitleDashboard, Path: "/dashboard", RequiresProject: true}` y `Routes{"GET /dashboard": …}`;
  - los pasos 1 a 6 de `data-model.md`: cualquier error va a `Fail` antes de escribir la respuesta.
- [x] T026 Crear `src/web/templates/dashboard.html`:
  - el mensaje único si `NoData`;
  - si no, los cinco bloques en orden, con `<svg role="img" aria-label="…" viewBox="…">` y un `<rect>` y un `<text>`
    por barra, cada gráfico con su `<table>`;
  - sin finalizados, el mensaje en lugar de velocidad y desviación;
  - sin `<script>` ni formularios.
- [x] T027 Agregar al final de los bloques de registro de `src/cmd/server/main.go` (hoy después del de HU-16,
  `RegisterPastSprints`) el bloque que llama a `web.RegisterDashboard` con `web.NewPostgreSQLDashboardCatalog(database)`,
  `web.NewPostgreSQLBoardCatalog(database)`, `metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(database))`
  y `web.NewHU07MetricsSource(database)`. Si falla, cerrar `ln` y `database`, como los
  otros bloques.
- [x] T028 Ejecutar T021–T023 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Scrum Master ve la salud del proyecto en una pantalla.

## Fase 6: Cierre

- [x] T029 REFACTOR:
  - revisar si la vista y el handler tienen duplicación y extraer una función local sin cambiar el comportamiento;
  - comprobar que ningún archivo de HU-07, HU-10, HU-14 ni HU-25 cambió y que no hay migraciones: `git diff --stat
    main` lista solo archivos nuevos y `main.go`.
- [x] T030 Correr `gofmt -l` sobre los archivos creados y sobre `main.go`, que no tiene que listar nada; después
  `go vet ./...` limpio y `go test ./... -p 1 -race` en verde, como en el CI.
- [x] T031 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde.
- [x] T032 Guardar `cobertura.txt` con `go test -count=1 -p 1 -cover ./...` y verificar al menos 80% en
  `src/metrics`.
- [x] T033 Verificar a mano los escenarios con la aplicación en ejecución (`quickstart.md`, verificación manual)
  y registrar el resultado en `evidencia-manual.txt`, con "Realizó:" y el "Recorrido del revisor".
- [x] T034 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas, y el manual
  breve de usuario con la sección "Dashboard".
- [x] T035 [P] Avisar:
  - a HU-09, que el Dashboard está en `main` y qué datos reúne;
  - a HU-10, HU-14 y HU-25, que no se tocaron sus archivos;
  - al equipo, que si hay conflicto en `main.go` al mergear se conservan todos los bloques.
- [x] T036 Abrir el Pull Request desde `HU-08` citando HU-08, con las evidencias y la confirmación de T001.
- [ ] T037 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- La **Fase 1** va primero. T001 bloquea todo: sin HU-10 en `main` no se crea la rama.
- El **Bloque Dominio de velocidad (Fase 2)** depende de la Fase 1.
- El **Bloque Catálogo (Fase 3)** depende de la Fase 1; puede ir en paralelo con la Fase 2.
- El **Bloque Vista y gráficos (Fase 4)** depende de T007 (usa `SummarizeVelocity`) y de T012 (usa los tipos del
  catálogo).
- El **Bloque Pantalla (Fase 5)** depende de las Fases 2 a 4.
- La **Fase 6** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- `main.go` es el único archivo compartido (T027). Si hay conflicto al mergear, se conservan todos los bloques.

### Oportunidades de paralelismo

- Fase 2 y Fase 3 completas.
- T015 y T016.
- T023 en paralelo con T021; T022 comparte archivo con T021 y va después.
- T018 y T019, después de T017.
- T034 y T035.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T022, T025 |
| FR-002 | T009, T010, T013, T015, T018, T021, T026 |
| FR-003 | T021, T019 |
| FR-004 | T016, T021 |
| FR-005 | T009, T016, T019, T021 |
| FR-006 | T005, T007, T016, T021 |
| FR-007 | T016, T019, T021 |
| FR-008 | T010, T013, T016, T021 |
| FR-009 | T016, T021, T026 |
| FR-010 | T016, T021, T026 |
| FR-011 | T009, T010, T013, T021 |
| FR-012 | T015, T018, T021, T026 |
| FR-013 | T015, T018 |
| FR-014 | T022, T025 |
| FR-015 | T013, T029 |
| FR-016 | T022, T025 |
| FR-017 | T005, T007, T012, T021, T029 |
| FR-018 | T019, T022 |
| SC-001 | T005, T010, T015, T016, T021 |
| SC-002 | T021 |
| SC-003 | T010, T021 |
| SC-004 | T022 |
| SC-005 | T022 |
| SC-006 | T021, T026 |
| SC-007 | T006, T011, T017, T024, T031, T032 |

## Estrategia de implementación

1. Fase 1: confirmar HU-10 (y con ella HU-06 y HU-15) en `main` (T001) y crear la rama.
2. Bloques Dominio de velocidad y Catálogo, en paralelo.
3. Bloque Vista y gráficos: los cinco bloques armados en memoria.
4. Bloque Pantalla: la sección "Dashboard". Es el MVP visible para la Sprint Review.
5. Cierre, evidencias y PR.
