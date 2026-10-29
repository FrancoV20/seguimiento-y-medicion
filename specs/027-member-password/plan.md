# Plan de implementación: HU-27 – Identificación con Contraseña

**Rama**: `HU-27` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/027-member-password/spec.md`
**Constitución**: v1.1.0

## Resumen

HU-27 agrega la contraseña a la identificación de HU-20b:

- Un paquete nuevo, `src/auth`, con las reglas puras de la contraseña (`HashPassword`, `VerifyPassword` y
  `ValidateNewPassword`) y el bloqueo en memoria (`Lockout`), más el puerto `PasswordStore` con su adaptador
  PostgreSQL.
- Una migración que agrega `members.password_hash`.
- `ContextStore` suma `MemberCredential`. `selectMember` verifica la contraseña antes de emitir la cookie firmada de
  HU-26, y se agrega `POST /contexto/salir` con su botón en la estructura común.
- Un comando nuevo, `src/cmd/set-password`, para que el Scrum Master configure la contraseña.
- Las 8 pruebas existentes que se identifican por `/contexto/integrante` se actualizan para enviar la contraseña.

`RegisterContext` y `src/cmd/server/main.go` no cambian.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1.
**Dependencias**:

- Biblioteca estándar: `crypto/pbkdf2`, `crypto/sha256`, `crypto/rand`, `crypto/subtle`, `encoding/base64`,
  `sync`, `time`, `flag`, `bufio`, `database/sql`, `net/http` y `html/template`.
- `github.com/jackc/pgx/v5/stdlib`, que ya está.

`go.mod` no cambia (`crypto/pbkdf2` verificado con `go doc`; research.md).
**Almacenamiento**: PostgreSQL. Una migración nueva (`members.password_hash TEXT NULL`) con up y down. HU-27 lee
`members` y `project_members`, y el comando escribe `members.password_hash`.
**Pruebas**:

- unitarias de `src/auth`: tablas de validación y de valores guardados; el bloqueo con reloj simulado y un caso
  concurrente para `-race`;
- una prueba con las 600.000 iteraciones reales (`TestHashPassword_IteracionesReales`); el resto usa 1.000 con
  `HashPasswordIterations`;
- integración contra PostgreSQL real: `PostgreSQLContextStore.MemberCredential`, `PostgreSQLPasswordStore`, la
  migración up y down en una base descartable, y el comando completo;
- handlers con `httptest` y `cookiejar`, con el `PostgreSQLContextStore` real, `RegisterContextWithLockout` y reloj
  simulado;
- BDD en `test/web_member_password_bdd_test.go`, con los datos del Issue;
- el comando: `setPassword` con un doble del puerto y `run` para el uso y el entorno (sin base), más una prueba de
  integración.

Se corren con `go test -count=1 -p 1 ./...`.
**CI**:

- `gofmt -l .` sin salida y `go vet ./...` limpio;
- `migrate up` antes de las pruebas;
- `go test ./... -p 1 -race` con cobertura.

Los archivos nuevos se formatean; los existentes se editan a mano respetando su fin de línea (CRLF en la copia de
Windows).
**Plataforma**: servidor web local (`src/cmd/server`) y comando de consola, en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript, más un comando.
**Objetivos de rendimiento**: identificarse con la contraseña correcta en menos de 1 s (SC-004, verificación manual
con 600.000 iteraciones). Con las fixtures de 1.000 iteraciones, las pruebas no se vuelven lentas.
**Restricciones**:

- Integrar al final, después de todas las demás historias.
- No cambiar la firma de `RegisterContext` ni `main.go`.
- La contraseña no aparece en ningún registro, página, mensaje ni salida.

**Escala**: proyectos del curso (decenas de integrantes); el mapa del bloqueo queda acotado por esa cantidad.

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Could Have, valor 60. Cada FR sale de un escenario del Issue o de una decisión del equipo. Roles, recuperación y vencimiento quedan fuera. | PASA |
| II. Calidad por construcción | Reglas puras en `src/auth`, separadas de la web y del comando. Se reutilizan `SetFlash`, `SanitizeReturnTo`, `Renderer.Fail/PageStatus`, `setContextCookie`, `clearContextCookie`, `validID`, `LoadConfig` y `CrossOriginGuard`, sin copiarlos. Los textos están en un solo archivo. | PASA |
| III. Primero las pruebas | 12 escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN. Las pruebas existentes se ven fallar con el handler nuevo (RED) antes de actualizarlas. | PASA |
| IV. Trabajo ágil y colaborativo | Se integra al final (nota 6). T003 avisa al equipo qué archivos de HU-05a, HU-20a, HU-20b, HU-23, HU-24 y HU-26 se tocan. T039 lo repite en el PR. | PASA |
| V. Medición | No agrega métricas; protege que el esfuerzo y los votos que alimentan las métricas sean de quien dice ser. | PASA (no aplica directamente) |
| VI. Persistencia PostgreSQL | Los dos puertos nuevos o ampliados (`ContextStore.MemberCredential` y `auth.PasswordStore`) tienen adaptador PostgreSQL y prueba de integración. La migración se prueba up y down. El bloqueo vive en memoria por decisión del equipo; no es un puerto de almacenamiento y la limitación está documentada. | PASA |

## Estructura

### Documentación

```text
specs/027-member-password/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
├── checklists/requirements.md
├── evidencia-red.txt        (durante la implementación)
├── evidencia-pruebas.txt    (cierre)
└── cobertura.txt            (cierre)
```

No hay carpeta `contracts/`. Las rutas, las respuestas, el formato guardado y el contrato del comando están en
[data-model.md](data-model.md).

### Código

Archivos nuevos:

```text
src/auth/password.go                          HashPassword, HashPasswordIterations, VerifyPassword,
                                              ValidateNewPassword, errores y constantes
