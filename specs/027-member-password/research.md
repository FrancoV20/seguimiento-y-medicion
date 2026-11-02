# Investigación: HU-27 – Identificación con Contraseña

## Datos verificados en `main` (`357f550`, 28/10/2026)

Verificados con `git grep -n`.

**HU-20b, contexto de trabajo**

- `src/web/context_store.go:6`: `ContextStore` con `ListProjects`, `ListMembers` y
  `ResolveContext(ctx, projectID, memberID) (ResolvedContext, error)` (línea 13).
- `src/web/postgres_context_store.go`:
  - `NewPostgreSQLContextStore(db *sql.DB)` (línea 18) y `ResolveContext` (línea 86);
  - `validID` deja afuera los identificadores fuera de 1..`MaxID` sin consultar (`src/web/context.go:21`);
  - los errores de la base se devuelven con `Unavailable(err)`.
- `src/web/context_handlers.go`:
  - `pathSelectMember = "/contexto/integrante"` (línea 11);
  - `RegisterContext(s *Server, store ContextStore)` (línea 16) registra el middleware y las cuatro rutas de
    selección;
  - `selectMember` (línea 143): integrante que no pertenece → `clearContextCookie(CookieMember)`,
    `SetFlash(FlashError, MsgMemberInvalid)` y 303 a `pathSelectMember?volver_a=…`; si pertenece →
    `setContextCookie(CookieMember, id)` y 303 al destino. Hoy no muestra ningún mensaje de éxito.
- `src/web/messages.go:32`: `MsgMemberInvalid = "El integrante elegido no pertenece al proyecto. Elegí uno de la
  lista."`. `MsgMemberRequired` es el texto de `RequireContext` cuando falta integrante.
- `src/web/context.go`:
  - `CookieMember = "ctx_integrante"` (línea 13);
  - `setContextCookie` (línea 78) firma con `SignedContextValue` (HU-26);
  - `clearContextCookie` (línea 83).
- `src/web/context_middleware.go`:
  - `contextMiddleware` valida las cookies en cada solicitud con `ResolveContext`;
  - `RequireContext` redirige a `pathSelectMember` con `MsgMemberRequired` cuando la sección tiene
    `RequiresMember: true`; las tienen `EffortSection` (`effort_handlers.go:17`) y la de Planning Poker
    (`poker_handlers.go:49`).
- `src/web/context_handlers.go`, `selectProject`: al cambiar a otro proyecto borra la cookie del integrante y al
  elegir el mismo la conserva. El escenario 8 ya se cumple con este código.
- `src/web/templates/select_member.html`:
  - formulario `POST /contexto/integrante` con `volver_a` y el parcial `member_picker` (radios `integrante_id`,
    marcado si `eq .ID $.ActiveID`);
  - el texto actual dice "No hace falta contraseña…".

**HU-20a, estructura común y errores**

- `src/web/templates/layout.html:19`:
  - `<p class="app-context">` muestra "Integrante: <strong>Nombre</strong>" y los enlaces "Cambiar integrante" o
    "Identificarme";
  - el layout no tiene formularios;
  - un `<form>` no puede ir dentro de un `<p>`.
- `src/web/static/app.css:41`: los estilos usan la clase `.app-context`, no la etiqueta.
- `src/web/render.go`:
  - `PageData` tiene `Flash *Flash` y `Form *FormView` (líneas 34 a 42);
  - `PageStatus(w, r, status, name, data)` (línea 107) usa `data.Flash` si viene cargado.
- `src/web/errors.go:115`: `Renderer.Fail` → 404, 503 (`KindUnavailable` o `IsDatabaseUnavailable`) o 500. El
  detalle va solo al registro.
- `src/web/forms.go`:
  - `FormView{Values, Errors}`, `AddError` y `Field`;
  - el parcial `field` del layout escribe `value="{{.Value}}"`, así que no sirve para la contraseña.
- `src/web/server.go`: `Server.route` responde 405 con la página común cuando la ruta existe con otro método.
- `src/web/security.go:10`: `CrossOriginGuard` (403) lo registra `main.go` para todo el servidor.
- `src/web/middleware.go`: el único registro por solicitud es el de pánicos (método, ruta, pánico y traza); no se
  registran formularios.
