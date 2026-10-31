# Plan de implementación: HU-19 – Resolver y reabrir defectos

**Rama**: HU-19 | **Spec**: specs/019-defect-resolution/spec.md
**Constitución**: v1.1.0

## Resumen

Se agregan al paquete `src/defect` las reglas de resolución y reapertura, un
puerto propio (`ResolutionStore`) con su adaptador PostgreSQL y un servicio. En
`src/web` se agregan dos handlers POST sobre el mismo manejador de `/defectos` de
HU-06, los textos de mensajes en un archivo propio y las acciones en
`templates/defects.html`. La lista no cambia: HU-06 ya entrega estado y Sprint de
resolución en `DefectRow`. No hay migración nueva.

## Supuestos tomados de `specs/006-defects` (verificar en T001)

HU-06 todavía no está en `main`. Estos nombres salen de su plan y se confirman con
el código real:

- `src/defect`: tipos `Severity`, `Status`; errores `ErrNoActiveSprint`,
  `ErrStoryNotFound`; puerto `Store`; servicio y `PostgreSQLStore`.
- `src/web`: `DefectRow` (estado, Sprint de detección, Sprint de resolución),
  `DefectCatalog`, `defects_handlers.go` con `GET` y `POST /defectos`,
  `templates/defects.html`, textos `MsgDefect…` en `defects_form.go`.
- Sección registrada con `Section{Name: "Defectos", Path: "/defectos", RequiresProject: true}`.
- Contratos de HU-20 ya en `main` (verificados): `Server.Register(section, Routes)`,
  `ActiveContextFrom`, `SetFlash`, `RedirectSeeOther`, `Renderer.PageStatus` con
  `PageData{Title, Form, Data, Flash}`, `Renderer.Fail`, `parseID`,
  `MsgOperationFailed`, middleware global `CrossOriginGuard`.
- Esquema de Sprints (verificado en la migración 004): `sprints(id, project_id,
  number, status 'Activo'|'Finalizado', …)` con `UNIQUE (project_id, number)`.

## Contexto técnico

- Go 1.27.1; `database/sql` con pgx/stdlib; sin dependencias nuevas.
- Pruebas: `go test -count=1 -p 1 ./...`; handlers con `httptest` y `cookiejar`;
  integración contra PostgreSQL real.
- Almacenamiento: tabla `defects` de HU-06, sin cambios de esquema.

## Verificación de la constitución

| Principio | Cumplimiento |
|---|---|
| SDD → BDD → TDD | spec.md → escenarios → pruebas rojas → código |
| Trazabilidad | Tabla de trazabilidad en spec.md |
| Persistencia PostgreSQL (VI) | `ResolutionStore` con adaptador PostgreSQL y pruebas de integración |
| Otras historias | Solo se editan los archivos de HU-06 listados abajo, coordinados |

## Estructura de archivos

Nuevos:

- `src/defect/resolution.go` – `CheckResolve`, `CheckReopen`, `Resolved`.
- `src/defect/resolution_errors.go` – `ErrDefectNotFound`, `ErrResolutionBeforeDetection`,
  `ErrAlreadyResolved`, `ErrAlreadyOpen`.
- `src/defect/resolution_store.go` – puerto `ResolutionStore`.
- `src/defect/resolution_service.go` – `ResolutionService`.
- `src/defect/postgres_resolution_store.go` – adaptador PostgreSQL.
- `src/defect/resolution_test.go` y `src/defect/postgres_resolution_store_integration_test.go`.
- `src/web/defects_resolution_messages.go` – textos de mensajes.
- `src/web/defects_resolution_handlers.go` – métodos `resolve` y `reopen` del manejador de HU-06.
- `src/web/defects_resolution_handlers_test.go`.
- `test/web_defect_resolution_bdd_test.go`.

Editados (coordinados con HU-06, nombres definitivos en T001):

- `src/web/defects_handlers.go` – agregar el servicio de resolución al manejador y
  las rutas `POST /defectos/resolver` y `POST /defectos/reabrir` al registrarlas.
- `src/web/templates/defects.html` – acciones por fila.
- `src/cmd/server/main.go` – construir `ResolutionService` y pasarlo al registro.

## Decisiones de diseño

1. **Reglas puras separadas.** `CheckResolve(status, detectionSprint, activeSprint)` y
   `CheckReopen(status)` devuelven el error de dominio sin base de datos. El
   adaptador las llama dentro de la transacción.
2. **Puerto propio.** `ResolutionStore.Resolve(ctx, projectID, defectID)` devuelve
   `Resolved{DefectID, SprintNumber}` y `ResolutionStore.Reopen(ctx, projectID,
   defectID)`. No se toca el puerto `Store` de HU-06.
3. **Transacción.** Resolver bloquea el defecto (`FOR UPDATE`), lee el Sprint activo
   con `FOR SHARE`, aplica las reglas y hace el UPDATE. Reabrir bloquea el defecto,
   aplica la regla y hace el UPDATE. Orden de bloqueos: defecto, luego Sprint.
4. **Sin catálogo nuevo.** La lista ya viene de `DefectCatalog.Defects`; la plantilla
   decide la acción según `DefectRow.Status`.
5. **Identificador por formulario.** Campo `defecto`, leído con `parseID`; un valor
   ausente, no numérico o fuera de rango se trata como inexistente (404) sin
   consultar la base.
6. **Errores con la pantalla.** En 404, 409 y 500 se arma la misma vista de HU-06
   y se agrega `Flash{Kind: FlashError, Text: …}` con `PageStatus` (igual que el
   500 de Backlog). El nombre de la función que arma la vista se toma en T001; si
   HU-06 no la separa, se extrae como refactor mínimo.
7. **Mapeo de errores.** `ErrNoActiveSprint`, `ErrResolutionBeforeDetection`,
   `ErrAlreadyResolved` y `ErrAlreadyOpen` → 409; `ErrDefectNotFound` → 404; otro
   error → 500 con `MsgOperationFailed`.
8. **Éxito.** `SetFlash` con el mensaje y `RedirectSeeOther` a `/defectos` (303).

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Nombres de HU-06 distintos a los supuestos | T001 los lee del código y los anota en `research.md` |
| Choque de nombres con `src/defect` o `src/web` | Los nombres nuevos llevan prefijo propio; T001 hace `git grep` |
| Conflicto de edición en `defects.html` y `defects_handlers.go` | HU-19 arranca con HU-06 mergeada y edita solo la fila de la tabla, las rutas y el campo del manejador |
| Fixtures de integración con el CHECK de coherencia | Los fixtures borran historias primero (los defectos caen en cascada) y después el proyecto |
| El escenario 4 casi no se da por pantalla | Se prueba con datos preparados directamente en la base |