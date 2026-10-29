# Modelo de datos: HU-28 – Roles del Proyecto (mínimos)

## Cambio de esquema

Migración nueva `db/migrations/<NNN>_add_project_member_role.up.sql` y `.down.sql`. `<NNN>` es la mayor versión en
`main` más 1, después de la de HU-27 (T001: HU-27). Lo fija T001 y lo reconfirma T033.

```sql
-- up
ALTER TABLE project_members
    ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'Integrante'
    CHECK (role IN ('Scrum Master', 'Integrante'));

-- down
ALTER TABLE project_members DROP COLUMN IF EXISTS role;
```

| Tabla | Columna | Tipo | Regla |
|---|---|---|---|
| `project_members` | `role` | `VARCHAR(20) NOT NULL DEFAULT 'Integrante'` | Solo 'Scrum Master' o 'Integrante'. Las filas existentes y las que inserten HU-01 y HU-11 quedan en 'Integrante'. Solo la cambia el comando `set-role`. |

No hay índice nuevo: el `EXISTS` filtra por `project_id`, primera columna de la PK.

## Paquete `src/role` (nuevo)

### Regla (`role.go`)

```go
type Role string

const (
    RoleScrumMaster Role = "Scrum Master"
    RoleMember      Role = "Integrante"
)

var ErrInvalidRole = errors.New("role: rol no válido")

func ParseRole(s string) (Role, error) // TrimSpace y coincidencia exacta; si no, ErrInvalidRole
func CanManage(r Role) bool             // r == RoleScrumMaster
```

| Entrada de `ParseRole` | Resultado |
|---|---|
| `"Scrum Master"`, `"  Scrum Master "` | `RoleScrumMaster` |
| `"Integrante"` | `RoleMember` |
| `"scrum master"`, `"SM"`, `"Product Owner"`, `""` | `ErrInvalidRole` |

| `CanManage` | Resultado |
|---|---|
| `RoleScrumMaster` | `true` |
| `RoleMember`, `""` | `false` |

### Puerto y adaptador (`store.go`, `postgres_store.go`)

```go
var (
    ErrProjectNotFound    = errors.New("role: el proyecto no existe")
    ErrMemberNotInProject = errors.New("role: el integrante no pertenece al proyecto")
)

type Store interface {
    ProjectName(ctx context.Context, projectID int64) (string, error)
    SetMemberRole(ctx context.Context, projectID, memberID int64, r Role) (memberName string, err error)
}

type PostgreSQLStore struct{ db *sql.DB }

func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore
```

```sql
-- ProjectName (sin filas → ErrProjectNotFound)
SELECT name FROM projects WHERE id = $1;

-- SetMemberRole (sin filas → ErrMemberNotInProject)
UPDATE project_members pm
SET role = $3
FROM members m
WHERE pm.project_id = $1 AND pm.member_id = $2 AND m.id = pm.member_id
RETURNING m.display_name;
```

Un identificador fuera de 1..2147483647 se responde con el error que corresponde, sin consultar.

## Paquete `src/web` (cambios)

### Contexto (HU-20b, se amplía)

```go
type ResolvedContext struct {
    ProjectExists  bool
    ProjectName    string
    MemberBelongs  bool
    MemberName     string
    MemberRole     role.Role // HU-28: vacío si el integrante no pertenece
    HasScrumMaster bool      // HU-28: el proyecto tiene al menos un Scrum Master
}

type ActiveContext struct {
    ProjectID      int64
    ProjectName    string
    MemberID       int64
    MemberName     string
    MemberRole     role.Role // HU-28
    HasScrumMaster bool      // HU-28
}
```

```sql
-- resolveContextQuery (se suman dos columnas)
SELECT p.name, m.display_name, pm.role,
       EXISTS (SELECT 1 FROM project_members sm
               WHERE sm.project_id = p.id AND sm.role = 'Scrum Master')
FROM projects p
LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.member_id = $2
LEFT JOIN members m ON m.id = pm.member_id
WHERE p.id = $1
```

- `contextMiddleware` copia `HasScrumMaster` siempre que el proyecto existe, y `MemberRole` solo cuando completa al
  integrante.
- Un valor de `role` que `ParseRole` no reconoce (imposible por el `CHECK`) se trata como `RoleMember`.

| Caso (datos de ejemplo) | `MemberRole` | `HasScrumMaster` |
|---|---|---|
| «Alfa», Ana | "Scrum Master" | true |
| «Alfa», Beto | "Integrante" | true |
| «Alfa», sin integrante | "" | true |
| «Beta», Dani | "Integrante" | false |
| «Alfa», Dani (no pertenece) | "" | true |

### Control (`src/web/scrum_master.go`, nuevo)

```go
const (
    TitleScrumMasterOnly = "Acción reservada al Scrum Master"
    MsgScrumMasterOnly   = "Esta acción la puede hacer solo el Scrum Master del proyecto"
    MsgNoScrumMaster     = "Este proyecto no tiene Scrum Master asignado. Pedí que se configure con el comando de roles"
    MsgScrumMasterOtherProject = "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master"
)

func (s *Server) RequireScrumMaster(h http.Handler) http.Handler
func (s *Server) RequireScrumMasterOfPathProject(h http.Handler) http.Handler // HU-11: {id} == proyecto activo
```

