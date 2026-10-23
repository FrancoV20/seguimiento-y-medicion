# Data Model: Interfaz web de registro de esfuerzo

Esta feature **no crea tablas ni migraciones**. El registro usa el servicio y el store de HU-03 (migración `003`).
Las lecturas de pantalla usan `sprints`, `sprint_story_assignments`, `user_stories`, `effort_logs` y `members`.

## Esquema leído

- `sprints(id, project_id, number, goal, start_date, end_date, status)`: estado `'Activo'` o `'Finalizado'`.
- `sprint_story_assignments(sprint_id, story_id)`: `UNIQUE (story_id)`.
- `user_stories(id, project_id, title, status, ...)`: estados `'Pendiente'`, `'En Sprint'`, `'En progreso'` y
  `'Hecho'`.
- `effort_logs`: registros de HU-03 con historia, integrante, fecha, actividad y horas.
- `members(id, display_name)`: catálogo global de integrantes.

> **A verificar (T001 y T004)**: los nombres exactos de las columnas de `effort_logs`. Se suponen `id`, `story_id`,
> `member_id`, `work_date`, `activity` y `hours`, con `hours` numérico. Si difieren, se ajustan las consultas y se
> anota acá.

Notas:

- Los identificadores son `BIGINT`; en Go se usan como `int64`.
- `user_stories.project_id` no tiene borrado en cascada; las pruebas borran primero las historias y después el
  proyecto.

## EffortForm

Datos del envío, para conservarlos al volver a mostrar el formulario.

### Fields

- `storyID`: texto del identificador de la historia.
- `date`: texto de la fecha tal como llegó.
- `activity`: texto de la actividad.
- `hours`: texto de las horas tal como llegó.

### Rules

- Los valores se devuelven tal como se enviaron cuando hay errores.
- Un campo ausente se trata como vacío.
- En la primera carga, `date` trae la fecha de hoy y `storyID` trae la historia elegida, si hay.

## EffortInput

Resultado de leer el envío para invocar a HU-03.

### Fields

- `storyID`: identificador numérico.
- `date`: fecha de calendario (`AAAA-MM-DD`); vacía si el campo vino vacío.
- `activity`: texto sin espacios en los bordes.
- `hours`: número leído del texto con coma o punto.
- `memberID`, `projectID`: del contexto del servidor.

### Rules

- Una fecha no vacía con otro formato genera `MsgEffortInvalidDate` junto a `fecha`.
- Un `storyID` vacío o no numérico genera `MsgEffortSelectStory` junto a `historia`.
- Un texto de horas que no es un número (letras, vacío, formato científico) genera el mensaje de horas inválidas de
  HU-03 junto a `horas`; un número fuera de rango lo informa HU-03.
- Las demás reglas (fecha requerida, actividad, rango de horas, integrante) las aplica HU-03.

## EffortActions (interfaz del lado web)

Adaptador sobre el servicio de HU-03.

### Operations

- `Validate(entry)`: devuelve todas las violaciones, cada una con su campo (`fecha`, `actividad`, `horas` o general)
  y el mensaje de HU-03.
- `Register(ctx, entry)`: registra la carga o devuelve el error de HU-03.

## EffortSprint

### Fields

- `sprintNumber`: número del Sprint activo.
- `stories`: lista de `EffortStoryRow`.

### Rules

- Sin Sprint activo no hay `EffortSprint` (el catálogo lo informa sin error).
- Con más de un Sprint activo (no debería ocurrir por FIX-HU-04) se toma el de mayor número.

## EffortStoryRow

### Fields

- `id`, `title`, `status`.
- `actualHours`: suma de las horas de todos sus registros (cero si no tiene).

### Rules

- Se ordenan por `id`.
- Solo historias asignadas al Sprint activo del proyecto.

## EffortEntryRow

### Fields

- `memberName`, `date`, `activity`, `hours`.

## EffortHistory

### Fields

- `storyID`, `title`.
- `entries`: lista de `EffortEntryRow`.
- `total`: suma de las horas de `entries`.

### Rules

- Se ordenan por fecha descendente y, a igual fecha, por orden de carga descendente.
- Una historia sin registros tiene `entries` vacía y `total` en cero.
- Solo existe para historias del Sprint activo del proyecto; otra devuelve "no encontrada" sin error.

## EffortCatalog

Interfaz de solo lectura del lado web.

### Operations

- `ActiveSprintStories(ctx, projectID)`: devuelve el `EffortSprint` o "sin Sprint activo".
- `StoryHistory(ctx, projectID, storyID)`: devuelve el `EffortHistory` o "no encontrada".

### Consultas

```sql
-- Sprint activo (mayor número si hubiera más de uno)
SELECT id, number FROM sprints
WHERE project_id = $1 AND status = 'Activo'
ORDER BY number DESC LIMIT 1;

-- Historias del Sprint activo con horas acumuladas
SELECT us.id, us.title, us.status, COALESCE(SUM(el.hours), 0) AS actual_hours
FROM sprint_story_assignments a
JOIN user_stories us ON us.id = a.story_id
LEFT JOIN effort_logs el ON el.story_id = us.id
WHERE a.sprint_id = $1 AND us.project_id = $2
GROUP BY us.id, us.title, us.status
ORDER BY us.id;

-- Historial de una historia del Sprint activo
SELECT m.display_name, el.work_date, el.activity, el.hours
FROM effort_logs el
JOIN members m ON m.id = el.member_id
WHERE el.story_id = $1
ORDER BY el.work_date DESC, el.id DESC;
```

### Rules

- Todas las lecturas se filtran por el proyecto del contexto del servidor.
- El historial solo se consulta si la historia figura entre las del Sprint activo del proyecto.
- Pruebas de integración contra PostgreSQL con fixtures propias, que se eliminan al terminar borrando primero las
  historias.

## Mensajes

- Éxito: `MsgEffortRecorded` ("Esfuerzo registrado correctamente").
- Errores de campo: `MsgEffortSelectStory` ("Seleccione una historia"), `MsgEffortStoryNotInSprint` ("La historia no
  pertenece al Sprint activo"), `MsgEffortInvalidDate` ("La fecha no es válida"); fecha requerida, actividad vacía,
  horas inválidas e integrante no especificado salen de HU-03.
- Vista: `MsgEffortNoSprint` ("No hay un Sprint activo"), `MsgEffortNoEntries` ("Todavía no hay horas registradas
  en esta historia").
- Falla de guardado: `MsgOperationFailed` de HU-20b.