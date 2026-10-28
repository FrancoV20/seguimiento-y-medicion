# Data Model: Registro y trazabilidad de defectos

Esta feature agrega **una tabla** (`defects`) con una migración nueva. El resto usa `projects`, `sprints` y
`user_stories`.

## Migración

Número: el último de `main` más uno (hoy `008`, o la siguiente si la de HU-15 ya está integrada); nadie edita una
migración ajena. Se verifica en T001 y T004.

```sql
-- NNN_create_defects.up.sql
CREATE TABLE defects (
    id                   BIGSERIAL PRIMARY KEY,
    project_id           INTEGER     NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    story_id             BIGINT      NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    description          TEXT        NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 1000),
    severity             VARCHAR(20) NOT NULL CHECK (severity IN ('Crítica', 'Alta', 'Media', 'Baja')),
    status               VARCHAR(20) NOT NULL DEFAULT 'Abierto' CHECK (status IN ('Abierto', 'Resuelto')),
    detection_sprint_id  BIGINT      NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    resolution_sprint_id BIGINT      REFERENCES sprints(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((status = 'Abierto' AND resolution_sprint_id IS NULL)
        OR (status = 'Resuelto' AND resolution_sprint_id IS NOT NULL))
);
CREATE INDEX idx_defects_project ON defects (project_id, id DESC);
CREATE INDEX idx_defects_story ON defects (story_id);
CREATE INDEX idx_defects_detection ON defects (detection_sprint_id);
CREATE INDEX idx_defects_resolution ON defects (resolution_sprint_id);

-- NNN_create_defects.down.sql
DROP TABLE defects;
```

> **Migración (T004)**: `origin/main` llega a `008`; no hay migración `009` en las ramas remotas visibles y el
> Product Builder confirmó que HU-15 todavía no tiene responsable ni implementación. `009` es el siguiente
> candidato; volver a coordinar con el equipo antes de crearla y si HU-15 se asigna.
>
> **Verificación (T005)**: PostgreSQL rechazó borrar el Sprint de resolución con `ON DELETE SET NULL`, porque
> dejaría un defecto "Resuelto" sin Sprint y violaría el `CHECK` de coherencia. Se usa `ON DELETE CASCADE` para
> que los defectos vinculados se eliminen junto con ese Sprint, igual que con el Sprint de detección.

## Verificación contra `main` (T001)

- `projects.id` es `SERIAL` (`INTEGER`); `user_stories.id` y `sprints.id` son `BIGSERIAL` (`BIGINT`). Los modelos
  Go actuales representan los identificadores como `int64`. Por eso `defects.project_id` usa `INTEGER`, mientras
  que las referencias a historias y Sprints usan `BIGINT`.
- `sprints.status` admite `'Activo'` y `'Finalizado'`. La migración 004 amplía los estados de historias con
  `'En Sprint'`; los otros estados relevantes son `'Pendiente'`, `'En progreso'` y `'Hecho'`.
- El código de `main` no contiene una operación de cierre de Sprint: los `FOR UPDATE` encontrados corresponden a
  asignación y remoción de historias. El contrato de cierre de HU-15 (`FOR UPDATE` sobre la fila del Sprint)
  queda pendiente de verificar contra esa historia; no se debe inferir de esos otros bloqueos.

## Esquema usado

- `projects(id, ...)`: dueño de los defectos (`ON DELETE CASCADE`).
- `sprints(id, project_id, number, status, ...)`: Sprint de detección y de resolución.
- `user_stories(id, project_id, title, status, ...)`: historia relacionada.
- `defects`: tabla nueva.

Notas:

- `projects.id` es `INTEGER`; las claves de historias, Sprints y defectos son `BIGINT`. En Go se usan como `int64`.
- `user_stories.project_id` no tiene borrado en cascada; las pruebas borran primero las historias (los defectos caen
  en cascada) y después el proyecto.

## Severity y Status

### Values

- `Severity`: `Crítica`, `Alta`, `Media`, `Baja`.
- `Status`: `Abierto`, `Resuelto` (HU-06 solo crea "Abierto").

## RegisterInput

### Fields

- `projectID`: del contexto del servidor.
- `storyID`: historia relacionada.
- `description`: texto.
- `severity`: una de las cuatro.

