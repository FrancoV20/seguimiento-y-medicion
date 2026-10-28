# Investigación: HU-05b – Revelación de Votos y Detección de Divergencia

## Datos verificados en el código (main `75ec8d9`, 28/10/2026)

Verificado con `git grep -n` en `src/poker`, `src/web/poker_*.go`, `src/web/postgres_poker_catalog.go`,
`src/web/templates/poker.html` y `db/migrations/011_create_poker.up.sql`.

- **Go y dependencias**: `go.mod` declara `go 1.27.1` y una sola dependencia directa, `github.com/jackc/pgx/v5`.
- **Dominio de HU-05a (`src/poker`)**:
  - `series.go`: `Series = []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}`, `InSeries(v int) bool`,
    `ParseValue(raw string) (int, error)`.
  - `round.go`: `RoundStatus`, `StatusOpen = "Abierta"`, `StatusClosed = "Cerrada"`, `CheckOpen(existing
    []RoundStatus) error`, `CheckVote(status RoundStatus) error`, `OpenedRound{RoundID, StoryID int64; StoryTitle
    string; Number int}`, `VoteOutcome{RoundID int64; Value int; Updated bool}`.
  - `errors.go`: `ErrValueRequired`, `ErrValueNotInSeries`, `ErrRoundClosed`, `ErrRoundNotFound`, `ErrStoryNotFound`,
    `ErrRoundAlreadyOpen`, `ErrStoryAlreadyRounded`, `ErrMemberRequired` (textos técnicos con prefijo `poker:`).
  - `store.go`: puerto `Store` con `OpenRound(ctx, projectID, storyID, memberID int64) (OpenedRound, error)` y
    `CastVote(ctx, projectID, roundID, memberID int64, value int) (VoteOutcome, error)`.
  - `service.go`: `Service`, `NewService(store Store) *Service`, `OpenRound` y `Vote` (identificador 0 →
    `ErrStoryNotFound`/`ErrRoundNotFound` sin llamar al almacén; `memberID == 0` → `ErrMemberRequired`).
  - `postgres_store.go`: `PostgreSQLStore`, `NewPostgreSQLStore`, `var _ Store = (*PostgreSQLStore)(nil)`;
    `CastVote` toma la ronda con `SELECT status FROM poker_rounds WHERE id = $1 AND project_id = $2 FOR SHARE`.
  - Pruebas: `fakeStore` en `service_test.go` (implementa `Store` con sus dos métodos); `newPokerFixture`,
    `insertClosedRound`, `count` y `cleanup` en `fixtures_integration_test.go`; `waitForLock` (espera a que alguna
    sesión quede en `wait_event_type = 'Lock'`) y `TestPostgresCastVote_EsperaAlCierre` en
    `postgres_store_integration_test.go`.
