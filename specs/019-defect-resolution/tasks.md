# Tareas: HU-19 – Resolver y reabrir defectos

Formato: `[ID] [P] descripción` ([P] = se puede hacer en paralelo con otras tareas de su fase).
Trazabilidad: cada tarea cita el escenario (E) o requisito (FR) que cubre.

## Fase 0 – Preparación

- [ ] T001 Con HU-06 ya mergeada: leer `src/defect` y `src/web` y anotar en `research.md` los nombres reales (manejador y función que arma la vista de `/defectos`, función de registro, valores de `Status`, puerto `Store`, fixtures de integración, archivo de textos). Comprobar con `git grep -n` que no existen `ErrDefectNotFound`, `ErrResolutionBeforeDetection`, `ErrAlreadyResolved`, `ErrAlreadyOpen`, `Resolved`, `ResolutionStore`, `ResolutionService`; ajustar `plan.md` si algún nombre cambió.
- [ ] T002 Crear la rama `feature/hu-19-resolucion-defectos` desde `main` actualizado.
- [ ] T003 `go test -count=1 -p 1 ./...` en verde antes de empezar.
- [ ] T004 Coordinar con quien hizo HU-06 los archivos que HU-19 edita (`defects_handlers.go`, `defects.html`, `main.go`) y con quien tenga HU-10.

## Fase 1 – RED: pruebas de dominio (src/defect/resolution_test.go)

- [ ] T005 [P] `TestCheckResolve_OK`: defecto "Abierto", Sprint activo 3, detección 3 → sin error (E1, FR-003).
- [ ] T006 [P] `TestCheckResolve_SinSprint`: sin Sprint activo → `ErrNoActiveSprint` (E2, FR-004).
- [ ] T007 [P] `TestCheckResolve_Anterior`: detección 3, activo 2 → `ErrResolutionBeforeDetection` (E4, FR-005).
- [ ] T008 [P] `TestCheckResolve_YaResuelto` → `ErrAlreadyResolved` (E7, FR-006).
- [ ] T009 [P] `TestCheckReopen_OK` y `TestCheckReopen_YaAbierto` → `ErrAlreadyOpen` (E3, E8, FR-007, FR-008).
- [ ] T010 [P] Prueba del servicio con un almacén de prueba: `Resolve` devuelve el número de Sprint y propaga los errores (E1, E6).

## Fase 2 – RED: integración (src/defect/postgres_resolution_store_integration_test.go)

- [ ] T011 Fixtures: dos proyectos, Sprints, historia y defectos; limpieza borrando primero las historias y después el proyecto (reutilizar los de HU-06 si existen).
- [ ] T012 [P] `TestPostgresResolve_OK`: estado "Resuelto" y `resolution_sprint_id` = Sprint activo; descripción, severidad, historia y Sprint de detección sin cambios (E1, FR-003, FR-013).
- [ ] T013 [P] `TestPostgresResolve_SinSprintActivo`: sin cambios (E2, FR-004).
- [ ] T014 [P] `TestPostgresResolve_Anterior`: Sprint 3 finalizado y Sprint 2 activo creados en la base; sin cambios (E4, FR-005).
- [ ] T015 [P] `TestPostgresResolve_Inexistente` y `TestPostgresResolve_OtroProyecto` (E5, FR-009, FR-014).
- [ ] T016 [P] `TestPostgresResolve_FalloNoDejaCambios`: forzar un error en el UPDATE y comprobar que nada cambia (E6, FR-010).
- [ ] T017 [P] `TestPostgresReopen_OK`: vuelve a "Abierto" y a NULL (E3, FR-007).
- [ ] T018 [P] `TestPostgresResolve_Concurrente`: dos resoluciones simultáneas del mismo defecto; una OK y otra `ErrAlreadyResolved` (FR-012, SC-007).

## Fase 3 – RED: handler y BDD

