# Plan de implementación: HU-18 – Planning Poker: Nueva Ronda y Estimación Acordada

**Rama**: `HU-18` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Constitución**: v1.1.0

## Resumen

Se agregan a `src/poker` dos reglas puras junto a `CheckOpen` (`CheckNextRound`, `CheckAgree`), tres errores
(`ErrNotLatestRound`, `ErrRoundStillOpen`, `ErrStoryInFinishedSprint`), dos métodos del puerto `Store`, del `Service` y
del adaptador `PostgreSQLStore` (`OpenNextRound`, `AgreeEstimate`), con el orden de bloqueos ronda → Sprint → historia.
En `src/web` se suman las rutas `POST /planning-poker/nueva-ronda` y `POST /planning-poker/estimacion` al `Routes` de
`RegisterPlanningPoker`, y la última ronda cerrada de cada historia en "Rondas cerradas" (HU-05b) muestra "Nueva ronda" y
el formulario "Estimación acordada", con recordatorio si hubo divergencia y deshabilitados con el motivo "Sprint
finalizado" si corresponde. Sin migración; no se edita `src/cmd/server/main.go`. **Se implementa después de que HU-05b
esté en `main`.**

## Contexto técnico

- **Lenguaje**: Go 1.27.1.
- **Dependencias**: biblioteca estándar y `github.com/jackc/pgx/v5/stdlib`, ya incorporado. Sin dependencias nuevas.
- **Almacenamiento**: PostgreSQL. Sin migración: `poker_rounds`/`poker_votes` (011), `user_stories.story_points` (002),
  `sprints` y `sprint_story_assignments` (004). Ver `data-model.md`.
- **Pruebas**: `go test -count=1 -p 1 ./...`. Reglas y servicio sin base (con `fakeStore`); adaptador y catálogo contra
  PostgreSQL real con datos propios (el Sprint finalizado de «Alta» se carga directo en la base; se omiten sin
  `DATABASE_URL`); handlers con `httptest` y `cookiejar` (proyecto e integrante por el flujo de HU-20b), con el Backlog
  registrado para comprobar los Story Points; BDD en `test/web_poker_consensus_bdd_test.go`.
- **Concurrencia**: dos "Nueva ronda" simultáneas (una OK y otra `ErrRoundAlreadyOpen`); estimación contra cierre de
  Sprint en los dos órdenes, sin bloqueo mutuo; "Nueva ronda" contra estimación desde la misma ronda (`research.md`,
  decisión 7).
- **CI**: `gofmt -l .` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. `gofmt -w` solo sobre
  archivos nuevos; los existentes se editan a mano sin cambiar su fin de línea.
- **Plataforma**: aplicación SSR sin JavaScript.
- **Rendimiento**: la pantalla responde en menos de 500 ms con 20 rondas abiertas, 50 cerradas y 10 integrantes
  (SC-005).
- **Alcance**: dos rutas nuevas, dos acciones en una sección existente, sin tablas nuevas.

## Verificación de la constitución

*Antes de Phase 0 y otra vez después del diseño.*

| Principio | Cumplimiento |
|---|---|
| I. Alcance guiado por valor | PASS. Cierra el ciclo de estimación: el resultado de Planning Poker llega al Backlog; roles y automatismos quedan afuera |
| II. Calidad por construcción | PASS. Reglas puras con el orden de rechazos en `src/poker`; transacciones en el adaptador; la web solo traduce errores; reutiliza `ParseValue`, `Spread` y la serie |
| III. Test-first | PASS. Cuatro bloques con pruebas vistas fallar (`evidencia-red.txt`) antes del código; escenarios en Dado/Cuando/Entonces |
| IV. Ejecución ágil | PASS. Bloques demostrables: reglas, nueva ronda, estimación, pantalla |
| V. Medición | PASS. Los Story Points acordados alimentan velocidad y métricas (HU-07, HU-10) |
| VI. PostgreSQL | PASS. `Store.OpenNextRound` y `Store.AgreeEstimate` con adaptador `PostgreSQLStore` y pruebas de integración; la lectura ampliada del catálogo, con su prueba de integración |
| Restricciones técnicas | PASS. Go + PostgreSQL, sin dependencias nuevas, sin migración |
| Otras historias | PASS. Solo archivos de HU-05a y HU-05b; sus pruebas siguen en verde sin cambiar expectativas (solo `fakeStore` y `failingPokerStore` suman los métodos del puerto) |

## Estructura de archivos

### Documentación