- **Pantalla de HU-05a (`src/web`)**:
  - `poker_catalog.go`: `PokerStoryOption`, `PokerParticipantView{MemberID int64; Name string; HasVoted, IsMe bool}`,
    `PokerRoundView{ID, StoryID int64; StoryTitle string; Number int; Participants []PokerParticipantView; OwnVote
    *int}`, `PokerBoard{Stories; Rounds}` y `PokerCatalog.PokerBoard(ctx, projectID, memberID int64)`.
  - `postgres_poker_catalog.go`: `PostgreSQLPokerCatalog`, cuatro consultas (`pokerStoriesQuery`,
    `pokerOpenRoundsQuery`, `pokerParticipantsQuery`, `pokerOwnVotesQuery`) en una transacción `ReadOnly` +
    `LevelRepeatableRead`; ayudante `pokerEach`; errores envueltos con `Unavailable`.
  - `poker_form.go`: `PokerFieldStory = "historia"`, `PokerFieldRound = "ronda"`, `PokerFieldValue = "valor"`,
    `pokerFormIDs(r) (story, round int64, value string)` (con `parseID`) y `pokerOutcomeFor(err) (status int, text
    string)`.
  - `poker_handlers.go`: `pathPoker = "/planning-poker"`, interfaz `PokerActions` (`OpenRound`, `Vote`),
    `RegisterPlanningPoker(s, actions, catalog)` con un único `s.Register(Section{...}, Routes{"GET /planning-poker",
    "POST /planning-poker/abrir", "POST /planning-poker/votar"})`, `show`, `open`, `vote`, `done` (`SetFlash` +
    `RedirectSeeOther`), `reject` (`pokerOutcomeFor` + `logger.Error` en 500) y `render` (`PageStatus` o `Fail`);
    tipos `pokerRoundPage` y `pokerPageData`.
  - `poker_messages.go`: `TitlePoker`, `MsgPoker…`, `LabelPokerVoted` "Votó", `LabelPokerNotVoted` "Sin votar" y
    `pokerRoundOpenedMessage(title)`.
  - `templates/poker.html`: sección "Abrir ronda" y, por ronda abierta, `<section class="poker-round"
    data-ronda="{{.ID}}">` con la lista `data-integrante` y el formulario de voto.
  - Pruebas: `failingPokerStore struct{ poker.Store }` (sobrescribe `OpenRound` y `CastVote`), `newPokerApp`,
    `pokerOption{store, catalog}`, `a.as`, `a.get`, `a.post`, `participantRow` en `poker_handlers_test.go`;
    `failingPokerCatalog` en `poker_view_test.go`; `newPokerWebFixture` con `round(t, project, story, open)` y
    `vote` en `postgres_poker_catalog_integration_test.go`.
- **Base (`011_create_poker.up.sql`)**: `poker_rounds.status` con `CHECK (status IN ('Abierta', 'Cerrada'))`,
  `closed_at TIMESTAMPTZ` y `chk_poker_rounds_closed_at CHECK ((status = 'Abierta') = (closed_at IS NULL))`;
  `poker_votes (round_id, member_id)` como clave primaria; `poker_votes.member_id` sin cascada a `members`;
  `project_members.member_id` con `ON DELETE CASCADE`. Ningún índice sobre `closed_at`.
- **Contratos web** (HU-20a/b): `Server.Register` falla si la sección (`Path`) ya existe (`registry.Add`), así que la
  ruta nueva va en el mismo `Routes`. `RequireContext` responde 303 a `/contexto/integrante?volver_a=%2Fplanning-poker`
  sin integrante. `Flash{Kind, Text}` con `Class()` ("alert alert-warning"), `Label()` ("Advertencia") y `Role()`
  ("status"); el parcial `alert` está en `templates/layout.html` y se parsea junto con cada página
  (`render.go`, `ParseFS(layout.html, página)`), así que `poker.html` puede usarlo.
- **Nombres libres**: no existen `CloseRound`, `ClosedRound`, `CheckClose`, `Spread`, `VoteSpread`, `ErrNoVotes`,
  `ClosedRounds`, `PokerClosedRoundView`, `PokerRevealedVoteView` ni la ruta `/planning-poker/cerrar`.
- **Registro en `main.go`**: `web.RegisterPlanningPoker(srv, poker.NewService(poker.NewPostgreSQLStore(database)),
  web.NewPostgreSQLPokerCatalog(database))`; la firma no cambia, así que `main.go` no se edita.
- **CI** (`.github/workflows/ci.yml`): `gofmt -l .` sin salida, `go vet ./...`, migraciones y `go test ./... -p 1 -race
  -coverprofile=coverage.out`.
- **Pruebas de HU-05a con rondas cerradas sin votos**: `TestHU05a_HistoriaYaTuvoRonda`, `TestBDD_HU05a_RondaCerrada` y
  `TestPostgresPokerCatalog_RondasAbiertas` preparan rondas `'Cerrada'` directamente en la base, algunas sin votos.
  Con HU-05b esas rondas aparecen en "Rondas cerradas": la pantalla no puede fallar con ellas (decisión 9).

## Decisiones