- [ ] T019 Preparar el servidor de prueba con `cookiejar`, proyecto elegido por el flujo de HU-20b, Sprint y defectos, y un almacén que falle para el escenario 6 (src/web/defects_resolution_handlers_test.go).
- [ ] T020 [P] `TestHU19_ResolucionExitosa`: 303 a `/defectos`, mensaje de una sola vista, la lista muestra "Resuelto" y "Sprint N" (E1, FR-011).
- [ ] T021 [P] `TestHU19_ResolverSinSprintActivo`: 409 y mensaje (E2).
- [ ] T022 [P] `TestHU19_Reabrir`: 303, "Abierto" y "—" en Sprint de resolución (E3).
- [ ] T023 [P] `TestHU19_SprintAnteriorAlDeDeteccion`: 409 y mensaje (E4).
- [ ] T024 [P] `TestHU19_DefectoInexistente`: identificador inexistente, vacío, no numérico y de otro proyecto dan 404 sin modificar nada (E5).
- [ ] T025 [P] `TestHU19_ErrorAlGuardar`: 500 y `MsgOperationFailed`; la lista no cambia y la respuesta no contiene texto de consultas (E6, FR-016).
- [ ] T026 [P] `TestHU19_YaResuelto` y `TestHU19_YaAbierto`: 409 y mensajes (E7, E8).
- [ ] T027 [P] `TestHU19_ListaMuestraAccionYSprint`: botón según estado y Sprint de resolución (E9, FR-002).
- [ ] T028 [P] Escenarios BDD equivalentes en `test/web_defect_resolution_bdd_test.go` para E1, E2, E3 y E5.
- [ ] T029 Ejecutar las pruebas de HU-19, confirmar que fallan por la razón esperada y guardar `evidencia-red.txt`.

## Fase 4 – GREEN: dominio

- [ ] T030 `resolution_errors.go` con los 4 errores nuevos.
- [ ] T031 `resolution.go` con `CheckResolve`, `CheckReopen` y `Resolved`.
- [ ] T032 `resolution_store.go` (puerto) y `resolution_service.go`; T005–T010 en verde.

## Fase 5 – GREEN: adaptador PostgreSQL

- [ ] T033 `postgres_resolution_store.go`: transacciones de resolver y reabrir con los bloqueos de `data-model.md`; llama a las reglas de T031.
- [ ] T034 T012–T018 en verde.

## Fase 6 – GREEN: web

- [ ] T035 `defects_resolution_messages.go` con los textos.
- [ ] T036 `defects_resolution_handlers.go`: métodos `resolve` y `reopen` que leen el campo `defecto` con `parseID`, usan el proyecto de `ActiveContextFrom`, mapean errores a 404/409/500 y vuelven a mostrar la pantalla con `PageStatus` y `Flash`.
- [ ] T037 Editar `defects_handlers.go` para agregar el servicio de resolución al manejador y las rutas `POST /defectos/resolver` y `POST /defectos/reabrir`.
- [ ] T038 Editar `templates/defects.html`: formulario con botón "Resolver" o "Reabrir" por fila, con el campo oculto `defecto`.
- [ ] T039 Editar `src/cmd/server/main.go` para construir `ResolutionService` y pasarlo al registro.
- [ ] T040 T019–T028 en verde.

## Fase 7 – REFACTOR y cierre

- [ ] T041 Revisar duplicación entre `resolve` y `reopen`; extraer una función local sin cambiar comportamiento.
- [ ] T042 No usar `gofmt -w` sobre archivos existentes (CRLF); archivos nuevos formateados; `go vet ./...` limpio.
- [ ] T043 `go test -count=1 -p 1 ./...` completo en verde; guardar `evidencia-pruebas.txt`.
- [ ] T044 `go test -count=1 -p 1 -cover` y guardar `cobertura.txt` (al menos 80% en `src/defect`).
- [ ] T045 Prueba manual de `quickstart.md`.
- [ ] T046 Actualizar la trazabilidad de `spec.md` con los nombres reales de las pruebas y actualizar el manual breve de usuario con las acciones de la pantalla "Defectos".
- [ ] T047 PR hacia `main` citando HU-19, con evidencias y la lista de archivos editados de HU-06.
- [ ] T048 Revisión por un compañero y merge con "Create a merge commit".
- [ ] T049 Avisar a quien tenga HU-10 que HU-19 está en `main`.