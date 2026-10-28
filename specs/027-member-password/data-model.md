# Modelo de datos: HU-27 – Identificación con Contraseña

## Cambio de esquema

Migración nueva `db/migrations/<NNN>_add_member_password.up.sql` y `.down.sql`. `<NNN>` es la mayor versión en `main`
más 1, con tres dígitos: hoy 012; con HU-06 y HU-15 integradas, 014. Lo fija T001 y lo reconfirma T038.

```sql
-- up
ALTER TABLE members ADD COLUMN password_hash TEXT NULL;

-- down
ALTER TABLE members DROP COLUMN IF EXISTS password_hash;
```

| Tabla | Columna | Tipo | Regla |
|---|---|---|---|
| `members` | `password_hash` | `TEXT NULL` | `NULL` = sin contraseña configurada. Si no es nula, tiene el formato de abajo. La escribe solo el comando `set-password`. |

Los integrantes existentes quedan con `NULL`. No hay índice: siempre se lee por `id`.

## Valor guardado

```text
pbkdf2-sha256$600000$<sal>$<hash>
```

- `<sal>`: 16 bytes de `crypto/rand`, en base64 estándar sin relleno (22 caracteres).
- `<hash>`: `pbkdf2.Key(sha256.New, contraseña, sal, iteraciones, 32)`, en base64 estándar sin relleno (43
  caracteres).
- Iteraciones: 600.000 en producción. Al verificar se usan las del valor guardado. Se aceptan valores de 1 a
  10.000.000; cualquier otro, o un formato distinto, es `ErrMalformedHash`.

## Paquete `src/auth` (nuevo)

### Contraseñas (`password.go`)

```go
const (
    DefaultIterations = 600000
    MinPasswordLength = 8   // runas
    MaxPasswordLength = 128 // runas
    saltLength        = 16
    keyLength         = 32
    maxIterations     = 10000000
    scheme            = "pbkdf2-sha256"
)

var (
    ErrPasswordTooShort = errors.New("La contraseña debe tener al menos 8 caracteres")
    ErrPasswordTooLong  = errors.New("La contraseña no puede superar 128 caracteres")
    ErrPasswordMismatch = errors.New("Las contraseñas no coinciden")
    ErrMalformedHash    = errors.New("auth: valor de contraseña guardado mal formado")
)

func HashPassword(password string) (string, error)                         // DefaultIterations
func HashPasswordIterations(password string, iterations int) (string, error) // pruebas y fixtures; iterations < 1 → error
func VerifyPassword(password, stored string) (bool, error)                   // subtle.ConstantTimeCompare
func ValidateNewPassword(password, confirm string) error                     // largo de password y después coincidencia
```

| Entrada de `ValidateNewPassword` | Resultado |
|---|---|
| `"corta12"` (7), `"corta12"` | `ErrPasswordTooShort` |
| `"caro-clave-2026"` (15), `"caro-clave-2026"` | `nil` |
| 8 runas exactas, iguales | `nil` |
| `"ñandú-12"` (8 runas, 10 bytes), iguales | `nil` |
| 128 runas, iguales | `nil` |
| 129 runas, iguales | `ErrPasswordTooLong` |
| `"caro-clave-2026"`, `"caro-clave-2027"` | `ErrPasswordMismatch` |
| `"corta12"`, `"otra"` | `ErrPasswordTooShort` (el largo primero) |
| `" caro-clave-2026 "`, igual | `nil` (17 runas; los espacios cuentan) |

| Entrada de `VerifyPassword` | Resultado |
|---|---|
| `"ana-clave-2026"` con el valor de Ana | `(true, nil)` |
| `"otra-clave"` con el valor de Ana | `(false, nil)` |
| `""` con el valor de Ana | `(false, nil)` (la pantalla no llega a llamar) |
| `"ana-clave-2026"` con un valor de 1.000 iteraciones | `(true, nil)` |
| `""`, `"abc"`, 3 o 5 partes, esquema `"bcrypt"`, iteraciones `"0"`, `"-1"`, `"abc"` o `"10000001"`, sal o hash que no son base64, sal de 8 bytes, hash de 16 bytes | `(false, ErrMalformedHash)` |

### Bloqueo (`lockout.go`)

```go
const (
    MaxFailedAttempts = 5
    LockDuration      = time.Minute
)

type Lockout struct {
    mu      sync.Mutex
    now     func() time.Time
    entries map[int64]*lockoutEntry
}

type lockoutEntry struct {
    failures    int
    lockedUntil time.Time // cero si no está bloqueado
}

func NewLockout(now func() time.Time) *Lockout // nil → time.Now
func (l *Lockout) Locked(memberID int64) bool  // borra la entrada si el bloqueo venció
func (l *Lockout) RecordFailure(memberID int64) // al llegar a 5: lockedUntil = now + LockDuration
func (l *Lockout) RecordSuccess(memberID int64) // borra la entrada
```

Estados de un integrante:

```text
libre(0) --fallo--> libre(1) ... libre(4) --fallo--> bloqueado(hasta = t5 + 60 s)
libre(n) --acierto--> libre(0)
bloqueado --now >= hasta--> libre(0)
bloqueado --intento--> bloqueado (429, sin verificar ni contar)
```