1. **Regla de divergencia en `src/poker/spread.go`.**
   - Decisión: `Spread(values []int) (VoteSpread, error)` con `VoteSpread{Low, High int; Divergent bool}`. El nivel
     es la posición en `Series` más 1; `Divergent` = `level(High) − level(Low) > 2`. Sin valores → `ErrNoVotes`; un
     valor fuera de la serie → `ErrValueNotInSeries`. No modifica el slice recibido.
   - Motivo: función pura, sin base, reutilizable por HU-18 (contrato en `data-model.md`). Reusa la serie de HU-05a y
     sus errores en lugar de duplicarlos.
   - Alternativas descartadas: devolver tres valores sueltos (`low, high int, divergent bool`), que obliga a HU-18 a
     repetir la validación de vacío; calcular la divergencia en SQL (mezcla la regla con la lectura y no se puede
     probar sin base); guardar el resultado al cerrar (exige migración y se desactualizaría si cambia la regla).

2. **Cierre con regla pura `CheckClose` y transacción en el adaptador.**
   - Decisión: `CheckClose(status RoundStatus, votes int) error`: ronda no abierta → `ErrRoundClosed`; abierta sin
     votos → `ErrNoVotes`. `PostgreSQLStore.CloseRound(ctx, projectID, roundID int64) (ClosedRound, error)`:
     1. `SELECT r.status, r.story_id, r.round_number, us.title … WHERE r.id = $1 AND r.project_id = $2 FOR UPDATE OF
        r` (sin fila → `ErrRoundNotFound`);
     2. `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`;
     3. `CheckClose(status, count)`;
     4. `UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`.
   - Motivo: es la transacción acordada (SELECT del proyecto con bloqueo exclusivo, cerrada → `ErrRoundClosed`,
     contar votos, una sola actualización). Se agregan al SELECT la historia y el número para armar "Ronda cerrada para
     la historia «título»" sin otra consulta; `FOR UPDATE OF r` bloquea solo la ronda, no la historia (no frena
     cargas de esfuerzo ni asignaciones a Sprint sobre esa historia). El orden 404 → 409 sale solo: sin fila no se
     llega a `CheckClose`.
   - El COUNT se hace también si la ronda está cerrada (una consulta barata dentro de la misma transacción); la
     precedencia "ya cerrada" antes que "sin votos" la decide `CheckClose` y la prueba `TestCheckClose`.
   - Alternativas descartadas: `UPDATE … WHERE status = 'Abierta' AND EXISTS (votos)` en una sola sentencia (no
     distingue los tres rechazos sin consultas extra); `SELECT … FOR UPDATE` sin `OF r` (bloquearía también la fila de
     `user_stories`).

3. **El puerto no recibe el integrante.**
   - Decisión: `Service.CloseRound(ctx, projectID, roundID, memberID int64) (ClosedRound, error)` valida `roundID ==
     0` → `ErrRoundNotFound` y `memberID == 0` → `ErrMemberRequired` sin llamar al almacén, y llama a
     `Store.CloseRound(ctx, projectID, roundID)`.
   - Motivo: no hay columna para quien cierra (sin migración); pasar un parámetro que el adaptador ignora confunde.
     El servicio conserva la defensa de HU-05a. Cuando HU-28 agregue roles, el servicio es el lugar para la
     comprobación de permisos.
   - Alternativa descartada: `Store.CloseRound(ctx, projectID, roundID, memberID)` por simetría con `OpenRound`.

