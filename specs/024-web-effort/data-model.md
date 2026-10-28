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

> **Verificado (T001 y T004, 2026-10-26)** contra `main` (migraciones 001 a 008):
>
> - `effort_logs(id BIGSERIAL, story_id BIGINT, member_id INTEGER, work_date DATE, activity TEXT, hours NUMERIC(5,2),
>   created_at TIMESTAMPTZ)`, con `CHECK (hours > 0 AND hours <= 24)` y `CHECK (btrim(activity) <> '')`.
> - `user_stories.active_sprint_id` (migración `004`) apunta al Sprint de la historia. HU-03 acepta una carga cuando
>   `active_sprint_id` es de un Sprint `'Activo'`, sin mirar el estado de la historia: también acepta "Hecho".
>   `sprint_story_assignments` se mantiene junto con `active_sprint_id` (HU-04), pero el catálogo de HU-24 usa
>   `active_sprint_id` para ofrecer exactamente lo que HU-03 acepta.
> - HU-03 vive en `src/effort`: `Service.LogEffort(ctx, LogEffortRequest)` devuelve `LogEffortResult` con
>   `ValidationErrors` (todas juntas, una por campo) y `StorageError` (`ErrStorage`, sin detalles). Las horas son
>   `effort.Hours` (centésimas de hora, enteras) y se leen con `effort.ParseHours` (coma o punto, hasta dos decimales,
>   mayor que 0 y hasta 24). `effort.ValidateInput(req, now)` ya devuelve todas las violaciones que no necesitan la
>   base, así que **T010 no agrega código en HU-03**: solo su prueba (`effort_actions_test.go`).
> - Mensajes de HU-03 (`src/effort/errors.go`): `ErrInvalidHours` "El valor de horas no es válido",
>   `ErrActivityRequired` "La descripción de la actividad es obligatoria", `ErrDateRequired` "La fecha es requerida",
>   `ErrDateInFuture` "La fecha no puede ser posterior a la fecha actual", `ErrMemberRequired` "Se debe especificar el
>   integrante responsable", `ErrStoryNotInSprint` "La historia no está asignada a un Sprint activo" y
>   `SuccessMessage` "Esfuerzo registrado correctamente".
> - HU-03 verifica que el integrante pertenezca al proyecto de la historia, pero no recibe el proyecto activo: por
>   eso el handler comprueba antes que la historia esté en el Sprint activo del proyecto del contexto.
> - En `main` no hay identificadores `Effort*` en el paquete `web` (HU-14, HU-15, HU-17, HU-21, HU-22 y HU-23 todavía
>   no están integradas) ni un formato de horas compartido: HU-24 muestra las horas con coma decimal
>   (`FormatEffortHours`: 250 → "2,5").

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
- `hours`: el texto tal como llegó; lo interpreta HU-03 con `effort.ParseHours` (coma o punto).
- `memberID`, `projectID`: del contexto del servidor.

### Rules

- Una fecha no vacía con otro formato genera `MsgEffortInvalidDate` junto a `fecha`.
- Un `storyID` vacío o no numérico genera `MsgEffortSelectStory` junto a `historia`.
- Las horas no se interpretan en la web: cualquier texto inválido (letras, vacío, formato científico, negativo, cero o
  más de 24) lo informa HU-03 con su mensaje junto a `horas`.
- Las demás reglas (fecha requerida, actividad, rango de horas, integrante) las aplica HU-03.

## EffortActions (interfaz del lado web)

Adaptador sobre el servicio de HU-03.

### Operations

- `Validate(entry)`: usa `effort.ValidateInput` y devuelve todas las violaciones, cada una con su campo (`fecha`, `actividad`, `horas` o general)
  y el mensaje de HU-03.
- `Register(ctx, entry)`: llama a `Service.LogEffort`; devuelve las violaciones de HU-03 o `ErrEffortNotSaved` (sin
  detalles) si falló el guardado. El integrante va como error general porque no es un campo de la pantalla.

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

-- Historias del Sprint activo con horas acumuladas (en centésimas, como HU-03)
SELECT us.id, us.title, us.status,
       COALESCE((SELECT SUM(el.hours) * 100 FROM effort_logs el WHERE el.story_id = us.id), 0)::bigint
FROM user_stories us
WHERE us.active_sprint_id = $1 AND us.project_id = $2
ORDER BY us.id;

-- La historia es del Sprint activo (de mayor número) del proyecto
SELECT us.title
FROM user_stories us
JOIN sprints s ON s.id = us.active_sprint_id
WHERE us.id = $1 AND us.project_id = $2 AND s.project_id = $2 AND s.status = 'Activo'
  AND s.number = (SELECT MAX(number) FROM sprints WHERE project_id = $2 AND status = 'Activo');

-- Historial de una historia del Sprint activo
SELECT m.display_name, el.work_date, el.activity, (el.hours * 100)::bigint
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
- `MsgEffortRecorded` es `effort.SuccessMessage` (no se copia el texto).
- Errores de campo: `MsgEffortSelectStory` ("Seleccione una historia"), `MsgEffortStoryNotInSprint` ("La historia no
  pertenece al Sprint activo"), `MsgEffortInvalidDate` ("La fecha no es válida"); fecha requerida, actividad vacía,
  horas inválidas e integrante no especificado salen de HU-03.
- Vista: `MsgEffortNoSprint` ("No hay un Sprint activo"), `MsgEffortNoStories` ("El Sprint activo no tiene historias
  asignadas"), `MsgEffortNoEntries` ("Todavía no hay horas registradas
  en esta historia").
- Falla de guardado: `MsgOperationFailed` de HU-20b.