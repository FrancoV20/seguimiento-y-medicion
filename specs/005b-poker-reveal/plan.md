# Plan de implementación: HU-05b – Revelación de Votos y Detección de Divergencia

**Rama**: `HU-05b` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Constitución**: v1.1.0

## Resumen

Se agrega a `src/poker` la regla pura de divergencia (`Spread`, archivo nuevo `spread.go`), la regla de cierre
(`CheckClose`), el error `ErrNoVotes`, el método `CloseRound` del puerto `Store`, del `Service` y del adaptador
`PostgreSQLStore` (ronda `FOR UPDATE OF r` → contar votos → un UPDATE con `status` y `closed_at`). En `src/web` se
suma la ruta `POST /planning-poker/cerrar` al `Routes` de `RegisterPlanningPoker`, el botón "Cerrar ronda" en cada
ronda abierta y la sección "Rondas cerradas", leída por el mismo `PokerCatalog` en su transacción de solo lectura,
con los votos revelados y el rango o la alerta de divergencia. Sin migración; no se edita `src/cmd/server/main.go`.

## Contexto técnico

- **Lenguaje**: Go 1.27.1.
- **Dependencias**: biblioteca estándar y `github.com/jackc/pgx/v5/stdlib`, ya incorporado. Sin dependencias nuevas.
- **Almacenamiento**: PostgreSQL. Sin migración: se usan `poker_rounds.status`, `poker_rounds.closed_at` y
  `poker_votes` de `011_create_poker` (ver `data-model.md`).
- **Pruebas**: `go test -count=1 -p 1 ./...`. Reglas y servicio sin base (con `fakeStore`); adaptador y catálogo contra
  PostgreSQL real con datos propios (se omiten sin `DATABASE_URL`, como el resto del repo); handlers con `httptest` y
  `cookiejar` (proyecto e integrante elegidos por el flujo de HU-20b); BDD en `test/web_poker_reveal_bdd_test.go`.
- **Concurrencia**: prueba determinista de un voto (`FOR SHARE`) que espera al cierre (`FOR UPDATE`) y termina en
  `ErrRoundClosed` sin cambiar el voto (`research.md`, decisión 4).
- **CI**: `gofmt -l .` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. `gofmt -w` solo sobre
  archivos nuevos; los existentes (CRLF) se editan a mano sin cambiar su fin de línea.
- **Plataforma**: aplicación SSR sin JavaScript; desarrollo en Windows con Docker Desktop.
- **Rendimiento**: la pantalla responde en menos de 500 ms con 20 rondas abiertas, 50 cerradas y 10 integrantes
  (SC-006).
- **Alcance**: una ruta nueva, una sección nueva en la pantalla existente, sin tablas nuevas.

## Verificación de la constitución

*Antes de Phase 0 y otra vez después del diseño.*

| Principio | Cumplimiento |
|---|---|
| I. Alcance guiado por valor | PASS. Cierra el ciclo de estimación de HU-05a (revelar y marcar divergencias); nueva ronda y estimación acordada quedan en HU-18, roles en HU-28 |
| II. Calidad por construcción | PASS. Reglas puras (`Spread`, `CheckClose`) en `src/poker`; transacción en el adaptador; lectura en el catálogo; la web solo traduce errores y arma textos. La serie no se duplica |
| III. Test-first | PASS. Tres bloques con pruebas escritas y vistas fallar (`evidencia-red.txt`) antes del código; escenarios en Dado/Cuando/Entonces |
| IV. Ejecución ágil | PASS. Bloques demostrables: la regla sola (HU-18 ya puede usarla), el cierre sin pantalla, la pantalla completa |
| V. Medición | PASS. La divergencia alimenta la discusión que mejora la calidad de la estimación (HU-18) |
| VI. PostgreSQL | PASS. El método nuevo `Store.CloseRound` → `PostgreSQLStore.CloseRound` y la lectura nueva de `PokerCatalog` → `PostgreSQLPokerCatalog`, ambos con pruebas de integración contra PostgreSQL |
| Restricciones técnicas | PASS. Go + PostgreSQL, sin dependencias nuevas, sin migración |
| Otras historias | PASS. Solo se editan archivos de HU-05a; sus pruebas siguen en verde sin cambios (salvo `failingPokerStore` y `fakeStore`, que suman el método del puerto) |

## Estructura de archivos

### Documentación