### Puerto para guardar (`password_store.go` y `postgres_password_store.go`)

```go
var ErrMemberNotFound = errors.New("auth: el integrante no existe")

type PasswordStore interface {
    MemberName(ctx context.Context, memberID int64) (string, error)          // ErrMemberNotFound si no existe
    SetPasswordHash(ctx context.Context, memberID int64, hash string) error  // ErrMemberNotFound si no existe
}

type PostgreSQLPasswordStore struct{ db *sql.DB }

func NewPostgreSQLPasswordStore(db *sql.DB) *PostgreSQLPasswordStore
```

```sql
-- MemberName
SELECT display_name FROM members WHERE id = $1;                      -- sin filas → ErrMemberNotFound
-- SetPasswordHash
UPDATE members SET password_hash = $2 WHERE id = $1;                   -- 0 filas → ErrMemberNotFound
```

Un `memberID` fuera de 1..2147483647 se responde con `ErrMemberNotFound`, sin consultar.

## Paquete `src/web` (cambios)

### `ContextStore` (HU-20b, se amplía)

```go
// MemberCredential es lo que la pantalla "Identificarme" necesita de un integrante (HU-27).
type MemberCredential struct {
    Belongs      bool   // pertenece al proyecto
    Name         string // display_name
    PasswordHash string // "" si password_hash es NULL
}

type ContextStore interface {
    ListProjects(ctx context.Context) ([]ProjectSummary, error)
    ListMembers(ctx context.Context, projectID int64) ([]MemberSummary, error)
    ResolveContext(ctx context.Context, projectID, memberID int64) (ResolvedContext, error)
    // MemberCredential lee el nombre y la contraseña guardada de un integrante del proyecto (HU-27).
    // Que no pertenezca no es un error; solo lo es la base no disponible (Unavailable).
    MemberCredential(ctx context.Context, projectID, memberID int64) (MemberCredential, error)
}
```

```sql
SELECT m.display_name, m.password_hash
FROM project_members pm
JOIN members m ON m.id = pm.member_id
WHERE pm.project_id = $1 AND pm.member_id = $2
-- sin filas → MemberCredential{} (Belongs=false); NULL → PasswordHash ""
```

`validID` en falso para cualquiera de los dos identificadores → `MemberCredential{}` sin consultar.

| Caso (datos de ejemplo) | `MemberCredential` |
|---|---|
| «Alfa», Ana | `{true, "Ana", "pbkdf2-sha256$…"}` |
| «Alfa», Caro | `{true, "Caro", ""}` |
| «Alfa», Dani (de «Beta») | `{false, "", ""}` |
| «Alfa», 0 o 2147483648 | `{false, "", ""}` sin consultar |
| proyecto inexistente, Ana | `{false, "", ""}` |
| base cerrada | error `Unavailable` |

### Registro y rutas

```go
const pathLogout = "/contexto/salir"

func RegisterContext(s *Server, store ContextStore) // sin cambios de firma: usa auth.NewLockout(nil)
func RegisterContextWithLockout(s *Server, store ContextStore, lockout *auth.Lockout)
```

| Método y ruta | Handler | Cambio |
|---|---|---|
| `GET /contexto/integrante` | `showMembers` | Mismo comportamiento; la plantilla suma el campo de contraseña. |
| `POST /contexto/integrante` | `selectMember` | Pide `contrasena` (tabla de respuestas). |
| `POST /contexto/salir` | `logout` (nuevo) | Borra `ctx_integrante`, avisa con `SetFlash` y redirige con 303. |
| `GET /contexto/salir` | — | 405 de HU-20a. |

### Datos de la pantalla

```go
type selectMemberData struct {
    Members     []MemberSummary
    ActiveID    int64  // integrante marcado: el identificado o, en un rechazo, el elegido
    ReturnTo    string
    ProjectName string
    NoMembers   string
}
```

- El error del campo va en `PageData.Form` (`Errors["contrasena"]`).
- Los demás rechazos van en `PageData.Flash` como `FlashError`.
- `Form.Values` nunca lleva `contrasena`.

### Respuestas de `POST /contexto/integrante`

| Paso | Condición | Estado | Pantalla y mensaje | Cookie `ctx_integrante` | Bloqueo |
|---|---|---|---|---|---|
| 1 | Sin proyecto activo | 303 a `/contexto/proyecto?volver_a=…` | `MsgChooseProjectFirst` (HU-20b) | sin cambios | — |
| 2 | Error al leer | 503 o 500 | página de error de HU-20a | sin cambios | — |
| 3 | `!Belongs` | 303 a `/contexto/integrante?volver_a=…` | `MsgMemberInvalid` (HU-20b) | se borra | — |
| 4 | `contrasena == ""` | 422 | selector, elegido marcado, "Ingresá tu contraseña" junto al campo | sin cambios | — |
| 5 | `Locked(id)` | 429 | selector, alerta "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar" | sin cambios | — |
| 6 | `PasswordHash == ""` | 422 | selector, alerta "<Nombre> todavía no tiene contraseña configurada. Pedile al Scrum Master que la configure" | sin cambios | — |
| 7 | `VerifyPassword` con error | 500 | página de error de HU-20a | sin cambios | — |
| 8 | Contraseña incorrecta | 401 | selector, alerta "Integrante o contraseña incorrectos" | sin cambios | `RecordFailure` |
| 9 | Contraseña correcta | 303 al destino saneado | flash de éxito "Te identificaste como <Nombre>" | se emite firmada | `RecordSuccess` |

