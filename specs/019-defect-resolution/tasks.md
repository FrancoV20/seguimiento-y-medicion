# Tareas: HU-19 – Resolver y reabrir defectos

**Entrada**: `specs/019-defect-resolution/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-06 mergeada en `main`; HU-04, FIX-HU-04, HU-20a y HU-20b ya integradas.

Formato: `[ID] [P] [E#] descripción`
- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario del Issue que cubre.
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en `specs/019-defect-resolution/evidencia-red.txt`.
- Antes de cada ejecución en PowerShell: `chcp 65001`.

## Fase 1: Preparación

- [x] T001 Con HU-06 ya mergeada: leer `src/defect` y `src/web` y anotar en `research.md` los nombres reales (manejador y función que arma la vista de `/defectos`, función de registro, valores de `Status`, puerto `Store`, fixtures de integración, archivo de textos); comprobar con `git grep -n` que no existen `ErrDefectNotFound`, `ErrResolutionBeforeDetection`, `ErrAlreadyResolved`, `ErrAlreadyOpen`, `Resolved`, `ResolutionStore` ni `ResolutionService`; ajustar `plan.md` si algún nombre cambió
- [x] T002 Crear la rama `HU-19` desde `main` actualizado
- [x] T003 `go test -count=1 -p 1 ./...` en verde antes de empezar (línea base)
- [x] T004 Coordinar con quien hizo HU-06 los archivos que HU-19 edita (`defects_handlers.go`, `defects.html`, `main.go`) y con quien tenga HU-10

## Fase 2: Fundamentos (dominio y almacenamiento; bloquea todo lo demás)

### Pruebas (RED)

- [x] T005 [P] [E1] `TestCheckResolve_OK`: defecto "Abierto", Sprint activo 3, detección 3 → sin error, en `src/defect/resolution_test.go`
- [x] T006 [P] [E2] `TestCheckResolve_SinSprint`: sin Sprint activo → `ErrNoActiveSprint`, en `src/defect/resolution_test.go`
- [x] T007 [P] [E4] `TestCheckResolve_Anterior`: detección 3, activo 2 → `ErrResolutionBeforeDetection`, en `src/defect/resolution_test.go`
- [x] T008 [P] [E7] `TestCheckResolve_YaResuelto` → `ErrAlreadyResolved`, en `src/defect/resolution_test.go`
- [x] T009 [P] [E3, E8] `TestCheckReopen_OK` y `TestCheckReopen_YaAbierto` → `ErrAlreadyOpen`, en `src/defect/resolution_test.go`
- [x] T010 [P] [E1, E6] Prueba del servicio con un almacén de prueba: `Resolve` devuelve el número de Sprint y propaga los errores, en `src/defect/resolution_test.go`
- [x] T011 Fixtures de integración: dos proyectos, Sprints, historia y defectos; limpieza borrando primero las historias y después el proyecto (reutilizar los de HU-06 si existen), en `src/defect/postgres_resolution_store_integration_test.go`
- [x] T012 [P] [E1] `TestPostgresResolve_OK`: estado "Resuelto" y `resolution_sprint_id` = Sprint activo; descripción, severidad, historia y Sprint de detección sin cambios
- [x] T013 [P] [E2] `TestPostgresResolve_SinSprintActivo`: sin cambios
- [x] T014 [P] [E4] `TestPostgresResolve_Anterior`: Sprint 3 finalizado y Sprint 2 activo creados en la base; sin cambios
- [x] T015 [P] [E5] `TestPostgresResolve_Inexistente` y `TestPostgresResolve_OtroProyecto`
- [x] T016 [P] [E6] `TestPostgresResolve_FalloNoDejaCambios`: forzar un error en el UPDATE y comprobar que nada cambia
- [x] T017 [P] [E3] `TestPostgresReopen_OK`: vuelve a "Abierto" y a NULL
- [x] T018 [P] `TestPostgresResolve_Concurrente`: dos resoluciones simultáneas del mismo defecto; una OK y otra `ErrAlreadyResolved`
- [x] T019 Ejecutar T005–T018, confirmar que fallan porque el código no existe y guardar la salida en `evidencia-red.txt` (sección "Fundamentos")

### Implementación (GREEN)

- [x] T020 Crear `ErrDefectNotFound`, `ErrResolutionBeforeDetection`, `ErrAlreadyResolved` y `ErrAlreadyOpen` en `src/defect/resolution_errors.go`
- [x] T021 Implementar `CheckResolve`, `CheckReopen` y `Resolved` en `src/defect/resolution.go`
- [x] T022 Definir el puerto `ResolutionStore` y el `ResolutionService` en `src/defect/resolution_store.go` y `src/defect/resolution_service.go`
- [ ] T023 Implementar el adaptador PostgreSQL con las transacciones y bloqueos de `data-model.md` (defecto `FOR UPDATE`, Sprint activo `FOR SHARE`) llamando a las reglas de T021, en `src/defect/postgres_resolution_store.go`
- [ ] T024 Ejecutar T005–T018 en verde

**Punto de control**: el dominio y el almacenamiento funcionan solos, sin tocar la pantalla.

## Fase 3: Bloque A – Resolver (escenarios 1, 2, 4 y 7) 🎯 MVP

**Objetivo**: resolver un defecto "Abierto" registrando el Sprint activo, con sus bloqueos.

**Prueba independiente**: con un Sprint activo, pulsar "Resolver" y ver "Resuelto" con "Sprint N"; sin Sprint activo, ver el bloqueo.

### Pruebas (RED)

- [ ] T025 Preparar el servidor de prueba con `cookiejar`, proyecto elegido por el flujo de HU-20b, Sprint y defectos, en `src/web/defects_resolution_handlers_test.go`
- [ ] T026 [P] [E1] `TestHU19_ResolucionExitosa`: 303 a `/defectos`, mensaje de una sola vista, la lista muestra "Resuelto" y "Sprint N"
- [ ] T027 [P] [E2] `TestHU19_ResolverSinSprintActivo`: 409 y mensaje, defecto "Abierto"
- [ ] T028 [P] [E4] `TestHU19_SprintAnteriorAlDeDeteccion`: 409 y mensaje, defecto "Abierto"
- [ ] T029 [P] [E7] `TestHU19_YaResuelto`: 409 y mensaje; el Sprint de resolución original no cambia
- [ ] T030 [P] [E1, E2] Escenarios BDD de resolución exitosa y sin Sprint activo en `test/web_defect_resolution_bdd_test.go`
- [ ] T031 Ejecutar T026–T030, confirmar que fallan (la ruta no existe) y guardar la salida en `evidencia-red.txt` (sección "Resolver")

### Implementación (GREEN)

- [ ] T032 Crear los textos de mensajes en `src/web/defects_resolution_messages.go`
- [ ] T033 Implementar el método `resolve` en `src/web/defects_resolution_handlers.go`: lee `defecto` con `parseID`, usa el proyecto de `ActiveContextFrom`, llama al servicio; éxito con `SetFlash` y `RedirectSeeOther` a `/defectos`; los errores 409 (`ErrNoActiveSprint`, `ErrResolutionBeforeDetection`, `ErrAlreadyResolved`) vuelven a mostrar la pantalla con `PageStatus` y `Flash`
- [ ] T034 Editar `src/web/defects_handlers.go`: agregar el servicio de resolución al manejador, `RegisterDefectsWithResolution` (`RegisterDefects` conserva su firma) y la ruta `POST /defectos/resolver`
- [ ] T035 Editar `src/web/templates/defects.html`: formulario con botón "Resolver" y campo oculto `defecto` en cada defecto "Abierto"
- [ ] T036 Editar `src/cmd/server/main.go` para construir `ResolutionService` y registrar con `RegisterDefectsWithResolution`
- [ ] T037 Ejecutar T026–T030 en verde y `go test -count=1 -p 1 ./...` completo

**Punto de control**: resolver funciona de punta a punta; HU-10 ya puede contar defectos resueltos.

## Fase 4: Bloque B – Reabrir (escenarios 3 y 8)

**Objetivo**: volver un defecto "Resuelto" a "Abierto" borrando su Sprint de resolución.

**Prueba independiente**: pulsar "Reabrir" en un defecto resuelto y ver "Abierto" con "—".

### Pruebas (RED)

- [ ] T038 [P] [E3] `TestHU19_Reabrir`: 303, "Abierto" y "—" en Sprint de resolución, también sin Sprint activo, en `src/web/defects_resolution_handlers_test.go`
- [ ] T039 [P] [E8] `TestHU19_YaAbierto`: 409 y mensaje; el defecto no cambia
- [ ] T040 [P] [E3] Escenario BDD de reabrir en `test/web_defect_resolution_bdd_test.go`
- [ ] T041 Ejecutar T038–T040, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Reabrir")

### Implementación (GREEN)

- [ ] T042 Implementar el método `reopen` en `src/web/defects_resolution_handlers.go` (éxito y 409 `ErrAlreadyOpen`)
- [ ] T043 Editar `src/web/defects_handlers.go`: agregar la ruta `POST /defectos/reabrir`
- [ ] T044 Editar `src/web/templates/defects.html`: botón "Reabrir" en cada defecto "Resuelto" (nunca junto a "Resolver")
- [ ] T045 Ejecutar T038–T040 en verde

## Fase 5: Bloque C – Errores y lista (escenarios 5, 6 y 9)

**Objetivo**: respuestas correctas ante identificadores inválidos y fallas, y lista coherente.

**Prueba independiente**: enviar un `defecto` inexistente, forzar una falla de guardado y revisar la lista con un defecto de cada estado.

### Pruebas (RED)

- [ ] T046 [P] [E5] `TestHU19_DefectoInexistente`: identificador inexistente, vacío, no numérico y de otro proyecto dan 404 sin modificar nada, en `src/web/defects_resolution_handlers_test.go`
- [ ] T047 [P] [E6] `TestHU19_ErrorAlGuardar`: con un almacén que falla, 500 y `MsgOperationFailed`; la lista no cambia y la respuesta no contiene texto de consultas ni rutas internas
- [ ] T048 [P] [E9] `TestHU19_ListaMuestraAccionYSprint`: botón según estado y Sprint de resolución, un defecto de cada estado
- [ ] T049 [P] [E5] Escenario BDD de defecto inexistente en `test/web_defect_resolution_bdd_test.go`
- [ ] T050 Ejecutar T046–T049, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Errores y lista")

### Implementación (GREEN)

- [ ] T051 Completar en `defects_resolution_handlers.go` el mapeo de `ErrDefectNotFound` a 404 y de cualquier otro error a 500 con `MsgOperationFailed`, con la pantalla y el `Flash` de error
- [ ] T052 Ajustar `defects.html` si hace falta para que cada defecto muestre una sola acción según `DefectRow.Status`
- [ ] T053 Ejecutar T046–T049 en verde

## Fase 6: Cierre

- [ ] T054 Revisar duplicación entre `resolve` y `reopen`; extraer una función local sin cambiar comportamiento
- [ ] T055 [P] No usar `gofmt -w` sobre archivos existentes (CRLF); archivos nuevos formateados; `go vet ./...` limpio; pipeline de GitHub Actions en verde
- [ ] T056 Guardar `evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en verde
- [ ] T057 Guardar `cobertura.txt` con `go test -count=1 -p 1 -cover ./...` verificando al menos 80% en `src/defect`
- [ ] T058 Verificar a mano los 9 escenarios con la aplicación en ejecución (`quickstart.md`)
- [ ] T059 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas y el manual breve de usuario con las acciones de la pantalla "Defectos"
- [ ] T060 Abrir el Pull Request desde `HU-19` citando HU-19, con evidencias y la lista de archivos editados de HU-06
- [ ] T061 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho"
- [ ] T062 Avisar a quien tenga HU-10 que HU-19 está en `main`

## Dependencias y orden de ejecución

- **Fase 1** primero. **Fase 2** depende de la Fase 1 y bloquea a las demás.
- **Bloque A (Fase 3)** depende de la Fase 2. Es el MVP.
- **Bloque B (Fase 4)** depende de T034 (el manejador ya extendido); puede avanzar en paralelo con el Bloque C.
- **Bloque C (Fase 5)** depende de T033; puede avanzar en paralelo con el Bloque B.
- **Fase 6** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar evidencia → implementación (GREEN).
- Los bloques B y C editan los mismos archivos que A (`defects_handlers.go`, `defects.html`, `defects_resolution_handlers.go`): si dos personas trabajan en paralelo, coordinar el orden de los cambios.

### Oportunidades de paralelismo

- T005–T010 y T012–T018 (después de T011).
- T026–T030, T038–T040 y T046–T049 dentro de su bloque.
- T055 y T059.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T048, T052 |
| FR-002 | T035, T044, T048 |
| FR-003 | T005, T010, T012, T026 |
| FR-004 | T006, T013, T027 |
| FR-005 | T007, T014, T028 |
| FR-006 | T008, T029 |
| FR-007 | T009, T017, T038 |
| FR-008 | T009, T039 |
| FR-009 | T015, T046, T051 |
| FR-010 | T016, T047, T051 |
| FR-011 | T026, T038 |
| FR-012 | T018, T023 |
| FR-013 | T012, T017 |
| FR-014 | T015, T033, T046 |
| FR-015 | T012, T017 |
| FR-016 | T047 |
| SC-001 | T012, T026 |
| SC-002 | T017, T038 |
| SC-003 | T013, T027 |
| SC-004 | T014, T028 |
| SC-005 | T015, T046 |
| SC-006 | T016, T047 |
| SC-007 | T018 |
| SC-008 | T019, T031, T041, T050, T056, T057 |

## Estrategia de implementación

1. Completar Fase 1 y Fase 2 (dominio y almacenamiento).
2. Bloque A: es el MVP; con él HU-10 ya puede contar defectos resueltos.
3. Bloques B y C, en paralelo si hay manos libres.
4. Cierre, evidencias y PR.