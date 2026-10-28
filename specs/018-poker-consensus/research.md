# Investigación: HU-18 – Planning Poker: Nueva Ronda y Estimación Acordada

## Datos verificados (main `d12067f`, 28/10/2026)

### HU-05a (en el código; `git grep -n`)

- `src/poker/series.go`: `Series`, `InSeries`, `ParseValue(raw string) (int, error)` (vacío → `ErrValueRequired`; no
  canónico o fuera de la serie → `ErrValueNotInSeries`).
- `src/poker/round.go`: `RoundStatus`, `StatusOpen = "Abierta"`, `StatusClosed = "Cerrada"`,
  `CheckOpen(existing []RoundStatus) error`, `CheckVote(status) error`, `OpenedRound{RoundID, StoryID int64; StoryTitle
  string; Number int}`, `VoteOutcome`.
- `src/poker/errors.go`: `ErrValueRequired`, `ErrValueNotInSeries`, `ErrRoundClosed`, `ErrRoundNotFound`,
  `ErrStoryNotFound`, `ErrRoundAlreadyOpen`, `ErrStoryAlreadyRounded`, `ErrMemberRequired`.
- `src/poker/store.go`: `Store` con `OpenRound` y `CastVote`. `src/poker/service.go`: `Service`, `NewService`,
  `OpenRound`, `Vote`.
- `src/poker/postgres_store.go`: `OpenRound` bloquea la historia con `SELECT title FROM user_stories WHERE id = $1 AND
  project_id = $2 FOR NO KEY UPDATE`, lee las rondas con `roundStatuses(ctx, tx, storyID)` y traduce el `23505` de
  `uq_poker_rounds_one_open_per_story` a `ErrRoundAlreadyOpen` y el de `uq_poker_rounds_story_number` a
  `ErrStoryAlreadyRounded`. `CastVote` toma la ronda `FOR SHARE`.
- `src/web/poker_form.go`: `PokerFieldRound = "ronda"`, `PokerFieldValue = "valor"`, `pokerFormIDs(r)`,
  `pokerOutcomeFor(err)`.
- `src/web/poker_handlers.go`: `pathPoker`, `PokerActions` (`OpenRound`, `Vote`), `RegisterPlanningPoker(s, actions,
  catalog)` con un solo `Routes`, `done`, `reject`, `render`.