### Rules

- La descripción se recorta en los bordes; tiene entre 1 y 1000 caracteres (se cuentan caracteres, no bytes).
- La severidad es obligatoria y una de la lista.
- `storyID` es obligatorio (mayor que cero).
- `Validate` devuelve todas las violaciones juntas.

## Registered

### Fields

- `id`, `sprintNumber`.

## Errores del dominio

`ErrDescriptionRequired`, `ErrDescriptionTooLong`, `ErrSeverityRequired`, `ErrSeverityInvalid`, `ErrStoryRequired`,
`ErrStoryNotFound`, `ErrNoActiveSprint`. Cualquier otro error es una falla de almacenamiento.

## Store (puerto del dominio)

### Operations

- `Create(ctx, input)`: registra el defecto o devuelve `ErrNoActiveSprint` o `ErrStoryNotFound`.

### Transacción de registro

```sql
BEGIN;
-- 1. Sprint activo del proyecto, bloqueado para que no se cierre mientras se registra
SELECT id, number FROM sprints
WHERE project_id = $1 AND status = 'Activo'
ORDER BY number DESC LIMIT 1
FOR SHARE;                                    -- sin fila => ErrNoActiveSprint

-- 2. La historia es del proyecto
SELECT id FROM user_stories WHERE id = $2 AND project_id = $1;   -- sin fila => ErrStoryNotFound

-- 3. Alta
INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
VALUES ($1, $2, $3, $4, 'Abierto', $5)
RETURNING id;
COMMIT;
```

### Rules

- Cualquier error revierte la transacción: no queda nada guardado.
- Pruebas de integración contra PostgreSQL con fixtures propias, que se eliminan al terminar borrando primero las
  historias.

## DefectRow (vista)

### Fields

- `id`, `description`, `severity`, `status`.
- `storyID`, `storyTitle`.
- `detectionSprint`: número del Sprint de detección.
- `resolutionSprint`: número del Sprint de resolución o vacío.

### Rules

- Se ordenan por `id` descendente.
- Solo defectos del proyecto activo.

## DefectStoryOption (vista)

### Fields

- `id`, `title`, `status`.

### Rules

- Todas las historias del proyecto, ordenadas por `id` descendente.

## DefectCatalog

Interfaz de solo lectura del lado web.

### Operations

- `Defects(ctx, projectID)`: lista de `DefectRow`.
- `StoryOptions(ctx, projectID)`: lista de `DefectStoryOption`.
- `ActiveSprintNumber(ctx, projectID)`: número del Sprint activo, o "no hay".

### Consultas

```sql
SELECT d.id, d.description, d.severity, d.status, us.id, us.title,
       ds.number AS detection_number, rs.number AS resolution_number
FROM defects d
JOIN user_stories us ON us.id = d.story_id
JOIN sprints ds ON ds.id = d.detection_sprint_id
LEFT JOIN sprints rs ON rs.id = d.resolution_sprint_id
WHERE d.project_id = $1
ORDER BY d.id DESC;

SELECT id, title, status FROM user_stories WHERE project_id = $1 ORDER BY id DESC;

SELECT number FROM sprints WHERE project_id = $1 AND status = 'Activo' ORDER BY number DESC LIMIT 1;
```

## DefectForm

### Fields

- `description`, `severity`, `storyID`: texto tal como llegó.

### Rules

- Los valores se devuelven tal como se enviaron cuando hay errores.
- Un campo ausente se trata como vacío.

## Mensajes

- Éxito: `MsgDefectRegistered` ("Defecto registrado correctamente. Sprint de detección: Sprint {N}").
- Errores de campo: `MsgDefectDescriptionRequired`, `MsgDefectDescriptionTooLong`, `MsgDefectSeverityRequired`,
  `MsgDefectSeverityInvalid`, `MsgDefectStoryRequired` y `MsgDefectStoryNotInProject` (textos en `spec.md`).
- Vista: `MsgDefectNoSprint` ("No hay un Sprint activo; no se puede registrar un defecto") y `MsgDefectNone`
  ("Aún no hay defectos").
- Falla de guardado: `MsgOperationFailed` de HU-20b.