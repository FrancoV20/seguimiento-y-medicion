# Tareas: HU-27 – Identificación con Contraseña

**Entrada**: `specs/027-member-password/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-20b y HU-26 en `main`. HU-27 se integra **después de todas las demás historias**.

Formato: `[ID] [P] [E#] descripción`
- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario del Issue que cubre.
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/027-member-password/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- Datos: los de "Datos de ejemplo" de `spec.md` y la tabla "Datos de prueba" de `data-model.md`. Las fixtures guardan
  las contraseñas con `auth.HashPasswordIterations(…, 1000)`.

## Fase 1: Preparación

- [x] T001 Hacer `git pull` de `main` y reconfirmar los "Datos a reconfirmar en T001" de `research.md`. Anotar el
  resultado en `research.md` ("Resultado de T001"):
  - la mayor migración en `db/migrations`; fijar `<NNN>` = mayor + 1 y reemplazarlo en `data-model.md` y `plan.md`;
  - la lista de pruebas que se identifican por `/contexto/integrante`
    (`git grep -n "contexto/integrante\|pathSelectMember" -- '*_test.go' test/`; hoy 8 archivos);
  - los dobles que implementan `ContextStore` completo (`git grep -n "func (.*) ResolveContext(" -- '*_test.go' test/`;
    hoy 2);
  - las pruebas que buscan que no haya `<form` o `type="password"` en una página;
  - que `go doc crypto/pbkdf2.Key` siga disponible con la versión de `go.mod`. Si no lo está, **detenerse y avisar**
    antes de seguir.
- [x] T002 Crear la rama `HU-27` desde `main` actualizado, solo si no hay cambios sin commitear.
- [x] T003 Avisar al equipo que HU-27 se integra al final y que toca:
  - de HU-20a: `layout.html` y `app.css`;
  - de HU-20b: `context_store.go`, `postgres_context_store.go`, `context_handlers.go` y `select_member.html`;
  - las 8 pruebas de T001 y los 2 dobles de HU-23.

  Pedir que nadie los edite mientras dure la rama.
- [x] T004 Línea base: `go test -count=1 -p 1 ./...` en verde antes de empezar.

## Fase 2: Bloque A – Dominio de contraseñas y bloqueo (`src/auth`)

**Objetivo**: reglas puras de la contraseña y bloqueo, sin base ni web.

**Prueba independiente**: `go test -count=1 -race ./src/auth/...` sin `DATABASE_URL`.

### Pruebas (RED)

- [x] T005 [P] [E1, E9, E11, E12] Escribir las pruebas de contraseñas en `src/auth/password_test.go`. Las tablas
  están en `data-model.md`.
  - `TestHashPassword_FormatoYSalDistinta`: dos valores de "ana-clave-2026" son distintos; los dos tienen 4 partes,
    esquema `pbkdf2-sha256`, sal de 16 bytes y hash de 32, y verifican; ninguno contiene la contraseña.
  - `TestVerifyPassword_CorrectaEIncorrecta`: "ana-clave-2026" sí; "otra-clave" y `""` no.
  - `TestVerifyPassword_LeeLasIteracionesGuardadas`: un valor de 1.000 iteraciones verifica con el código de
    producción.
  - `TestVerifyPassword_ValorGuardadoMalFormado`: tabla de valores mal formados → `ErrMalformedHash`, incluidas las
    iteraciones `"0"` y `"10000001"`.
  - `TestHashPassword_IteracionesReales`: `HashPassword` guarda `600000` y verifica; sin medir tiempo.
  - `TestValidateNewPassword`: la tabla de largo (8 y 128 sí; 7 y 129 no; "ñandú-12" cuenta 8 runas; los espacios
    cuentan), "no coinciden", y que el largo se informa antes que la coincidencia. Cada error tiene el texto exacto
    del Issue.
- [x] T006 [P] [E5] Escribir las pruebas del bloqueo, con reloj simulado, en `src/auth/lockout_test.go`.
  - `TestLockout_CincoFallosBloqueanUnMinuto`: 4 fallos, no bloqueado; el 5.º bloquea; a los 59 s del 5.º sigue; a
    los 60 s, libre; un fallo nuevo no bloquea (el contador volvió a cero).
  - `TestLockout_AciertoReiniciaElContador`: 4 fallos, un acierto y 4 fallos: no bloqueado.
  - `TestLockout_PorIntegrante`: con Ana bloqueada, Beto está libre.
  - `TestLockout_UsoConcurrente`: goroutines que llaman a `Locked`, `RecordFailure` y `RecordSuccess` sobre varios
    integrantes, sin carreras con `-race`.
- [x] T007 Ejecutar T005–T006, confirmar que fallan porque el paquete no existe y guardar la salida en
  `evidencia-red.txt` (sección "Dominio").

### Implementación (GREEN)

- [x] T008 Implementar `src/auth/password.go` según `data-model.md`:
  - constantes, `HashPassword`, `HashPasswordIterations` (iteraciones < 1 → error) y `VerifyPassword`;
  - `VerifyPassword` usa `subtle.ConstantTimeCompare`, base64 `RawStdEncoding`, iteraciones del valor guardado de 1
    a 10.000.000 y `ErrMalformedHash`;
  - `ValidateNewPassword` (largo en runas con `utf8.RuneCountInString`, sin recortar);
  - los errores con el texto exacto.
- [x] T009 Implementar `src/auth/lockout.go`: `Lockout` con `sync.Mutex`, `NewLockout(now)` (`nil` → `time.Now`),
  `Locked` (borra la entrada vencida), `RecordFailure` (al 5.º, `lockedUntil = now + LockDuration`) y `RecordSuccess`
  (borra la entrada).
- [x] T010 Ejecutar `go test -count=1 -race ./src/auth/...`: T005–T006 en verde.

**Punto de control**: las reglas funcionan solas, sin tocar otras historias.

## Fase 3: Bloque B – Migración y almacenamiento

**Objetivo**: columna `members.password_hash` y los dos adaptadores PostgreSQL.

**Prueba independiente**: pruebas de integración con `DATABASE_URL`, sin pantalla.

### Pruebas (RED)

- [x] T011 [P] [E9] `TestMigracion_PasswordHash_UpYDown` en `db/member_password_migration_integration_test.go`, sobre
  una base descartable con el mismo patrón que `TestMigraciones_SeAplicanEnOrdenSobreUnaBaseDescartable`:
  - aplica las `up` hasta `<NNN>` y comprueba la columna (`data_type = 'text'`, `is_nullable = 'YES'`, filas previas
    en `NULL`);
  - aplica `<NNN>…down.sql` y comprueba que no está;
  - vuelve a aplicar la `up` sin error.
- [x] T012 [P] [E6, E12] Pruebas de `MemberCredential` en `src/web/postgres_context_store_integration_test.go`, con
  «Alfa» (Ana con valor guardado, Caro sin valor) y «Beta» (Dani):
  - `TestPostgreSQLContextStore_MemberCredential`: los casos de la tabla de `data-model.md`, incluidos 0 y 2147483648
    sin consultar;
  - `TestPostgreSQLContextStore_MemberCredentialBaseCaida`: base cerrada → `KindUnavailable`.
- [x] T013 [P] [E10] `TestPostgreSQLPasswordStore` en `src/auth/postgres_password_store_integration_test.go`:
  - `MemberName` de Caro → "Caro"; de un integrante inexistente → `ErrMemberNotFound`;
  - `SetPasswordHash` guarda el valor y un integrante inexistente da `ErrMemberNotFound`;
  - base cerrada → error, distinto de `ErrMemberNotFound`.
- [x] T014 Ejecutar T011–T013, confirmar que fallan (falta la migración, el método y el adaptador) y guardar la salida
  en `evidencia-red.txt` (sección "Almacenamiento").

### Implementación (GREEN)

- [ ] T015 Crear `db/migrations/<NNN>_add_member_password.up.sql`
  (`ALTER TABLE members ADD COLUMN password_hash TEXT NULL;`) y `.down.sql`
  (`ALTER TABLE members DROP COLUMN IF EXISTS password_hash;`). Aplicarla a la base local con `migrate … up`.
- [ ] T016 Agregar `MemberCredential` (tipo y método) a `ContextStore` en `src/web/context_store.go` e implementarlo
  en `src/web/postgres_context_store.go`:
  - la consulta de `data-model.md`, `validID` en los dos identificadores, `NULL` → `""` y errores con `Unavailable`;
  - en el mismo paso, sumar el método a los dobles `sprintHandlerContextStore` (`src/web/sprints_handlers_test.go`) y
    `sprintBDDContextStore` (`test/web_sprints_bdd_test.go`) para que compilen.
- [ ] T017 Implementar `src/auth/password_store.go` (`PasswordStore`, `ErrMemberNotFound`) y
  `src/auth/postgres_password_store.go` (`NewPostgreSQLPasswordStore`, `MemberName` y `SetPasswordHash` con las
  consultas de `data-model.md`; un identificador fuera de 1..2147483647 → `ErrMemberNotFound` sin consultar).
- [ ] T018 Ejecutar T011–T013 en verde y `go vet ./...` (todo compila).

## Fase 4: Bloque C – Comando `set-password` (escenarios 10 y 11)

**Objetivo**: el Scrum Master configura la contraseña desde la consola.

**Prueba independiente**: `go test ./src/cmd/set-password/...` con y sin `DATABASE_URL`.

### Pruebas (RED)

- [ ] T019 [E10, E11] Escribir `src/cmd/set-password/main_test.go`:
  - `TestSetPassword_Configura`: con un doble de `auth.PasswordStore` y "caro-clave-2026\r\ncaro-clave-2026\n" →
    código 0, "Contraseña configurada para Caro", un valor guardado que verifica, y la contraseña no aparece en la
    salida;
  - `TestSetPassword_ContrasenaInvalida`: corta, de 129 runas y distintas → el texto exacto, código 1, sin llamar a
    `SetPasswordHash`; también la entrada cerrada antes de tiempo;
  - `TestSetPassword_IntegranteInexistente`: "El integrante no existe", código 1, sin leer la entrada;
  - `TestSetPassword_ErrorAlGuardar`: "No se pudo acceder a la base de datos", código 1;
  - `TestRun_UsoYEntorno`: sin `-integrante`, con `abc`, `0` o `-1` → uso y código 2; sin `DATABASE_URL` → el mensaje
    de `LoadConfig` y código 1; con una `DATABASE_URL` inalcanzable (por ejemplo, `127.0.0.1:1`) → "No se pudo
    acceder a la base de datos", código 1, sin imprimir la dirección;
  - `TestRun_ConfiguraContrasena` (integración; se omite sin `DATABASE_URL`): crea a Caro, corre `run` con
    "caro-clave-2026" dos veces y comprueba en la base que `password_hash` empieza con `pbkdf2-sha256$600000$` y
    verifica.
- [ ] T020 Ejecutar T019, confirmar que falla porque el comando no existe y guardar la salida en `evidencia-red.txt`
  (sección "Comando").

### Implementación (GREEN)

- [ ] T021 Implementar `src/cmd/set-password/main.go` según `data-model.md`:
  - `main`, `run(ctx, args, stdin, stdout, getenv)` y `setPassword(ctx, store, memberID, stdin, stdout)`;
  - `flag.NewFlagSet` con `ContinueOnError` y salida descartada;
  - `web.LoadConfig` y `sql.Open("pgx", …)` con `PingContext`;
  - las líneas se leen con `bufio.Reader` y se les quita solo `\n` y `\r` finales;
  - validación con `auth.ValidateNewPassword`, `auth.HashPassword` y `SetPasswordHash`;
  - los textos fijos y los códigos 0, 1 y 2.
- [ ] T022 Ejecutar T019 en verde.

## Fase 5: Bloque D – Pantalla "Identificarme" y "Salir" (escenarios 1 a 9 y 12) 🎯 MVP

**Objetivo**: identificarse exige la contraseña; "Salir" cierra la identificación.

**Prueba independiente**: las pruebas nuevas de handler y BDD con PostgreSQL. Las pruebas existentes de
identificación fallan hasta el Bloque E.

### Pruebas (RED)

- [ ] T023 [P] [E1–E5, E7, E9, E12] Escribir las pruebas de handler.
  - Primero, `src/web/member_password_helpers_test.go`: `testMemberPassword`, `setTestPassword(t, db, memberID)`
    (guarda `HashPasswordIterations(testMemberPassword, 1000)`) y `memberLoginValues(id)` (`integrante_id` y
    `contrasena`).
  - Después, `src/web/member_password_handlers_test.go`, con `httptest`, `cookiejar`, `PostgreSQLContextStore` real,
    «Alfa» (Ana, Beto, Caro) y «Beta» (Dani), y `RegisterContextWithLockout` con reloj simulado.
  - `TestMemberPassword_Correcta`: 303 al destino, cookie firmada y flash con "Te identificaste como Ana".
  - `TestMemberPassword_Incorrecta`: 401, el mensaje, Ana `checked`, el campo sin `value` y sin cookie; una
    contraseña de 129 caracteres también da 401 y cuenta como fallo.
  - `TestMemberPassword_Vacia`: 422 y el error junto al campo (`field-error`, `aria-describedby`).
  - `TestMemberPassword_SinConfigurar`: 422 y el mensaje con "Caro".
  - `TestMemberPassword_Bloqueo`: la tabla "Cuentas del bloqueo" de `spec.md`; durante el bloqueo, 429 con la
    contraseña correcta; con el reloj a los 60 s del 5.º fallo, 303; los rechazos 422 no cuentan.
  - `TestMemberPassword_RechazoConservaIdentificacionPrevia`: identificado como Beto, un 401 con Ana deja a Beto.
  - `TestMemberPassword_Salir`: `POST /contexto/salir` borra solo `ctx_integrante`, deja el flash "Cerraste tu
    identificación" y redirige con 303 a "/"; con `volver_a` externo, también a "/"; sin estar identificado responde
    igual; `GET` responde 405; el botón "Salir" aparece solo con integrante.
  - `TestMemberPassword_NuncaMuestraLaContrasena`: en cada respuesta, ni el cuerpo ni las cabeceras contienen la
    contraseña enviada.
  - `TestMemberPassword_ErrorDeLectura`: un doble que falla en `MemberCredential` da 503 sin detalles internos (lista
    de `TestContexto_ErroresNoExponenDetallesInternos`), sin cookie y sin contar; un valor guardado mal formado da
    500.
- [ ] T024 [P] [E1–E10, E12] Escribir las pruebas BDD.
  - Primero, `test/member_password_helpers_test.go`: `claveDePrueba`, `configurarClave(t, db, id, clave)` y
    `formIdentificacion(id, clave)`.
  - Después, `test/web_member_password_bdd_test.go`, con los datos del Issue y un comentario Dado/Cuando/Entonces por
    paso:
    - `TestHU27_IdentificacionCorrecta`, `TestHU27_ContrasenaIncorrecta`, `TestHU27_ContrasenaVacia`,
      `TestHU27_IntegranteSinContrasena`, `TestHU27_DemasiadosIntentos` (reloj simulado),
      `TestHU27_IntegranteDeOtroProyecto` (Dani con "dani-clave-2026"; mensaje `MsgMemberInvalid`);
    - `TestHU27_Salir`: después de salir, la sección con `RequiresMember` lleva a "Identificarme", que tiene
      `type="password"`;
    - `TestHU27_CambioDeProyecto`;
    - `TestHU27_ContrasenaProtegida`: Beto con "ana-clave-2026" solo en esta prueba; se revisan la base, las páginas
      y el registro del servidor capturado con `slog` en un búfer; los dos valores son distintos;
    - `TestHU27_ConfigurarContrasena`: Caro guardada con `auth.HashPassword("caro-clave-2026")` mediante
      `auth.NewPostgreSQLPasswordStore`, como lo hace el comando, y después se identifica;
    - `TestHU27_ErrorAlVerificar`: store envuelto que falla en `MemberCredential`.
- [ ] T025 Ejecutar T023–T024, confirmar que fallan por la razón esperada (hoy se identifica sin contraseña y no
  existe "Salir") y guardar la salida en `evidencia-red.txt` (sección "Pantalla").

### Implementación (GREEN)

- [ ] T026 Crear los textos y adaptar los handlers.
  - `src/web/member_password_messages.go`: la tabla de textos de `data-model.md`.
  - En `src/web/context_handlers.go`:
    - `RegisterContextWithLockout(s, store, lockout)`; `RegisterContext` delega con `auth.NewLockout(nil)`;
    - la ruta `POST /contexto/salir` (`logout`);
    - `selectMember` con el orden de `data-model.md` ("Respuestas de `POST /contexto/integrante`");
    - los rechazos 401, 422 y 429 con `PageStatus`, `ActiveID` = elegido, `PageData.Form` o `PageData.Flash`, sin
      tocar la cookie previa;
    - el acierto con `RecordSuccess`, `setContextCookie`, `SetFlash(FlashSuccess, IdentifiedMessage(nombre))` y 303.
  - No cambiar `ResolveContext`, el middleware ni `RequireContext`.
- [ ] T027 Editar las plantillas y el estilo:
  - `src/web/templates/select_member.html`: nueva introducción y campo
    `<input type="password" id="contrasena" name="contrasena" autocomplete="current-password">` sin `value`, con su
    error;
  - `src/web/templates/layout.html`: `<p class="app-context">` → `<div class="app-context">` y el formulario "Salir"
    junto al nombre, solo con `.MemberName`;
  - `src/web/static/app.css`: `.app-context form { display: inline; }`.
- [ ] T028 Ejecutar T023–T024 en verde (`go test -count=1 -p 1 -run "MemberPassword|HU27" ./src/web/... ./test/...`).

**Punto de control**: la identificación pide contraseña. Las pruebas existentes de identificación quedan en rojo
hasta el Bloque E; no subir la rama en este estado.

## Fase 6: Bloque E – Actualización de pruebas existentes

**Objetivo**: las pruebas de las otras historias se identifican con contraseña; solo cambian las expectativas "sin
contraseña" de HU-20b.

**Prueba independiente**: `go test -count=1 -p 1 ./...` completo.

- [ ] T029 Ejecutar `go test -count=1 -p 1 ./...` después del Bloque D. Confirmar que fallan exactamente las pruebas de
  los archivos de T001 (identificación sin contraseña y las dos que esperan que no haya `type="password"`) y guardar
  la salida en `evidencia-red.txt` (sección "Pruebas existentes"). Si falla otra, anotarla y revisarla antes de
  seguir.
- [ ] T030 [P] Actualizar las pruebas de `src/web`: `context_member_test.go`, `effort_handlers_test.go`,
  `poker_handlers_test.go` y `poker_view_test.go` (más las que agregue T001).
  - Configurar con `setTestPassword` la contraseña de los integrantes que se identifican.
  - Enviar `memberLoginValues` en lugar de solo `integrante_id`.
  - En `TestIntegrante_SelectorMuestraSoloLosDelProyectoActivo`, la expectativa "no hay `type="password"`" pasa a
    "hay `type="password"` con `autocomplete="current-password"`".
  - Las demás expectativas no cambian.
- [ ] T031 [P] Actualizar las pruebas de `test/`: `context_bdd_test.go` (`elegirIntegrante` y la expectativa "sin
  contraseña" de la línea 505), `web_effort_bdd_test.go` (`dadoQueTrabajoEn` y la línea 298),
  `web_poker_bdd_test.go` y `web_signed_cookies_bdd_test.go`. Usar `configurarClave` y `formIdentificacion`; las demás
  expectativas no cambian.
- [ ] T032 Ejecutar `go test -count=1 -p 1 ./...` completo en verde.

## Fase 7: Cierre

- [ ] T033 Revisión de seguridad.
  - Con `git grep -n -i "contrasena\|password" src test`, comprobar que la contraseña:
    - no se registra con `slog`;
    - no entra en `FormView.Values`;
    - no se incluye en mensajes ni errores;
    - no se imprime en el comando.
  - Comprobar además que:
    - la comparación usa `subtle.ConstantTimeCompare`;
    - la plantilla no tiene `value` en el campo;
    - `POST /contexto/salir` pasa por el `CrossOriginGuard`.
  - Anotar el resultado en esta tarea.
- [ ] T034 [P] Formatear con `gofmt` solo los archivos nuevos, sin `gofmt -w` sobre archivos existentes (CRLF);
  `go vet ./...` limpio; `go test ./... -p 1 -race` en verde; pipeline de GitHub Actions en verde.
- [ ] T035 [P] Documentar en `README.md`, junto a la migración:
  - configurar contraseñas con `go run ./src/cmd/set-password -integrante N` después de migrar (nota 7);
  - las limitaciones: bloqueo en memoria, contraseña visible al escribirla en el comando y sin cambio ni
    recuperación por web.

  En `docs/hu-20b-deuda-tecnica.md`, marcar la decisión 1 ("Sin contraseña") como resuelta por HU-27.
- [ ] T036 Guardar `evidencia-pruebas.txt` (`go test -count=1 -p 1 -v ./...`) y `cobertura.txt`
  (`go test -count=1 -p 1 -cover ./...`).
- [ ] T037 Verificar a mano los 12 escenarios con la aplicación en ejecución (`quickstart.md`), incluido el tiempo de
  identificación menor a 1 segundo (SC-004).
- [ ] T038 Antes del PR, hacer `git pull` de `main` y reconfirmar que `<NNN>` sigue siendo la mayor migración + 1 (si
  otra historia lo tomó, renombrar los dos archivos y la prueba de T011). Reconfirmar también que los nombres de las
  pruebas coinciden con la trazabilidad de `spec.md`.
- [ ] T039 Abrir el Pull Request desde `HU-27` citando HU-27, con:
  - las evidencias y la lista de archivos de otras historias que se editan (`plan.md`);
  - el aviso para la demo (migrar y configurar las contraseñas).

  Pedir la revisión de otro integrante, mergear con "Create a merge commit" y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- **Fase 1** primero. T001 puede detener todo si `crypto/pbkdf2` no está disponible.
- **Bloque A** no depende de nada más que la Fase 1.
- **Bloque B**:
  - T011 no depende de A;
  - T012 y T016 usan `auth.HashPasswordIterations` en las fixtures, así que dependen de T008;
  - T013 y T017 dependen de T008.
- **Bloque C** depende de A (T008) y de B (T015 y T017).
- **Bloque D** depende de A (T008 y T009) y de B (T015 y T016).
- **Bloque E** depende de D: su RED es la suite fallando con el handler nuevo.
- **Cierre** depende de todos los bloques. T038 se repite justo antes de abrir el PR.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar evidencia → implementación (GREEN).
- No subir la rama entre T028 y T032: la suite completa está en rojo.

### Oportunidades de paralelismo

- T005 y T006 entre sí.
- T011, T012 y T013 entre sí (después de T008).
- El Bloque C con el Bloque D, después de B.
- T023 y T024 entre sí.
- T030 y T031 entre sí.
- T034 y T035 entre sí.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T023, T024, T027, T030 |
| FR-002 | T023, T024, T026 |
| FR-003 | T006, T023, T024, T026 |
| FR-004 | T023, T024, T026 |
| FR-005 | T023, T024, T026 |
| FR-006 | T006, T009, T023, T024 |
| FR-007 | T006, T009, T035 |
| FR-008 | T012, T016, T024, T026 |
| FR-009 | T023, T026 |
| FR-010 | T023, T024, T026, T027 |
| FR-011 | T024 |
| FR-012 | T024, T031 |
| FR-013 | T005, T008, T024 |
| FR-014 | T005, T008, T023, T024 |
| FR-015 | T019, T023, T024, T033 |
| FR-016 | T012, T023, T024 |
| FR-017 | T013, T017, T019, T021, T024 |
| FR-018 | T005, T019, T021 |
| FR-019 | T013, T017, T019, T021 |
| FR-020 | T011, T015 |
| FR-021 | T029, T030, T031, T032 |
| SC-001 | T007, T014, T020, T025, T029, T036 |
| SC-002 | T024, T033 |
| SC-003 | T006, T023, T024 |
| SC-004 | T037 |
| SC-005 | T032, T034, T036 |
| SC-006 | T011, T018 |

## Estrategia de implementación

1. Fase 1 y Bloque A: las reglas, sin riesgo para otras historias.
2. Bloque B: la migración y los adaptadores. Todo sigue en verde, porque nadie usa todavía la columna.
3. Bloque C: el comando; ya se pueden configurar contraseñas.
4. Bloques D y E juntos (MVP): la identificación exige contraseña y las pruebas existentes se actualizan en la misma
   rama, antes de subirla.
5. Cierre y PR, al final de todas las demás historias.
