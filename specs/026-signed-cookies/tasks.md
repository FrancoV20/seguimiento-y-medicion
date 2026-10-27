# Tareas: HU-26 – Cookies firmadas

Formato: `[ID] [P] descripción` ([P] = se puede hacer en paralelo con otras tareas de su fase).
Trazabilidad: cada tarea cita el escenario (E) o requisito (FR) que cubre.

## Fase 0 – Preparación

- [ ] T001 Hacer `git pull` de `main` y comprobar con `git grep -n "CookieProject\|CookieMember" -- '*_test.go' test/` qué pruebas arman o leen cookies de contexto; confirmar la lista de `research.md` y la preparación de `test/context_bdd_test.go` (`nuevaSeleccionApp`, `formNavegador`, `poner`, `valorCookie`); anotar diferencias.
- [ ] T002 Crear la rama `feature/hu-26-cookies-firmadas` desde `main` actualizado.
- [ ] T003 `go test -count=1 -p 1 ./...` en verde antes de empezar.

## Fase 1 – RED: pruebas unitarias del firmador y de la clave

- [ ] T004 [P] `TestCookieSigner_RoundTrip`: lo firmado se verifica y devuelve el valor (E1, FR-001, FR-002).
- [ ] T005 [P] `TestCookieSigner_ValorModificado` y `TestCookieSigner_FirmaModificada` (E2, E3, FR-002, FR-003).
- [ ] T006 [P] `TestCookieSigner_LigadoAlNombre`: la firma de `ctx_proyecto` no vale en `ctx_integrante` (E4, FR-006).
- [ ] T007 [P] `TestCookieSigner_SinFirma` y `TestCookieSigner_MalFormada` con `7`, `""`, `.`, `7.`, `.abc` (E5, FR-003).
- [ ] T008 [P] `TestCookieSigner_OtraClave`: una firma hecha con otra clave no vale (FR-002, SC-003).
- [ ] T009 [P] `TestLoadCookieSecret_Valida`, `_Ausente` (marca `temporary`) y `_Corta` (E6, E7, E8, FR-008, FR-009, FR-010).
- [ ] T010 [P] `TestLoadCookieSecret_NoExponeLaClave`: el mensaje de error no contiene el valor configurado (FR-011).
- [ ] T011 [P] `TestReadIDCookie_FirmaInvalidaEsManipulada`: devuelve `(0, true)` ante las entradas de la tabla de `data-model.md` (FR-003).

## Fase 2 – RED: escenarios de aplicación (test/web_signed_cookies_bdd_test.go)

- [ ] T012 Preparar la aplicación de prueba con la misma preparación de `test/context_bdd_test.go`.
- [ ] T013 [P] `TestHU26_ContextoValidoSeConserva`: elegir proyecto e integrante por la pantalla, navegar y ver ambos en la estructura común (E1, FR-013).
- [ ] T014 [P] `TestHU26_CookieProyectoModificada`: cambiar el número de `ctx_proyecto`; se pide elegir proyecto y ambas cookies se borran (E2, FR-004).
- [ ] T015 [P] `TestHU26_CookieIntegranteModificada`: cambiar `ctx_integrante`; se conserva el proyecto y se pide identificarse (E3, FR-005).
- [ ] T016 [P] `TestHU26_ValorCopiadoEntreCookies`: copiar el valor firmado de proyecto a integrante; se rechaza (E4, FR-006).
- [ ] T017 [P] `TestHU26_CookiesSinFirmaSeDescartan`: cookies con solo el número; se descartan sin error técnico (E5, FR-003).
- [ ] T018 [P] `TestHU26_ReinicioConMismaClave` (en `src/web`): con `SetCookieSecret(k)` dos veces el contexto se conserva y con otra clave se pierde (E6, SC-003).
- [ ] T019 [P] `TestRun_ClaveCortaNoArranca` en `src/cmd/server/main_test.go`: `run` con `DATABASE_URL` y `COOKIE_SECRET` corta devuelve 1, nombra el largo mínimo y no abre el puerto (E8, FR-010).
- [ ] T020 Ejecutar las pruebas HU-26, confirmar que fallan por la razón esperada y guardar `evidencia-red.txt` (`chcp 65001`).

## Fase 3 – GREEN

- [ ] T021 `src/web/cookie_signer.go`: `CookieSigner`, clave global inicializada al azar, `SetCookieSecret`, `SignedContextValue`; T004–T008 en verde.
- [ ] T022 `src/web/cookie_secret.go`: `LoadCookieSecret` y `ShortCookieSecretError`; T009–T010 en verde.
- [ ] T023 Editar `src/web/context.go`: `setContextCookie` firma y `readIDCookie` verifica (T011 en verde). No tocar `newContextCookie`, `clearContextCookie` ni `flash.go`.
- [ ] T024 Actualizar los helpers `idCookie` (`src/web/context_handlers_test.go`) y `cookieID` (`test/context_bdd_test.go`) para que usen `SignedContextValue`.
- [ ] T025 Actualizar las pruebas existentes que dependían del valor crudo: `TestReadIDCookie_Valores` (firmar cada valor antes de agregarlo, conservando los casos que rechaza `parseID`), `TestSetContextCookie_AtributosDeSesion` (verificar la firma en lugar de `"42"`), la comparación con `id64(...)` en `context_handlers_test.go` y `context_member_test.go`, y las cookies de `projects_handlers_test.go` (usar `idCookie`).
- [ ] T026 Editar `src/cmd/server/main.go`: justo después de `LoadConfig`, llamar a `web.LoadCookieSecret`; si hay error, `fail(err.Error())`; si la clave es temporal, `logger.Warn` con el aviso; luego `web.SetCookieSecret`.
- [ ] T027 Ejecutar T013–T019 en verde.
- [ ] T028 `go test -count=1 -p 1 ./...` completo: todo lo existente sigue en verde (SC-004).

## Fase 4 – REFACTOR y cierre

- [ ] T029 Confirmar con `git grep -n "r.Cookie(\|http.SetCookie" src` que el único acceso a cookies de contexto pasa por `readIDCookie` y `setContextCookie` y que la clave no se imprime en ningún registro.
- [ ] T030 No usar `gofmt -w` sobre archivos existentes (CRLF); los archivos nuevos formateados; `go vet ./...` limpio.
- [ ] T031 Guardar `evidencia-pruebas.txt` y `cobertura.txt` (`chcp 65001`).
- [ ] T032 Probar a mano `quickstart.md`.
- [ ] T033 Agregar al `README.md`, junto a `DATABASE_URL`, una nota sobre `COOKIE_SECRET` (mínimo 32 caracteres; ausente = clave temporal).
- [ ] T034 Actualizar la trazabilidad de `spec.md` con los nombres reales de las pruebas.
- [ ] T035 PR hacia `main` citando HU-26, con evidencias y la lista de archivos editados; avisar en el equipo a quienes tengan pruebas con cookies armadas a mano.
- [ ] T036 Revisión por un compañero y merge con "Create a merge commit".