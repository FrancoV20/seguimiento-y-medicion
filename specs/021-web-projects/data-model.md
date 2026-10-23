# Data Model: Interfaz web de proyectos

Esta feature **no crea tablas ni migraciones**. El alta usa el esquema de HU-01 (migración `001`) a
través del `ProjectService` y el `ProjectStore` existentes. Las lecturas de pantalla usan las
tablas `projects`, `project_members` y `members`. Los integrantes se cargan con
`db/seeds/members.sql`, que no es una migración.

## Verificación de HU-01 (T001)

- Paquete: `src/project`.
- `ProjectService.CreateProject(req ProjectCreationRequest) (*ProjectCreationResult, error)`.
- `ProjectStore` define `Create(p *Project) error` y `MemberExists(id int) (bool, error)`.
- Errores y textos exactos: `ErrEmptyName` ("project name cannot be empty"),
  `ErrMissingMembers` ("the member list is required"), `ErrInvalidPeriod`
  ("La fecha de finalización debe ser posterior a la fecha de inicio"),
  `ErrMemberNotFound` ("member does not exist") y `ErrDuplicateMember`
  ("duplicate members are not allowed"). El servicio convierte identificadores no numéricos a
  `ErrMemberNotFound`.
- HU-01 no normaliza el nombre: rechaza únicamente `""`, acepta texto solo con espacios y no limita
  su longitud en el servicio. La columna `projects.name` limita el almacenamiento a 255 caracteres.
  Por decisión para cumplir esta spec sin modificar HU-01, el formulario de HU-21 recortará los
  espacios extremos antes del alta y rechazará el resultado vacío con el texto exacto de
  `ErrEmptyName`. La validación adicional de más de 255 caracteres sigue siendo la tarea extra T027.
- El mensaje de éxito de `ProjectCreationResult` es "Proyecto creado exitosamente".
- FIX-HU-01 está en `main` (merge `bb4e1e5`), rechaza fin igual o anterior al inicio y define el
  texto de `ErrInvalidPeriod` indicado arriba. La migración `008` quedó aplicada en la base local
  durante esta verificación (versión actual: 8).

## Tablas leídas (definidas por HU-01)

- `members(id SERIAL PK, display_name VARCHAR(255) NOT NULL)`: catálogo global; el nombre no es
  único.
- `projects(id SERIAL PK, name VARCHAR(255) NOT NULL, start_date DATE NOT NULL, end_date DATE NOT
  NULL, status VARCHAR(50) DEFAULT 'Activo')` con `CHECK (end_date > start_date)` desde FIX-HU-01
  (antes `>=`).
- `project_members(project_id, member_id)` con clave primaria compuesta y borrado en cascada.

`projects.status` no se usa en esta historia.

## ProjectForm

Datos del envío, para conservarlos al volver a mostrar el formulario.

### Fields

- `name`: texto del nombre.
- `startDate`, `endDate`: texto de cada fecha tal como llegó.
- `memberIDs`: identificadores de integrantes marcados, como texto.

### Rules

- Los valores se devuelven tal como se enviaron cuando hay errores.
- Un campo ausente en el envío se trata como vacío.

## ProjectFormInput

Resultado de parsear el envío para invocar el alta.

### Fields

- `name`.
- `startDate`, `endDate`: fechas de calendario (`AAAA-MM-DD`).
- `memberIDs`: identificadores sin repetidos.

### Rules

- Una fecha vacía o con formato distinto de `AAAA-MM-DD` genera un error junto a su campo.
- Un identificador no numérico genera un error junto al campo de integrantes.
- Los identificadores repetidos se toman una sola vez.
- El nombre se recorta antes del alta; si queda vacío, se muestra `ErrEmptyName` sin invocar HU-01.
- La coherencia de fechas e integrantes existentes la valida HU-01.

## ProjectRow

Fila de la lista de proyectos.

### Fields

- `id`.
- `name`.
- `startDate`, `endDate`.
- `memberCount`: cantidad de integrantes del proyecto.

### Rules

- Se ordenan por `id` descendente (los más recientes primero).
- Un proyecto sin integrantes (no debería existir por HU-01) muestra `memberCount` en cero.

## MemberOption

Integrante ofrecido en el formulario.

### Fields

- `id`.
- `displayName`.

### Rules

- Se ordenan por nombre y luego por `id`, porque el nombre puede repetirse.

## FieldErrors

Mensajes de error por campo para `FormView` de HU-20b.

### Fields

- Claves: `nombre`, `fecha_inicio`, `fecha_fin`, `integrantes`.
- Valores: texto del mensaje.

### Messages

- Nombre, fecha de fin anterior o igual a la de inicio (FIX-HU-01) y sin integrantes: los de HU-01,
  sin cambios; se toman de su código (T001).
- Fecha faltante: "Falta la fecha de inicio" o "Falta la fecha de fin" (`MsgMissingStartDate`,
  `MsgMissingEndDate`; propuesta).
- Formato de fecha inválido: "La fecha no tiene un formato válido" (`MsgInvalidDateFormat`).
- Integrante no válido: "El integrante seleccionado no es válido" (`MsgInvalidMember`).
- Sin integrantes cargados: "No hay integrantes disponibles" (`MsgNoMembersAvailable`).
- Éxito: "Proyecto creado exitosamente" (`MsgProjectCreated`, coincide con el resultado de HU-01).
- Falla de guardado: `MsgOperationFailed` de HU-20b.

## ProjectCatalog

Interfaz interna de solo lectura para la pantalla.

### Operations

- `ListProjects(ctx)`: devuelve los `ProjectRow`, con la cantidad de integrantes de cada uno.
- `ListMembers(ctx)`: devuelve los `MemberOption` de todo el catálogo.

### Rules

- Distingue errores de lectura de resultados vacíos: una lista vacía no es un error.

## PostgreSQLProjectCatalog

Implementación concreta de `ProjectCatalog` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` mediante `database/sql`.

### Queries

```sql
-- ListProjects
SELECT p.id, p.name, p.start_date, p.end_date, COUNT(pm.member_id)
FROM projects p
LEFT JOIN project_members pm ON pm.project_id = p.id
GROUP BY p.id
ORDER BY p.id DESC;

-- ListMembers
SELECT id, display_name FROM members ORDER BY display_name, id;
```

### Rules

- Solo realiza lecturas; el alta pasa por el `ProjectStore` de HU-01.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse; las pruebas crean
  sus propios proyectos e integrantes y los eliminan al terminar.

## Semilla de integrantes

`db/seeds/members.sql` inserta los cinco integrantes del equipo sin duplicar si se ejecuta más de
una vez.

### Rules

- No es una migración: no lleva número, no tiene par `.down.sql` y no la ejecuta `migrate`.
- Se ejecuta a mano una vez por base local (ver `quickstart.md`).
