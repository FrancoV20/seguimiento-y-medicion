# Data Model: Interfaz web de métricas del Sprint

Esta feature **no crea tablas ni migraciones**. La lista de Sprints se lee de `sprints`; los cálculos los hace el
servicio de HU-07 con su propio almacenamiento (`user_stories`, `sprint_story_assignments`, `effort_logs`).

## Esquema leído

- `sprints(id, project_id, number, goal, start_date, end_date, status)`: estado `'Activo'` o `'Finalizado'`.

Notas:

- Los identificadores son `BIGINT`; en Go se usan como `int64`.
- `user_stories.project_id` no tiene borrado en cascada; las pruebas borran primero las historias y después el
  proyecto.

## Revisión de dependencias en `main` (T001, T002 y T004)

Revisado el 31/10/2026 sobre la rama `HU-25`, sin diferencias con `origin/main` (`git merge origin/main`: ya
estaba al día).

### HU-20a y HU-20b (T002)

Están integradas y se reutilizan sin cambios:

- `Server.Register(Section, Routes)` (`src/web/server.go`), con rutas `"GET /..."` de `http.ServeMux`. Aplica
  `RequireContext(section)` a todas las rutas de la sección.
- `Section.RequiresProject` (`src/web/sections.go`). Sin proyecto activo válido, `RequireContext` redirige con 303 a
  `/contexto/proyecto?volver_a=...` y un aviso (`MsgProjectRequired` o el del proyecto eliminado).
- `ActiveContextFrom(ctx)` (`src/web/render.go`) y `projectIDFrom(r)` (`src/web/board_handlers.go`, HU-14).
- `Renderer.Page`, `Renderer.PageStatus`, `Renderer.RenderError` y `Renderer.Fail` (`src/web/render.go` y
  `src/web/errors.go`). `Fail` responde 404, 503 (base no disponible o `Unavailable`) o 500, y el detalle solo va al
  registro del servidor.
- Página 405: `Server.route` reemplaza el 405 de `ServeMux` por `RenderError(405, TitleMethodNotAllowed,
  MsgMethodNotAllowed)` cuando la dirección existe solo con otro método.

### HU-07 (paquete `src/metrics`)

- Servicio: `metrics.NewService(store MetricsStore)` y `(*Service).CalculateMetrics(sprintID int64)
  (metrics.SprintMetrics, error)`. **No recibe el proyecto ni un `context.Context`**: el filtro por proyecto lo
  garantiza HU-25 pidiendo solo Sprints que el catálogo devolvió para el proyecto activo.
- Exige Sprint finalizado: si `status <> 'Finalizado'` devuelve `metrics.ErrSprintNotFinished`.
- Antes de calcular aplica `ValidateSprintClosure`: una historia "Hecho" con Story Points inválidos u horas
  negativas da `ClosureValidationError`; una "Hecho" sin Story Points da `CalculationError`.
- Resultado `metrics.SprintMetrics`:
  - `Velocity int`: suma de Story Points de las historias "Hecho" (0 si no hay ninguna).
  - `DeviationPercentage string`: texto con dos decimales y **punto**, ya redondeado por HU-07 (`"20.00"`,
    `"-8.50"`, `"0.00"`), o el texto `"No disponible"` cuando las horas estimadas usadas suman 0.
  - `CalculationUsedCount` (X) y `CalculationTotalCount` (Y): historias "Hecho" con horas estimadas **y** horas
    reales, sobre el total de historias "Hecho".
  - `IsPartial bool` (X < Y) y `Warning string`: el texto ya armado por HU-07, `"Cálculo parcial: basado en X de Y
    historias"`.
- Fórmula de HU-07: (horas reales − horas estimadas) ÷ horas estimadas × 100, con los totales de las historias
  usadas; redondeo a dos decimales "mitad hacia arriba" (`roundToTwoDecimals`).
