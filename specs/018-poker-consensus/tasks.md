# Tareas: HU-18 – Planning Poker: Nueva Ronda y Estimación Acordada

**Entrada**: `specs/018-poker-consensus/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-05b integrada en `main` (y con ella HU-05a), HU-02, HU-20a y HU-20b. Al 28/10/2026 HU-05b
solo tiene sus documentos en `main` (`d12067f`): **HU-18 no se empieza hasta que T001 confirme HU-05b en el código.**

Formato: `[ID] [P] [E#] descripción`

- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (1 a 12).
- Cada bloque sigue RED → evidencia → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se
  guarda en `specs/018-poker-consensus/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`. Pruebas siempre con `go test -count=1 -p 1`.
- CI (`.github/workflows/ci.yml`): `gofmt -l .` sin salida, `go vet ./...` y `go test ./... -p 1 -race` con cobertura.
  `gofmt -w` solo sobre los archivos que se crean; los de HU-05a/HU-05b se editan a mano, sin cambiar su fin de línea.
- Datos de prueba: los del Issue (`data-model.md`, "Datos de prueba"), con nombres exactos; cada prueba arma su propio
  «Alfa» y lo borra al final (asignaciones → historias → Sprints → proyectos → integrantes).
- Cada GREEN corre también las pruebas de HU-05a y HU-05b sin cambiar sus expectativas.
- Los nombres marcados "(T001: HU-05b)" salen de `specs/005b-poker-reveal/data-model.md` y se confirman en T001.

## Fase 1: Preparación

- [x] T001 Confirmar que HU-05b está en `main` y sus nombres. `git switch main; git pull`; `git grep -n` en `src/poker`,
  `src/web/poker_*.go`, `src/web/postgres_poker_catalog.go` y `src/web/templates/poker.html`:
  - de HU-05b: `Spread`, `VoteSpread`, `ErrNoVotes`, `CheckClose`, `ClosedRound`, `Store.CloseRound`,
    `Service.CloseRound`, `PokerClosedRoundView`, `PokerRevealedVoteView`, `PokerBoard.ClosedRounds`,
    `pokerClosedRoundsQuery`, `pokerClosedRoundPage` (con `Alert` y `Range`), `PokerActions.CloseRound`, la ruta
    `/planning-poker/cerrar`, la sección "Rondas cerradas" con `data-ronda-cerrada`, y en las pruebas `newAlfaFixture`,
    `waitForWaiters`, `newAlfaWebFixture`, `closeAt`, `roundRow`, `closedRow` y las pruebas falsas del puerto;
  - de HU-05a: los de `research.md` ("Datos verificados");
  - en las pruebas de HU-05b (`poker_reveal_view_test.go`, `poker_close_handlers_test.go`), si alguna cuenta formularios,
    radios `valor` o avisos `alert-warning` de una ronda cerrada: HU-18 agrega en la última ronda cerrada dos formularios,
    diez radios sin `checked` y, con divergencia, un segundo `alert-warning`; si alguna prueba se rompería, avisar antes
    de empezar (no se cambian sus expectativas);
  - que no existan `OpenNextRound`, `AgreeEstimate`, `CheckNextRound`, `CheckAgree`, `StoryRound`, `AgreedEstimate`,
    `ErrNotLatestRound`, `ErrRoundStillOpen`, `ErrStoryInFinishedSprint`, `/planning-poker/nueva-ronda` ni
    `/planning-poker/estimacion`; y si HU-13, HU-15 o HU-17 ya integraron una regla de "Sprint finalizado", anotarlo en
    `research.md` y reutilizar su criterio.

  **Si `CloseRound` o `Spread` no están en `src/poker` de `main`, detener HU-18 y avisar al equipo.** Si algún nombre
  de HU-05b cambió, ajustar `data-model.md`, `plan.md` y este archivo antes de seguir.
- [x] T002 Crear la rama `HU-18` desde `main` actualizado, solo si no hay cambios sin commitear (`git switch -c HU-18`).
  Línea base: `go test -count=1 -p 1 ./...`, `gofmt -l .` y `go vet ./...` en verde.

## Fase 2: Bloque Reglas

**Objetivo**: las decisiones de "Nueva ronda" y de la estimación, con su orden de rechazos, como funciones puras.

**Prueba independiente**: `go test -count=1 ./src/poker/ -run "CheckNextRound|CheckAgree"` pasa sin PostgreSQL.

### Pruebas (RED)

- [x] T003 [E1, E7, E8, E9, E10] Escribir `src/poker/consensus_rules_test.go` con las dos pruebas de tabla de
  `data-model.md`: `TestCheckNextRound` («Login» 1 → 2 y 2 → 3; desde la 1 con la 2 cerrada → `ErrNotLatestRound`;
  «Reportes» desde la 1 y desde la 2 → `ErrRoundAlreadyOpen`; «Alta» → `ErrStoryInFinishedSprint`) y `TestCheckAgree`
  («Pagos» → nil; «Reportes» 2 → `ErrRoundStillOpen`; «Reportes» 1 → `ErrNotLatestRound`; «Alta» →
  `ErrStoryInFinishedSprint`; abierta, no última y con Sprint finalizado → `ErrRoundStillOpen`), con `errors.Is`.
- [x] T004 Ejecutar T003, confirmar que falla porque los tipos, las reglas y los errores no existen y guardar la salida
  en `evidencia-red.txt`, sección "Reglas".

### Implementación (GREEN)

- [x] T005 Agregar en `src/poker/errors.go` `ErrNotLatestRound`, `ErrRoundStillOpen` y `ErrStoryInFinishedSprint` (textos
  de `data-model.md`), y en `src/poker/round.go`, junto a `CheckOpen`, `StoryRound{Number int; Status RoundStatus}`,
  `CheckNextRound(from int, rounds []StoryRound, inFinishedSprint bool) (next int, err error)`, `CheckAgree(status
  RoundStatus, number, latest int, inFinishedSprint bool) error` y `AgreedEstimate{StoryID int64; StoryTitle string;
  StoryPoints int}`, con el orden de rechazos de `data-model.md`.
- [x] T006 Ejecutar T003 en verde junto con `go test -count=1 -p 1 ./src/poker/...`.

## Fase 3: Bloque Nueva ronda (dominio y almacenamiento)

**Objetivo**: abrir la ronda siguiente de una historia en el dominio y en PostgreSQL, con sus rechazos y su
concurrencia.

**Prueba independiente**: `TestService_OpenNextRound_*` y `TestPostgresOpenNextRound_*` pasan sin pantalla.

### Pruebas (RED)

- [x] T007 Crear `src/poker/consensus_fixtures_integration_test.go` con `newConsensusFixture(t)` (usa `openTestDB`): los
  datos de `data-model.md` («Alfa» con Ana, Beto y Caro; «Login», «Pagos», «Reportes» y «Alta» con sus rondas cerradas
  cargadas en la base con `closed_at` explícito y sus votos; la ronda 2 abierta de «Reportes» con el 5 de Ana; Sprint 1
  "Finalizado" con «Alta» asignada y en "Hecho"; «Beta» con Dani y «Tablero» con su ronda 1 cerrada), ayudantes
  `rounds(story)`, `storyPoints(story)`, `assignToActiveSprint(story, status)` (Sprint 2 "Activo", solo para las pruebas
  de concurrencia) y `closeSprintLikeHU15(t, tx, sprintID)` (`research.md`, decisión 7). Limpieza en `t.Cleanup`.
- [x] T008 [E1, E8, E10, E11, E12] Escribir las pruebas de "Nueva ronda":
  - en `src/poker/service_test.go`, `fakeStore` suma `OpenNextRound` (registra proyecto, ronda e integrante) y
    `TestService_OpenNextRound_OK` / `TestService_OpenNextRound_Validaciones` (`roundID == 0` → `ErrRoundNotFound` y
    `memberID == 0` → `ErrMemberRequired` sin llamar al almacén; errores propagados para `errors.Is`);
  - en `src/poker/next_round_integration_test.go`: `TestPostgresOpenNextRound_OK` («Login» desde la 1 → ronda 2
    "Abierta", `opened_by` = Beto, sin votos, `project_id` de «Alfa»; la ronda 1 sigue cerrada con 2, 13 y 5; cerrada la
    2, desde la 2 → ronda 3), `_NoEsLaUltima` (con las rondas 1 y 2 de «Login» cerradas, desde la 1 →
    `ErrNotLatestRound`), `_ConRondaAbierta` («Reportes» desde la 1 y desde la 2 → `ErrRoundAlreadyOpen`, sigue con dos
    rondas), `_SprintFinalizado` («Alta» → `ErrStoryInFinishedSprint`, sin ronda 2), `_Inexistente` y `_OtroProyecto`
    (id 2147483647 y la ronda de «Tablero» → `ErrRoundNotFound`), `_Concurrente` (dos a la vez desde la 1 de «Login»
    con una barrera → una ronda 2 y un `ErrRoundAlreadyOpen`), `_FalloNoDejaCambios` (integrante 2147483647 → error que
    no es del dominio y ninguna ronda nueva).
- [x] T009 Ejecutar T008, confirmar que falla porque `OpenNextRound` no existe y guardar la salida en
  `evidencia-red.txt`, sección "Nueva ronda".

### Implementación (GREEN)

- [x] T010 En un solo commit: `OpenNextRound(ctx, projectID, roundID, memberID int64) (OpenedRound, error)` en `Store`
  (`src/poker/store.go`); `Service.OpenNextRound` en `src/poker/service.go` (validaciones de `data-model.md`);
  `PostgreSQLStore.OpenNextRound` en `src/poker/postgres_store.go` con la transacción de `data-model.md` (ronda base
  `FOR SHARE`; asignación; Sprint `FOR SHARE` si hay; historia `FOR NO KEY UPDATE OF us` con relectura de la asignación
  y una repetición; `SELECT round_number, status FROM poker_rounds WHERE story_id = $1`; `CheckNextRound`; `INSERT`;
  `23505` de `uq_poker_rounds_one_open_per_story` → `ErrRoundAlreadyOpen` y de `uq_poker_rounds_story_number` →
  `ErrNotLatestRound`), con ayudantes `storySprint` y `lockStory` que reutiliza T015; `fakeStore` y `failingPokerStore`
  (`src/web/poker_handlers_test.go`, falla con `errPokerStoreInternal`) con el método nuevo.
- [x] T011 Ejecutar T008 en verde y `go test -count=1 -p 1 ./src/poker/... ./src/web/...` completo (HU-05a y HU-05b
  sin cambios).

## Fase 4: Bloque Estimación acordada (dominio y almacenamiento)

**Objetivo**: fijar los Story Points desde la última ronda cerrada, con la regla del Sprint finalizado y sin bloqueos
mutuos con el cierre de Sprint.

**Prueba independiente**: `TestService_AgreeEstimate_*` y `TestPostgresAgreeEstimate_*` pasan sin pantalla.

### Pruebas (RED)

- [x] T012 [E3, E4, E5, E6, E7, E9, E10, E11, E12] Escribir las pruebas de la estimación:
  - en `src/poker/service_test.go`, `fakeStore` suma `AgreeEstimate` y `TestService_AgreeEstimate_OK` /
    `TestService_AgreeEstimate_Validaciones` (valor "" → `ErrValueRequired` y "4" → `ErrValueNotInSeries` antes que
    `roundID == 0`, sin llamar al almacén; `roundID == 0` → `ErrRoundNotFound`; `memberID == 0` →
    `ErrMemberRequired`; pasa el valor ya convertido);
  - en `src/poker/agree_estimate_integration_test.go`: `TestPostgresAgreeEstimate_OK` («Pagos» 5; después 8 desde la
    misma ronda → 8; otra vez 8 → éxito; solo cambia `story_points`: estado, `active_sprint_id` y rondas iguales),
    `_ConDivergencia` («Login» desde la 1 → 5), `_RondaAbierta` («Reportes» 2 → `ErrRoundStillOpen`, sigue 3),
    `_NoEsLaUltima` («Reportes» 1 → `ErrNotLatestRound`, sigue 3), `_SprintFinalizado` («Alta» → `ErrStoryInFinishedSprint`,
    sigue 3), `_SprintActivo` («Pagos» en el Sprint 2 "Activo" → 5), `_Inexistente` y `_OtroProyecto`
    (`ErrRoundNotFound`), `_FalloNoDejaCambios` (una transacción de prueba toma «Pagos» `FOR SHARE` y la estimación con
    un contexto de 200 ms falla con un error que no es del dominio; «Pagos» sigue sin Story Points).
- [x] T013 [E9, E10] Escribir en `src/poker/agree_estimate_integration_test.go` las pruebas de concurrencia de
  `research.md`, decisión 7: `TestPostgresAgreeEstimate_CierreDeSprintPrimero` (casos «Pagos» "Hecho" →
  `ErrStoryInFinishedSprint` y «Pagos» "En progreso" → vuelve al Backlog, la estimación repite y queda 5),
  `TestPostgresAgreeEstimate_EstimacionPrimero` (estimación y cierre terminan bien, sin `40P01`, «Pagos» 5 y el Sprint
  finalizado) y `TestPostgresNextRoundYEstimacion_Serializadas` (con `LOCK TABLE poker_rounds IN SHARE ROW EXCLUSIVE
  MODE`: la ronda 2 de «Pagos» se abre y la estimación desde la 1 recibe `ErrNotLatestRound`). Sincronización solo con
  `waitForWaiters` (T001: HU-05b), plazo de 5 s.
- [x] T014 Ejecutar T012–T013, confirmar que fallan porque `AgreeEstimate` no existe y guardar la salida en
  `evidencia-red.txt`, sección "Estimación".

### Implementación (GREEN)

- [ ] T015 En un solo commit: `AgreeEstimate(ctx, projectID, roundID int64, value int) (AgreedEstimate, error)` en
  `Store`; `Service.AgreeEstimate(ctx, projectID, roundID, memberID int64, rawValue string)` con `ParseValue` primero;
  `PostgreSQLStore.AgreeEstimate` con la transacción de `data-model.md` (ronda `FOR UPDATE`; `SELECT max(round_number)
  FROM poker_rounds WHERE story_id = $1`; `storySprint`/`lockStory` de T010 con la relectura; `CheckAgree`; `UPDATE
  user_stories SET story_points = $2 WHERE id = $1 RETURNING title`); `fakeStore` y `failingPokerStore` con el método
  nuevo.
- [ ] T016 Ejecutar T012–T013 en verde y `go test -count=1 -p 1 ./src/poker/... ./src/web/...` completo.

**Punto de control**: las dos acciones funcionan en el dominio y en PostgreSQL, sin pantalla.

## Fase 5: Bloque Pantalla 🎯 MVP visible

**Objetivo**: "Nueva ronda" y "Estimación acordada" en la última ronda cerrada de cada historia, con recordatorio y
acciones deshabilitadas para un Sprint finalizado.

**Prueba independiente**: con los datos del Issue, abrir la ronda 2 de «Login» desde la pantalla, votarla, cerrarla y
registrar 8; ver 8 en el Backlog.

### Pruebas (RED)

- [ ] T017 Crear `src/web/poker_consensus_fixtures_test.go` con `newConsensusWebFixture(t, db)` (usa
  `openContextTestDB`; los datos de T007) y `newConsensusApp(t, opts...)`: el servidor de `newPokerApp` más
  `RegisterBacklog(srv, backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(db)), NewPostgreSQLStoryCatalog(db))`,
  con la opción de `failingPokerStore`. Limpieza en `t.Cleanup`.
- [ ] T018 [P] [E1, E10] Escribir en `src/web/postgres_poker_consensus_integration_test.go`:
  `TestPostgresPokerCatalog_UltimaRonda` (ronda 1 de «Login», «Pagos» y «Alta» con `Latest`; ronda 1 de «Reportes» sin
  `Latest` porque la 2 está abierta; abierta la ronda 2 de «Login», su ronda 1 pierde `Latest`) y
  `TestPostgresPokerCatalog_SprintFinalizado` («Alta» con `InFinishedSprint`; «Login» sin Sprint y «Pagos» en un Sprint
  "Activo", sin; el caso de una historia devuelta al Backlog lo cubre T013).
- [ ] T019 [P] [E1–E12] Escribir en `src/web/poker_consensus_handlers_test.go` (con `newConsensusApp`, `roundRow` y
  `closedRow` de HU-05b, T001: HU-05b):
  - `TestHU18_AccionesSoloEnLaUltimaRonda`: dentro de `data-ronda-cerrada` de «Login», «Pagos» y «Alta», los formularios
    `action="/planning-poker/nueva-ronda"` y `action="/planning-poker/estimacion"` con `ronda` oculto, diez radios
    `valor` sin `checked`, "Nueva ronda" y "Registrar estimación"; en la ronda 1 de «Reportes», ninguno;
  - `TestHU18_NuevaRonda` (E1: 303; una vez "Ronda 2 abierta para la historia «Login»"; la ronda 2 entre las abiertas
    con todos "Sin votar"; `closedRow` de la ronda 1 con Ana 2, Beto 13 y Caro 5; un campo `integrante` se ignora);
  - `TestHU18_VotosOcultosEnLaNuevaRonda` (E2: Ana vota 8 en la ronda 2; `roundRow` de Ana en la página de Beto dice
    "Votó" sin "8"; Ana cambia a 21 y la página de Beto es idéntica byte a byte);
  - `TestHU18_RegistrarEstimacion` (E3: ronda 2 de «Login» cerrada con 8, 8 y 5; registrar 8 → 303, "Estimación
    registrada: «Login» queda con 8 Story Points" y `GET /backlog` muestra «Login» con 8), `TestHU18_ConsensoDirecto`
    (E4: «Pagos» 5), `TestHU18_RecordatorioDeDivergencia` (E5: en «Login» el formulario tiene `alert-warning`,
    "Advertencia:" y el recordatorio; en «Pagos» y «Alta», no; registrar 5 → «Login» 5);
  - `TestHU18_ValorFueraDeSerie` (E6: "4", "abc", "05" → 422 "El valor no pertenece a la serie de Fibonacci"; vacío →
    422 "Seleccioná un valor de la serie"; vacío con ronda inexistente → 422; «Pagos» sin cambios);
  - `TestHU18_RondaSinCerrar` (E7: 409, «Reportes» 3), `TestHU18_NuevaRondaConRondaAbierta` (E8: 409, dos rondas),
    `TestHU18_RondaQueNoEsLaUltima` (E9: 409 "Hay una ronda más reciente de esta historia", «Reportes» 3);
  - `TestHU18_SprintFinalizado` (E10: en «Alta» los dos botones y el `fieldset` con `disabled` y "Sprint finalizado";
    enviar la estimación 5 y "Nueva ronda" igual → 409 con el mensaje; «Alta» sigue con 3 y una sola ronda);
  - `TestHU18_RondaInexistente` (E11: en las dos acciones, inexistente, vacía, "abc", "05" y la ronda de «Tablero» →
    404 "La ronda no existe"; nada cambia);
  - `TestHU18_ErrorAlGuardar` (E12: con `failingPokerStore`, las dos acciones → 500 "La operación no pudo completarse",
    sin mensaje de éxito, sin "poker_", "SQL", "pgx" ni el texto del error; Story Points y rondas sin cambios);
  - `TestHU18_SinIdentificarme` (303 a `/contexto/integrante?volver_a=%2Fplanning-poker` en las dos acciones),
    `TestHU18_RecargaSinReenviar`, `TestHU18_TituloEscapado` (título con `<script>` escapado en los dos mensajes de
    éxito) y `TestHU18_Rendimiento` (20 rondas abiertas, 50 cerradas, 10 integrantes → `GET /planning-poker` en menos de
    500 ms).
- [ ] T020 [P] [E1–E11] Escribir `test/web_poker_consensus_bdd_test.go` con su propio «Alfa» (Dado/Cuando/Entonces como
  comentarios, servidor y servicio reales, Backlog registrado): `TestBDD_HU18_NuevaRondaTrasDivergencia`,
  `_VotosOcultosEnLaNuevaRonda`, `_RegistrarEstimacionAcordada`, `_ConsensoDirecto`, `_RegistrarDespuesDeDebatir`,
  `_ValorFueraDeLaSerie`, `_RondaSinCerrar`, `_NuevaRondaConRondaAbierta`, `_RondaQueYaNoEsLaUltima`,
  `_HistoriaDeUnSprintFinalizado` y `_RondaInexistente`. En el escenario 3, la ronda 2 se abre, se vota y se cierra
  desde la pantalla.
- [ ] T021 Ejecutar T018–T020, confirmar que fallan (no existen `Latest`, las acciones ni las rutas) y guardar la
  salida en `evidencia-red.txt`, sección "Pantalla".

### Implementación (GREEN)

- [ ] T022 Agregar `Latest bool` e `InFinishedSprint bool` a `PokerClosedRoundView` en `src/web/poker_catalog.go` y las
  columnas `latest` e `in_finished_sprint` a `pokerClosedRoundsQuery` en `src/web/postgres_poker_catalog.go`, con el
  SQL de `data-model.md` (T001: HU-05b).
- [ ] T023 [P] Agregar en `src/web/poker_messages.go` los textos de la tabla de `data-model.md`
  (`MsgPokerNotLatestRound`, `MsgPokerRoundStillOpen`, `MsgPokerStoryInFinishedSprint`, `MsgPokerDivergenceReminder`,
  `LabelPokerFinishedSprint`, `pokerNextRoundOpenedMessage(number, title)`, `pokerEstimateRegisteredMessage(title,
  points)`), y en `src/web/poker_form.go` los casos `ErrNotLatestRound`, `ErrRoundStillOpen` y
  `ErrStoryInFinishedSprint` → 409 en `pokerOutcomeFor`.
- [ ] T024 En `src/web/poker_handlers.go`: `OpenNextRound` y `AgreeEstimate` en `PokerActions`; `"POST " + pathPoker +
  "/nueva-ronda"` y `"POST " + pathPoker + "/estimacion"` en el mismo `Routes` de `RegisterPlanningPoker`; handlers
  `nextRound` y `agree` (flujo de `plan.md`); y en `render`, `pokerClosedRoundPage` suma `Actions`, `Locked` y
  `Reminder` (`FlashWarning` con `MsgPokerDivergenceReminder` si es la última y `poker.Spread` dice `Divergent`), y
  `pokerPageData` suma `FinishedSprint` (T001: HU-05b).
- [ ] T025 Completar `src/web/templates/poker.html` con el bloque `poker-actions` de `data-model.md` dentro de cada
  `data-ronda-cerrada` con `.Actions` (radios sin `checked`, `disabled` y "Sprint finalizado" si `.Locked`). Sin
  JavaScript.
- [ ] T026 Ejecutar T018–T020 en verde y `go test -count=1 -p 1 ./...` completo, incluidas las pruebas de HU-05a y
  HU-05b sin cambiar sus expectativas.

**Punto de control**: el Scrum Master vuelve a votar o fija los Story Points desde Planning Poker.

## Fase 6: Cierre

- [ ] T027 REFACTOR sin cambiar comportamiento (duplicación entre `OpenNextRound` y `AgreeEstimate`, y entre los
  handlers). Comprobar con `git diff --stat main` que no hay migraciones, que `src/cmd/server/main.go` no cambió y que
  solo se editaron archivos de HU-05a/HU-05b. `gofmt -l` sobre los archivos creados (sin salida), `go vet ./...` y
  `go test ./... -p 1 -race` en verde.
- [ ] T028 Guardar `specs/018-poker-consensus/evidencia-pruebas.txt` (`go test -count=1 -p 1 -v ./...`) y
  `specs/018-poker-consensus/cobertura.txt` (`go test -count=1 -p 1 -cover ./...`), con al menos 80% en `src/poker`.
- [ ] T029 [P] Verificar a mano los 12 escenarios (`quickstart.md`, "Verificación manual"); actualizar la trazabilidad
  de `spec.md` con los nombres reales de las pruebas y marcar `checklists/requirements.md`.
- [ ] T030 Abrir el Pull Request desde `HU-18` citando HU-18, con las evidencias, sin migración y con la lista de
  archivos de HU-05a/HU-05b editados; revisión y aprobación de otro integrante, merge con "Create a merge commit" y
  tarjeta a "Hecho". Avisar a HU-13, HU-15 y HU-17 que HU-18 aplica "Sprint finalizado" con la asignación y el orden
  ronda → Sprint → historia, y a HU-28 que las acciones no tienen roles.

## Dependencias y orden de ejecución

- **T001 bloquea todo**: sin HU-05b en `main`, HU-18 no empieza.
- **Bloque Reglas (Fase 2)** depende de la Fase 1.
- **Bloque Nueva ronda (Fase 3)** depende de T005. **Bloque Estimación (Fase 4)** depende de T005 y de T010 (reutiliza
  `storySprint` y `lockStory`); sus pruebas se pueden escribir mientras se hace la Fase 3.
- **Bloque Pantalla (Fase 5)** depende de T010 y T015. GREEN en orden T022 → T023 (en paralelo con T022) → T024 →
  T025 → T026.
- **Fase 6** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar la evidencia → implementación (GREEN).
- T010 y T015 cambian el puerto: cada uno va en un solo commit con el adaptador y las dos pruebas falsas.

### Oportunidades de paralelismo

- T003 (Fase 2) y T007 (Fase 3) se pueden escribir a la vez.
- T012–T013 mientras se implementa T010.
- T018, T019 y T020 (archivos distintos, después de T017).
- T022 y T023.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T018, T019, T022, T024, T025 |
| FR-002 | T003, T005, T008, T010, T019 |
| FR-003 | T003, T008, T010, T019 |
| FR-004 | T003, T008, T012, T019 |
| FR-005 | T008, T019, T020 |
| FR-006 | T012, T015, T019, T020 |
| FR-007 | T019, T024, T025 |
| FR-008 | T012, T015, T019 |
| FR-009 | T003, T012, T019 |
| FR-010 | T003, T008, T012, T013, T018, T019 |
| FR-011 | T008, T012, T019 |
| FR-012 | T008, T012, T019 |
| FR-013 | T008, T013 |
| FR-014 | T019 |
| FR-015 | T008, T012, T019 |
| FR-016 | T026 (`TestHU05a_ErrorAlLeer` sigue en verde con la lectura ampliada) |
| FR-017 | T019 |
| FR-018 | T011, T016, T026, T027 |
| FR-019 | T019, T023, T027 |
| SC-001 | T003, T008, T012, T019 |
| SC-002 | T008, T012, T019 |
| SC-003 | T008, T013 |
| SC-004 | T019 |
| SC-005 | T019 |
| SC-006 | T004, T009, T014, T021, T028 |

## Estrategia de implementación

1. T001: confirmar HU-05b en `main` (si no está, se espera) y crear la rama.
2. Bloque Reglas: el orden de los rechazos, sin base.
3. Bloques Nueva ronda y Estimación: las dos operaciones en PostgreSQL, con la concurrencia probada.
4. Bloque Pantalla: las acciones en "Rondas cerradas". Es el MVP visible para la Sprint Review.
5. Cierre, evidencias y PR.
