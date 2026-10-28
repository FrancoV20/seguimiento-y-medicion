# Modelo de datos: HU-11 – Modificación de Proyecto (web)

## Esquema

Sin migración nueva. HU-11 usa estas tablas existentes:

| Tabla | Columnas usadas | Operación de HU-11 |
|---|---|---|
| `projects` | `id`, `name`, `start_date`, `end_date` | lee con bloqueo y actualiza nombre y fechas |
| `project_members` | `project_id`, `member_id` | lee, borra las bajas e inserta las altas |
| `members` | `id`, `display_name` | lee (existencia y nombre) |
| `effort_logs` | `story_id`, `member_id` | lee (horas de las bajas) |
| `user_stories` | `id`, `project_id` | lee (une horas con proyecto) |
| `sprints` | `project_id`, `number`, `start_date`, `end_date` | lee (solo si cambian las fechas) |

`projects.status` y `next_sprint_number` no se tocan. `CHECK (end_date > start_date)` (008) es la segunda barrera de
R2.

## Dominio (`src/project`)

### Tipos nuevos (`update.go`)

```go
type ProjectUpdateRequest struct {
    ID        int
    Name      string    // la web lo recorta y el servicio lo vuelve a recortar
    Members   []string  // identificadores de integrantes
    StartDate time.Time
    EndDate   time.Time
}

type ProjectUpdate struct { // lo que recibe el puerto
    ID        int
    Name      string
    MemberIDs []int // sin repetidos
    StartDate time.Time
    EndDate   time.Time
}

type SprintPeriod struct {
    Number    int
    StartDate time.Time
    EndDate   time.Time
}

type DateField string

const (
    DateFieldStart DateField = "inicio"
    DateFieldEnd   DateField = "fin"
)
```

### Errores nuevos (`update_errors.go`)

| Error | Cuándo | Datos |
|---|---|---|
| `ErrProjectNotFound` | `ID <= 0` o el proyecto no existe al guardar | — |
| `*MemberHasEffortError` | una baja tiene horas en historias del proyecto | `MemberName string` |
| `*SprintOutOfRangeError` | un Sprint que estaba dentro queda fuera | `Number int`, `Field DateField` |

Se reutilizan `ErrEmptyName`, `ErrMissingMembers`, `ErrInvalidPeriod`, `ErrMemberNotFound` y `ErrDuplicateMember`.

### Reglas puras (`update_rules.go`)

- `DatesChanged(oldStart, oldEnd, newStart, newEnd time.Time) bool`: compara año, mes y día (R7).
- `MembershipChanges(current, requested []int) (added, removed []int)`: ambas listas ordenadas y sin repetidos.
- `SprintsLeftOutside(oldStart, oldEnd, newStart, newEnd time.Time, sprints []SprintPeriod) *SprintOutOfRangeError`
  (R6). Devuelve `nil` si ningún Sprint queda fuera. Si no, devuelve el de menor número que cumple las dos
  condiciones:
  - estaba dentro de las fechas actuales: `!s.Start.Before(oldStart) && !s.End.After(oldEnd)`;
  - queda fuera de las nuevas: `s.Start.Before(newStart) || s.End.After(newEnd)`.

  `Field = DateFieldStart` si `s.Start.Before(newStart)`; si no, `DateFieldEnd`.

### Servicio (`update_service.go`)

`func (s *ProjectService) UpdateProject(ctx context.Context, req ProjectUpdateRequest) error`:

1. `req.ID <= 0` → `ErrProjectNotFound`.
2. `name := strings.TrimSpace(req.Name)`: R1 también vale para quien llama al servicio sin pasar por la web.
3. `parseMemberIDs(req.Members)`, extraída de `CreateProject` → `ErrMemberNotFound` o `ErrDuplicateMember`.
4. `NewProject(name, req.Members, req.StartDate, req.EndDate)` → `ErrEmptyName`, `ErrMissingMembers` o
   `ErrInvalidPeriod` (la misma regla que el alta).