- Textos: `"No disponible"` llega como valor de `DeviationPercentage` (HU-07 no exporta una constante; también lo
  devuelve `metrics.CalculateDeviation(nil, nil)`, que las pruebas usan para no reescribirlo). `"Cálculo parcial:
  ..."` llega en `Warning`.
- Sprint sin historias "Hecho": `Velocity = 0`, X = Y = 0, `IsPartial = false` y `DeviationPercentage = "No
  disponible"`, sin error.
- Lectura (`metrics.NewPostgreSQLMetricsStore(db).LoadSprintData`): `sprints` (id, status) y las historias de
  `sprint_story_assignments` unidas a `user_stories` (status, story_points, estimated_hours) con la suma de
  `effort_logs.hours`.
- **Escritura**: `CalculateMetrics` termina con `SaveMetrics`, que hace un *upsert* en `sprint_metrics`. Como HU-25
  no guarda métricas (Edge Cases, SC-006, T035), el adaptador usa el servicio real con un store propio de solo
  lectura: `LoadSprintData` delega en `PostgreSQLMetricsStore` y `SaveMetrics` no escribe nada (decisión del
  Product Builder, 31/10/2026). No se modifica ningún archivo de HU-07.

### HU-15 (cierre de Sprint)

- `sprint.NewCloseSprintService(sprint.NewPostgreSQLSprintCloser(db), validador)` y `CloseSprint(ctx, projectID,
  sprintID)`. El validador de `main` (`sprintClosureValidator` en `src/cmd/server/main.go`, copiado en las pruebas
  como `closeHandlerValidator`) aplica `metrics.ValidateSprintClosure`.
- Al cerrar: deja el Sprint en `'Finalizado'`; las historias "Hecho" conservan su asignación en
  `sprint_story_assignments`; las incompletas pierden la asignación y vuelven a "Pendiente" con
  `active_sprint_id = NULL`. Por eso **HU-07 no cuenta las historias devueltas**. También guarda
  `sprint_story_results` (migración 014), que HU-25 no lee.

### HU-17 (horas estimadas)

- `user_stories.estimated_hours NUMERIC(6,2)` (> 0, opcional). HU-17 no deja modificarlas en historias de un Sprint
  finalizado (`EstimateLockedReason = "Sprint finalizado"`).

### Estado de Sprint finalizado

- Valor exacto: `'Finalizado'` (`sprint.SprintFinished`, CHECK de la migración 004). El activo es `'Activo'`
  (`sprint.SprintActive`).

### Identificadores y fechas del paquete `web`

- No hay ningún identificador con el prefijo `Metrics` ni la ruta `/metricas` (HU-14, HU-15, HU-21 a HU-24, HU-12).
  Ya existen y no se redefinen: `projectIDFrom`, `newCatProject`, `newCatSprint`, `newCatStory`, `addCatEffort`,
  `newCtxApp`, `openContextTestDB`, `id64`, `closeHandlerValidator` y `closeHandlerStore` (estos dos en
  `close_handlers_test.go`).
- Las pantallas muestran las fechas como `dd/mm/aaaa` (`"02/01/2006"`): HU-12 (`projectStatusDateLayout`) y HU-15
  (`close_form.go`). HU-25 usa el mismo formato con una constante propia.
- HU-12 ya pasa porcentajes de punto a coma (`"50.00"` → `"50,00 %"`); HU-25 hace lo mismo y agrega el signo.

### Migraciones (T004)

`main` tiene 001, 002, 003, 004, 007, 008, 011, 012, 013 y 014. El esquema anotado coincide con 004 (`sprints`,
`sprint_story_assignments`, `user_stories.active_sprint_id`); HU-07 lee además `user_stories` (002), `effort_logs`
(003) y escribe `sprint_metrics` (007). HU-25 no necesita migraciones nuevas y no hubo cambios que avisar al equipo.

## FinishedSprint

Fila de la lista de Sprints finalizados.

### Fields

- `id`, `number`, `goal`, `startDate`, `endDate`.

