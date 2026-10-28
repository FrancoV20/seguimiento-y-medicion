# Tareas: HU-26 – Cookies firmadas

**Entrada**: `specs/026-signed-cookies/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-20b en `main`.

Formato: `[ID] [P] [E#] descripción`
- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario del Issue que cubre.
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y se guarda la salida en `specs/026-signed-cookies/evidencia-red.txt`.
- Antes de cada ejecución en PowerShell: `chcp 65001`.

## Fase 1: Preparación

- [X] T001 Hacer `git pull` de `main` y comprobar con `git grep -n "CookieProject\|CookieMember" -- '*_test.go' test/` qué pruebas arman o leen cookies de contexto; confirmar la lista de `research.md` y la preparación de `test/context_bdd_test.go` (`nuevaSeleccionApp`, `formNavegador`, `poner`, `valorCookie`); anotar diferencias en `research.md` — Resultado anotado en `research.md` ("Resultado de T001")
- [X] T002 Crear la rama `feature/hu-26-cookies-firmadas` desde `main` actualizado — Se trabaja en la rama `HU-26`, la que existe en el repositorio
- [ ] T003 `go test -count=1 -p 1 ./...` en verde antes de empezar (línea base)

## Fase 2: Fundamentos (firmador y clave; bloquea todo lo demás)

### Pruebas (RED)

- [X] T004 [P] [E1] `TestCookieSigner_RoundTrip`: lo firmado se verifica y devuelve el valor, en `src/web/cookie_signer_test.go`
- [X] T005 [P] [E2, E3] `TestCookieSigner_ValorModificado` y `TestCookieSigner_FirmaModificada` en `src/web/cookie_signer_test.go` — Cambia caracteres al principio, en el medio y al final de la firma; la decodificación es estricta (`RawURLEncoding.Strict()`)
- [X] T006 [P] [E4] `TestCookieSigner_LigadoAlNombre`: la firma de `ctx_proyecto` no vale en `ctx_integrante`, en `src/web/cookie_signer_test.go`
- [X] T007 [P] [E5] `TestCookieSigner_SinFirma` y `TestCookieSigner_MalFormada` con `7`, `""`, `.`, `7.`, `.abc`, en `src/web/cookie_signer_test.go`
- [X] T008 [P] `TestCookieSigner_OtraClave`: una firma hecha con otra clave no vale, en `src/web/cookie_signer_test.go`
- [X] T009 [P] [E6, E7, E8] `TestLoadCookieSecret_Valida`, `_Ausente` (marca `temporary` y largo 32) y `_Corta` (error `ShortCookieSecretError`), en `src/web/cookie_secret_test.go`
- [X] T010 [P] `TestLoadCookieSecret_NoExponeLaClave`: el mensaje de error no contiene el valor configurado, en `src/web/cookie_secret_test.go`
- [ ] T011 Ejecutar T004–T010, confirmar que fallan porque el código no existe y guardar la salida en `evidencia-red.txt` (sección "Fundamentos")

### Implementación (GREEN)

- [X] T012 Implementar `CookieSigner`, la clave global (`atomic.Pointer`) inicializada al azar, `SetCookieSecret` y `SignedContextValue` en `src/web/cookie_signer.go`; T004–T008 en verde
- [X] T013 Implementar `EnvCookieSecret`, `MinCookieSecretLength`, `ShortCookieSecretError` y `LoadCookieSecret` en `src/web/cookie_secret.go`; T009–T010 en verde

**Punto de control**: el firmador y la carga de la clave funcionan solos, sin tocar HU-20b.

## Fase 3: Bloque A – Contexto firmado (escenarios 1 a 5) 🎯 MVP

**Objetivo**: el servidor solo acepta cookies de contexto que él emitió.

**Prueba independiente**: elegir proyecto e integrante, comprobar que las cookies tienen la forma `número.firma`, modificarlas o dejarlas sin firma y comprobar que se descartan.

### Pruebas (RED)

- [X] T014 [P] [E2, E5] `TestReadIDCookie_FirmaInvalidaEsManipulada`: devuelve `(0, true)` ante las entradas de la tabla de `data-model.md`, en `src/web/context_test.go`
- [X] T015 [P] [E1] `TestHU26_ContextoValidoSeConserva`: elegir proyecto e integrante por la pantalla, navegar y ver ambos en la estructura común, en `test/web_signed_cookies_bdd_test.go`
- [X] T016 [P] [E2] `TestHU26_CookieProyectoModificada`: cambiar el número de `ctx_proyecto`; se pide elegir proyecto y ambas cookies se borran, en `test/web_signed_cookies_bdd_test.go`
- [X] T017 [P] [E3] `TestHU26_CookieIntegranteModificada`: cambiar `ctx_integrante`; se conserva el proyecto y se pide identificarse en una acción que lo requiere, en `test/web_signed_cookies_bdd_test.go` — Usa un segundo integrante del proyecto para que la prueba falle si se aceptan cookies sin firmar
- [X] T018 [P] [E4] `TestHU26_ValorCopiadoEntreCookies`: copiar el valor firmado de proyecto a integrante; se rechaza, en `test/web_signed_cookies_bdd_test.go`
- [X] T019 [P] [E5] `TestHU26_CookiesSinFirmaSeDescartan`: cookies con solo el número; se descartan sin error técnico, en `test/web_signed_cookies_bdd_test.go`
- [ ] T020 Ejecutar T014–T019, confirmar que fallan por la razón esperada (las cookies sin firma todavía se aceptan) y guardar la salida en `evidencia-red.txt` (sección "Contexto firmado")

### Implementación (GREEN)

- [X] T021 Editar `src/web/context.go`: `setContextCookie` firma el valor y `readIDCookie` verifica la firma (devuelve `(0, true)` si falla y `parseID` del valor si pasa); no tocar `newContextCookie`, `clearContextCookie` ni `flash.go`
- [X] T022 Actualizar los helpers `idCookie` (`src/web/context_handlers_test.go`) y `cookieID` (`test/context_bdd_test.go`) para que usen `SignedContextValue`
- [X] T023 Actualizar las pruebas existentes que dependían del valor crudo: `TestReadIDCookie_Valores` (firmar cada valor antes de agregarlo, conservando los casos que rechaza `parseID`), `TestSetContextCookie_AtributosDeSesion` (verificar la firma en lugar de `"42"`; los atributos no cambian), la comparación con `id64(...)` en `context_handlers_test.go` y `context_member_test.go`, y las cookies de `projects_handlers_test.go` (usar `idCookie`)
- [ ] T024 Ejecutar T014–T019 en verde y `go test -count=1 -p 1 ./...` completo: todo lo existente sigue en verde

**Punto de control**: con la clave temporal por defecto, la aplicación ya funciona firmada.

## Fase 4: Bloque B – Clave y arranque (escenarios 6 a 8)

**Objetivo**: la clave se configura con `COOKIE_SECRET` y un error de configuración se detecta al arrancar.

**Prueba independiente**: arrancar con clave válida, ausente y corta, y reiniciar con la misma clave.

### Pruebas (RED)

- [X] T025 [P] [E6] `TestHU26_ReinicioConMismaClave`: con `SetCookieSecret(k)` dos veces el contexto se conserva y con otra clave se pierde; restaurar la clave con `t.Cleanup`; sin `t.Parallel`, en `src/web/cookie_signer_test.go`
- [X] T026 [P] [E8] `TestRun_ClaveCortaNoArranca`: `run` con `DATABASE_URL` y `COOKIE_SECRET` corta devuelve 1, nombra el largo mínimo, no imprime la clave y no abre el puerto, en `src/cmd/server/main_test.go`
- [ ] T027 Ejecutar T025–T026, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Clave y arranque")

### Implementación (GREEN)

- [X] T028 Editar `src/cmd/server/main.go`: justo después de `LoadConfig`, llamar a `web.LoadCookieSecret(getenv)`; con error, `fail(err.Error())`; con clave temporal, `logger.Warn` avisando que las sesiones se perderán al reiniciar; luego `web.SetCookieSecret`
- [ ] T029 Ejecutar T025–T026 en verde

## Fase 5: Cierre

- [X] T030 Revisar con `git grep -n "r.Cookie(\|http.SetCookie" src` que el único acceso a cookies de contexto pasa por `readIDCookie` y `setContextCookie`, que la verificación usa `hmac.Equal` y que la clave no se imprime en ningún registro ni mensaje — Solo `readIDCookie` y `setContextCookie` acceden a las cookies de contexto; `Verify` usa `hmac.Equal`; `flash` sigue sin firma; la clave solo se lee en `LoadCookieSecret` y no se registra
- [ ] T031 [P] No usar `gofmt -w` sobre archivos existentes (CRLF); archivos nuevos formateados; `go vet ./...` limpio; pipeline de GitHub Actions en verde
- [ ] T032 Guardar `evidencia-pruebas.txt` (`go test -count=1 -p 1 -v ./...`) y `cobertura.txt` (`go test -count=1 -p 1 -cover ./...`)
- [X] T033 Verificar a mano los 8 escenarios con la aplicación en ejecución (`quickstart.md`), incluido el aviso de clave temporal del escenario 7
- [X] T034 [P] Agregar al `README.md`, junto a `DATABASE_URL`, una nota sobre `COOKIE_SECRET` (mínimo 32 caracteres; ausente = clave temporal)
- [X] T035 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas — Los nombres de las pruebas coinciden con la tabla de `spec.md`
- [X] T036 Abrir el Pull Request desde `HU-26` citando HU-26, con evidencias y la lista de archivos editados; avisar en el equipo a quienes tengan pruebas con cookies armadas a mano
- [ ] T037 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho"

## Dependencias y orden de ejecución

- **Fase 1** primero. **Fase 2** depende de la Fase 1 y bloquea a las demás.
- **Fase 3 (Bloque A)** depende de la Fase 2 (T012 y T013). Es el MVP.
- **Fase 4 (Bloque B)** depende de T013 y de T021 (para que T025 use cookies firmadas); T028 puede hacerse en paralelo con la Fase 3 si otra persona edita solo `main.go`.
- **Fase 5** depende de las fases que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar evidencia → implementación (GREEN).

### Oportunidades de paralelismo

- T004–T010 entre sí.
- T014–T019 entre sí.
- T025 y T026 entre sí.
- T031, T034 y T035 entre sí.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T004, T012, T021 |
| FR-002 | T004, T005, T008, T014, T021 |
| FR-003 | T005, T007, T014, T019 |
| FR-004 | T016 |
| FR-005 | T017 |
| FR-006 | T006, T018 |
| FR-007 | T012, T030 |
| FR-008 | T009, T025 |
| FR-009 | T009, T033 |
| FR-010 | T009, T026 |
| FR-011 | T010, T026, T030 |
| FR-012 | T023, T024 |
| FR-013 | T015, T024 |
| SC-001 | T016, T017, T018, T019 |
| SC-002 | T015, T024 |
| SC-003 | T008, T025 |
| SC-004 | T024, T032 |
| SC-005 | T011, T020, T027, T032 |

## Estrategia de implementación

1. Completar Fase 1 y Fase 2.
2. Bloque A: es el MVP; con él la aplicación ya rechaza cookies falsificadas.
3. Bloque B: agrega la configuración y la falla temprana.
4. Cierre y PR.