4. **Concurrencia con los votos.**
   - `CastVote` toma la ronda `FOR SHARE` y `CloseRound` `FOR UPDATE`: se excluyen. Si el cierre toma la ronda
     primero, el voto espera; al confirmar el cierre, PostgreSQL (READ COMMITTED) vuelve a leer la fila, ve `'Cerrada'`
     y `CheckVote` devuelve `ErrRoundClosed` sin escribir. Si el voto toma la ronda primero, el cierre espera y su
     `COUNT` (una sentencia nueva, con una foto nueva) incluye el voto confirmado.
   - Dos cierres simultáneos: el segundo espera, relee `'Cerrada'` y recibe `ErrRoundClosed`; `closed_at` queda con la
     fecha del primero.
   - **Prueba determinista del voto que espera al cierre** (`TestPostgresCloseRound_VotoEsperaAlCierre`): para que el
     cierre real (no una transacción de prueba) tenga la ronda bloqueada mientras llega el voto, una transacción de
     prueba toma `LOCK TABLE poker_votes IN ACCESS EXCLUSIVE MODE`; `CloseRound` (en una goroutine) toma la ronda `FOR
     UPDATE` y queda esperando en el `COUNT`; recién entonces `CastVote` de Caro con 8 (otra goroutine) queda esperando
     la ronda. Con dos sesiones en espera (`waitForWaiters(t, f, 2)`, generalización local de `waitForLock`), la
     transacción de prueba hace `ROLLBACK`: el cierre termina bien y el voto recibe `ErrRoundClosed`; el voto de Caro
     sigue en 5. Con `-p 1` el bloqueo de tabla no afecta a otros paquetes.
   - **Falla real al guardar** (`TestPostgresCloseRound_FalloNoDejaCambios`): una transacción de prueba toma la ronda
     `FOR SHARE` (como un voto en curso) y `CloseRound` corre con un contexto de 200 ms: queda esperando, el contexto
     vence y devuelve un error que no es del dominio; al soltar el bloqueo la ronda sigue `'Abierta'` con `closed_at`
     NULL.

5. **Lectura de las rondas cerradas en el catálogo de HU-05a.**
   - Decisión: `PokerBoard` suma `ClosedRounds []PokerClosedRoundView`, leído en la misma transacción `READ ONLY
     REPEATABLE READ` con dos consultas nuevas (rondas cerradas y votos revelados). Ambas filtran por `project_id` y
     `status = 'Cerrada'`; la de votos lee `poker_votes.value` solo a través de un CTE `closed` con ese filtro.
   - Motivo: misma pantalla, misma foto de la base; el catálogo de lectura es el patrón de HU-05a (decisión 3 de su
     research).
   - Alternativa descartada: un método `ClosedRounds` aparte en `PokerCatalog` (dos transacciones: la pantalla podría
     mostrar una ronda a la vez como abierta y cerrada).

6. **Revelación por construcción.**
   - Los tipos de rondas abiertas de HU-05a no cambian (`PokerParticipantView` sigue sin campo de valor). El valor
     revelado vive solo en `PokerRevealedVoteView.Value` (`*int`, nil = "No votó"), que solo se llena desde la consulta
     de rondas cerradas.
   - Pruebas: `TestPostgresPokerCatalog_SoloRondasCerradas` (ninguna ronda abierta en `ClosedRounds` y ningún valor de
     una ronda abierta en sus votos) y `TestHU05b_VotoAbiertoNuncaSeRevela` (con «Login» cerrada y «Pagos» abierta, la
     fila de Ana en «Pagos» dice "Votó" sin valor en la página de Beto, y esa página es idéntica byte a byte cuando Ana
     cambia su voto de «Pagos» de 3 a 21, un valor que no está en ninguna ronda cerrada).

7. **Lista revelada: integrantes actuales más quien votó** (clarificación del 28/10).
   - Decisión: la consulta une los integrantes actuales del proyecto (`project_members`) con quienes tienen un voto en
     la ronda (`UNION`), ordenados por `display_name, id`, con `LEFT JOIN poker_votes` para el valor.
   - Motivo: lo que se ve es exactamente lo que se calcula (el rango usa todos los votos); una ronda cerrada por la
     aplicación nunca queda sin valores visibles. `poker_votes.member_id` no tiene cascada, así que el integrante
     sigue existiendo en `members` aunque lo quiten del proyecto.

