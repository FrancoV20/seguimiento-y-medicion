# Tareas: HU-05a – Emisión de voto oculto en Planning Poker

**Entrada**: `specs/005a-poker-voting/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-01, HU-02, HU-20a y HU-20b en `main` (verificado el 27/10/2026).

Formato: `[ID] [P] [E#] descripción`
- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario de la spec que cubre (E8a/E8b = escenario 8, ronda / historia).
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se guarda
  en `specs/005a-poker-voting/evidencia-red.txt` (una sección por bloque).
- Antes de cada ejecución en PowerShell: `chcp 65001`. Pruebas siempre con `go test -count=1 -p 1`.
- No usar `gofmt -w` sobre archivos existentes (CRLF); formatear solo los archivos nuevos.
- Cada tarea entra en un día o menos.

## Fase 1: Preparación

- [x] T001 Crear la rama `HU-05a` desde `main` actualizado: `git switch main; git pull` (la `main` local está
  detrás de `origin/main`, falta HU-14); como `HU-05a` ya existe localmente, confirmar con `git log main..HU-05a`
  que no tiene commits propios y hacer `git switch HU-05a; git merge --ff-only origin/main` (si no existiera:
  `git switch -c HU-05a`)
- [x] T002 `git fetch --all` y confirmar en `origin/main` y en las ramas remotas el último número de migración
  (`git ls-tree --name-only <rama> db/migrations/`): 009 y 010 siguen reservadas para HU-06 y HU-15 y nadie usa
  011; anotar el resultado en `research.md` (sección "Datos verificados") y, si cambió, ajustar el número en
  `plan.md`, `data-model.md` y este archivo
- [x] T003 [P] `git grep -n -i poker -- src` sin coincidencias; confirmar en `origin/main` los contratos listados
  en `research.md` (`RequireContext`, `parseID`, `PageStatus`, `Fail`, `SetFlash`, `MsgOperationFailed`, parcial
  `alert`) y la firma de `backlog.ValidateStoryPoints(*int) error`
- [x] T004 Línea base: `DATABASE_URL` definida, `migrate … up` aplicado y `go test -count=1 -p 1 ./...` en verde
- [ ] T005 [P] Avisar al equipo que HU-05a toma la migración 011 y edita `src/cmd/server/main.go` (coordinar con
  quien tenga HU-06 y HU-15, que también lo editan)

## Fase 2: Fundamentos (dominio, migración y almacenamiento; bloquea todo lo demás)

### Pruebas (RED)

- [x] T006 [P] [E2, E5, E6] En `src/poker/series_test.go`: `TestParseValue_Serie` (cada uno de 1, 2, 3, 5, 8, 13,
  21, 34, 55, 89 devuelve su número), `TestParseValue_Vacio` ("" → `ErrValueRequired`) y
  `TestParseValue_FueraDeSerie` (0, 4, 90, -1, "abc", "05", " 5", "5.0" → `ErrValueNotInSeries`)
- [x] T007 [P] [E5] `TestSeries_CoincideConStoryPoints` en `src/poker/series_test.go`: `poker.Series` tiene 10
  valores en orden creciente; `backlog.ValidateStoryPoints(&v)` devuelve nil para cada uno y
  `errors.Is(err, backlog.ErrInvalidStoryPoints)` para 0, 4 y 90 (sin editar `src/backlog`)
- [x] T008 [P] [E1, E7, E9, E10] En `src/poker/round_test.go`: `TestCheckOpen_SinRondas` (nil),
  `TestCheckOpen_Abierta` (`ErrRoundAlreadyOpen`, también si además hay una cerrada), `TestCheckOpen_Cerrada`
  (`ErrStoryAlreadyRounded`), `TestCheckVote_Abierta` (nil) y `TestCheckVote_Cerrada` (`ErrRoundClosed`)
- [x] T009 [P] [E1, E2, E3, E8a, E8b, E11] En `src/poker/service_test.go`, con un `Store` de prueba:
  `TestService_OpenRound_OK` (pasa proyecto, historia e integrante al almacén), `storyID == 0` →
  `ErrStoryNotFound` sin llamar al almacén, `memberID == 0` → `ErrMemberRequired`; `TestService_Vote_*`: valor
  vacío → `ErrValueRequired` antes del almacén, `roundID == 0` → `ErrRoundNotFound`, devuelve `Updated` del
  almacén y propaga sus errores sin envolverlos de forma que `errors.Is` falle
- [x] T010 Fixtures de integración en `src/poker/fixtures_integration_test.go`: `openTestDB` (se omite sin
  `DATABASE_URL`), proyectos A y B, integrantes Ana y Beto (en A) y Caro (en B) con `project_members`, historias
  en A («Login» "Pendiente" sin Story Points, «Pagos» "Hecho" con 5 SP) y en B, `insertClosedRound(storyID)`
  (`status = 'Cerrada', closed_at = now()`); limpieza: historias → proyectos → integrantes
- [x] T011 [P] `TestMigracionPoker_Restricciones` en `src/poker/postgres_store_integration_test.go`: la base
  rechaza `status` distinto de `'Abierta'`/`'Cerrada'`, una ronda `'Cerrada'` con `closed_at` NULL, una
  `'Abierta'` con `closed_at`, dos rondas `'Abierta'` de la misma historia, `(story_id, round_number)` repetido y
  un voto con `value` 4
- [x] T012 [P] [E1] `TestPostgresOpenRound_OK` (ronda 1, `'Abierta'`, `opened_by` = Ana, `closed_at` NULL,
  `project_id` = A, título devuelto) y `TestPostgresOpenRound_NoModificaHistoria` (título, estado y
  `story_points` de «Pagos» sin cambios; una historia "Hecho" con SP se puede abrir)
- [x] T013 [P] [E8b] `TestPostgresOpenRound_Inexistente` y `TestPostgresOpenRound_OtroProyecto` →
  `ErrStoryNotFound` y ninguna fila en `poker_rounds`
- [x] T014 [P] [E9, E10] `TestPostgresOpenRound_ConRondaAbierta` → `ErrRoundAlreadyOpen` y
  `TestPostgresOpenRound_YaTuvoRonda` (con `insertClosedRound`) → `ErrStoryAlreadyRounded`; la cantidad de rondas
  no cambia
- [x] T015 [P] [E9] `TestPostgresOpenRound_Concurrente`: dos `OpenRound` simultáneos (goroutines con una barrera)
  sobre «Login» → uno OK y otro `ErrRoundAlreadyOpen`; queda una sola ronda. Y
  `TestPostgresCastVote_EsperaAlCierre`: una transacción de prueba toma la ronda con `FOR UPDATE` y la cierra
  (como hará HU-05b); un `CastVote` concurrente espera al commit y termina en `ErrRoundClosed` sin dejar voto
- [x] T016 [P] [E2, E3] `TestPostgresCastVote_Nuevo` (`Updated == false`, una fila con el valor),
  `TestPostgresCastVote_Reemplaza` (`Updated == true`, sigue una fila, valor nuevo, `updated_at` posterior) y
  `TestPostgresCastVote_DosIntegrantes` (Ana y Beto votan a la vez y quedan dos votos)
- [x] T017 [P] [E7, E8a] `TestPostgresCastVote_RondaCerrada` (`ErrRoundClosed`),
  `TestPostgresCastVote_Inexistente` y `TestPostgresCastVote_OtroProyecto` (`ErrRoundNotFound`); los votos no
  cambian
- [x] T018 [P] [E11] `TestPostgresCastVote_FalloNoDejaCambios`: un voto con un integrante que no existe en
  `members` falla por la clave foránea, devuelve un error que no es del dominio y no deja filas; lo mismo para
  `OpenRound`
- [x] T019 [P] [E1, E13] En `src/web/postgres_poker_catalog_integration_test.go` (fixtures propios con
  `openContextTestDB`): `TestPostgresPokerCatalog_Historias` (todas las del proyecto, cualquier estado, ordenadas
  por id, ninguna de otro proyecto), `TestPostgresPokerCatalog_RondasAbiertas` (solo `'Abierta'` del proyecto, en
  orden de apertura) y `TestPostgresPokerCatalog_Vacio` (proyecto sin historias ni rondas)
- [x] T020 [P] [E2, E4] En el mismo archivo: `TestPostgresPokerCatalog_Participantes` (todos los integrantes del
  proyecto por nombre, `HasVoted` e `IsMe` correctos), `TestPostgresPokerCatalog_VotoPropio` (`OwnVote` del
  integrante pedido; nil si no votó) y `TestPostgresPokerCatalog_SinValoresAjenos` (Ana 13, Beto 3: el tablero de
  Beto tiene `OwnVote` 3 en la ronda y nada que contenga 13)
- [x] T021 [P] [E4] `TestPokerViewModel_SinCampoDeValorAjeno` en `src/web/poker_catalog_test.go`: con `reflect`,
  `PokerParticipantView` no tiene campos llamados `Value`/`Vote` ni numéricos distintos de `MemberID`
- [x] T022 Ejecutar T006–T021, confirmar que fallan porque el paquete, las tablas y el catálogo no existen y
  guardar la salida en `evidencia-red.txt` (sección "Fundamentos")

### Implementación (GREEN)

- [x] T023 Crear `db/migrations/011_create_poker.up.sql` y `.down.sql` con el SQL exacto de `data-model.md`
  (`status VARCHAR(20) NOT NULL DEFAULT 'Abierta' CHECK (status IN ('Abierta', 'Cerrada'))`,
  `CHECK ((status = 'Abierta') = (closed_at IS NULL))`, `UNIQUE (story_id, round_number)`, índice único parcial
  `(story_id) WHERE status = 'Abierta'`, `value INTEGER NOT NULL CHECK (value IN (1, 2, 3, 5, 8, 13, 21, 34, 55,
  89))`, `PRIMARY KEY (round_id, member_id)`); probar `up`, `down 1`, `up` y `go test -count=1 -p 1 ./db/...`
- [x] T024 [P] Crear `ErrValueRequired`, `ErrValueNotInSeries`, `ErrRoundClosed`, `ErrRoundNotFound`,
  `ErrStoryNotFound`, `ErrRoundAlreadyOpen`, `ErrStoryAlreadyRounded` y `ErrMemberRequired` en
  `src/poker/errors.go`
- [x] T025 [P] Implementar `Series`, `InSeries` y `ParseValue` (texto canónico: `strconv.Itoa(n) == raw`) en
  `src/poker/series.go`
- [x] T026 [P] Implementar `RoundStatus` (`StatusOpen = "Abierta"`, `StatusClosed = "Cerrada"`), `CheckOpen`,
  `CheckVote`, `OpenedRound` y `VoteOutcome` en `src/poker/round.go`
- [x] T027 Definir el puerto `Store` en `src/poker/store.go` y `Service`/`NewService` en `src/poker/service.go`
  (validaciones de `data-model.md`, sección "Tipos de dominio")
- [x] T028 Implementar `PostgreSQLStore.OpenRound` en `src/poker/postgres_store.go`: transacción de
  `data-model.md` (historia `FOR NO KEY UPDATE`, rondas, `CheckOpen`, INSERT de la ronda 1) y traducción de
  `23505` sobre `uq_poker_rounds_one_open_per_story` a `ErrRoundAlreadyOpen` y sobre
  `uq_poker_rounds_story_number` a `ErrStoryAlreadyRounded`
- [x] T029 Implementar `PostgreSQLStore.CastVote` en `src/poker/postgres_store.go`: ronda `FOR SHARE`,
  `CheckVote` y el upsert con CTE `previous` de `data-model.md`
- [x] T030 Crear los tipos de vista y la interfaz `PokerCatalog` en `src/web/poker_catalog.go` y
  `PostgreSQLPokerCatalog` en `src/web/postgres_poker_catalog.go` con las cuatro lecturas de `data-model.md` en
  una transacción `READ ONLY ISOLATION LEVEL REPEATABLE READ`; la consulta de participantes no lee
  `poker_votes.value`
- [x] T031 Ejecutar T006–T021 en verde y `go test -count=1 -p 1 ./...` completo

**Punto de control**: el dominio, la migración y las lecturas funcionan solos, sin pantalla.

## Fase 3: Bloque A – Abrir ronda (escenarios 1, 9 y 10) 🎯 MVP (parte 1)

**Objetivo**: abrir la ronda 1 de una historia desde la pantalla y ver los rechazos de apertura.

**Prueba independiente**: elegir una historia, abrir la ronda y verla con todos los integrantes "Sin votar";
volver a abrirla y ver el 409.

### Pruebas (RED)

- [ ] T032 Preparar el servidor de prueba en `src/web/poker_handlers_test.go`: servidor real con
  `CrossOriginGuard`, `RegisterContext`, `poker.NewService(poker.NewPostgreSQLStore(db))` y
  `NewPostgreSQLPokerCatalog(db)`; fixtures (proyecto con Ana y Beto, «Login», «Pagos», otro proyecto); cliente
  con `cookiejar` que elige proyecto e integrante por el flujo de HU-20b; ayudante para un `poker.Store` que falla
- [ ] T033 [P] [E1] `TestHU05a_AbrirRonda`: `POST /planning-poker/abrir` con `historia` → 303 a
  `/planning-poker`; el GET muestra una sola vez "Ronda abierta para la historia «Login»", "Ronda 1" y a Ana y
  Beto "Sin votar"; en la base `opened_by` es el integrante identificado aunque se envíe un campo `integrante`
  con otro id
- [ ] T034 [P] [E9] `TestHU05a_HistoriaConRondaAbierta`: 409, "Ya hay una ronda abierta para esta historia" y una
  sola ronda
- [ ] T035 [P] [E10] `TestHU05a_HistoriaYaTuvoRonda`: con una ronda cerrada preparada en la base, 409, "La
  historia ya tuvo una ronda de votación" y ninguna ronda nueva
- [ ] T036 [P] [E1] `TestHU05a_SelectorTodasLasHistorias`: el selector lista «Login» y «Pagos» con su estado
  (también después de abrir la ronda de «Login») y ninguna historia de otro proyecto
- [ ] T037 [P] [E1, E9] `TestBDD_HU05a_AbrirRonda` y `TestBDD_HU05a_HistoriaConRondaAbierta` en
  `test/web_poker_bdd_test.go` (Dado/Cuando/Entonces como comentarios, igual que `test/web_backlog_bdd_test.go`)
- [ ] T038 Ejecutar T033–T037, confirmar que fallan (la sección no existe) y guardar la salida en
  `evidencia-red.txt` (sección "Abrir ronda")

### Implementación (GREEN)

- [ ] T039 [P] Crear `src/web/poker_messages.go` con `TitlePoker` "Planning Poker", `MsgPokerVoteRegistered` "Voto
  registrado", `MsgPokerVoteUpdated` "Voto actualizado", `MsgPokerSelectValue` "Seleccioná un valor de la serie",
  `MsgPokerValueNotInSeries` "El valor no pertenece a la serie de Fibonacci", `MsgPokerRoundClosed` "La ronda ya
  está cerrada", `MsgPokerRoundNotFound` "La ronda no existe", `MsgPokerStoryNotFound` "La historia no existe",
  `MsgPokerRoundAlreadyOpen` "Ya hay una ronda abierta para esta historia", `MsgPokerStoryAlreadyRounded` "La
  historia ya tuvo una ronda de votación", `MsgPokerNoStories` "Aún no hay historias", `MsgPokerNoOpenRounds` "No
  hay rondas abiertas", `LabelPokerVoted` "Votó", `LabelPokerNotVoted` "Sin votar" y
  `pokerRoundOpenedMessage(title)` → "Ronda abierta para la historia «title»"
- [ ] T040 [P] Crear `src/web/poker_form.go`: lectura de `historia`, `ronda` (con `parseID`) y `valor` (texto), y
  `pokerOutcomeFor(err) (status int, text string)` con la tabla de `plan.md` (decisión 7)
- [ ] T041 Crear `src/web/poker_handlers.go`: `PokerActions`, `RegisterPlanningPoker(s, actions, catalog)` con
  `Section{Name: TitlePoker, Path: "/planning-poker", RequiresProject: true, RequiresMember: true}`, `GET
  /planning-poker` (`show`), `POST /planning-poker/abrir` (`open`) y `render(w, r, status, flash)` que arma la
  vista con `PageStatus` y usa `Fail` si la lectura falla; éxito con `SetFlash` + `RedirectSeeOther`
- [ ] T042 Crear `src/web/templates/poker.html`: formulario "Abrir ronda" (selector `historia` con título y
  estado) y, por ronda abierta, título, "Ronda N" y la lista de integrantes (`data-integrante="{{.MemberID}}"`,
  "Votó"/"Sin votar")
- [ ] T043 Editar `src/cmd/server/main.go`: construir el servicio y el catálogo y llamar a
  `web.RegisterPlanningPoker` después del Tablero, con el mismo manejo de error (sin reformatear el archivo)
- [ ] T044 Ejecutar T033–T037 en verde y `go test -count=1 -p 1 ./...` completo

## Fase 4: Bloque B – Votar y cambiar voto (escenarios 2, 3 y 4) 🎯 MVP (parte 2)

**Objetivo**: votar y cambiar el voto en una ronda abierta, sin que nadie vea los valores ajenos.

**Prueba independiente**: votar 5 y luego 8 y ver "Tu voto: 8"; con otro integrante que votó, ver solo "Votó".

### Pruebas (RED)

- [ ] T045 [P] [E2] `TestHU05a_VotoRegistrado` en `src/web/poker_handlers_test.go`: `POST /planning-poker/votar`
  con `ronda` y `valor=5` → 303; el GET muestra "Voto registrado" una vez, la fila propia "Votó" y "Tu voto: 5";
  en la base hay un voto del integrante identificado aunque se envíe un campo `integrante` con otro id
- [ ] T046 [P] [E3] `TestHU05a_VotoActualizado`: votar 5 y después 8 → "Voto actualizado", "Tu voto: 8" y una sola
  fila en `poker_votes` para el integrante
- [ ] T047 [P] [E4] `TestHU05a_VotosAjenosOcultos` en `src/web/poker_view_test.go` (Ana 13, Beto 3; página de
  Beto): (1) la fila `data-integrante` de Ana dice "Votó" y no contiene "13", la de Beto muestra 3; (2) el único
  `checked` es el valor 3; (3) Ana cambia a 21 y la página de Beto es idéntica byte a byte a la anterior
- [ ] T048 [P] [E4] `TestHU05a_CambioDeIdentificacion` en `src/web/poker_view_test.go`: al identificarse como Ana,
  la pantalla muestra "Tu voto: 13" y no el 3 de Beto
- [ ] T049 [P] [E2, E3, E4] `TestBDD_HU05a_VotoOculto`, `TestBDD_HU05a_CambioDeVoto` y
  `TestBDD_HU05a_VotosOcultos` en `test/web_poker_bdd_test.go`
- [ ] T050 Ejecutar T045–T049, confirmar que fallan (la ruta de voto no existe) y guardar la salida en
  `evidencia-red.txt` (sección "Votar")

### Implementación (GREEN)

- [ ] T051 Agregar `POST /planning-poker/votar` (`vote`) en `src/web/poker_handlers.go`: `Service.Vote` con
  proyecto e integrante de `ActiveContextFrom`; mensaje según `VoteOutcome.Updated`
- [ ] T052 Completar `src/web/templates/poker.html`: "Tu voto: N" solo en la fila con `IsMe`; formulario con
  `ronda` oculto, diez radios `valor` de `poker.Series` y `checked` solo en `OwnVote`
- [ ] T053 Ejecutar T045–T049 en verde

**Punto de control**: con los Bloques A y B el equipo ya puede votar de forma oculta (MVP); HU-05b puede empezar.

## Fase 5: Bloque C – Errores y estados (escenarios 5, 6, 7, 8, 11 y 12)

**Objetivo**: cada rechazo con su código y su mensaje, sin modificar datos.

**Prueba independiente**: enviar un valor 4, un voto vacío, un voto a una ronda cerrada y a una inexistente, y
forzar una falla de guardado.

### Pruebas (RED)

- [ ] T054 [P] [E5] `TestHU05a_ValorFueraDeSerie` en `src/web/poker_handlers_test.go`: `valor` 4, 0, 90 y "abc" →
  422 y "El valor no pertenece a la serie de Fibonacci"; un voto anterior de 5 se conserva
- [ ] T055 [P] [E6] `TestHU05a_VotoSinValor`: sin `valor` → 422 y "Seleccioná un valor de la serie"; nada
  registrado
- [ ] T056 [P] [E7] `TestHU05a_RondaCerrada`: ronda cerrada en la base → 409 y "La ronda ya está cerrada"; los
  votos no cambian
- [ ] T057 [P] [E8a, E8b] `TestHU05a_RondaInexistente` y `TestHU05a_HistoriaInexistente`: identificador
  inexistente, vacío, no numérico, "05" y de otro proyecto → 404 con "La ronda no existe" / "La historia no
  existe"; nada cambia
- [ ] T058 [P] [E11] `TestHU05a_ErrorAlGuardar`: con un `poker.Store` que falla detrás del `Service` real, votar y
  abrir → 500 y "La operación no pudo completarse", sin mensaje de éxito; el cuerpo no contiene "poker_", "SQL",
  "pgx" ni el texto del error
- [ ] T059 [P] [E12] `TestHU05a_SinIdentificarme`: con proyecto y sin integrante, `POST /planning-poker/votar` → 303
  a `/contexto/integrante?volver_a=%2Fplanning-poker`, ningún voto; al identificarse, el regreso lleva a
  `/planning-poker`
- [ ] T060 [P] [E7, E12] `TestBDD_HU05a_RondaCerrada` y `TestBDD_HU05a_SinIdentificarme` en
  `test/web_poker_bdd_test.go`
- [ ] T061 Ejecutar T054–T060, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección "Errores y
  estados")

### Implementación (GREEN)

- [ ] T062 Completar en `src/web/poker_handlers.go` los rechazos: `pokerOutcomeFor` → `render` con su código y
  `Flash{Kind: FlashError}`; 500 con `MsgOperationFailed` y `logger.Error` con el detalle; ajustar
  `poker_form.go` si alguna prueba lo pide
- [ ] T063 Ejecutar T054–T060 en verde

## Fase 6: Bloque D – Pantalla y estados vacíos (escenario 13)

**Objetivo**: la pantalla se ve bien sin datos, escapa los textos y responde rápido.

**Prueba independiente**: abrir Planning Poker en un proyecto sin historias y en uno con historias sin rondas.

### Pruebas (RED)

- [ ] T064 [P] [E13] `TestHU05a_PantallaVacia` en `src/web/poker_view_test.go`: proyecto sin historias → "Aún no
  hay historias" y "No hay rondas abiertas", sin formulario de apertura; con historias y sin rondas → selector y
  "No hay rondas abiertas"
- [ ] T065 [P] `TestHU05a_TituloEscapado`: un título y un nombre de integrante con `<script>` se ven como texto, en
  la página y en el mensaje de apertura
- [ ] T066 [P] `TestHU05a_ErrorAlLeer`: con un `PokerCatalog` que falla, la página de error de HU-20a (500, o 503 si
  la base no responde) y nada de la pantalla
- [ ] T067 [P] [E2] `TestHU05a_RecargaSinReenviar`: después de votar, dos GET seguidos → el mensaje aparece una sola
  vez y hay un solo voto
- [ ] T068 [P] `TestHU05a_Rendimiento`: con 50 historias, 20 rondas abiertas y 10 integrantes, `GET
  /planning-poker` responde en menos de 500 ms con la base local (SC-007)
- [ ] T069 [P] [E13] `TestBDD_HU05a_PantallaVacia` en `test/web_poker_bdd_test.go`
- [ ] T070 Ejecutar T064–T069, confirmar que fallan y guardar la salida en `evidencia-red.txt` (sección
  "Pantalla")

### Implementación (GREEN)

- [ ] T071 Completar `src/web/templates/poker.html` con los estados vacíos (`MsgPokerNoStories`,
  `MsgPokerNoOpenRounds`) y ocultar el formulario de apertura sin historias
- [ ] T072 Ajustar `render` en `src/web/poker_handlers.go` si T066 lo requiere (lectura fallida → `Fail`, sin
  escribir la respuesta a medias)
- [ ] T073 Ejecutar T064–T069 en verde

## Fase 7: Cierre

- [ ] T074 Refactor sin cambiar comportamiento: duplicación entre `open` y `vote`, nombres y comentarios de
  `src/poker` y `src/web/poker_*`; todas las pruebas en verde
- [ ] T075 [P] `gofmt -l` solo sobre los archivos nuevos (sin `-w` en existentes), `go vet ./...` limpio y pipeline
  de GitHub Actions en verde
- [ ] T076 Guardar `specs/005a-poker-voting/evidencia-pruebas.txt` con `go test -count=1 -p 1 -v ./...` completo en
  verde
- [ ] T077 Guardar `specs/005a-poker-voting/cobertura.txt` con `go test -count=1 -p 1 -cover ./...` y verificar al
  menos 80% en `src/poker`
- [ ] T078 Verificar a mano los 13 escenarios con la aplicación en ejecución (`quickstart.md`, "Verificación
  manual")
- [ ] T079 [P] Actualizar la tabla de trazabilidad de `spec.md` con los nombres reales de las pruebas y marcar
  `checklists/requirements.md`
- [ ] T080 Antes del PR, `git fetch --all` y volver a confirmar que 011 sigue libre y que 009/010 ya están en
  `main` o que el equipo acordó el orden de merge (riesgo de `research.md`, decisión 1); si no, renumerar solo la
  migración propia
- [ ] T081 Abrir el Pull Request desde `HU-05a` citando HU-05a, con evidencias, la migración y el único archivo
  ajeno editado (`src/cmd/server/main.go`)
- [ ] T082 Revisión y aprobación de otro integrante, merge con "Create a merge commit" y pasar la tarjeta a "Hecho"
- [ ] T083 Avisar a quien tenga HU-05b y HU-18 que las tablas y el servicio están en `main` (HU-05b cierra con
  `FOR UPDATE` y `status = 'Cerrada', closed_at = now()` en el mismo UPDATE)

## Dependencias y orden de ejecución

- **Fase 1** primero (T001 antes que todo). **Fase 2** depende de la Fase 1 y bloquea a las demás.
- Dentro de la Fase 2: T010 antes de T011–T018; T019–T020 necesitan la migración para pasar, no para escribirse.
  GREEN en orden T023 → T024–T026 → T027 → T028–T029 → T030.
- **Bloque A (Fase 3)** depende de la Fase 2. **Bloque B (Fase 4)** depende de T041–T042 (sección y plantilla
  base). A + B son el MVP.
- **Bloque C (Fase 5)** depende de T051 (ruta de voto); puede avanzar en paralelo con el Bloque D.
- **Bloque D (Fase 6)** depende de T042; puede avanzar en paralelo con el Bloque C.
- **Fase 7** depende de los bloques que se entreguen.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar evidencia → implementación (GREEN).
- Los bloques B, C y D editan los mismos archivos que A (`poker_handlers.go`, `poker.html`): si dos personas
  trabajan en paralelo, coordinar el orden de los cambios.

### Oportunidades de paralelismo

- T003 y T005.
- T006–T009 entre sí; T011–T018 (después de T010); T019–T021.
- T024–T026.
- T033–T037, T045–T049, T054–T060 y T064–T069 dentro de su bloque.
- T039 y T040.
- T075 y T079.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T041, T043, T059 |
| FR-002 | T019, T036, T042 |
| FR-003 | T020, T042, T045, T047, T052 |
| FR-004 | T012, T028, T033 |
| FR-005 | T009, T013, T057 |
| FR-006 | T008, T014, T034 |
| FR-007 | T008, T014, T035 |
| FR-008 | T016, T029, T045, T051, T067 |
| FR-009 | T016, T046 |
| FR-010 | T006, T009, T054, T055 |
| FR-011 | T008, T017, T056 |
| FR-012 | T009, T017, T057 |
| FR-013 | T020, T021, T030, T047, T048 |
| FR-014 | T018, T058, T062 |
| FR-015 | T066, T072 |
| FR-016 | T059 |
| FR-017 | T019, T064, T071 |
| FR-018 | T013, T017, T019, T033, T045 |
| FR-019 | T015, T028, T029 |
| FR-020 | T065 |
| FR-021 | T012 |
| FR-022 | T007, T025 |
| FR-023 | T058 |
| SC-001 | T012, T033 |
| SC-002 | T016, T046 |
| SC-003 | T020, T047 |
| SC-004 | T013, T014, T017, T054–T057 |
| SC-005 | T015 |
| SC-006 | T018, T058 |
| SC-007 | T068 |
| SC-008 | T022, T038, T050, T061, T070, T076, T077 |

## Estrategia de implementación

1. Fases 1 y 2: rama, migración, dominio, adaptador y catálogo con sus pruebas.
2. Bloques A y B: es el MVP; con ellos el equipo vota de forma oculta y HU-05b puede empezar.
3. Bloques C y D, en paralelo si hay manos libres.
4. Cierre, evidencias y PR.