5. `s.store.Update(ctx, ProjectUpdate{Name: name, ...})`. Los errores de dominio se devuelven tal cual; cualquier otro se
   envuelve con `fmt.Errorf("persist project update: %w", err)`.

### Puerto (`store.go`)

```go
type ProjectStore interface {
    Create(p *Project) error
    MemberExists(id int) (bool, error)
    Update(ctx context.Context, u ProjectUpdate) error // HU-11
}
```

`TestProjectStore` suma:

- `Update`, que guarda la última `ProjectUpdate` en `Updates []ProjectUpdate`;
- `ShouldFailUpdate bool`;
- `UpdateErr error`, para devolver errores de dominio en las pruebas del servicio y de los handlers.

## Transacción de guardado (`PostgreSQLProjectStore.Update`)

`BeginTx(ctx, nil)` (`READ COMMITTED`), `defer tx.Rollback()`:

```sql
-- 1. Bloquear el proyecto y leer las fechas actuales (sin filas → ErrProjectNotFound)
SELECT start_date, end_date FROM projects WHERE id = $1 FOR NO KEY UPDATE;

-- 2. Todos los integrantes pedidos existen (conteo distinto de len → ErrMemberNotFound)
SELECT COUNT(*) FROM members WHERE id = ANY($1::int[]);

-- 3. Pertenencia actual
SELECT member_id FROM project_members WHERE project_id = $1 ORDER BY member_id;
--    MembershipChanges(current, requested) → added, removed

-- 4. Bajas primero (toma el bloqueo de fila que espera a las cargas de horas en curso)
DELETE FROM project_members WHERE project_id = $1 AND member_id = ANY($2::int[]);

-- 5. Horas de las bajas en este proyecto (una fila → MemberHasEffortError)
SELECT m.display_name
FROM effort_logs el
JOIN user_stories us ON us.id = el.story_id
JOIN members m ON m.id = el.member_id
WHERE us.project_id = $1 AND el.member_id = ANY($2::int[])
ORDER BY m.display_name, m.id
LIMIT 1;

-- 6. Solo si DatesChanged(...)
SELECT number, start_date, end_date FROM sprints WHERE project_id = $1 ORDER BY number;
--    SprintsLeftOutside(...) → SprintOutOfRangeError

-- 7. Datos del proyecto
UPDATE projects SET name = $2, start_date = $3, end_date = $4 WHERE id = $1;

-- 8. Altas
INSERT INTO project_members (project_id, member_id)
SELECT $1, unnest($2::int[]) ON CONFLICT DO NOTHING;

COMMIT;
```

- Los pasos 4 y 5 se saltean si no hay bajas, y el 8 si no hay altas.
- Un guardado sin cambios ejecuta los pasos 1, 2, 3 y 7 y confirma: es un éxito (R9).
- Un error en cualquier paso deshace todo (R8).
- Los identificadores se pasan como `[]int32`, porque las columnas son `INTEGER`. Que pgx convierte el slice de Go
  a `int[]` con `database/sql` se verifica en la primera prueba de integración (T011). Si no lo hace, se cambia a
  un bucle de sentencias con un solo identificador.
- Un error 23503 (clave foránea) en el paso 8, por un integrante borrado entre los pasos 2 y 8, se traduce a
  `ErrMemberNotFound`.

## Cambio en HU-03 (`PostgreSQLEffortStore.SaveEntry`)

Dentro de la transacción existente, antes del INSERT:

```sql
SELECT 1
FROM user_stories us
JOIN project_members pm ON pm.project_id = us.project_id
WHERE us.id = $1 AND pm.member_id = $2
FOR SHARE OF pm;
```

- Sin filas: `errMemberNotInProject` (nuevo, sin exportar, en `src/effort/errors.go`), y no se guarda nada.
- `Service.LogEffort`: si `SaveEntry` devuelve `errMemberNotInProject`, responde
  `ValidationErrors: []FieldError{{Field: FieldMember, Err: ErrMemberRequired}}` en lugar de `StorageError`.