```text
specs/018-poker-consensus/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML (rutas, campos y códigos en `spec.md` y abajo).

### Código

Nuevos:

```text
src/poker/
├── consensus_rules_test.go                     # TestCheckNextRound, TestCheckAgree
├── consensus_fixtures_integration_test.go      # newConsensusFixture (datos del Issue), closeSprintLikeHU15
├── next_round_integration_test.go              # TestPostgresOpenNextRound_*
└── agree_estimate_integration_test.go          # TestPostgresAgreeEstimate_*, TestPostgresNextRoundYEstimacion_Serializadas
src/web/
├── poker_consensus_fixtures_test.go            # newConsensusWebFixture, servidor con Backlog
├── postgres_poker_consensus_integration_test.go # TestPostgresPokerCatalog_UltimaRonda, _SprintFinalizado
└── poker_consensus_handlers_test.go            # TestHU18_*
test/
└── web_poker_consensus_bdd_test.go             # TestBDD_HU18_*
```

Editados:

| Archivo | Historia | Cambio |
|---|---|---|
| `src/poker/errors.go` | HU-05a | `ErrNotLatestRound`, `ErrRoundStillOpen`, `ErrStoryInFinishedSprint` |
| `src/poker/round.go` | HU-05a | `StoryRound`, `CheckNextRound`, `CheckAgree`, `AgreedEstimate` |
| `src/poker/store.go` | HU-05a | `OpenNextRound`, `AgreeEstimate` en `Store` |
| `src/poker/service.go` | HU-05a | `Service.OpenNextRound`, `Service.AgreeEstimate` |
| `src/poker/postgres_store.go` | HU-05a/05b | `OpenNextRound`, `AgreeEstimate`, ayudantes de asignación y rondas |
| `src/poker/service_test.go` | HU-05a | `fakeStore` suma los dos métodos; `TestService_OpenNextRound_*`, `TestService_AgreeEstimate_*` |
| `src/web/poker_catalog.go` | HU-05b | `PokerClosedRoundView.Latest`, `.InFinishedSprint` (T001: HU-05b) |
| `src/web/postgres_poker_catalog.go` | HU-05b | dos columnas en `pokerClosedRoundsQuery` (T001: HU-05b) |
| `src/web/poker_form.go` | HU-05a | tres errores → 409 en `pokerOutcomeFor` |
| `src/web/poker_messages.go` | HU-05a/05b | textos de `data-model.md` |
| `src/web/poker_handlers.go` | HU-05a/05b | `PokerActions` suma dos métodos; rutas y handlers `nextRound`, `agree`; `pokerClosedRoundPage` suma `Actions`, `Locked`, `Reminder` (T001: HU-05b) |
| `src/web/poker_handlers_test.go` | HU-05a | `failingPokerStore` falla también en los dos métodos nuevos |
| `src/web/templates/poker.html` | HU-05a/05b | acciones en la última ronda cerrada (T001: HU-05b) |

## Direcciones y campos

| Método | Dirección | Campos | Éxito | Rechazos |
|---|---|---|---|---|
| POST | `/planning-poker/nueva-ronda` | `ronda` | 303 a `/planning-poker` + "Ronda N abierta para la historia «…»" | 404, 409, 500 |
| POST | `/planning-poker/estimacion` | `ronda`, `valor` | 303 a `/planning-poker` + "Estimación registrada: «…» queda con N Story Points" | 422, 404, 409, 500 |
| GET, POST | `/planning-poker`, `/abrir`, `/votar`, `/cerrar` | sin cambios (HU-05a, HU-05b) | | |

Sin integrante, `RequireContext` responde 303 a `/contexto/integrante?volver_a=%2Fplanning-poker`.

## Decisiones de diseño

Detalle y alternativas en `research.md`.

1. **Reglas puras** `CheckNextRound` y `CheckAgree` con el orden de rechazos de la spec (R5 y R6b).
2. **Nueva ronda**: ronda base `FOR SHARE` → asignación → Sprint `FOR SHARE` → historia `FOR NO KEY UPDATE OF us` con
   relectura → rondas → `CheckNextRound` → `INSERT` de `mayor + 1`; `23505` → `ErrRoundAlreadyOpen` /
   `ErrNotLatestRound`.
3. **Estimación**: `ParseValue` en el servicio; ronda `FOR UPDATE` → `max(round_number)` → asignación → Sprint `FOR
   SHARE` → historia `FOR NO KEY UPDATE OF us` con relectura → `CheckAgree` → `UPDATE user_stories SET story_points`.
4. **Sprint de la historia** = su fila en `sprint_story_assignments`; si cambia entre la lectura y el bloqueo de la
   historia, se repite la transacción una vez (patrón de HU-13).
5. **Orden de bloqueos** ronda → Sprint → historia en todas las operaciones (tabla en `research.md`, decisión 6).
6. **Traducción de errores** (`pokerOutcomeFor`, se agregan tres filas):

   | Error | Código | Texto |
   |---|---|---|
   | `ErrValueRequired` / `ErrValueNotInSeries` | 422 | ya existen (HU-05a) |
   | `ErrRoundNotFound` | 404 | "La ronda no existe" (ya existe) |
   | `ErrRoundAlreadyOpen` | 409 | "Ya hay una ronda abierta para esta historia" (ya existe) |
   | `ErrNotLatestRound` | 409 | `MsgPokerNotLatestRound` (nuevo) |
   | `ErrRoundStillOpen` | 409 | `MsgPokerRoundStillOpen` (nuevo) |
   | `ErrStoryInFinishedSprint` | 409 | `MsgPokerStoryInFinishedSprint` (nuevo) |
   | cualquier otro | 500 | `MsgOperationFailed` con `logger.Error` (ya existe) |

7. **Lectura**: dos columnas (`latest`, `in_finished_sprint`) en la consulta de rondas cerradas de HU-05b, en la misma
   transacción.
8. **Vista**: `Actions`, `Locked` y `Reminder` en `pokerClosedRoundPage`; recordatorio con el parcial `alert`
   (`FlashWarning`); acciones deshabilitadas con el motivo "Sprint finalizado".
9. **Sin cambios de comportamiento en HU-05a/HU-05b**: "Abrir ronda" sigue rechazando historias con rondas; votar y
   cerrar no cambian; las pruebas de las dos historias corren en cada GREEN.

## Flujo de los POST

`/planning-poker/nueva-ronda`:

1. `RequireContext` (proyecto e integrante; si no, 303 de HU-20b).
2. `pokerFormIDs(r)` → `ronda` (inválido → 0).
3. `Service.OpenNextRound(ctx, ac.ProjectID, round, ac.MemberID)`.
4. Error → `reject` (`pokerOutcomeFor`). Éxito → `done(w, r, pokerNextRoundOpenedMessage(opened.Number,
   opened.StoryTitle))`.

`/planning-poker/estimacion`: igual con `ronda` y `valor`, `Service.AgreeEstimate(…, value)` y
`pokerEstimateRegisteredMessage(agreed.StoryTitle, agreed.StoryPoints)`.

## Contratos que consume

- HU-05a: lo listado en `research.md` ("Datos verificados").
- HU-05b (T001: HU-05b): `poker.Spread`, `poker.VoteSpread`, `Store.CloseRound`, `PokerClosedRoundView`,
  `PokerBoard.ClosedRounds`, `pokerClosedRoundsQuery`, `pokerClosedRoundPage`, la sección "Rondas cerradas",
  `waitForWaiters`, `roundRow`, `closedRow`.
- HU-02: `user_stories.story_points` y su `CHECK`. HU-04: `sprints`, `sprint_story_assignments`. HU-22: `GET /backlog`
  (`RegisterBacklog`, `NewPostgreSQLStoryCatalog`) solo en pruebas.
- HU-20a/HU-20b: `Server.Register`, `Renderer.PageStatus`, `Renderer.Fail`, parcial `alert`, `RequireContext`,
  `ActiveContextFrom`, `SetFlash`, `RedirectSeeOther`, `parseID`, `MsgOperationFailed`, `FlashWarning`.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| HU-05b no está en `main` o cambió sus nombres | T001 confirma cada nombre marcado "(T001: HU-05b)" y detiene todo si HU-05b no está integrada; si un nombre cambió, se ajustan `data-model.md` y `tasks.md` antes de seguir |
| Bloqueo mutuo con el cierre de Sprint (HU-15) | Orden ronda → Sprint → historia en todas las operaciones (`research.md`, decisión 6) y `TestPostgresAgreeEstimate_EstimacionPrimero` |
| La asignación cambia entre la lectura y el bloqueo de la historia | Relectura y una repetición (decisión 4); `TestPostgresAgreeEstimate_CierreDeSprintPrimero`, caso "En progreso" |
| HU-15 o HU-13 entran después con otra regla de Sprint finalizado | La regla es la de sus specs (asignación a un Sprint `'Finalizado'`); T030 les avisa para reutilizar el mismo criterio |
| Las acciones nuevas rompen pruebas de HU-05a/HU-05b que cuentan formularios, radios o alertas | Los radios de la estimación salen sin `checked`; las acciones solo aparecen en rondas cerradas. Una ronda cerrada con divergencia pasa a tener dos avisos `alert-warning` (la alerta de HU-05b y el recordatorio): T001 revisa si alguna prueba de HU-05b cuenta exactamente uno y, si es así, lo avisa antes de empezar (no se cambian sus expectativas). Las pruebas de las dos historias corren en cada GREEN |
| Pruebas de concurrencia inestables | Orden forzado con bloqueos de prueba y `waitForWaiters`, plazo de 5 s; nada de `time.Sleep` como sincronización |
| `main.go` | No se edita: `RegisterPlanningPoker` mantiene su firma |

## Complexity Tracking

Sin violaciones de la constitución que justificar.

## Verificación posterior al diseño

- I–V: PASS (sin cambios respecto de la verificación inicial).
- VI: PASS. Dos métodos nuevos del puerto con adaptador PostgreSQL y pruebas de integración; lectura ampliada con su
  prueba de integración; sin migración.
- Ningún `NEEDS CLARIFICATION` pendiente (`research.md`, "Preguntas abiertas": ninguna).