### Rules

- Se ordenan por `number` descendente (el más reciente primero).
- Solo Sprints del proyecto activo con estado "Finalizado".

## MetricsCatalog

Interfaz de solo lectura del lado web.

### Operations

- `FinishedSprints(ctx, projectID)`: lista de `FinishedSprint` (vacía si no hay).
- `ActiveSprintNumber(ctx, projectID)`: número del Sprint activo, o "no hay".

### Consultas

```sql
SELECT id, number, goal, start_date, end_date
FROM sprints
WHERE project_id = $1 AND status = 'Finalizado'
ORDER BY number DESC;

SELECT number FROM sprints
WHERE project_id = $1 AND status = 'Activo'
ORDER BY number DESC LIMIT 1;
```

### Rules

- Todas las lecturas se filtran por el proyecto del contexto del servidor.
- Pruebas de integración contra PostgreSQL con fixtures propias, que se eliminan al terminar borrando primero las
  historias.

## SprintMetrics

Resultado de `MetricsSource`, tomado de HU-07.

### Fields

- `velocity`: Story Points completados (`Velocity` de HU-07; cero si no hay historias "Hecho").
- `deviation`: el texto `DeviationPercentage` de HU-07 tal cual (`"20.00"` o `"No disponible"`).
- `partialUsed`, `partialTotal`: X e Y del cálculo (`CalculationUsedCount`, `CalculationTotalCount`).
- `partial` y `partialAlert`: `IsPartial` y `Warning` de HU-07.

### Rules

- `deviation` no está disponible cuando ninguna historia terminada tiene horas estimadas (HU-07 devuelve "No
  disponible").
- El cálculo es parcial cuando `partialUsed` es menor que `partialTotal` (HU-07 decide el criterio).

## MetricsSource

Interfaz del lado web; el adaptador llama al servicio de HU-07.

### Operations

- `Metrics(ctx, projectID, sprintID)`: devuelve `SprintMetrics` o un error de lectura o cálculo.

### Rules

- El adaptador no calcula ni redondea; traduce los tipos de HU-07.
- Usa el servicio real de HU-07 con un store de solo lectura: lee con `PostgreSQLMetricsStore` y no guarda en
  `sprint_metrics`.
- Solo se consulta por Sprints que el catálogo devolvió para el proyecto.

## MetricsView

Datos de la página.

### Fields

- `sprints`: lista de `FinishedSprint` con el elegido marcado.
- `selected`: Sprint elegido (puede no haber).
- `velocityText`: por ejemplo `18 Story Points`.
- `deviationText`: por ejemplo `+20,00 %`, `-8,50 %`, `0,00 %` o "No disponible".
- `partialAlert`: "Cálculo parcial: basado en X de Y historias" o vacío.
- `activeSprintNumber` y `activeHint`: aviso del Sprint en curso.
- `emptyMessage`: "Aún no hay Sprints finalizados" cuando no hay lista.

### Rules

- El Sprint elegido es el de `sprint` si es numérico y está en la lista; si no, el primero. Sin lista, no hay
  elegido.
- Con `selected` vacío no se pide nada a HU-07.
- La desviación se presenta a partir del texto de HU-07: si es un número, se cambia el punto por coma, se agrega
  ` %` y el signo `+` solo si es positiva (el cero no lleva signo); si no es un número ("No disponible"), se muestra
  el texto de HU-07 sin cambios. No se recalcula ni se redondea.
- Las fechas se muestran como `dd/mm/aaaa`.

## Mensajes

- Vista: `MsgMetricsNoSprints` ("Aún no hay Sprints finalizados"), `MsgMetricsActiveHint` ("El Sprint {N} sigue en
  curso; sus métricas se calculan al cerrarlo").
- De HU-07: "No disponible" (`DeviationPercentage`) y "Cálculo parcial: basado en X de Y historias" (`Warning`).
- Falla de lectura o de cálculo: página de error de HU-20a.