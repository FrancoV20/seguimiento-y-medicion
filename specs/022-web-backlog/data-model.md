# Data Model: Interfaz web del Product Backlog

Esta feature **no crea tablas ni migraciones**. El alta usa el esquema de HU-02 (migración `002`) a
través del servicio y el store existentes. La lista lee las historias del proyecto activo.

## Esquema confirmado (migración `002`)

- `user_stories(id BIGSERIAL PK, project_id BIGINT NOT NULL REFERENCES projects(id), title TEXT,
  description TEXT, priority TEXT, story_points INTEGER, status TEXT DEFAULT 'Pendiente',
  estimated_hours NUMERIC(6,2), created_at TIMESTAMPTZ)`.
  - `priority` solo admite `'Alta'`, `'Media'` o `'Baja'`.
  - `story_points` es opcional y solo admite 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89.
  - `status` solo admite `'Pendiente'`, `'En progreso'` o `'Hecho'`.
  - `estimated_hours` es opcional y debe ser mayor que 0.
  - `title` y `description` no pueden estar en blanco y no tienen límite de largo.
- `acceptance_criteria(id BIGSERIAL PK, story_id BIGINT REFERENCES user_stories(id) ON DELETE
  CASCADE, position INTEGER, content TEXT)` con `UNIQUE (story_id, position)` y contenido no en blanco.

Notas:

- Los identificadores de historias son `BIGINT`; en Go se usan como `int64`.
- `user_stories.project_id` no tiene borrado en cascada: eliminar un proyecto con historias falla. Las
  pruebas borran primero las historias (los criterios caen en cascada) y después el proyecto.
- `NUMERIC(6,2)` redondea sin error un valor con más de dos decimales (1,555 pasa a 1,56): la
  validación debe hacerse antes de guardar (HU-02 o HU-17; se confirma en T001).

## Datos leídos (definidos por HU-02)

- Historia del Backlog: identificador, proyecto al que pertenece, título, descripción, prioridad,
  Story Points (puede faltar), horas estimadas (pueden faltar) y estado (`'Pendiente'`,
  `'En progreso'` o `'Hecho'`).
- Criterios de aceptación de la historia: uno o más textos con su posición (`acceptance_criteria`).

Esta pantalla muestra título, prioridad, Story Points, horas estimadas y estado; no muestra
descripción ni criterios.

## StoryForm

Datos del envío, para conservarlos al volver a mostrar el formulario.

### Fields

- `title`, `description`: texto de cada campo.
- `priority`: valor elegido entre `Alta`, `Media` y `Baja`.
- `storyPoints`: valor elegido entre 1, 2, 3, 5, 8, 13, 21, 34, 55 y 89, o vacío.
- `estimatedHours`: texto ingresado (puede estar vacío).
- `criteria`: lista ordenada de textos de criterios, uno por campo mostrado.

### Rules

- Los valores se devuelven tal como se enviaron cuando hay errores o cuando se agrega o quita un
  criterio.
- El formulario siempre muestra al menos un campo de criterio.

## StoryFormInput

Resultado de armar el envío para invocar el alta.

### Fields

- `projectID`: tomado del contexto del servidor; el campo del formulario se descarta.
- `title`, `description`, `priority`.
- `storyPoints`: ausente si el campo estaba vacío.
- `estimatedHours`: ausente si el campo estaba vacío.
- `criteria`: criterios con texto, sin los vacíos ni los que solo tienen espacios.

### Rules

- Story Points vacío es válido (se crea con advertencia); un valor que no es válido para HU-02 genera
  un error junto al campo.
- Horas estimadas vacías son válidas y no generan advertencia; cero, negativas, no numéricas o con más
  de dos decimales generan un error junto al campo (regla de HU-02 y HU-17, se confirma en T001).
- Las demás reglas (título, descripción, prioridad, al menos un criterio) las aplica HU-02.

## FormAction

Acción pedida por el botón con el que se envió el formulario.

### Values

- `guardar`: valor por defecto y el de la tecla Enter.
- `agregar_criterio`: agrega un campo vacío al final.
- `quitar_criterio:N`: quita el campo de índice `N`.

### Rules

- Solo `guardar` persiste datos.
- `quitar_criterio:N` no quita el último campo restante; un índice fuera de rango o no numérico se
  ignora y se vuelve a mostrar el formulario sin cambios.
- Una acción desconocida se trata como `guardar`.

## StoryRow

Fila de la lista del Backlog.

### Fields

- `id`.
- `title`.
- `priority`.
- `storyPoints`: ausente si no se estimó; se muestra "Sin estimar".
- `estimatedHours`: ausente si no se estimó; se muestra "Sin estimar".
- `status`: texto del estado.

### Rules

- Se ordenan por `id` descendente (las más recientes primero).
- Solo incluye historias del proyecto pedido.

## FieldErrors

Mensajes de error por campo para `FormView` de HU-20b.

### Fields

- Claves: `titulo`, `descripcion`, `prioridad`, `story_points`, `horas_estimadas`, `criterio`.
- Valores: texto del mensaje.

### Messages

- Título, descripción, prioridad y criterios: los de HU-02, sin cambios.
- Story Points inválidos: el de HU-02 si lo define, o un mensaje propio de HU-22.
- Horas estimadas inválidas: el de HU-02 o HU-17 ("El valor de horas no es válido") si lo definen, o
  un mensaje propio de HU-22.
- Éxito: mensaje de alta (`MsgStoryCreated`), tomado de HU-02 si lo define.
- Advertencia por falta de Story Points: la de HU-02 si la define (`MsgMissingStoryPoints`).
- Falla de guardado: `MsgOperationFailed` de HU-20b.

## StoryCatalog

Interfaz interna de solo lectura para la pantalla.

### Operations

- `ListStories(ctx, projectID)`: devuelve los `StoryRow` del proyecto.

### Rules

- Distingue errores de lectura de resultados vacíos: una lista vacía no es un error.

## PostgreSQLStoryCatalog

Implementación concreta de `StoryCatalog` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` mediante `database/sql`.

### Query

```sql
SELECT id, title, priority, story_points, estimated_hours, status
FROM user_stories
WHERE project_id = $1
ORDER BY id DESC;
```

### Rules

- Solo realiza lecturas; el alta pasa por el servicio y el store de HU-02.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse; las pruebas crean sus
  propios proyectos e historias y los eliminan al terminar, borrando primero las historias porque
  `user_stories.project_id` no tiene borrado en cascada.