Orden de los bloqueos (decisión 6 de `research.md`):

| Llega primero | Qué pasa |
|---|---|
| La carga de horas | Bloquea la fila de pertenencia. El DELETE de HU-11 (paso 4) espera su `COMMIT`; el paso 5 ve las horas, HU-11 deshace todo y responde 409. |
| HU-11 | Borra la fila (paso 4). El `FOR SHARE` de HU-03 espera. Si HU-11 confirma, la fila no existe y la carga se rechaza; si HU-11 se deshace, la carga sigue. |

## Lectura de pantalla (`src/web`)

```go
// projects_catalog.go
type ProjectEdit struct {
    ID        int64
    Name      string
    StartDate time.Time
    EndDate   time.Time
    MemberIDs []int64 // ordenados
}

type ProjectCatalog interface {
    ListProjects(ctx context.Context) ([]ProjectRow, error)
    ListMembers(ctx context.Context) ([]MemberOption, error)
    ProjectForEdit(ctx context.Context, id int64) (ProjectEdit, error) // HU-11
}
```

```sql
SELECT p.name, p.start_date, p.end_date, pm.member_id
FROM projects p
LEFT JOIN project_members pm ON pm.project_id = p.id
WHERE p.id = $1
ORDER BY pm.member_id;
```

Sin filas: `NotFound`. Error de la base: `Unavailable`. `member_id` nulo (proyecto sin integrantes): `MemberIDs`
vacío.

## Modelo de vista (`src/web`)

```go
type projectEditPageData struct {
    ID              int64
    Members         []MemberOption
    SelectedMembers map[string]bool // reutiliza memberOptionKey
}
```

- `PageData.Title = TitleProjectEdit` ("Modificar proyecto").
- `Form`: `nombre`, `fecha_inicio` y `fecha_fin` con los valores actuales en el `GET` o los enviados en el `POST`.
- En un rechazo, `SelectedMembers` refleja los integrantes enviados (FR-017).

## Textos (`src/web/projects_form.go`)

| Constante | Texto |
|---|---|
| `TitleProjectEdit` | "Modificar proyecto" |
| `MsgProjectUpdated` | "Proyecto modificado exitosamente" |
| `MsgMemberHasEffort` | "No se puede quitar a %s: tiene horas registradas en este proyecto" |
| `MsgSprintOutOfRange` | "El Sprint %d queda fuera de las nuevas fechas del proyecto" |
| `LinkEditProject` | "Modificar" |

## Mapeo de errores en el handler

| Error | Campo | Código | Texto |
|---|---|---|---|
| errores de `parseProjectForm` | el del error | 422 | los de HU-21 |
| `ErrEmptyName` | `nombre` | 422 | `ErrEmptyName.Error()` |
| `ErrMissingMembers` | `integrantes` | 422 | `ErrMissingMembers.Error()` |
| `ErrInvalidPeriod` | `fecha_fin` | 422 | `ErrInvalidPeriod.Error()` |
| `ErrMemberNotFound` | `integrantes` | 422 | `MsgInvalidMember` |
| `ErrDuplicateMember` | `integrantes` | 422 | `ErrDuplicateMember.Error()` (no se alcanza: la web quita repetidos) |
| `*MemberHasEffortError` | `integrantes` | 409 | `MsgMemberHasEffort` con el nombre |
| `*SprintOutOfRangeError` | `fecha_inicio` o `fecha_fin` | 409 | `MsgSprintOutOfRange` con el número |
| `ErrProjectNotFound` | — | 404 | `Fail(NotFound(...))` |
| otro | — | 500 | `Flash{FlashError, MsgOperationFailed}` |

Los primeros cinco ya los resuelve `projectFieldError` de HU-21, que se reutiliza.