| Paso | Condición | Respuesta | ¿Llama a `h`? |
|---|---|---|---|
| 1 | Sin proyecto activo | `SetFlash(FlashWarning, MsgProjectRequired)` y 303 a `/contexto/proyecto?volver_a=/<segmento>` | No |
| 2 | Sin integrante identificado | `SetFlash(FlashWarning, MsgMemberRequired)` y 303 a `/contexto/integrante?volver_a=/<segmento>` | No |
| 2b | Solo en `…OfPathProject`: `parseID(r.PathValue("id")) != ProjectID` (distinto, inválido o inexistente; no se consulta el rol en otro proyecto) | 403, `TitleScrumMasterOnly`, `MsgScrumMasterOtherProject` | No |
| 3 | `role.CanManage(MemberRole)` | la acción | Sí |
| 4 | `!HasScrumMaster` | 403, `TitleScrumMasterOnly`, `MsgNoScrumMaster` | No |
| 5 | En otro caso | 403, `TitleScrumMasterOnly`, `MsgScrumMasterOnly` | No |

`<segmento>` es `firstSegment(r.URL.Path)`.

### Rutas que se envuelven

| Archivo | Historia | Patrón | Envoltorio |
|---|---|---|---|
| `src/web/projects_handlers.go` | HU-11 (T001) | `POST /proyectos/{id}/modificar` | `RequireScrumMasterOfPathProject` |
| `src/web/sprints_handlers.go` | HU-23 | `POST /sprints`, `POST /sprints/historias`, `POST /sprints/historias/quitar` | `RequireScrumMaster` |
| `src/web/close_handlers.go` | HU-15 (T001) | `POST /cierre` | `RequireScrumMaster` |
| `src/web/poker_handlers.go` | HU-05a | `POST /planning-poker/abrir` | `RequireScrumMaster` |
| `src/web/poker_handlers.go` | HU-05b (T001) | `POST /planning-poker/cerrar` | `RequireScrumMaster` |
| `src/web/poker_handlers.go` | HU-18 (T001) | `POST /planning-poker/nueva-ronda`, `POST /planning-poker/estimacion` | `RequireScrumMaster` |

### Estructura común (`layout.html`, HU-20a/HU-27)

```text
Integrante: <strong>{{.MemberName}} ({{.MemberRole}})</strong>
```

Solo si hay `MemberName`. Ejemplos: "Integrante: Ana (Scrum Master)" e "Integrante: Beto (Integrante)".

## Comando `src/cmd/set-role` (nuevo)

```go
func main() // os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Getenv))
func run(ctx context.Context, args []string, stdout io.Writer, getenv func(string) string) int
func assignRole(ctx context.Context, store role.Store, projectID, memberID int64, r role.Role, stdout io.Writer) int
```

| Caso | Salida | Código | Cambia la base |
|---|---|---|---|
| Falta `-proyecto`, `-integrante` o `-rol`, o un número no es entero de 1 a 2147483647 | "Uso: go run ./src/cmd/set-role -proyecto P -integrante N -rol \"Scrum Master\"" | 2 | No |
| `-rol` no válido | "El rol no es válido" (sin conectarse) | 1 | No |
| Sin `DATABASE_URL` | "falta la variable de entorno obligatoria DATABASE_URL" | 1 | No |
| No se puede conectar o falla la consulta | "No se pudo acceder a la base de datos" | 1 | No |
| Proyecto inexistente | "El proyecto no existe" | 1 | No |
| Integrante inexistente o de otro proyecto | "El integrante no pertenece al proyecto" | 1 | No |
| Beto, «Alfa», "Scrum Master" | "Beto ahora es Scrum Master de «Alfa»" | 0 | `role` de Beto en «Alfa» |
| Ana, «Alfa», "Integrante" | "Ana ahora es Integrante de «Alfa»" | 0 | `role` de Ana en «Alfa» |

## Datos de prueba

| Dato | Valor | Uso |
|---|---|---|
| «Alfa» | Ana "Scrum Master"; Beto y Caro "Integrante"; Sprint 2 en curso con «Login»; ronda 1 de «Login» abierta; «Perfil» en el Backlog sin Sprint | Escenarios 1 a 8, 10 y 12 |
| «Beta» | Dani "Integrante", sin Scrum Master; «Reportes» en el Backlog | Escenario 9 |
| Contraseñas (T001: HU-27) | "ana-clave-2026", "beto-clave-2026", "caro-clave-2026", "dani-clave-2026", guardadas con 1.000 iteraciones | Identificación |
| Sprint 1 de «Alfa» | ya finalizado, para que el Sprint 2 sea el número correcto | Escenarios 2, 3, 8 y 10 |

- Cada prueba crea sus datos y los borra en `t.Cleanup`, en este orden: rondas y votos, asignaciones, esfuerzo,
  `sprint_metrics`, historias, Sprints, `project_members`, proyectos e integrantes.
- Los integrantes se llaman exactamente "Ana", "Beto", "Caro" y "Dani"; las pruebas buscan por identificador.