`POST /contexto/salir`: `clearContextCookie(CookieMember)`, `SetFlash(FlashSuccess, MsgLoggedOut)` y 303 a
`SanitizeReturnTo(volver_a)` ("/" por defecto). No exige proyecto ni integrante.

### Textos (`src/web/member_password_messages.go`, nuevo)

| Constante o función | Texto |
|---|---|
| `LabelPassword` | "Contraseña" |
| `MsgPasswordRequired` | "Ingresá tu contraseña" |
| `MsgWrongCredentials` | "Integrante o contraseña incorrectos" |
| `MsgTooManyAttempts` | "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar" |
| `NoPasswordMessage(name)` | "<name> todavía no tiene contraseña configurada. Pedile al Scrum Master que la configure" |
| `IdentifiedMessage(name)` | "Te identificaste como <name>" |
| `MsgLoggedOut` | "Cerraste tu identificación" |
| `ButtonLogout` | "Salir" |
| Introducción de `select_member.html` | "Proyecto: <nombre>. Elegí tu nombre e ingresá tu contraseña. Tus acciones (cargar esfuerzo, votar) quedan a tu nombre hasta que salgas." |

### Plantillas

- `select_member.html`:
  - nueva introducción;
  - después de `member_picker`, un bloque `field` propio con
    `<input type="password" id="contrasena" name="contrasena" autocomplete="current-password">`, sin `value`, y el
    error con las clases `field-invalid` y `field-error` y `aria-describedby`;
  - botón "Identificarme".
- `layout.html` (HU-20a):
  - `<p class="app-context">` pasa a `<div class="app-context">`;
  - si `.MemberName`, después del nombre va
    `<form method="post" action="/contexto/salir" class="logout-form"><button type="submit">Salir</button></form>`.
- `static/app.css` (HU-20a): `.app-context form { display: inline; }`.

## Comando `src/cmd/set-password` (nuevo)

```go
func main() // os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Getenv))
func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) int
func setPassword(ctx context.Context, store auth.PasswordStore, memberID int64, stdin io.Reader, stdout io.Writer) int
```

| Caso | Salida | Código | Cambia la base |
|---|---|---|---|
| Sin `-integrante`, o con un valor no numérico, ≤ 0 o > 2147483647 | "Uso: go run ./src/cmd/set-password -integrante N" | 2 | No |
| Sin `DATABASE_URL` | "falta la variable de entorno obligatoria DATABASE_URL" | 1 | No |
| No se puede conectar o falla la consulta | "No se pudo acceder a la base de datos" (sin la dirección) | 1 | No |
| Integrante inexistente | "El integrante no existe" (sin pedir la contraseña) | 1 | No |
| Contraseña corta | "La contraseña debe tener al menos 8 caracteres" | 1 | No |
| Contraseña de más de 128 | "La contraseña no puede superar 128 caracteres" | 1 | No |
| No coinciden | "Las contraseñas no coinciden" | 1 | No |
| Caro con "caro-clave-2026" dos veces | "Contraseña configurada para Caro" | 0 | `password_hash` de Caro |

Pide "Contraseña: " y "Repetí la contraseña: " en `stdout`; lee dos líneas de `stdin` y quita solo `\n` y `\r`
finales. Nunca imprime lo que lee.

## Datos de prueba

| Dato | Valor | Uso |
|---|---|---|
| «Alfa» | Ana, Beto, Caro | Escenarios 1 a 9 y 12 |
| Ana | "ana-clave-2026" | Escenarios 1, 2, 5, 7, 8, 9 |
| Beto | "beto-clave-2026"; en el escenario 9, "ana-clave-2026" | Escenarios 3 y 9 |
| Caro | sin contraseña; en el escenario 10 queda "caro-clave-2026" | Escenarios 4, 10 y 11 |
| «Beta» | Dani, "dani-clave-2026" (agregada) | Escenarios 6 y 8 |
| Contraseña incorrecta | "otra-clave" | Escenarios 2 y 5 |
| Iteraciones de las fixtures | 1.000 (`HashPasswordIterations`) | Todas, salvo `TestHashPassword_IteracionesReales` y las que pasan por el comando |

Cada prueba crea sus proyectos e integrantes y los borra en `t.Cleanup`, en este orden: `project_members`, proyectos e
integrantes. Los integrantes se llaman exactamente "Ana", "Beto", "Caro" y "Dani" (`display_name` no es único), porque
los mensajes esperados los nombran. Solo los proyectos llevan un sufijo único si hace falta para no chocar con datos de
otras pruebas. Las pruebas buscan por identificador, nunca por nombre.