src/auth/password_test.go
src/auth/lockout.go                           Lockout, NewLockout, Locked, RecordFailure, RecordSuccess
src/auth/lockout_test.go
src/auth/password_store.go                    PasswordStore, ErrMemberNotFound
src/auth/postgres_password_store.go           PostgreSQLPasswordStore
src/auth/postgres_password_store_integration_test.go
db/migrations/<NNN>_add_member_password.up.sql
db/migrations/<NNN>_add_member_password.down.sql
db/member_password_migration_integration_test.go
src/cmd/set-password/main.go                  main, run, setPassword
src/cmd/set-password/main_test.go
src/web/member_password_messages.go           textos de HU-27
src/web/member_password_handlers_test.go      handlers con httptest y cookiejar
src/web/member_password_helpers_test.go       setTestPassword, memberLoginValues, testMemberPassword
test/web_member_password_bdd_test.go          escenarios del Issue
test/member_password_helpers_test.go          configurarClave, formIdentificacion, claveDePrueba
```

Archivos existentes que se editan, por historia:

| Historia | Archivo | Cambio |
|---|---|---|
| HU-20b | `src/web/context_store.go` | `MemberCredential` (tipo y método de la interfaz) |
| HU-20b | `src/web/postgres_context_store.go` | implementación de `MemberCredential` |
| HU-20b | `src/web/postgres_context_store_integration_test.go` | pruebas de `MemberCredential` |
| HU-20b | `src/web/context_handlers.go` | `RegisterContextWithLockout`, `selectMember` con contraseña, `logout` |
| HU-20b | `src/web/templates/select_member.html` | introducción, campo de contraseña y su error |
| HU-20b | `src/web/context_member_test.go`, `test/context_bdd_test.go` | contraseña en las identificaciones; "sin contraseña" pasa a "con contraseña" |
| HU-20a | `src/web/templates/layout.html`, `src/web/static/app.css` | botón "Salir" (`<p>` → `<div>`) |
| HU-23 | `src/web/sprints_handlers_test.go`, `test/web_sprints_bdd_test.go` | `MemberCredential` en los dos dobles de `ContextStore` |
| HU-24 | `src/web/effort_handlers_test.go`, `test/web_effort_bdd_test.go` | contraseña en las identificaciones |
| HU-05a | `src/web/poker_handlers_test.go`, `src/web/poker_view_test.go`, `test/web_poker_bdd_test.go` | contraseña en las identificaciones |
| HU-26 | `test/web_signed_cookies_bdd_test.go` | contraseña en la identificación |
| Documentación | `README.md`, `docs/hu-20b-deuda-tecnica.md` | comando, migración, limitaciones; la decisión 1 queda resuelta |

No se tocan `src/cmd/server/main.go`, `src/web/context.go`, `context_middleware.go`, `cookie_signer.go` ni las
migraciones existentes.

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. PBKDF2-SHA256 de la biblioteca estándar, formato `pbkdf2-sha256$600000$<sal>$<hash>` en base64 sin relleno, con
   las iteraciones leídas del valor guardado (de 1 a 10.000.000).
2. `HashPasswordIterations` solo para pruebas y fixtures; `HashPassword` siempre usa 600.000.
3. `auth.Lockout` con reloj inyectable y `sync.Mutex`; al vencer el bloqueo, el contador vuelve a cero.
4. `RegisterContext` delega en `RegisterContextWithLockout(s, store, auth.NewLockout(nil))`.
5. `MemberCredential` reemplaza a `ResolveContext` dentro de `selectMember` (una sola consulta).
6. Orden: pertenencia → vacía → bloqueo → sin contraseña → verificación. Solo la contraseña incorrecta cuenta.
7. Los rechazos 401, 422 y 429 vuelven a mostrar el selector con `PageStatus`, con el elegido marcado y el campo sin
   `value`. No tocan la cookie que ya había.
8. "Salir" es un formulario POST en `layout.html`, visible solo con integrante.
9. El comando: `run` lee la opción y el entorno, y `setPassword` contiene la lógica sobre el puerto. Códigos 0, 1 y 2.
10. Número de la migración: la mayor en `main` + 1 (sin huecos).

## Contratos que se consumen

- **HU-20b**:
  - `ContextStore`, `PostgreSQLContextStore` y `validID`;
  - `setContextCookie`, `clearContextCookie`, `CookieMember` y `contextMiddleware`;
  - `RequireContext` (sin cambios);
  - `SetFlash`, `FormView`, `SanitizeReturnTo`, `MsgMemberInvalid`, `MsgChooseProjectFirst`, `CrossOriginGuard`.
- **HU-26**: `SignedContextValue` dentro de `setContextCookie` (sin cambios).
- **HU-20a**: `Server.HandleFunc`, el 405 de `Server.route`, `Renderer.Page/PageStatus/Fail`, `PageData.Flash/Form`,
  `layout.html` y `LoadConfig`.
- **HU-01**: `members(id, display_name)` y `project_members(project_id, member_id)`.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Rompe la identificación de las pruebas de todas las historias, incluidas las de ramas abiertas | Se integra al final (nota 6). T001 recuenta los archivos (hoy 8, no 6). T003 avisa al equipo; los ayudantes de T023 y T024 sirven a quien llegue después. |
| El número de migración choca con HU-06, HU-15 u otra | T001 lo fija con la mayor en `main` + 1; T038 lo reconfirma antes del PR y renombra si hace falta. |
| El servidor no arranca si no se migró (verificación de esquema de HU-20a) | Está en el quickstart y en el README: migrar y después configurar las contraseñas (nota 7). |
| Nadie puede identificarse en la demo si no se configuran las contraseñas | Quickstart, README y aviso en el PR: el comando, por integrante, después de migrar. |
| La verificación con 600.000 iteraciones es lenta con `-race` en el CI | Las fixtures usan 1.000. Solo `TestHashPassword_IteracionesReales`, la prueba de integración del comando y `TestHU27_ConfigurarContrasena` usan las reales, sin medir tiempo. |
| Una ráfaga de intentos en paralelo supera los 5 antes del bloqueo | Limitación documentada (research.md, decisión 5); el bloqueo es una defensa contra la prueba manual de contraseñas. |
| El bloqueo se pierde al reiniciar, y cualquiera puede bloquear a otro 1 minuto | Limitaciones aceptadas por el equipo; documentadas en el README. |
| Las contraseñas se ven al escribirlas en el comando | Limitación documentada; ocultarlas exige `golang.org/x/term` (dependencia nueva). |
| Conflictos en `context_handlers.go`, `select_member.html` y `layout.html` | Se integra al final; los cambios se limitan a `selectMember`, una ruta nueva y dos bloques de plantilla. |
| Pruebas futuras que buscan que no haya `<form` en páginas con integrante identificado | Hoy solo una prueba (`projects_handlers_test.go:222`) lo busca, en una página de error sin integrante. T001 lo vuelve a buscar. |
| Fin de línea CRLF en los archivos existentes de la copia de Windows | Se editan a mano, sin `gofmt -w`; el CI corre `gofmt -l` sobre el checkout con LF. |
| "Planificación de proyecto.pdf" no está en el repositorio | Se usaron el Issue y las notas; si la planificación dice otra cosa, prevalece y se corrige la spec antes de T002. |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: `MemberCredential` y `PasswordStore` tienen adaptador PostgreSQL y prueba de integración (T012,
  T013). La migración se prueba up y down (T011).
- **Principio II**: las reglas puras están en `src/auth` y la web y el comando no las duplican.
- **Principio III**: los cinco bloques siguen RED → evidencia → GREEN. En el bloque de pruebas existentes, el RED es
  la suite fallando con el handler nuevo.

Sigue en **PASA**, sin violaciones que justificar.