- `src/web/poker_messages.go`: `MsgPokerSelectValue`, `MsgPokerValueNotInSeries`, `MsgPokerRoundNotFound`,
  `MsgPokerRoundAlreadyOpen`, `MsgPokerStoryAlreadyRounded`, `pokerRoundOpenedMessage(title)` ("Ronda abierta para la
  historia «…»", sin número: no cambia).
- Pruebas: `fakeStore` (`src/poker/service_test.go`), `failingPokerStore` y `errPokerStoreInternal`
  (`src/web/poker_handlers_test.go`), `newPokerApp`, `pokerOption`, `a.as`, `a.get`, `a.post`.
- `db/migrations/011_create_poker.up.sql`: `uq_poker_rounds_story_number UNIQUE (story_id, round_number)`, índice
  único parcial `uq_poker_rounds_one_open_per_story`, `round_number INTEGER NOT NULL CHECK (round_number > 0)`.

### HU-05b (todavía no está en el código; nombres de `specs/005b-poker-reveal/data-model.md`)

Cada uno se confirma en T001; si HU-05b no está en `main`, HU-18 no empieza.

- `poker.Spread(values []int) (VoteSpread, error)`, `poker.VoteSpread{Low, High int; Divergent bool}`,
  `poker.ErrNoVotes` (T001: HU-05b).
- `poker.CheckClose`, `poker.ClosedRound`, `Store.CloseRound(ctx, projectID, roundID)`, `Service.CloseRound(ctx,
  projectID, roundID, memberID)` (T001: HU-05b).
- `web.PokerClosedRoundView{ID, StoryID int64; StoryTitle string; Number int; Votes []PokerRevealedVoteView}`,
  `web.PokerRevealedVoteView`, `PokerBoard.ClosedRounds`, `pokerClosedRoundsQuery`, `pokerRevealedVotesQuery`
  (T001: HU-05b).
- `pokerClosedRoundPage{PokerClosedRoundView; Alert *Flash; Range string}` en `render` y la sección "Rondas cerradas"
  de `poker.html` (al final de la página, `data-ronda-cerrada`, `data-integrante`) (T001: HU-05b).
- `PokerActions.CloseRound`, ruta `POST /planning-poker/cerrar`, `MsgPokerNoVotes`, `pokerDivergenceMessage`
  (T001: HU-05b).
- Pruebas: `newAlfaFixture` y `waitForWaiters(t, f, n)` en `src/poker/reveal_fixtures_integration_test.go`;
  `newAlfaWebFixture`, `closeAt`, `roundRow`, `closedRow` en `src/web/poker_reveal_fixtures_test.go` (T001: HU-05b).

### Esquema de Sprints (`004_create_sprints.up.sql`)

- `sprints(id BIGSERIAL, project_id BIGINT, number, goal TEXT NOT NULL CHECK (btrim(goal) <> ''), start_date,
  end_date CHECK (end_date > start_date), status VARCHAR(20) CHECK (status IN ('Activo', 'Finalizado')))`,
  `UNIQUE (project_id, number)`.
- `sprint_story_assignments(sprint_id, story_id)`, `PRIMARY KEY (sprint_id, story_id)` y **`UNIQUE (story_id)`**: una
  historia está asignada a lo sumo a un Sprint.
- `user_stories.active_sprint_id BIGINT REFERENCES sprints(id) ON DELETE SET NULL`.
- **Cómo se sabe el Sprint de una historia**: por su fila en `sprint_story_assignments`. Es la fuente que usan
  HU-13 y HU-17 (`specs/013-backlog-edit/data-model.md`, `specs/017-story-estimates/data-model.md`). No se usa
  `active_sprint_id`: HU-04 lo llena al asignar, pero el cierre de HU-15 conserva la asignación de las historias
  completadas y borra la de las no completadas (que vuelven a "Pendiente" con `active_sprint_id = NULL`); la asignación
  es la que dice, después del cierre, a qué Sprint finalizado pertenece una historia.
- HU-13, HU-15 y HU-17 no están en el código de `main` (`git grep` sin `ErrStoryInFinishedSprint`, `CloseSprint` ni
  `sprint_story_results`): HU-18 implementa la regla por su cuenta, con el mismo criterio y el mismo orden de bloqueos.
- Orden de bloqueos acordado (HU-13, HU-15): **Sprint antes que historia**. HU-15 toma el Sprint `FOR UPDATE` y sus
  historias `FOR UPDATE OF us`; HU-13 toma el Sprint `FOR SHARE` y la historia `FOR NO KEY UPDATE OF us`, con relectura
  de la asignación.
- HU-04 (`src/sprint/postgres_sprint_store.go`) asigna con `UPDATE user_stories SET status = $1, active_sprint_id = $2
  …` en la misma transacción que la asignación: una historia bloqueada `FOR NO KEY UPDATE` no puede cambiar de
  asignación hasta el `COMMIT`.

### Backlog (para "el Backlog muestra «Login» con 8 Story Points")

- `web.RegisterBacklog(s, creator *backlog.BacklogService, catalog StoryCatalog)`, `backlog.NewBacklogService(
  backlog.NewPostgreSQLBacklogStore(db))`, `web.NewPostgreSQLStoryCatalog(db)`; `GET /backlog` lista `story_points` en
  la tabla (`templates/backlog.html`). No se edita.

## Decisiones

1. **Reglas puras junto a `CheckOpen`** (`src/poker/round.go`).
   - `StoryRound{Number int; Status RoundStatus}`.
   - `CheckNextRound(from int, rounds []StoryRound, inFinishedSprint bool) (next int, err error)`: alguna ronda
     abierta → `ErrRoundAlreadyOpen`; `from` no es el mayor número → `ErrNotLatestRound`; Sprint finalizado →
     `ErrStoryInFinishedSprint`; si no, `next = mayor + 1`.
   - `CheckAgree(status RoundStatus, number, latest int, inFinishedSprint bool) error`: abierta → `ErrRoundStillOpen`;
     `number != latest` → `ErrNotLatestRound`; Sprint finalizado → `ErrStoryInFinishedSprint`.
   - Motivo: el orden de los rechazos (escenario 8: "hay una abierta" gana sobre "no es la última"; R5 y R6b de la
     spec) queda en un solo lugar y se prueba sin base. El adaptador junta los datos dentro de la transacción y llama a
     la regla una vez.
   - Alternativa descartada: comprobaciones sueltas en el adaptador (el orden de precedencia solo se probaría contra
     PostgreSQL).

2. **Errores nuevos** en `src/poker/errors.go`: `ErrNotLatestRound`, `ErrRoundStillOpen`, `ErrStoryInFinishedSprint`.
   `ErrRoundAlreadyOpen`, `ErrRoundNotFound`, `ErrValueRequired`, `ErrValueNotInSeries` y `ErrMemberRequired` se
   reutilizan.

3. **Transacción de "Nueva ronda"** (`PostgreSQLStore.OpenNextRound`): ronda base del proyecto `FOR SHARE` →
   asignación de la historia → Sprint `FOR SHARE` (si tiene) → historia `FOR NO KEY UPDATE OF us` con relectura de la
   asignación → rondas de la historia → `CheckNextRound` → `INSERT` de la ronda `next`. Detalle en `data-model.md`.
   - La ronda base se toma `FOR SHARE` (decisión de este plan; el equipo solo pidió buscarla): así una "Nueva ronda" y
     una estimación (`FOR UPDATE`) desde la misma ronda se serializan, y la estimación ve la ronda nueva y responde
     "Hay una ronda más reciente". Dos "Nueva ronda" desde la misma ronda no se frenan en la ronda (`FOR SHARE` es
     compatible consigo mismo) sino en la historia: la segunda lee las rondas después del `COMMIT` de la primera y
     recibe `ErrRoundAlreadyOpen`.
   - El Sprint se lee porque la clarificación del 28/10 rechaza la nueva ronda de una historia de un Sprint finalizado
     (comprobado dentro de la transacción, Sprint antes que historia).
   - `23505` sobre `uq_poker_rounds_one_open_per_story` → `ErrRoundAlreadyOpen` (como HU-05a). `23505` sobre
     `uq_poker_rounds_story_number` (otra transacción creó el mismo número; no debería pasar con la historia
     bloqueada) → `ErrNotLatestRound`.
   - La ronda nueva copia el `project_id` de la ronda base y `opened_by` es el integrante identificado.

4. **Transacción de la estimación** (`PostgreSQLStore.AgreeEstimate`): ronda del proyecto `FOR UPDATE` → número mayor
   de la historia → asignación → Sprint `FOR SHARE` (si tiene) → historia `FOR NO KEY UPDATE OF us` con relectura de la
   asignación → `CheckAgree` → `UPDATE user_stories SET story_points = $2 WHERE id = $1 RETURNING title`.
   - Motivo: es la transacción acordada (ronda, Sprint, historia, comprobaciones, UPDATE). La ronda `FOR UPDATE`
     también se serializa con el cierre de HU-05b (`FOR UPDATE OF r`) y con los votos (`FOR SHARE`).
   - El valor se valida antes, en el servicio (`ParseValue`): 422 antes que 404/409, como el voto.
   - Un valor igual al actual es un éxito (el UPDATE se ejecuta igual).

5. **Relectura de la asignación** (patrón de HU-13). La asignación se lee sin bloquear para saber qué Sprint bloquear;
   después de bloquear la historia se vuelve a leer. Si cambió (HU-04 la asignó o HU-15 la devolvió al Backlog entre
   las dos lecturas), se revierte y se repite la transacción desde el principio **una sola vez**; si vuelve a cambiar,
   error interno (500). Con la historia bloqueada la asignación ya no cambia (HU-04 actualiza la historia en la misma
   transacción y HU-15 la bloquea `FOR UPDATE`).

6. **Orden de bloqueos sin ciclos.**

   | Operación | Orden |
   |---|---|
   | Nueva ronda (HU-18) | ronda base `FOR SHARE` → Sprint `FOR SHARE` → historia `FOR NO KEY UPDATE` |
   | Estimación (HU-18) | ronda `FOR UPDATE` → Sprint `FOR SHARE` → historia `FOR NO KEY UPDATE` |
   | Abrir ronda 1 (HU-05a) | historia `FOR NO KEY UPDATE` |
   | Votar (HU-05a) | ronda `FOR SHARE` |
   | Cerrar ronda (HU-05b) | ronda `FOR UPDATE OF r` |
   | Cierre de Sprint (HU-15) | Sprint `FOR UPDATE` → historias `FOR UPDATE OF us` |
   | Edición del Backlog (HU-13) | Sprint `FOR SHARE` → historia `FOR NO KEY UPDATE` |

   Todas respetan ronda → Sprint → historia; ninguna toma una ronda después de la historia o del Sprint.

7. **Pruebas de concurrencia deterministas** (con `waitForWaiters` de HU-05b, T001: HU-05b):
   - `TestPostgresOpenNextRound_Concurrente`: dos `OpenNextRound` desde la ronda 1 de «Login» a la vez (barrera) → uno
     devuelve la ronda 2 y el otro `ErrRoundAlreadyOpen`; «Login» queda con dos rondas.
   - Ayudante de prueba `closeSprintLikeHU15(tx, sprintID)`: los pasos de la transacción de HU-15 que tocan las tablas
     existentes (Sprint `FOR UPDATE`; historias `FOR UPDATE OF us`; borra la asignación de las no "Hecho" y las pasa a
     "Pendiente" con `active_sprint_id = NULL`; `UPDATE sprints SET status = 'Finalizado'`). No escribe
     `sprint_story_results`, que todavía no existe.
   - `TestPostgresAgreeEstimate_CierreDeSprintPrimero`: «Pagos» asignada a un Sprint 2 "Activo" (solo en esta prueba).
     Una transacción de prueba corre `closeSprintLikeHU15` sin confirmar; `AgreeEstimate(5)` en una goroutine queda
     esperando el Sprint (`waitForWaiters(1)`); `COMMIT`. Dos casos:
     - «Pagos» en "Hecho": sigue asignada al Sprint, ahora finalizado → `ErrStoryInFinishedSprint` y sigue sin Story
       Points;
     - «Pagos» en "En progreso": vuelve al Backlog; la estimación ve que la asignación cambió, repite la transacción y
       termina bien con 5 (decisión 5).
   - `TestPostgresAgreeEstimate_EstimacionPrimero` («Pagos» en "Hecho" en el Sprint 2 "Activo"): una transacción de
     prueba toma «Pagos» `FOR SHARE`;
     `AgreeEstimate(5)` toma la ronda y el Sprint y queda esperando la historia (`waitForWaiters(1)`); el cierre de
     Sprint (ayudante de prueba con los pasos de HU-15) queda esperando el Sprint (`waitForWaiters(2)`); `ROLLBACK` de
     la prueba → la estimación termina bien, después el cierre; ninguno recibe `40P01` (deadlock) y «Pagos» queda con 5
     en un Sprint finalizado.
   - `TestPostgresNextRoundYEstimacion_Serializadas`: una transacción de prueba toma `LOCK TABLE poker_rounds IN SHARE
     ROW EXCLUSIVE MODE` (frena el `INSERT` pero no los `SELECT … FOR SHARE/UPDATE`); `OpenNextRound` desde la ronda 1
     de «Pagos» queda esperando en el `INSERT` con la ronda tomada `FOR SHARE`; `AgreeEstimate` desde la misma ronda
     queda esperando la ronda (`waitForWaiters(2)`); `ROLLBACK` de la prueba → la ronda 2 se abre y la estimación
     recibe `ErrNotLatestRound`.
   - Falla al guardar: `OpenNextRound` con un integrante que no existe en `members` (falla la clave foránea) y
     `AgreeEstimate` con un contexto de 200 ms mientras una transacción de prueba tiene la historia `FOR SHARE` → errores
     que no son del dominio y nada cambia.

8. **Lectura: última ronda y Sprint finalizado en la consulta de rondas cerradas.** `pokerClosedRoundsQuery`
   (T001: HU-05b) suma dos columnas calculadas en la misma lectura: `latest` (`NOT EXISTS` una ronda de la historia con
   número mayor, abierta o cerrada) y `in_finished_sprint` (`EXISTS` asignación a un Sprint `'Finalizado'`).
   `PokerClosedRoundView` (T001: HU-05b) suma `Latest` e `InFinishedSprint`. Sin consultas nuevas ni transacciones
   nuevas.

9. **Vista.** `pokerClosedRoundPage` (T001: HU-05b) suma `Actions` (= `Latest`), `Locked` (= `InFinishedSprint`) y
   `Reminder *Flash` (`FlashWarning` con `MsgPokerDivergenceReminder` cuando la ronda es la última y `poker.Spread`
   dice `Divergent`). La plantilla muestra, solo si `Actions`:
   - un formulario `POST /planning-poker/nueva-ronda` con `ronda` oculto y el botón "Nueva ronda";
   - un formulario `POST /planning-poker/estimacion` con `ronda` oculto, el recordatorio (parcial `alert`), los diez
     radios `valor` de `poker.Series` sin ninguno marcado y el botón "Registrar estimación";
   - si `Locked`, los botones y los radios con `disabled` y el motivo "Sprint finalizado" junto a las acciones.

10. **Textos** en `poker_messages.go`: `MsgPokerNotLatestRound`, `MsgPokerRoundStillOpen`, `MsgPokerStoryInFinishedSprint`,
    `MsgPokerDivergenceReminder`, `LabelPokerFinishedSprint` "Sprint finalizado", `pokerNextRoundOpenedMessage(number,
    title)` y `pokerEstimateRegisteredMessage(title, points)`. "Abrir ronda" de HU-05a conserva su mensaje sin número.

11. **Rutas en el mismo `Routes`.** `"POST " + pathPoker + "/nueva-ronda"` y `"POST " + pathPoker + "/estimacion"` en
    `RegisterPlanningPoker`. `PokerActions` suma `OpenNextRound` y `AgreeEstimate`.

12. **Traducción de errores.** `pokerOutcomeFor` suma `ErrNotLatestRound`, `ErrRoundStillOpen` y
    `ErrStoryInFinishedSprint` → 409. El resto ya está.

13. **Pruebas falsas en el mismo commit que el puerto.** `fakeStore` suma `OpenNextRound` y `AgreeEstimate`;
    `failingPokerStore` los sobrescribe para fallar con `errPokerStoreInternal`; si HU-05b agregó otras pruebas falsas
    del puerto, se completan igual (T001: HU-05b).

14. **Datos de prueba del Issue** (`data-model.md`, "Datos de prueba"): fixtures nuevas (`newConsensusFixture` en
    `src/poker`, `newConsensusWebFixture` en `src/web`, su propio mundo en `test/`), porque las de HU-05b arman rondas
    abiertas y HU-18 necesita rondas cerradas, una ronda 2 y un Sprint finalizado; no se tocan las de HU-05b. Las
    rondas cerradas se cargan directo en la base con `closed_at` explícito. El Sprint 1 de «Alta» se carga como
    `'Finalizado'` con su asignación y «Alta» en "Hecho" (como lo deja HU-15).

## Preguntas abiertas

Ninguna. La única duda (historias de un Sprint finalizado) se respondió el 28/10 (ver "Clarifications" en `spec.md`).