8. **Divergencia calculada en la web al armar la vista.**
   - Decisión: `render` arma, por ronda cerrada, un `pokerClosedRoundPage` con los valores emitidos, llama a
     `poker.Spread` y elige el texto: con divergencia, `Alert = &Flash{Kind: FlashWarning, Text:
     pokerDivergenceMessage(low, high)}` (la plantilla usa `{{template "alert" .Alert}}`: "Advertencia:" + texto, clase
     `alert alert-warning`, `role="status"`); sin divergencia, `Range = pokerNoDivergenceMessage(low, high)` en un
     párrafo.
   - Motivo: el catálogo solo lee; la regla está en el dominio; el formato de alerta es el de HU-20b sin código nuevo
     de presentación. `FlashWarning` (no `FlashError`): no es un error de la operación, es un aviso para la discusión.

9. **Ronda cerrada sin votos (solo con datos cargados fuera de la aplicación).**
   - Decisión: si `Spread` devuelve `ErrNoVotes`, la ronda se muestra con todos en "No votó", sin rango ni alerta, y la
     pantalla no falla. Un valor fuera de la serie no puede existir (CHECK de la base); si `Spread` devolviera otro
     error, se trata igual (sin rango) para no romper la pantalla por una ronda.
   - Motivo: tres pruebas de HU-05a preparan rondas cerradas sin votos y deben seguir en verde.

10. **Orden de las rondas cerradas.** `ORDER BY r.closed_at DESC, r.id DESC`. Sin índice nuevo: con decenas de rondas
    por proyecto alcanza `idx_poker_rounds_project_status` (SC-006 lo mide).

11. **Ruta en el mismo `Routes`.** `"POST " + pathPoker + "/cerrar": http.HandlerFunc(h.close)` dentro de
    `RegisterPlanningPoker`; un segundo `Register` con el mismo `Path` falla.

12. **Traducción de errores.** `pokerOutcomeFor` suma `ErrNoVotes` → 409 `MsgPokerNoVotes`. `ErrRoundNotFound` (404),
    `ErrRoundClosed` (409) y el resto (500 `MsgOperationFailed` con `logger.Error`) ya están. El flujo de rechazo
    (`reject` → `render` con `PageStatus`) se reutiliza sin cambios.

13. **Textos** en `poker_messages.go`: `MsgPokerNoVotes` "La ronda no tiene votos", `MsgPokerNoClosedRounds` "Aún no
    hay rondas cerradas", `LabelPokerDidNotVote` "No votó", `pokerRoundClosedMessage(title)` → "Ronda cerrada para la
    historia «title»", `pokerDivergenceMessage(low, high)` → "Divergencia: los votos van de X a Y. Debatan antes de
    acordar la estimación" y `pokerNoDivergenceMessage(low, high)` → "Sin divergencia: los votos van de X a Y". Los
    encabezados y botones ("Rondas cerradas", "Cerrar ronda") van en la plantilla, como "Abrir ronda" y "Votar" en
    HU-05a.

14. **Pruebas falsas en el mismo commit que el puerto.** Al agregar `CloseRound` a `poker.Store`: `fakeStore`
    (`service_test.go`) suma `CloseRound`; `failingPokerStore` (`poker_handlers_test.go`) lo sobrescribe para fallar
    (si no, el embebido llamaría al almacén real). `failingPokerCatalog` no cambia (devuelve `PokerBoard{}`).

15. **Datos de prueba del Issue.** Cada paquete (`src/poker`, `src/web`, `test`) arma su proyecto «Alfa» con Ana,
    Beto y Caro (nombres exactos: la base no exige nombres únicos y todo se filtra por identificador), las historias
    «Login», «Pagos», «Reportes», «Perfil» y «Ayuda» con sus rondas 1 abiertas y los votos del Issue, y un proyecto
    «Beta» con su integrante, una ronda abierta con votos y una cerrada. Las rondas que se preparan cerradas llevan
    `closed_at` explícito para que el orden sea determinista. Limpieza: historias → proyectos → integrantes (sin
    cascada desde `poker_votes`).

## Preguntas abiertas

Ninguna. La única duda (voto de un integrante quitado del proyecto) se respondió el 28/10 (ver "Clarifications" en
`spec.md`).
