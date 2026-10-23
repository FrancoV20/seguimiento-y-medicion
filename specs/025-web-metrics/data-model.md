# Data Model: Interfaz web de métricas del Sprint

Esta feature **no crea tablas ni migraciones**. La lista de Sprints se lee de `sprints`; los cálculos los hace el
servicio de HU-07 con su propio almacenamiento (`user_stories`, `sprint_story_assignments`, `effort_logs`).

## Esquema leído

- `sprints(id, project_id, number, goal, start_date, end_date, status)`: estado `'Activo'` o `'Finalizado'`.

> **A verificar (T001 y T004)**: el valor exacto del estado de Sprint finalizado (se supone `'Finalizado'`) y las
> tablas que lee HU-07.

Notas:

- Los identificadores son `BIGINT`; en Go se usan como `int64`.
- `user_stories.project_id` no tiene borrado en cascada; las pruebas borran primero las historias y después el
  proyecto.

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

- `velocity`: Story Points completados (entero; cero si no hay historias "Hecho").
- `deviation`: valor de la desviación en porcentaje, con indicador de si está disponible.
- `partialUsed`, `partialTotal`: X e Y del cálculo parcial; cero si el cálculo no es parcial.

### Rules

- `deviation` no está disponible cuando ninguna historia terminada tiene horas estimadas.
- El cálculo es parcial cuando `partialUsed` es menor que `partialTotal` (HU-07 decide el criterio).

## MetricsSource

Interfaz del lado web; el adaptador llama al servicio de HU-07.

### Operations

- `Metrics(ctx, projectID, sprintID)`: devuelve `SprintMetrics` o un error de lectura o cálculo.

### Rules

- El adaptador no calcula ni redondea; traduce los tipos de HU-07.
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
- El signo `+` se muestra solo en desviaciones positivas; el cero no lleva signo.

## Mensajes

- Vista: `MsgMetricsNoSprints` ("Aún no hay Sprints finalizados"), `MsgMetricsActiveHint` ("El Sprint {N} sigue en
  curso; sus métricas se calculan al cerrarlo").
- De HU-07 (o con su texto): "No disponible" y "Cálculo parcial: basado en X de Y historias".
- Falla de lectura o de cálculo: página de error de HU-20a.