```text
specs/005b-poker-reveal/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML (rutas, campos y códigos en `spec.md` y abajo) y el
contrato para HU-18 (`poker.Spread`) está en `data-model.md`.

### Código

Nuevos:

```text
src/poker/
├── spread.go                                   # VoteSpread, Spread, level
├── spread_test.go                              # TestSpread, TestSpread_Errores, TestSpread_NoModificaLosValores
├── reveal_fixtures_integration_test.go         # newAlfaFixture («Alfa», «Beta», votos del Issue) y waitForWaiters
└── close_integration_test.go                   # TestPostgresCloseRound_*
src/web/
├── poker_reveal_fixtures_test.go               # newAlfaWebFixture, closeAt, removeMember, roundRow, closedRow
├── postgres_poker_reveal_integration_test.go   # TestPostgresPokerCatalog_RondasCerradas, _VotosRevelados, …
├── poker_reveal_view_test.go                   # TestHU05b_ divergencia, para todos, seguridad, vacío, escape, rendimiento
└── poker_close_handlers_test.go                # TestHU05b_ cierre y rechazos (E1, E5, E6, E8–E12)
test/
└── web_poker_reveal_bdd_test.go                # TestBDD_HU05b_*
```

Editados (todos de HU-05a):

| Archivo | Cambio |
|---|---|
| `src/poker/errors.go` | `ErrNoVotes` |
| `src/poker/round.go` | `CheckClose`, `ClosedRound` |
| `src/poker/store.go` | `CloseRound(ctx, projectID, roundID int64) (ClosedRound, error)` en `Store` |
| `src/poker/service.go` | `Service.CloseRound(ctx, projectID, roundID, memberID int64)` |
| `src/poker/postgres_store.go` | `PostgreSQLStore.CloseRound` (transacción de `data-model.md`) |
| `src/poker/round_test.go` | `TestCheckClose` |
| `src/poker/service_test.go` | `fakeStore.CloseRound` y `TestService_CloseRound_*` |
| `src/web/poker_catalog.go` | `PokerRevealedVoteView`, `PokerClosedRoundView`, `PokerBoard.ClosedRounds` |
| `src/web/postgres_poker_catalog.go` | dos consultas nuevas en la misma transacción |
| `src/web/poker_form.go` | `ErrNoVotes` → 409 en `pokerOutcomeFor` |
| `src/web/poker_messages.go` | textos nuevos (ver decisión 6) |
| `src/web/poker_handlers.go` | `PokerActions.CloseRound`, ruta y handler `close`, `pokerClosedRoundPage` en `render` |
| `src/web/poker_handlers_test.go` | `failingPokerStore.CloseRound` (falla) |
| `src/web/templates/poker.html` | botón "Cerrar ronda" y sección "Rondas cerradas" |

## Direcciones y campos

| Método | Dirección | Campos | Éxito | Rechazos |
|---|---|---|---|---|
| GET | `/planning-poker` | — | 200 (ahora con "Rondas cerradas") | lectura fallida → `Fail` |
| POST | `/planning-poker/cerrar` | `ronda` | 303 a `/planning-poker` + "Ronda cerrada para la historia «…»" | 404, 409, 500 |
| POST | `/planning-poker/abrir`, `/planning-poker/votar` | sin cambios (HU-05a) | | |

Sin integrante, `RequireContext` responde 303 a `/contexto/integrante?volver_a=%2Fplanning-poker` antes del handler
(escenario 11).

## Decisiones de diseño

Detalle y alternativas en `research.md`.

1. **Regla pura de divergencia** `poker.Spread(values []int) (VoteSpread, error)` en `spread.go` (contrato para HU-18).
2. **Regla pura de cierre** `CheckClose(status, votes)`: cerrada antes que sin votos.
3. **Transacción de cierre**: ronda del proyecto `FOR UPDATE OF r` con historia y número → `COUNT(*)` de votos →
   `CheckClose` → `UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`. Sin fila →
   `ErrRoundNotFound`, por eso el 404 se decide antes que el 409.
4. **Servicio**: `roundID == 0` → `ErrRoundNotFound`, `memberID == 0` → `ErrMemberRequired`, sin llamar al almacén; el
   puerto no recibe el integrante (no se guarda quién cierra).
5. **Traducción de errores** (`pokerOutcomeFor`, se agrega una fila):

   | Error | Código | Texto |
   |---|---|---|
   | `ErrRoundNotFound` | 404 | `MsgPokerRoundNotFound` "La ronda no existe" (ya existe) |
   | `ErrRoundClosed` | 409 | `MsgPokerRoundClosed` "La ronda ya está cerrada" (ya existe) |
   | `ErrNoVotes` | 409 | `MsgPokerNoVotes` "La ronda no tiene votos" (nuevo) |
   | cualquier otro | 500 | `MsgOperationFailed`; el detalle va a `logger.Error` (ya existe) |

6. **Textos nuevos** en `poker_messages.go`: `MsgPokerNoVotes`, `MsgPokerNoClosedRounds` "Aún no hay rondas cerradas",
   `LabelPokerDidNotVote` "No votó", `pokerRoundClosedMessage(title)`, `pokerDivergenceMessage(low, high)` y
   `pokerNoDivergenceMessage(low, high)`.
7. **Lectura**: `PokerBoard.ClosedRounds` en la misma transacción `READ ONLY REPEATABLE READ`; rondas cerradas por
   `closed_at DESC, id DESC`; votos revelados = integrantes actuales ∪ quienes votaron, por nombre; `value` solo desde
   el CTE de rondas `'Cerrada'`.
8. **Vista**: `render` arma `pokerClosedRoundPage` con `poker.Spread`: alerta `FlashWarning` con el parcial `alert` de
   HU-20b si diverge; párrafo "Sin divergencia…" si no; nada si `Spread` falla (ronda cerrada sin votos de datos
   externos).
9. **Plantilla**: formulario "Cerrar ronda" (`ronda` oculto) en cada ronda abierta; sección "Rondas cerradas" al
   final, después de las rondas abiertas, con `data-ronda-cerrada` por ronda y `data-integrante` por fila. Como
   `data-integrante` se repite entre secciones, las pruebas de HU-05b buscan cada fila dentro de su ronda
   (`roundRow`, `closedRow`); `participantRow` de HU-05a sigue encontrando primero las rondas abiertas.
10. **Sin cambios de comportamiento en HU-05a**: abrir y votar, los tipos de rondas abiertas y el ocultamiento siguen
    iguales; las pruebas de HU-05a corren en cada GREEN.

## Flujo del POST `/planning-poker/cerrar`

1. `RequireContext` garantiza proyecto e integrante (si no, 303 de HU-20b).
2. `pokerFormIDs(r)` lee `ronda` con `parseID` (inválido → 0).
3. `Service.CloseRound(ctx, ac.ProjectID, round, ac.MemberID)`: `round == 0` → `ErrRoundNotFound` →
   `Store.CloseRound`.
4. Error → `reject` → `pokerOutcomeFor` → pantalla con su código y mensaje (500 con `logger.Error`).
5. Éxito → `done(w, r, pokerRoundClosedMessage(closed.StoryTitle))`: `SetFlash` + 303 a `/planning-poker`.

## Contratos que consume

- HU-05a: todo lo listado en `research.md` ("Datos verificados").
- HU-20a: `Server.Register`, `Renderer.PageStatus`, `Renderer.Fail`, parcial `alert`.
- HU-20b: `RequireContext`, `ActiveContextFrom`, `SetFlash`, `RedirectSeeOther`, `parseID`, `MsgOperationFailed`,
  `Flash`/`FlashWarning`, `CrossOriginGuard`.

## Contrato que se ofrece (HU-18)

`poker.Spread(values []int) (poker.VoteSpread, error)`, `poker.VoteSpread{Low, High int; Divergent bool}`,
`poker.ErrNoVotes`, `poker.ClosedRound` y `Store.CloseRound` / `Service.CloseRound` (firmas en `data-model.md`).

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Un error de plantilla o de consulta revela el voto de una ronda abierta | El valor solo se lee desde el CTE de rondas `'Cerrada'` y vive solo en `PokerRevealedVoteView`; `TestPostgresPokerCatalog_SoloRondasCerradas` y la página idéntica byte a byte de `TestHU05b_VotoAbiertoNuncaSeRevela` |
| Las rondas cerradas sin votos que preparan las pruebas de HU-05a rompen la pantalla | `render` tolera el error de `Spread` (decisión 8) y `TestHU05b_RondaCerradaSinVotos` lo fija; las pruebas de HU-05a corren en cada GREEN |
| Prueba de concurrencia inestable | Orden forzado con `LOCK TABLE poker_votes` y espera explícita de dos sesiones bloqueadas (`research.md`, decisión 4), con plazo de 5 s; nada de `time.Sleep` como sincronización |
| El `LOCK TABLE` de la prueba bloquea a otras pruebas | `-p 1`; la transacción se revierte siempre (`defer tx.Rollback()`) |
| Cambiar el puerto rompe la compilación de pruebas de HU-05a | El puerto, el adaptador y las dos pruebas falsas van en el mismo commit (T012–T013) |
| HU-18 empieza antes y usa otro nombre para la regla | El contrato queda en `data-model.md`; T030 avisa a HU-18 los nombres finales |
| Conflictos con quien edite `poker.html` o `poker_handlers.go` en paralelo (HU-18) | HU-18 va después de HU-05b (notas del equipo) |

## Complexity Tracking

Sin violaciones de la constitución que justificar.

## Verificación posterior al diseño

- I–V: PASS (sin cambios respecto de la verificación inicial).
- VI: PASS. Un método nuevo del puerto y una lectura nueva del catálogo, ambos con adaptador PostgreSQL y pruebas de
  integración; sin migración.
- Ningún `NEEDS CLARIFICATION` pendiente (`research.md`, "Preguntas abiertas": ninguna).