- `src/web/flash.go:62`: `SetFlash(w, r, kind, text)`; tipos `FlashSuccess`, `FlashError` y `FlashWarning`.
- `src/web/return_to.go:8`: `SanitizeReturnTo(raw, sections)` devuelve "/" si el destino no es seguro.
- `src/web/config.go:30`: `LoadConfig(getenv)` devuelve `*MissingEnvError` ("falta la variable de entorno
  obligatoria DATABASE_URL").

**Servidor, base y CI**

- `src/cmd/server/main.go`:
  - llama a `web.RegisterContext(srv, web.NewPostgreSQLContextStore(database))`;
  - después de `OpenDatabase` compara la versión del esquema con `db.LatestVersion()`, así que la migración nueva
    exige migrar antes de arrancar.
- Base de datos:
  - `db/migrations/001_create_projects.up.sql` define `members (id SERIAL, display_name VARCHAR(255) NOT NULL)` sin
    columnas de contraseña;
  - migraciones presentes: 001, 002, 003, 004, 007, 008 y 011;
  - `db/migrations_integration_test.go` aplica todas las `up` sobre una base descartable
    (`TestMigraciones_SeAplicanEnOrdenSobreUnaBaseDescartable`) y exige el par up/down
    (`TestMigraciones_VersionUnicaYParUpDown`).
- `.github/workflows/ci.yml`:
  - `gofmt -l .`, `go vet ./...` y `migrate … up`;
  - después, `go test ./... -p 1 -race -coverprofile=coverage.out`.
- `go.mod`: `go 1.27.1`, una sola dependencia directa (`pgx/v5`).
- `go doc crypto/pbkdf2.Key` con el Go del proyecto:
  `func Key[Hash hash.Hash](h func() Hash, password string, salt []byte, iter, keyLength int) ([]byte, error)`. Está
  disponible y no hace falta avisar. `crypto/subtle.ConstantTimeCompare` también existe.
- `src/effort/service.go:17`: `NewService(store, now func() time.Time)` es el patrón del repositorio para inyectar el
  reloj (`nil` → `time.Now`).
- Los adaptadores de dominio viven en su paquete (`effort.NewPostgreSQLEffortStore(db *sql.DB)`,
  `poker.NewPostgreSQLStore`) e importan `_ "github.com/jackc/pgx/v5/stdlib"`.

**Pruebas que se identifican por `POST /contexto/integrante`**

Hoy son **8 archivos**, no 6 como dice la nota 6 del equipo; los dos de más son `effort_handlers_test.go` y
`poker_view_test.go`:

| Archivo | Historia | Dónde |
|---|---|---|
| `src/web/context_member_test.go` | HU-20b | `selectMember` (línea 16); además, línea 69: espera que no haya `type="password"` |
| `src/web/effort_handlers_test.go` | HU-24 | líneas 78 y 288 (`postTo`) |
| `src/web/poker_handlers_test.go` | HU-05a | líneas 80 y 416 |
| `src/web/poker_view_test.go` | HU-05a | línea 65 |
| `test/context_bdd_test.go` | HU-20b | `elegirIntegrante` (línea 437); además, línea 505: espera que no haya `type="password"` |
| `test/web_effort_bdd_test.go` | HU-24 | `dadoQueTrabajoEn` (línea 196) y línea 298 |
| `test/web_poker_bdd_test.go` | HU-05a | líneas 104 y 305 |
| `test/web_signed_cookies_bdd_test.go` | HU-26 | línea 71 |

`src/web/context_guard_test.go` nombra `pathSelectMember` solo para comprobar redirecciones y no cambia.

**Implementaciones de `ContextStore` en pruebas**

Al ampliar la interfaz dejan de compilar dos dobles completos de HU-23:

- `sprintHandlerContextStore` (`src/web/sprints_handlers_test.go:39`);
- `sprintBDDContextStore` (`test/web_sprints_bdd_test.go:33`).

Los que embeben la interfaz (`sinProyectos`, `listaFija`) siguen compilando.

**Llamadas a `RegisterContext`**: 15 (1 en `main.go` y 14 en 11 archivos de prueba). Por eso no se cambia su firma.

## Decisiones

1. **`src/auth` con funciones puras y un adaptador en el mismo paquete** (decisión del equipo).
   - Motivo: las reglas de la contraseña no dependen de la web y las usan la pantalla y el comando. El puerto
     `PasswordStore` y `PostgreSQLPasswordStore` van en `src/auth`, como `effort.EffortStore` y su adaptador.
   - Alternativa descartada: ponerlo todo en `src/web`; el comando tendría que importar la capa web para guardar.

2. **Formato `pbkdf2-sha256$<iteraciones>$<sal>$<hash>` con base64 estándar sin relleno** (`RawStdEncoding`).
   - Motivo: el `$` no aparece en base64; sin relleno queda más corto (sal de 22 caracteres, hash de 43, unos 88 en
     total). Las iteraciones del valor guardado permiten subirlas más adelante sin invalidar lo guardado.
   - Validación al leer: 4 partes, esquema `pbkdf2-sha256`, iteraciones enteras entre 1 y 10.000.000, sal de 16 bytes
     y hash de 32. Si no cumple → `ErrMalformedHash` → 500. El tope evita que un valor cargado a mano deje colgado el
     servidor.
   - Alternativa descartada: bcrypt o argon2, porque necesitan `golang.org/x/crypto` (dependencia nueva).

3. **Iteraciones inyectables solo para pruebas y fixtures.**
   - `HashPassword` usa siempre `DefaultIterations = 600000`.
   - `HashPasswordIterations(password, iterations)` existe para pruebas y fixtures (como `SignedContextValue` en
     HU-26).
   - Como `VerifyPassword` lee las iteraciones del valor guardado, las fixtures con pocas iteraciones (por ejemplo,
     1.000) se verifican con el mismo código de producción.
   - Una sola prueba usa las 600.000 reales (`TestHashPassword_IteracionesReales`), sin medir tiempo: con `-race` la
     duración varía mucho.
   - Alternativa descartada: una variable de paquete con las iteraciones, porque es estado global que las pruebas
     tendrían que restaurar.

4. **Mensajes de validación como errores exportados de `src/auth`**: `ErrPasswordTooShort`, `ErrPasswordTooLong` y
   `ErrPasswordMismatch`, con el texto exacto que se muestra.
   - Motivo: el comando imprime `err.Error()` sin traducir.
   - Costo: los textos empiezan con mayúscula y llevan tildes, lo que contradice la convención de Go para errores.
     `go vet` no lo controla, así que el CI no falla.
   - Orden de validación: primero el largo de la contraseña y después la coincidencia.

5. **Bloqueo `auth.Lockout`** con `NewLockout(now func() time.Time)` (`nil` → `time.Now`, como
   `effort.NewService`) y los métodos `Locked(memberID) bool`, `RecordFailure(memberID)` y
   `RecordSuccess(memberID)`.
   - Un mapa por integrante protegido con `sync.Mutex`. `Locked` borra la entrada vencida, así que al terminar el
     bloqueo el contador vuelve a cero.
   - El tamaño del mapa queda acotado por la cantidad de integrantes, porque `RecordSuccess` borra la entrada.
   - Límite aceptado: `Locked` y `RecordFailure` son llamadas separadas, así que una ráfaga en paralelo puede
     verificar algunas contraseñas más antes del bloqueo. Resolverlo exigiría reservar intentos en curso, demasiado
     para una historia Could Have.

6. **`RegisterContext` no cambia de firma.**
   - `RegisterContext(s, store)` pasa a llamar a `RegisterContextWithLockout(s, store, auth.NewLockout(nil))`.
   - Las pruebas que necesitan reloj simulado usan `RegisterContextWithLockout`.
   - Motivo: hay 15 llamadas y `main.go` no se toca, lo que evita conflictos con las ramas abiertas.

7. **Lectura de la credencial con un solo método nuevo de `ContextStore`.**
   - Método: `MemberCredential(ctx, projectID, memberID) (MemberCredential, error)`, que devuelve `Belongs`, `Name`
     y `PasswordHash`; `""` si es `NULL`.
   - Una sola consulta (`project_members JOIN members`). Los identificadores inválidos se descartan con `validID`
     sin consultar. Los errores se devuelven con `Unavailable(err)`, como `ResolveContext`.
   - Reemplaza a `ResolveContext` en `selectMember`, que pasaría a hacer una sola consulta en lugar de dos.

8. **Orden de las comprobaciones** (regla 1 de la spec): pertenencia → vacía → bloqueo → sin contraseña →
   verificación.
   - La pertenencia va primero para conservar el comportamiento de HU-20b (escenario 6).
   - El bloqueo va antes de la verificación para no gastar 600.000 iteraciones ni dar pistas mientras dura.
   - "Sin contraseña" va después del bloqueo: un integrante sin contraseña nunca acumula fallos, así que nunca está
     bloqueado y el orden entre esas dos no cambia el resultado.

9. **Respuestas de rechazo sin redirección.**
   - 401, 422 y 429 vuelven a mostrar `select_member.html` con `PageStatus`, el integrante elegido marcado
     (`ActiveID` = el elegido) y el campo vacío.
   - El error del campo va en `PageData.Form` ("Ingresá tu contraseña"); los demás mensajes, como alerta de error en
     `PageData.Flash`.
   - No se usa el parcial `field`, porque escribe `value=`; el campo de contraseña se escribe en la plantilla sin
     `value`.
   - No se agrega `WWW-Authenticate` al 401 ni `Retry-After` al 429; no lo pide el Issue.

10. **Un rechazo por contraseña no toca la identificación previa** (regla 5).
    - Motivo: la cookie que ya había la emitió el servidor después de verificar una contraseña.
    - Alternativa descartada: borrarla. Castigaría al integrante ya identificado por un intento de otra persona en el
      mismo navegador, sin ganar seguridad.

11. **"Salir" en `layout.html` (HU-20a).**
    - `<p class="app-context">` pasa a `<div class="app-context">`, porque un formulario no puede ir dentro de un
      `<p>`; los estilos usan la clase y no cambian.
    - El formulario `POST /contexto/salir` va junto al nombre, solo si hay integrante. No lleva `volver_a`: la
      estructura común no conoce la dirección actual, así que vuelve al inicio.
    - Una regla en `app.css` (`.app-context form { display: inline; }`) lo deja en la misma línea.
    - Alternativa descartada: un enlace `GET /contexto/salir`, porque una acción que cambia estado no va por GET.

12. **El comando separa el arranque de la lógica.**
    - `run(ctx, args, stdin, stdout, getenv) int` lee la opción y `DATABASE_URL` (con `web.LoadConfig`), abre la base
      y delega en `setPassword(ctx, store auth.PasswordStore, memberID, stdin, stdout) int`.
    - Las pruebas sin base usan `setPassword` con un doble del puerto, y `run` para el uso y el entorno. La prueba de
      integración corre `run` completo.
    - Códigos de salida: 0 si configuró, 1 si rechazó o falló, 2 si la opción falta o es inválida.
    - Las líneas se leen con `bufio.Reader`, quitando solo `\n` y `\r` finales (Windows).
    - La decisión del equipo dice `run(args, stdin, stdout, getenv)`; se agrega `ctx` al principio, igual que en
      `src/cmd/server`.

13. **Sin normalización Unicode ni recorte de espacios.**
    - Motivo: la biblioteca estándar no normaliza (haría falta `golang.org/x/text`). Recortar cambiaría la
      contraseña elegida.

14. **Número de la migración: la mayor en `main` + 1, sin rellenar huecos.**
    - Motivo: `golang-migrate up` solo aplica versiones mayores a la actual. Una 005 agregada sobre una base en la 011
      no se aplicaría nunca.
    - Hoy correspondería 012; si HU-06 y HU-15 entran antes, 014. T001 lo fija y T038 lo vuelve a confirmar.

15. **Prueba de la migración en una base descartable, con un archivo nuevo**
    (`db/member_password_migration_integration_test.go`).
    - Aplica todas las `up` hasta la nueva, comprueba la columna (`text`, admite nulos, filas existentes en `NULL`),
      aplica la `down`, comprueba que la columna no está y vuelve a aplicar la `up`.
    - Reutiliza el patrón de la prueba de HU-20a sin editar ese archivo.

16. **Ayudantes de prueba para las pruebas existentes.**
    - En `src/web`: `setTestPassword(t, db, memberID)` y `memberLoginValues(id)` (devuelve `integrante_id` y
      `contrasena`), con `testMemberPassword` y 1.000 iteraciones.
    - En `test/`: `configurarClave(t, db, id, clave)` y `formIdentificacion(id, clave)`, con `claveDePrueba` para las
      pruebas existentes y las claves del Issue para las de HU-27.
    - Los ayudantes se crean con las pruebas nuevas (T023 y T024) y los reutiliza el Bloque E.
    - Cada archivo actualizado configura la contraseña de sus integrantes de prueba y usa el ayudante para enviarla.

17. **La contraseña nunca se escribe.**
    - El formulario no la guarda en `FormView`, la plantilla no tiene `value` y ningún mensaje la incluye.
    - Las páginas de error no muestran detalles.
    - El comando solo imprime los textos fijos.
    - `TestHU27_ContrasenaProtegida` revisa la base, las páginas y un registro capturado con `slog` en un búfer.

## Datos a reconfirmar en T001 (el código pudo cambiar)

- La mayor migración en `main` y el número que toma HU-27.
- La lista de pruebas que se identifican por `/contexto/integrante`:
  `git grep -n "contexto/integrante\|pathSelectMember" -- '*_test.go' test/`.
- Los dobles que implementan `ContextStore` completo: `git grep -n "func (.*) ResolveContext(" -- '*_test.go' test/`.
- Que nadie más haya cambiado `context_handlers.go`, `context_store.go`, `postgres_context_store.go`,
  `select_member.html` ni `layout.html`.

## Preguntas abiertas

Ninguna.

## Resultado de T001 (2026-11-01)

Reconfirmado sobre `main` en `04f1f0c` (después del merge de HU-09, #69):

- **Migración**: la mayor es `014_sprint_story_results`; HU-27 toma **015** (`015_add_member_password`). Se
  reemplazó `<NNN>` en `plan.md` y `data-model.md`. T038 lo vuelve a confirmar antes del PR.
- **Pruebas que se identifican por `/contexto/integrante`**: hoy son **16 archivos**, no 8, porque después de
  escribir la spec se integraron HU-11, HU-15, HU-18 y otras. De esos 16, **12** envían `integrante_id` y se
  actualizan en el Bloque E:
  - `src/web`: `context_member_test.go` (su auxiliar `selectMember` también lo usa `context_guard_test.go`),
    `effort_handlers_test.go`, `poker_handlers_test.go` (su auxiliar `as` también lo usan
    `poker_close_handlers_test.go` y `poker_consensus_handlers_test.go`), `poker_view_test.go` y
    `projects_update_handlers_test.go`;
  - `test/`: `context_bdd_test.go`, `web_close_bdd_test.go`, `web_effort_bdd_test.go`, `web_poker_bdd_test.go`,
    `web_poker_consensus_bdd_test.go`, `web_poker_reveal_bdd_test.go` y `web_signed_cookies_bdd_test.go`.

  Los otros 4 (`context_guard_test.go`, `poker_close_handlers_test.go`, `poker_consensus_handlers_test.go` y
  `web_project_update_bdd_test.go`) solo nombran la ruta o usan esos auxiliares, y no cambian.
- **Dobles que implementan `ContextStore` completo**: 2, `sprintHandlerContextStore`
  (`src/web/sprints_handlers_test.go`) y `sprintBDDContextStore` (`test/web_sprints_bdd_test.go`).
- **Pruebas que buscan que no haya `type="password"`**: 2, `TestIntegrante_SelectorMuestraSoloLosDelProyectoActivo`
  (`context_member_test.go`) y el escenario de `context_bdd_test.go`; las dos pasan a exigir el campo. Las que
  buscan que no haya `<form` (HU-10, HU-12, HU-16, HU-09) no se identifican, así que el botón "Salir" del
  encabezado no las afecta.
- **`crypto/pbkdf2`**: disponible (`go doc crypto/pbkdf2.Key`), con la versión de `go.mod` (1.27.1).
- Nadie más cambió `context_handlers.go`, `context_store.go`, `postgres_context_store.go`, `select_member.html` ni
  `layout.html` desde HU-26.
