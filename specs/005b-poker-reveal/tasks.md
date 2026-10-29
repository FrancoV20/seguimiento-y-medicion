# Tareas: HU-05b – Revelación de Votos y Detección de Divergencia

**Entrada**: `specs/005b-poker-reveal/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-05a, HU-20a y HU-20b integradas en `main` (verificado el 28/10/2026, `75ec8d9`).

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 12).
- Cada bloque sigue RED → evidencia → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se
  guarda en `specs/005b-poker-reveal/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`. Pruebas siempre con `go test -count=1 -p 1`.
- CI (`.github/workflows/ci.yml`): `gofmt -l .` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` solo sobre los archivos que se crean; los de HU-05a se editan a mano, sin cambiar su fin de línea.
- Datos de prueba: los del Issue (`data-model.md`, "Datos de prueba"), con nombres exactos; cada prueba arma su propio
  proyecto «Alfa» y lo borra al final (historias → proyectos → integrantes).
- Cada GREEN corre también las pruebas de HU-05a: abrir y votar no cambian.

## Fase 1: Preparación

- [x] T001 Confirmar con `git grep -n` en `src/poker`, `src/web/poker_*.go`, `src/web/postgres_poker_catalog.go`,
  `src/web/templates/poker.html` y `db/migrations/011_create_poker.up.sql` los nombres de `research.md` ("Datos
  verificados"): `Series`, `InSeries`, `RoundStatus`, `StatusOpen`/`StatusClosed`, `CheckVote`, `OpenedRound`, los ocho
  `Err…`, `Store`, `Service`, `PostgreSQLStore`, `fakeStore`, `waitForLock`, `PokerBoard`, `PokerCatalog`,
  `pokerEach`, `PokerFieldRound`, `pokerFormIDs`, `pokerOutcomeFor`, `PokerActions`, `RegisterPlanningPoker`,
  `pathPoker`, `done`, `reject`, `render`, `pokerPageData`, `failingPokerStore`, `failingPokerCatalog`,
  `newPokerApp`, `participantRow`, `newPokerWebFixture`, y el parcial `alert` de `templates/layout.html`. Confirmar que
  no existen `CloseRound`, `CheckClose`, `Spread`, `VoteSpread`, `ErrNoVotes`, `ClosedRounds` ni
  `/planning-poker/cerrar`. Si algo cambió, ajustar `data-model.md` y este archivo antes de seguir.
- [x] T002 Crear la rama `HU-05b` desde `main` actualizado, solo si no hay cambios sin commitear:
  `git switch main; git pull; git switch -c HU-05b`. Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y
  `go vet ./...` en verde.

## Fase 2: Bloque Divergencia (regla pura)

**Objetivo**: la regla de divergencia como función pura del dominio, lista para HU-18.

**Prueba independiente**: `go test -count=1 ./src/poker/ -run Spread` pasa sin PostgreSQL.

### Pruebas (RED)

- [x] T003 [P] [E2, E3, E4, E5] Escribir `src/poker/spread_test.go`:
  - `TestSpread`: prueba de tabla con los once casos de `data-model.md` (2-13-5 → 2, 13, true; 3-5-8 → 3, 8, false;
    3-13-3 → 3, 13, true; 5-5 → 5, 5, false; 8 → 8, 8, false; 1-3 false; 1-5 true; 55-89 false; 1-89 true);
  - `TestSpread_Errores`: sin valores (nil y vacío) → `errors.Is(err, ErrNoVotes)`; 4 y {2, 0} →
    `ErrValueNotInSeries`;
  - `TestSpread_NoModificaLosValores`: el slice {13, 2, 5} sigue igual después de llamar a `Spread`.
- [x] T004 Ejecutar T003, confirmar que falla porque `Spread`, `VoteSpread` y `ErrNoVotes` no existen y guardar la
  salida en `evidencia-red.txt`, sección "Divergencia".

### Implementación (GREEN)

- [x] T005 Agregar `ErrNoVotes = errors.New("poker: la ronda no tiene votos")` en `src/poker/errors.go` y crear
  `src/poker/spread.go` con `VoteSpread{Low, High int; Divergent bool}`, `level(v)` (posición en `Series` + 1),
  `maxLevelGap = 2` y `Spread(values []int) (VoteSpread, error)` según `data-model.md` (sin ordenar ni modificar
  `values`).
- [x] T006 Ejecutar T003 en verde junto con las pruebas existentes de `./src/poker/...`.

**Punto de control**: HU-18 ya puede programar contra `poker.Spread`.

## Fase 3: Bloque Cierre (dominio y almacenamiento)

**Objetivo**: cerrar una ronda en el dominio y en PostgreSQL, con sus rechazos y su concurrencia con los votos.

**Prueba independiente**: las pruebas de `CheckClose`, `Service.CloseRound` y `TestPostgresCloseRound_*` pasan sin
pantalla.

### Pruebas (RED)

- [x] T007 [P] [E1, E6, E9, E10] Escribir las pruebas unitarias:
  - en `src/poker/round_test.go`, `TestCheckClose`: (Abierta, 3) → nil; (Cerrada, 3) → `ErrRoundClosed`; (Cerrada, 0)
    → `ErrRoundClosed` (cerrada antes que sin votos); (Abierta, 0) → `ErrNoVotes`;
  - en `src/poker/service_test.go`, `fakeStore` suma `closeCalls`, `gotClose [2]int64` y `closed ClosedRound`, y
    `TestService_CloseRound_OK` (pasa proyecto y ronda al almacén y devuelve su `ClosedRound`) y
    `TestService_CloseRound_Validaciones` (`roundID == 0` → `ErrRoundNotFound` y `memberID == 0` →
    `ErrMemberRequired` sin llamar al almacén; propaga `ErrNoVotes`, `ErrRoundClosed` y un error cualquiera de forma
    que `errors.Is` funcione).
- [x] T008 Crear `src/poker/reveal_fixtures_integration_test.go` con `newAlfaFixture(t)` (usa `openTestDB`): «Alfa»
  con Ana, Beto y Caro; «Login», «Pagos», «Reportes», «Perfil» y «Ayuda» con su ronda 1 abierta por Ana y los votos del
  Issue (Login 2/13/5, Pagos 3/5/8, Reportes 3/13/3, Perfil 5/5 sin Caro, Ayuda sin votos); «Beta» con Dani, «Tablero»
  con una ronda abierta (Dani 8) y «Agenda» con una ronda cerrada sin votos. Ayudantes `vote(round, member)`,
  `status(round)`, `closedAt(round)` y `waitForWaiters(t, f, n)` (como `waitForLock`, pero espera `n` sesiones en
  `wait_event_type = 'Lock'`, plazo 5 s). Limpieza en `t.Cleanup`: historias → proyectos → integrantes.
- [x] T009 [E1, E5, E6, E9, E10] Escribir en `src/poker/close_integration_test.go`:
  - `TestPostgresCloseRound_OK`: cerrar «Login» → `ClosedRound{StoryTitle: "Login", Number: 1}`, `status =
    'Cerrada'`, `closed_at` no NULL, los tres votos sin cambios y la historia (título, estado, `story_points`) sin
    cambios;
  - `TestPostgresCloseRound_SinTodosLosVotos`: «Perfil» se cierra con dos votos;
  - `TestPostgresCloseRound_SinVotos`: «Ayuda» → `ErrNoVotes`, sigue `'Abierta'` con `closed_at` NULL;
  - `TestPostgresCloseRound_YaCerrada`: cerrar «Login» dos veces → `ErrRoundClosed` y `closed_at` igual a la del primer
    cierre;
  - `TestPostgresCloseRound_Inexistente` y `TestPostgresCloseRound_OtroProyecto`: id 2147483647, la ronda abierta de
    «Tablero» y la cerrada sin votos de «Agenda» desde «Alfa» → `ErrRoundNotFound` (404 antes que 409) y nada cambia.
- [x] T010 [E8, E9, E12] Escribir en `src/poker/close_integration_test.go` las pruebas de concurrencia y falla
  (`research.md`, decisión 4):
  - `TestPostgresCloseRound_VotoEsperaAlCierre`: transacción de prueba con `LOCK TABLE poker_votes IN ACCESS EXCLUSIVE
    MODE`; `CloseRound` de «Login» en una goroutine; `waitForWaiters(1)`; `CastVote` de Caro con 8 en otra goroutine;
    `waitForWaiters(2)`; `ROLLBACK` de la prueba → el cierre devuelve nil, el voto `ErrRoundClosed` y Caro sigue en 5;
  - `TestPostgresCloseRound_Concurrente`: dos `CloseRound` de «Pagos» a la vez (barrera) → uno nil y otro
    `ErrRoundClosed`;
  - `TestPostgresCloseRound_EsperaAlVotoEnCurso`: transacción de prueba que toma «Ayuda» `FOR SHARE` e inserta un voto
    de Ana con 3; `CloseRound` espera (`waitForWaiters(1)`); `COMMIT` → el cierre devuelve nil y la ronda queda cerrada
    con ese voto;
  - `TestPostgresCloseRound_FalloNoDejaCambios`: transacción de prueba que toma «Login» `FOR SHARE`; `CloseRound` con
    un contexto de 200 ms → error que no es del dominio (`errors.Is` falso para los `Err…`); después del `ROLLBACK`,
    «Login» sigue `'Abierta'` con `closed_at` NULL.
- [x] T011 Ejecutar T007–T010, confirmar que fallan porque `CheckClose`, `ClosedRound` y `CloseRound` no existen y
  guardar la salida en `evidencia-red.txt`, sección "Cierre".

### Implementación (GREEN)

T012 y T013 van en el mismo commit: el puerto, el adaptador y las dos pruebas falsas cambian juntos.

- [x] T012 Agregar en `src/poker/round.go` `CheckClose(status RoundStatus, votes int) error` (`status != StatusOpen` →
  `ErrRoundClosed`; `votes == 0` → `ErrNoVotes`) y `ClosedRound{RoundID, StoryID int64; StoryTitle string; Number
  int}`; el método `CloseRound(ctx context.Context, projectID, roundID int64) (ClosedRound, error)` en `Store`
  (`src/poker/store.go`); `Service.CloseRound(ctx, projectID, roundID, memberID int64) (ClosedRound, error)` en
  `src/poker/service.go` (validaciones de `data-model.md`); `fakeStore.CloseRound` en `src/poker/service_test.go` y
  `failingPokerStore.CloseRound` (devuelve `errPokerStoreInternal`) en `src/web/poker_handlers_test.go`.
- [x] T013 Implementar `PostgreSQLStore.CloseRound` en `src/poker/postgres_store.go` con la transacción de
  `data-model.md`: `SELECT r.status, r.story_id, r.round_number, us.title … WHERE r.id = $1 AND r.project_id = $2 FOR
  UPDATE OF r` (`sql.ErrNoRows` → `ErrRoundNotFound`), `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`,
  `CheckClose` y `UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`; errores técnicos
  envueltos con `fmt.Errorf("…: %w", err)` como en `CastVote`.
- [x] T014 Ejecutar T007–T010 en verde y `go test -count=1 -p 1 ./src/poker/... ./src/web/...` completo (HU-05a sin
  cambios).

**Punto de control**: el cierre funciona en el dominio y en PostgreSQL, sin pantalla.

## Fase 4: Bloque Pantalla ("Cerrar ronda" y "Rondas cerradas") 🎯 MVP visible

**Objetivo**: cerrar desde Planning Poker y ver los votos revelados, el rango y la alerta, igual para todos.

**Prueba independiente**: con los datos del Issue, cerrar «Login» desde la pantalla y ver Ana 2, Beto 13, Caro 5 y la
alerta "de 2 a 13"; entrar como Beto y ver lo mismo.

### Pruebas (RED)

- [ ] T015 Crear `src/web/poker_reveal_fixtures_test.go` con `newAlfaWebFixture(t, db)` (usa `openContextTestDB`): los
  mismos datos de T008 y un ayudante `closeAt(round, ts)` que hace `UPDATE poker_rounds SET status = 'Cerrada',
  closed_at = $2 WHERE id = $1` (fecha explícita para el orden), `removeMember(member)` (borra su fila de
  `project_members`), y `roundRow(t, body, round, member)` / `closedRow(t, body, round, member)`, que devuelven el
  texto de la fila `data-integrante` dentro de la sección `data-ronda` / `data-ronda-cerrada` de esa ronda (el atributo
  se repite entre secciones). Limpieza en `t.Cleanup`.
- [ ] T016 [P] [E1, E5, E7, E10] Escribir en `src/web/postgres_poker_reveal_integration_test.go`:
  - `TestPostgresPokerCatalog_RondasCerradas`: con «Login» cerrada a las 10:00 y «Pagos» a las 10:05, `ClosedRounds` =
    [Pagos, Login] con historia, id y número 1; con el mismo `closed_at`, la de id mayor primero;
  - `TestPostgresPokerCatalog_VotosRevelados`: «Login» → Ana 2, Beto 13, Caro 5 en ese orden; «Perfil» cerrada → Caro
    con `Value` nil; los tableros de Ana y de Beto tienen `ClosedRounds` iguales;
  - `TestPostgresPokerCatalog_IntegranteQuitado`: se quita a Ana de «Alfa» y se cierra «Login» → Ana sigue con 2;
    «Perfil» cerrada sin Caro en el proyecto → Caro no figura;
  - `TestPostgresPokerCatalog_SoloRondasCerradas`: «Pagos» abierta no está en `ClosedRounds`, ningún voto de
    `ClosedRounds` es de «Pagos», y `Rounds`/`Participants` de HU-05a no cambian; las rondas de «Beta» no aparecen.
- [ ] T017 [P] [E2, E3, E4, E5, E7] Escribir en `src/web/poker_reveal_view_test.go` (servidor de `newPokerApp`, datos
  de T015, rondas cerradas con `closeAt`):
  - `TestHU05b_AlertaDeDivergencia`: «Login» y «Reportes» → dentro de `data-ronda-cerrada` de cada una, un `div` con
    `alert alert-warning`, "Advertencia:" y "Divergencia: los votos van de 2 a 13. Debatan antes de acordar la
    estimación" / "… de 3 a 13. …";
  - `TestHU05b_SinDivergenciaEnElLimite`: «Pagos» → "Sin divergencia: los votos van de 3 a 8" y, en esa ronda, ni
    `alert-warning` ni "Divergencia:"; «Perfil» → Caro "No votó" y "Sin divergencia: los votos van de 5 a 5";
  - `TestHU05b_VotosParaTodos`: la sección "Rondas cerradas" de la página de Ana y la de Beto son idénticas byte a byte;
  - `TestHU05b_VotoAbiertoNuncaSeRevela`: con «Login» cerrada y «Pagos» abierta, `roundRow` de Ana en «Pagos» de la
    página de Beto dice "Votó" sin "3"; Ana cambia su voto de «Pagos» a 21 y la página de Beto es
    idéntica byte a byte a la anterior;
  - `TestHU05b_SinRondasCerradas` ("Aún no hay rondas cerradas"), `TestHU05b_RondaCerradaSinVotos` (una ronda cerrada
    sin votos preparada en la base: 200, todos "No votó", sin rango ni alerta), `TestHU05b_TextosEscapados` (título e
    integrante con `<script>` escapados en "Rondas cerradas" y en el mensaje de cierre) y `TestHU05b_Rendimiento` (20
    rondas abiertas, 50 cerradas y 10 integrantes → `GET /planning-poker` en menos de 500 ms).
- [ ] T018 [P] [E1, E5, E6, E8, E9, E10, E11, E12] Escribir en `src/web/poker_close_handlers_test.go`:
  - `TestHU05b_BotonCerrarRonda`: cada ronda abierta tiene un formulario `action="/planning-poker/cerrar"` con su
    `ronda` oculto y el botón "Cerrar ronda"; las rondas cerradas no;
  - `TestHU05b_CerrarRonda` (E1): Ana cierra «Login» → 303 a `/planning-poker`; el GET muestra una sola vez "Ronda
    cerrada para la historia «Login»", «Login» ya no está entre las abiertas y en "Rondas cerradas" figuran Ana 2, Beto
    13 y Caro 5; un campo `integrante` con otro id se ignora;
  - `TestHU05b_CerrarSinTodosLosVotos` (E5), `TestHU05b_RondaSinVotos` (E6: 409, "La ronda no tiene votos", «Ayuda»
    sigue abierta), `TestHU05b_VotarEnRondaCerrada` (E8: Caro vota 8 → 409, "La ronda ya está cerrada", sigue 5),
    `TestHU05b_CerrarDosVeces` (E9: 409, `closed_at` sin cambios);
  - `TestHU05b_RondaInexistente` (E10): `ronda` inexistente, vacío, "abc", "05" y de «Beta» (abierta y cerrada) → 404
    "La ronda no existe", nada cambia;
  - `TestHU05b_SinIdentificarme` (E11): con proyecto y sin integrante → 303 a
    `/contexto/integrante?volver_a=%2Fplanning-poker` y «Login» sigue abierta;
  - `TestHU05b_ErrorAlCerrar` (E12): con `failingPokerStore` → 500, "La operación no pudo completarse", sin "Ronda
    cerrada para"; «Login» sigue abierta, "Aún no hay rondas cerradas" y `roundRow` de Beto en «Login» dice "Votó" sin "13"; el
    cuerpo no contiene "poker_", "SQL", "pgx" ni el texto del error;
  - `TestHU05b_RecargaSinReenviar`: después de cerrar, dos GET → el mensaje aparece una sola vez.
- [ ] T019 [P] [E1–E11] Escribir `test/web_poker_reveal_bdd_test.go` con su propio «Alfa» (Dado/Cuando/Entonces como
  comentarios, igual que `test/web_poker_bdd_test.go`, con el servidor y el servicio reales):
  `TestBDD_HU05b_RevelacionDeVotos`, `_AlertaDeDivergencia`, `_SinDivergenciaEnElLimite`, `_DivergenciaDeTresNiveles`,
  `_CerrarSinQueTodosVoten`, `_RondaSinVotos`, `_VotosParaTodos`, `_VotarEnRondaCerrada`, `_CerrarDosVeces`,
  `_RondaInexistente` y `_SinIdentificarme`. Las rondas cerradas se cierran desde la pantalla (en `_VotosParaTodos`,
  Ana cierra «Login» y Beto abre "Planning Poker", como dice el escenario 7).
- [ ] T020 Ejecutar T016–T019, confirmar que fallan (no existen `ClosedRounds`, la sección ni la ruta) y guardar la
  salida en `evidencia-red.txt`, sección "Pantalla".

### Implementación (GREEN)

- [ ] T021 Agregar en `src/web/poker_catalog.go` `PokerRevealedVoteView{MemberID int64; Name string; Value *int}`,
  `PokerClosedRoundView{ID, StoryID int64; StoryTitle string; Number int; Votes []PokerRevealedVoteView}` y
  `PokerBoard.ClosedRounds`; en `src/web/postgres_poker_catalog.go`, las consultas `pokerClosedRoundsQuery` y
  `pokerRevealedVotesQuery` de `data-model.md` dentro de la misma transacción de `PokerBoard`, con `pokerEach` y un
  índice por id (`ClosedRounds` nunca nil).
- [ ] T022 [P] Agregar en `src/web/poker_messages.go` `MsgPokerNoVotes` "La ronda no tiene votos",
  `MsgPokerNoClosedRounds` "Aún no hay rondas cerradas", `LabelPokerDidNotVote` "No votó",
  `pokerRoundClosedMessage(title)` → "Ronda cerrada para la historia «title»", `pokerDivergenceMessage(low, high)` →
  "Divergencia: los votos van de X a Y. Debatan antes de acordar la estimación" y `pokerNoDivergenceMessage(low,
  high)` → "Sin divergencia: los votos van de X a Y"; y en `src/web/poker_form.go` el caso `poker.ErrNoVotes` → 409
  `MsgPokerNoVotes` en `pokerOutcomeFor`.
- [ ] T023 En `src/web/poker_handlers.go`: `CloseRound(ctx, projectID, roundID, memberID int64) (poker.ClosedRound,
  error)` en `PokerActions`; `"POST " + pathPoker + "/cerrar": http.HandlerFunc(h.close)` en el mismo `Routes` de
  `RegisterPlanningPoker`; `close` (lee `ronda` con `pokerFormIDs`, llama a `CloseRound` con proyecto e integrante de
  `ActiveContextFrom`, `reject` o `done(w, r, pokerRoundClosedMessage(closed.StoryTitle))`); y en `render`,
  `pokerClosedRoundPage{PokerClosedRoundView; Alert *Flash; Range string}` armado con `poker.Spread` (`data-model.md`,
  "Modelo de vista": divergencia → `FlashWarning`; sin divergencia → `Range`; error → ninguno) y `pokerPageData` con
  `ClosedRounds`, `NoClosedRounds` y `DidNotVote`.
- [ ] T024 Completar `src/web/templates/poker.html` según `data-model.md`, "Pantalla": formulario "Cerrar ronda" en
  cada ronda abierta y, al final de la página, la sección "Rondas cerradas" (`data-ronda-cerrada`, `data-integrante`,
  valor o "No votó",
  `{{with .Alert}}{{template "alert" .}}{{end}}` o `<p class="poker-range">`, "Aún no hay rondas cerradas"). Sin
  JavaScript.
- [ ] T025 Ejecutar T016–T019 en verde y `go test -count=1 -p 1 ./...` completo, incluidas las pruebas de HU-05a.

**Punto de control**: el Scrum Master cierra rondas y el equipo ve los votos y las divergencias.

## Fase 5: Cierre

- [ ] T026 REFACTOR sin cambiar comportamiento: duplicación entre `open`, `vote` y `close`, y entre los ayudantes de
  fixtures; nombres y comentarios de `src/poker` y `src/web/poker_*`. Comprobar con `git diff --stat main` que no hay
  migraciones, que `src/cmd/server/main.go` no cambió y que solo se editaron archivos de HU-05a. Todas las pruebas en
  verde.
- [ ] T027 `gofmt -l` sobre los archivos creados (sin salida), `go vet ./...` limpio y `go test ./... -p 1 -race` en
  verde, como en el CI; guardar `specs/005b-poker-reveal/evidencia-pruebas.txt` (`go test -count=1 -p 1 -v ./...`) y
  `specs/005b-poker-reveal/cobertura.txt` (`go test -count=1 -p 1 -cover ./...`), verificando al menos 80% en
  `src/poker`.
- [ ] T028 Verificar a mano los 12 escenarios con la aplicación en ejecución (`quickstart.md`, "Verificación manual").
- [ ] T029 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas y marcar
  `checklists/requirements.md`.
- [ ] T030 Abrir el Pull Request desde `HU-05b` citando HU-05b, con las evidencias, sin migración y con la lista de
  archivos de HU-05a editados; revisión y aprobación de otro integrante, merge con "Create a merge commit" y tarjeta a
  "Hecho". Avisar a HU-18 los nombres finales `poker.Spread`, `poker.VoteSpread`, `poker.ErrNoVotes` y
  `Service.CloseRound`, y a HU-28 que el cierre no tiene roles.

## Dependencias y orden de ejecución

- La **Fase 1** va primero (T001 antes que todo).
- **Bloque Divergencia (Fase 2)** depende de la Fase 1.
- **Bloque Cierre (Fase 3)** depende de T005 (`ErrNoVotes`); sus pruebas se pueden escribir en paralelo con la Fase 2.
  GREEN en orden T012 → T013 (mismo commit) → T014.
- **Bloque Pantalla (Fase 4)** depende de T005 (`Spread`) y de T012–T013 (`CloseRound`). GREEN en orden T021 → T022
  (en paralelo con T021) → T023 → T024 → T025.
- **Fase 5** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- Los archivos editados son de HU-05a; HU-18 va después de HU-05b (notas del equipo), así que no hay otro trabajo en
  paralelo sobre ellos.

### Oportunidades de paralelismo

- T003 (Fase 2) y T007–T008 (Fase 3) se pueden escribir a la vez.
- T016, T017, T018 y T019 (archivos distintos, después de T015).
- T021 y T022.
- T029 junto con T028.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T018, T023, T024 |
| FR-002 | T009, T013, T018, T023 |
| FR-003 | T007, T009, T012, T013, T018 |
| FR-004 | T007, T009, T010, T012, T018 |
| FR-005 | T007, T009, T012, T018, T022 |
| FR-006 | T003, T005, T030 |
| FR-007 | T016, T017, T021, T024 |
| FR-008 | T016, T017, T021, T024 |
| FR-009 | T003, T017, T022, T023, T024 |
| FR-010 | T016, T017, T019 |
| FR-011 | T016, T017, T021 |
| FR-012 | T010, T018 |
| FR-013 | T010, T018, T023 |
| FR-014 | T025 (`TestHU05a_ErrorAlLeer` sigue en verde con la lectura ampliada) |
| FR-015 | T018, T019 |
| FR-016 | T009, T016, T018 |
| FR-017 | T017 |
| FR-018 | T009, T014, T025, T026 |
| FR-019 | T018, T022, T026 |
| SC-001 | T003, T017 |
| SC-002 | T009, T010, T018 |
| SC-003 | T016, T017 |
| SC-004 | T016, T017 |
| SC-005 | T010 |
| SC-006 | T017 |
| SC-007 | T004, T011, T020, T027 |

## Estrategia de implementación

1. Fase 1: confirmar nombres y crear la rama.
2. Bloque Divergencia: la regla sola. Al terminar, HU-18 ya puede usar el contrato.
3. Bloque Cierre: el cierre en el dominio y en PostgreSQL, con la concurrencia probada.
4. Bloque Pantalla: el botón y la sección "Rondas cerradas". Es el MVP visible para la Sprint Review.
5. Cierre, evidencias y PR.
