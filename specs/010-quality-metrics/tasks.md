# Tareas: HU-10 – Cálculo de Métricas de Calidad y Avance

**Entrada**: `specs/010-quality-metrics/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-06, HU-07, HU-14, HU-15, HU-20a y HU-20b integradas en `main`. HU-19 hace falta solo
para la demo.
**Orden de implementación**: después de HU-06 (tabla `defects`) y HU-15 (tabla `sprint_story_results`). Sin las dos
en `main` no se crea la rama (T001).

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 9, 10a, 10b, 10c, 11 y 12).
- Cada bloque sigue RED → GREEN. Las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/010-quality-metrics/evidencia-red.txt`.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- CI (`.github/workflows/ci.yml`): `gofmt -l` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` se corre solo sobre los archivos que se crean o se tocan.
- Las pruebas de integración arman sus propios datos y los borran al final: historias primero y el proyecto
  después.

## Fase 1: Preparación

- [ ] T001 Confirmar con `git grep -n` los nombres que usa HU-10, ajustar `data-model.md` y las consultas si alguno
  cambió, y quitar las marcas "(T001)":
  - de HU-06: tabla `defects` y columnas `project_id`, `status` ('Abierto' y 'Resuelto'), `detection_sprint_id` y
    `resolution_sprint_id`;
  - de HU-15: tabla `sprint_story_results` y columnas `sprint_id`, `story_id`, `final_status` y `story_points`;
  - de HU-07: `roundToTwoDecimals`, `ErrSprintNotFinished`, `NewService`, `NewPostgreSQLMetricsStore` y
    `CalculateMetrics` en `src/metrics`;
  - que no existan `ErrSprintNotFound`, `QualityStore`, `QualityService` ni `SprintQuality` en `src/metrics`, ni
    tipos `Quality*` en `src/web`.

  Si HU-06 o HU-15 no están en `main`, no seguir y avisar al equipo.
- [ ] T002 Crear la rama `HU-10` desde `main` actualizado, solo si no hay cambios sin commitear:
  `git switch main; git pull; git switch -c HU-10`.
- [ ] T003 Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y `go vet ./...` en verde antes de empezar.
- [ ] T004 Coordinar dos cosas:
  - con quien tenga HU-08, HU-09 y HU-25, el bloque de registro que se suma en `src/cmd/server/main.go`;
  - con HU-08 y HU-09, el contrato `metrics.NewQualityService` / `SprintQuality` de `data-model.md`.

## Fase 2: Bloque Dominio (cálculo y servicio)

**Objetivo**: calcular la calidad y el avance a partir de datos crudos, sin base de datos.

**Prueba independiente**: las pruebas unitarias de `src/metrics` pasan sin PostgreSQL.

### Pruebas (RED)

- [ ] T005 [P] [E1, E2, E3, E4, E5, E6, E7] Escribir `TestCalculateSprintQuality` como prueba de tabla en
  `src/metrics/quality_test.go`. Casos:
  - 3 de 4 → "75.00";
  - 0 de 3 → "0.00";
  - sin historias → "No disponible";
  - 2 de 3 → "66.67", 1 de 8 → "12.50" y 1 de 3 → "33.33";
  - Story Points 3, 5, 8 y 2 con 3 y 5 en "Hecho" → 18 planificados y 8 completados;
  - una y dos historias sin Story Points → `StoriesWithoutPoints` 1 y 2, sin sumar;
  - una "Hecho" sin Story Points → cuenta como completada y en `StoriesWithoutPoints`;
  - todas sin Story Points → 0 y 0;
  - los conteos de defectos se copian sin cambios.
- [ ] T006 [P] [E3] Escribir `TestQualityService_SprintQuality` con un almacén de prueba local en
  `src/metrics/quality_service_test.go`. Casos:
  - `projectID` o `sprintID` en 0 → `ErrSprintNotFound`, sin llamar al almacén;
  - `ErrSprintNotFound` del almacén se devuelve sin cambios;
  - Sprint "Activo" → `errors.Is(err, ErrSprintNotFinished)`;
  - otra falla se envuelve: `errors.Is` con la causa original;
  - almacén `nil` → error;
  - con un Sprint "Finalizado", el resultado es el de `calculateSprintQuality`.
- [ ] T007 Ejecutar T005 y T006, confirmar que fallan porque el código no existe y guardar la salida en
  `evidencia-red.txt`, sección "Dominio".

### Implementación (GREEN)

- [ ] T008 Crear `src/metrics/quality.go` con:
  - `ErrSprintNotFound = errors.New("sprint not found")`;
  - `QualityStoryResult`, `QualityData` y `SprintQuality` (campos de `data-model.md`);
  - `calculateSprintQuality`.

  Reglas del cálculo:
  - "No disponible" si no hay historias planificadas;
  - el porcentaje es `roundToTwoDecimals(big.NewRat(completadas*100, planificadas)).FloatString(2)`;
  - se suman solo los Story Points no nulos;
  - `StoriesWithoutPoints` cuenta los nulos.
- [ ] T009 Crear el puerto `QualityStore` en `src/metrics/quality_store.go`, y en `src/metrics/quality_service.go`
  crear `QualityService`, `NewQualityService` y `SprintQuality` con los pasos 1 a 5 de `data-model.md`.
- [ ] T010 Ejecutar T005 y T006 en verde junto con las pruebas existentes de `./src/metrics/...`. Las de HU-07 no
  cambian.

**Punto de control**: el cálculo y el servicio funcionan solos; HU-08 y HU-09 ya pueden programar contra
`SprintQuality`.

## Fase 3: Bloque Almacenamiento (adaptador PostgreSQL)

**Objetivo**: leer el Sprint, el registro de cierre y los defectos del proyecto en una lectura consistente.

**Prueba independiente**: las pruebas de integración de `src/metrics` pasan contra PostgreSQL con datos propios.

### Pruebas (RED)

- [ ] T011 Crear las fixtures en `src/metrics/postgres_quality_store_integration_test.go`:
  - proyectos «Alfa» y «Beta»;
  - en «Alfa», Sprints 1 y 2 "Finalizado" y Sprint 3 "Activo";
  - historias del Sprint 1: de 3, 5, 8 y 2 Story Points, con "Hecho" las de 3 y 5 (2 de 4, 18 planificados y 8
    completados);
  - historias del Sprint 2: tres "Hecho" de 1 Story Point y una sin estimar que volvió al Backlog (3 de 4, "75.00",
    3 planificados y 3 completados, advertencia de 1);
  - filas de `sprint_story_results`, y las "Hecho" asignadas, como las deja HU-15;
  - defectos de «Alfa»: tres detectados en el Sprint 2 (uno resuelto en él); uno detectado en el Sprint 1 y resuelto
    en el 2; uno detectado en el Sprint 1 que se resolvió y se reabrió ('Abierto', sin Sprint de resolución). Así el
    Sprint 2 da 3 detectados y 2 resueltos, y el Sprint 1 da 2 detectados y 0 resueltos;
  - un defecto de «Beta» cuyo Sprint de detección y de resolución es el Sprint 2 de «Alfa» (la clave foránea lo
    permite), para probar que no se cuenta.

  Los casos de T012 y T013 que necesitan otro reparto (ninguna "Hecho", sin registro de cierre, sin defectos) usan
  Sprints finalizados propios dentro del mismo subtest, con números distintos del Sprint activo; no se reutiliza un
  Sprint para dos repartos distintos.

  Limpieza, en este orden: defectos, registro de cierre, asignaciones, `sprint_metrics`, historias, Sprints y
  proyectos.
- [ ] T012 [P] [E1, E2, E3, E4, E5] Escribir `TestPostgresQuality_Historias` con un subtest por caso:
  - Sprint 2: 3 de 4 → "75.00";
  - Sprint sin "Hecho" → "0.00";
  - Sprint finalizado sin registro → "No disponible";
  - Sprint 1: 3, 5, 8 y 2 → 18 y 8;
  - Sprint 2: una sin Story Points → advertencia de 1 y 3 planificados.
- [ ] T013 [P] [E6, E7, E8] Escribir `TestPostgresQuality_Defectos`:
  - Sprint 2 → 3 detectados y 2 resueltos;
  - Sprint sin defectos → 0 y 0;
  - el reabierto cuenta como detectado en su Sprint y no como resuelto.
- [ ] T014 [P] [E9] Escribir `TestPostgresQuality_OtroProyecto`:
  - un Sprint de «Alfa» pedido con el proyecto «Beta» → `ErrSprintNotFound`;
  - un defecto de «Beta» que apunta a un Sprint de «Alfa» no cuenta.
- [ ] T015 [P] Escribir `TestPostgresQuality_SprintActivoEInexistente`:
  - el Sprint 3 activo devuelve solo su estado, y el servicio responde `ErrSprintNotFinished`;
  - el identificador 2147483647 → `ErrSprintNotFound`.
- [ ] T016 [P] [E4] Escribir `TestQuality_CoherenciaConVelocidadHU07`: en el mismo Sprint, `PointsCompleted` es igual
  a `NewService(NewPostgreSQLMetricsStore(db)).CalculateMetrics(id).Velocity`. La prueba borra la fila que HU-07
  guarda en `sprint_metrics`.

  El Sprint es el de 3, 5, 8 y 2 Story Points, y sus historias "Hecho" tienen Story Points y horas no negativas.
  Así pasa `ValidateSprintClosure` de HU-07, que rechaza historias "Hecho" sin Story Points.
- [ ] T017 Ejecutar T012–T016, confirmar que fallan y guardar la salida en `evidencia-red.txt`, sección
  "Almacenamiento".

### Implementación (GREEN)

- [ ] T018 Implementar `PostgreSQLQualityStore` y `NewPostgreSQLQualityStore` en
  `src/metrics/postgres_quality_store.go`:
  - `BeginTx` con `ReadOnly: true` y `Isolation: sql.LevelRepeatableRead`;
  - las tres consultas de `data-model.md`: el Sprint del proyecto (sin fila → `ErrSprintNotFound`), el registro de
    cierre ordenado por `story_id`, y los dos `COUNT(*) FILTER` con `WHERE project_id = $2`;
  - si el Sprint no está "Finalizado", no leer el registro ni los defectos;
  - errores envueltos con `%w`.
- [ ] T019 Ejecutar T012–T016 en verde.

**Punto de control**: `SprintQuality` devuelve los números reales de un Sprint cerrado.

## Fase 4: Bloque Pantalla ("Calidad y avance") 🎯 MVP visible

**Objetivo**: la sección `GET /calidad?sprint=ID`, con la lista de Sprints finalizados y los números del elegido.

**Prueba independiente**: con tres Sprints finalizados y uno activo, abrir "Calidad y avance", elegir cada Sprint
y comprobar los números; también con un proyecto sin finalizados.

### Pruebas (RED)

- [ ] T020 [P] [E10a, E10c] Escribir `TestPostgresQualityCatalog` en
  `src/web/postgres_quality_catalog_integration_test.go`. Casos:
  - solo los Sprints "Finalizado", del de mayor número al de menor;
  - `ActiveNumber` con el Sprint activo, o 0 si no hay;
  - nada de otro proyecto;
  - proyecto sin Sprints.
- [ ] T021 [P] [E1, E5] Escribir `TestQualityView` en `src/web/quality_view_test.go`:
  - `qualityPercentText`: "75.00" → "75,00 %" y "No disponible" sin cambios;
  - la advertencia en singular, en plural y vacía con 0;
  - `selectQualitySprint` con "", "abc", "0", "-1", "007", el identificador del activo, uno de otro proyecto y uno
    válido.
- [ ] T022 [E1, E4, E5, E6, E9, E10a, E10b, E10c, E11] Escribir el servidor de prueba y las pruebas de pantalla en
  `src/web/quality_handlers_test.go`:
  - `TestHU10_ListaYSprintPorDefecto`;
  - `TestHU10_ElegirOtroSprint`;
  - `TestHU10_SprintActivoNoFigura`, con el aviso "El Sprint 4 sigue en curso; sus métricas se calculan al
    cerrarlo";
  - `TestHU10_SinSprintsFinalizados`;
  - `TestHU10_SprintInvalidoSeIgnora`;
  - `TestHU10_NumerosDelSprint`, con dos Sprints como los de T011:
    - en uno, "3 de 4 historias completadas", "75,00 %", la advertencia y los defectos;
    - en el de 3, 5, 8 y 2, "2 de 4 historias completadas", "50,00 %", 18 y 8;
    - con un Sprint sin registro de cierre, "No disponible" (escenario 3), y ningún "NaN" ni "%" suelto;
  - `TestHU10_OtroProyecto`.

  Las pruebas usan el servicio y el catálogo reales con datos propios.
- [ ] T023 [P] [E12] Escribir las pruebas de errores y bordes en `src/web/quality_handlers_test.go`:
  - `TestHU10_ErrorAlLeer`: con un catálogo que falla y con un `QualityReader` que falla, se ve la página de error
    de HU-20a, sin números ni texto interno;
  - `TestHU10_SinProyecto`: 303 a `/contexto/proyecto?volver_a=%2Fcalidad`;
  - `TestHU10_PostNoPermitido`: 405;
  - `TestHU10_Rendimiento`: 20 Sprints, 50 historias por Sprint y 100 defectos, en menos de 500 ms.
- [ ] T024 [P] [E1, E6, E10a, E11] Escribir los escenarios BDD en `test/web_quality_bdd_test.go`: "Porcentaje de
  avance", "Defectos detectados y resueltos", "Lista de Sprints y Sprint por defecto" y "Sin Sprints finalizados".
- [ ] T025 Ejecutar T020–T024, confirmar que fallan (la ruta y el catálogo no existen) y guardar la salida en
  `evidencia-red.txt`, sección "Pantalla".

### Implementación (GREEN)

- [ ] T026 Crear el catálogo de la pantalla:
  - en `src/web/quality_catalog.go`: `QualitySprintOption`, `QualitySprintList`, `QualityCatalog` y
    `QualityReader`;
  - en `src/web/postgres_quality_catalog.go`: `PostgreSQLQualityCatalog` y `NewPostgreSQLQualityCatalog`, con la
    consulta de `data-model.md`; un error de la base devuelve `Unavailable`.
- [ ] T027 Crear `src/web/quality_view.go` con:
  - los textos de la tabla de `data-model.md`;
  - `qualityPercentText` y `qualityWithoutPointsWarning`;
  - `selectQualitySprint`, que usa `parseID` y la lista de finalizados;
  - el armado de `qualityPageData`.
- [ ] T028 Crear `RegisterQuality(s, reader, catalog)` en `src/web/quality_handlers.go`:
  - `Section{Name: "Calidad y avance", Path: "/calidad", RequiresProject: true}`;
  - `Routes{"GET /calidad": ...}`;
  - si falta algún argumento, devolver un error;
  - si falla el catálogo o el cálculo, `Fail` sin renderizar números.
- [ ] T029 Crear `src/web/templates/quality.html` con:
  - la lista de enlaces `/calidad?sprint=ID` y `aria-current="page"` en el Sprint elegido;
  - los avisos;
  - los números del Sprint elegido.

  La pantalla no tiene formularios ni JavaScript.
- [ ] T030 Agregar en `src/cmd/server/main.go`, después del último bloque de registro (hoy el de HU-24, `RegisterEffort`), el bloque que construye
  `metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(database))` y
  `web.NewPostgreSQLQualityCatalog(database)` y llama a `web.RegisterQuality`. Si falla, cerrar `ln` y `database`,
  como los otros bloques.
- [ ] T031 Ejecutar T020–T024 en verde y `go test -count=1 -p 1 ./...` completo.

**Punto de control**: el Scrum Master ve la calidad y el avance de cada Sprint cerrado.

## Fase 5: Cierre

- [ ] T032 REFACTOR:
  - revisar si el cálculo y la vista tienen duplicación y extraer una función local sin cambiar el comportamiento;
  - comprobar que ningún archivo de HU-07 ni de HU-25 cambió: `git diff --stat main` lista solo archivos nuevos y
    `main.go`.
- [ ] T033 Correr `gofmt -l` sobre los archivos creados y sobre `main.go`, que no tiene que listar nada; después
  `go vet ./...` limpio y `go test ./... -p 1 -race` en verde, como en el CI.
- [ ] T034 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde.
- [ ] T035 Guardar `cobertura.txt` con `go test -count=1 -p 1 -cover ./...` y verificar al menos 80% en
  `src/metrics`.
- [ ] T036 Verificar a mano los escenarios con la aplicación en ejecución (`quickstart.md`, verificación manual).
- [ ] T037 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas, y el manual
  breve de usuario con la sección "Calidad y avance".
- [ ] T038 [P] Avisar:
  - a HU-08 y HU-09, que el contrato `SprintQuality` está en `main`;
  - a HU-25, que no se tocaron sus archivos;
  - al equipo, que si hay conflicto en `main.go` al mergear se conservan todos los bloques.
- [ ] T039 Abrir el Pull Request desde `HU-10` citando HU-10, con las evidencias y la confirmación de T001.
- [ ] T040 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- La **Fase 1** va primero. T001 bloquea todo: sin HU-06 y HU-15 en `main` no se crea la rama.
- El **Bloque Dominio (Fase 2)** depende de la Fase 1.
- El **Bloque Almacenamiento (Fase 3)** depende de T008 y T009, porque usa los tipos y el puerto.
- El **Bloque Pantalla (Fase 4)** depende de la Fase 3, porque las pruebas usan el servicio real. T021 y T027
  (vista) se pueden adelantar apenas está T008.
- La **Fase 5** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- `main.go` es el único archivo compartido (T030). Si hay conflicto al mergear, se conservan ambos bloques.

### Oportunidades de paralelismo

- T005 y T006.
- T012–T016, después de T011.
- T020, T021, T023 y T024. T022 comparte archivo con T023: va primero.
- T026 y T027.
- T037 y T038.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T023, T028 |
| FR-002 | T020, T022, T026, T029 |
| FR-003 | T021, T022, T027 |
| FR-004 | T020, T022, T027 |
| FR-005 | T022, T024, T027 |
| FR-006 | T005, T012, T008 |
| FR-007 | T005, T012, T021, T008, T027 |
| FR-008 | T005, T012, T008, T018 |
| FR-009 | T005, T021, T022, T027 |
| FR-010 | T016 |
| FR-011 | T005, T013, T018 |
| FR-012 | T013 |
| FR-013 | T014, T018, T022 |
| FR-014 | T006, T009, T014, T015 |
| FR-015 | T018, T023, T028 |
| FR-016 | T018, T032 |
| FR-017 | T023 |
| FR-018 | T009, T038 |
| FR-019 | T023, T027 |
| SC-001 | T005, T012, T013, T022 |
| SC-002 | T016 |
| SC-003 | T014, T020, T022 |
| SC-004 | T023 |
| SC-005 | T023 |
| SC-006 | T007, T017, T025, T034, T035 |

## Estrategia de implementación

1. Fase 1: confirmar HU-06 y HU-15 en `main` (T001) y crear la rama.
2. Bloque Dominio: el contrato `SprintQuality` queda listo para HU-08 y HU-09.
3. Bloque Almacenamiento: números reales desde PostgreSQL, con la prueba de coherencia con HU-07.
4. Bloque Pantalla: la sección "Calidad y avance". Es el MVP visible para la Sprint Review.
5. Cierre, evidencias y PR.
