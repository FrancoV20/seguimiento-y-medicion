# Plan de implementación: HU-05a – Emisión de voto oculto en Planning Poker

**Rama**: `HU-05a` | **Fecha**: 2026-10-27 | **Spec**: [spec.md](spec.md)
**Constitución**: v1.1.0

## Resumen

Se crea el paquete de dominio `src/poker` con la serie de votación, las reglas de apertura y de voto, los
errores, el puerto `Store`, el `Service` y el adaptador `PostgreSQLStore`. Una migración nueva (`011`) crea
`poker_rounds` y `poker_votes`. En `src/web` se agrega la sección "Planning Poker" (`/planning-poker`, con
proyecto e integrante requeridos): un catálogo de solo lectura (`PokerCatalog` + `PostgreSQLPokerCatalog`), los
handlers de `GET /planning-poker`, `POST /planning-poker/abrir` y `POST /planning-poker/votar`, los textos en un
único archivo y la plantilla `poker.html`. El modelo de vista no tiene ningún campo para valores de votos ajenos.
Solo se edita `src/cmd/server/main.go` para registrar la sección.

## Contexto técnico

- **Lenguaje**: Go 1.27.1.
- **Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`) y
  `github.com/jackc/pgx/v5/stdlib`, ya incorporado. Sin dependencias nuevas.
- **Almacenamiento**: PostgreSQL (Docker Compose). Migración `011_create_poker` (ver `data-model.md`).
- **Pruebas**: `go test -count=1 -p 1 ./...`. Reglas y servicio con un almacén de prueba; adaptador y catálogo
  contra PostgreSQL real (se omiten sin `DATABASE_URL`, como el resto del repo); handlers con `httptest` y
  `cookiejar` (proyecto e integrante elegidos por el flujo de HU-20b); BDD en `test/web_poker_bdd_test.go`.
- **Plataforma**: aplicación SSR multiplataforma; desarrollo en Windows con Docker Desktop.
- **Rendimiento**: la pantalla responde en menos de 500 ms (SC-007).
- **Restricciones**: sin JavaScript; sin token en formularios (`CrossOriginGuard`); proyecto e integrante siempre
  del contexto del servidor; ningún detalle interno en las respuestas; no se usa `gofmt -w` sobre archivos
  existentes (CRLF), solo sobre los nuevos.
- **Alcance**: una pantalla, tres rutas, dos tablas.

## Verificación de la constitución

*Antes de Phase 0 y otra vez después del diseño.*

| Principio | Cumplimiento |
|---|---|
| I. Alcance guiado por valor | PASS. Cubre la estimación oculta del README; revelar, nueva ronda y estimación acordada quedan en HU-05b y HU-18 |
| II. Calidad por construcción | PASS. Reglas puras en `src/poker`, adaptador separado, catálogo de lectura y handlers en `src/web`; la serie se comprueba contra HU-02 sin duplicar su validación en la web |
| III. Test-first | PASS. Cada bloque de `tasks.md` escribe las pruebas, guarda `evidencia-red.txt` y recién después implementa; escenarios en Dado/Cuando/Entonces |
| IV. Ejecución ágil | PASS. Cuatro bloques demostrables; el Bloque A + B es el MVP |
| V. Medición | PASS. Los votos guardados son la base de la estimación acordada (HU-18) y de la velocidad |
| VI. PostgreSQL | PASS. `poker.Store` → `PostgreSQLStore` y `web.PokerCatalog` → `PostgreSQLPokerCatalog`, ambos con pruebas de integración contra PostgreSQL; migración probada en ambos sentidos |
| Restricciones técnicas | PASS. Go + PostgreSQL, sin dependencias nuevas |
| Otras historias | PASS. Solo se edita `src/cmd/server/main.go`; `src/backlog` se usa en una prueba sin modificarse |

## Estructura de archivos

### Documentación

```text
specs/005a-poker-voting/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML; las direcciones, campos y códigos están en
`spec.md` (Sección SDD) y abajo.

### Código

Nuevos:

```text
db/migrations/
├── 011_create_poker.up.sql
└── 011_create_poker.down.sql
src/poker/
├── series.go                              # Series, InSeries, ParseValue
├── round.go                               # RoundStatus, CheckOpen, CheckVote, OpenedRound, VoteOutcome
├── errors.go                              # Err… del dominio
├── store.go                               # puerto Store
├── service.go                             # Service
├── postgres_store.go                      # PostgreSQLStore (transacciones de data-model.md)
├── series_test.go                         # incluye TestSeries_CoincideConStoryPoints (usa src/backlog)
├── round_test.go
├── service_test.go                        # con un Store de prueba
├── fixtures_integration_test.go           # proyectos, integrantes, historias, limpieza
└── postgres_store_integration_test.go
src/web/
├── poker_messages.go                      # TitlePoker y MsgPoker…, único archivo de textos
├── poker_catalog.go                       # PokerCatalog y tipos de vista
├── postgres_poker_catalog.go              # PostgreSQLPokerCatalog
├── poker_form.go                          # lectura de `historia`, `ronda`, `valor` y traducción de errores
├── poker_handlers.go                      # RegisterPlanningPoker, show, open, vote
├── templates/poker.html
├── poker_catalog_test.go                  # el modelo de vista no tiene campo para valores ajenos
├── poker_handlers_test.go                 # escenarios por handler
├── poker_view_test.go                     # ocultamiento y estados vacíos
└── postgres_poker_catalog_integration_test.go
test/
└── web_poker_bdd_test.go
```

Editado:

- `src/cmd/server/main.go`: construir `poker.NewService(poker.NewPostgreSQLStore(database))` y
  `web.NewPostgreSQLPokerCatalog(database)` y llamar a `web.RegisterPlanningPoker`, igual que Backlog y Tablero.

## Direcciones y campos

| Método | Dirección | Campos | Éxito | Rechazos |
|---|---|---|---|---|
| GET | `/planning-poker` | — | 200 | lectura fallida → `Fail` |
| POST | `/planning-poker/abrir` | `historia` | 303 a `/planning-poker` + "Ronda abierta para la historia «…»" | 404, 409, 500 |
| POST | `/planning-poker/votar` | `ronda`, `valor` | 303 a `/planning-poker` + "Voto registrado" / "Voto actualizado" | 422, 404, 409, 500 |

Sección: `Section{Name: TitlePoker ("Planning Poker"), Path: "/planning-poker", RequiresProject: true,
RequiresMember: true}`. Sin integrante, `RequireContext` responde 303 a `/contexto/integrante?volver_a=/planning-poker`
antes de llegar al handler (escenario 12).

## Decisiones de diseño

Detalle y alternativas descartadas en `research.md`.

1. **Reglas puras**: `ParseValue`, `CheckOpen(statuses)` y `CheckVote(status)` sin base de datos; el adaptador las
   llama dentro de la transacción.
2. **Puerto de escritura** `poker.Store` (`OpenRound`, `CastVote`) y **catálogo de lectura** `web.PokerCatalog`
   (`PokerBoard`), siguiendo HU-14.
3. **Abrir**: historia `FOR NO KEY UPDATE` → rondas → `CheckOpen` → INSERT ronda 1; índice único parcial como
   segunda defensa.
4. **Votar**: ronda `FOR SHARE` → `CheckVote` → upsert con CTE que distingue alta y cambio.
5. **Ocultamiento por construcción**: la consulta de participantes devuelve `EXISTS`, nunca `value`; el voto
   propio se consulta filtrando por el integrante identificado; `PokerParticipantView` no tiene campo de valor.
6. **Interfaz del lado web** `PokerActions` (`OpenRound`, `Vote`), cumplida por `*poker.Service`, para que los
   handlers dependan de un contrato pequeño.
7. **Traducción de errores** en `poker_form.go` (`pokerOutcomeFor(err) (status int, text string)`):

   | Error | Código | Texto |
   |---|---|---|
   | `ErrValueRequired` | 422 | `MsgPokerSelectValue` "Seleccioná un valor de la serie" |
   | `ErrValueNotInSeries` | 422 | `MsgPokerValueNotInSeries` "El valor no pertenece a la serie de Fibonacci" |
   | `ErrRoundClosed` | 409 | `MsgPokerRoundClosed` "La ronda ya está cerrada" |
   | `ErrRoundNotFound` | 404 | `MsgPokerRoundNotFound` "La ronda no existe" |
   | `ErrStoryNotFound` | 404 | `MsgPokerStoryNotFound` "La historia no existe" |
   | `ErrRoundAlreadyOpen` | 409 | `MsgPokerRoundAlreadyOpen` "Ya hay una ronda abierta para esta historia" |
   | `ErrStoryAlreadyRounded` | 409 | `MsgPokerStoryAlreadyRounded` "La historia ya tuvo una ronda de votación" |
   | cualquier otro (incluido `ErrMemberRequired`) | 500 | `MsgOperationFailed`; el detalle va a `logger.Error` |

8. **Rechazos con la pantalla**: en 404, 409, 422 y 500 se vuelve a armar la pantalla con `PageStatus` y
   `Flash{Kind: FlashError}`. Si esa lectura falla, `Renderer.Fail`.
9. **Éxitos**: `SetFlash(FlashSuccess, …)` + `RedirectSeeOther("/planning-poker")`. El texto de apertura se arma con
   `pokerRoundOpenedMessage(title)` en `poker_messages.go`.
10. **Textos de vista**: `MsgPokerNoStories` "Aún no hay historias", `MsgPokerNoOpenRounds` "No hay rondas abiertas",
    `LabelPokerVoted` "Votó", `LabelPokerNotVoted` "Sin votar", `MsgPokerVoteRegistered` "Voto registrado",
    `MsgPokerVoteUpdated` "Voto actualizado".
11. **Plantilla**: formulario "Abrir ronda" (selector `historia` con título y estado, oculto si no hay historias);
    por cada ronda abierta, título, "Ronda N", lista de integrantes con `data-integrante` y "Votó"/"Sin votar",
    "Tu voto: N" solo en la fila propia y un formulario con `ronda` oculto y diez radios `valor` (marcado solo el
    voto propio). Sin JavaScript.

## Flujo del POST `/planning-poker/votar`

1. `RequireContext` garantiza proyecto e integrante (si no, 303 de HU-20b).
2. Leer `ronda` con `parseID` y `valor` como texto.
3. `Service.Vote(ctx, projectID, roundID, memberID, valor)`: `ParseValue` → (`roundID == 0` → `ErrRoundNotFound`)
   → `Store.CastVote`.
4. Error → `pokerOutcomeFor` → pantalla con su código y mensaje.
5. Éxito → `SetFlash` con "Voto registrado" o "Voto actualizado" según `VoteOutcome.Updated` → 303.

`POST /planning-poker/abrir` es igual con `historia` y `Service.OpenRound`.

## Contratos que consume

- HU-20a: `Server.Register`, `Renderer.PageStatus`, `Renderer.Fail`, layout y parcial `alert`.
- HU-20b: `RequireContext` (vía `Section`), `ActiveContextFrom`, `SetFlash`, `RedirectSeeOther`, `parseID`,
  `MsgOperationFailed`, `CrossOriginGuard`.
- HU-01: `members`, `project_members`, `projects`. HU-02: `user_stories` y `backlog.ValidateStoryPoints` (solo en
  una prueba).

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Número de migración: 011 llega a `main` antes que 009/010 y las bases locales ya migradas no aplican 009/010 | T002 y T080 confirman los números en `origin/main` y avisan al equipo; si hace falta, se renumera la migración propia (ver `research.md`, decisión 1) |
| `main` local desactualizada (le falta HU-14) | T001 hace `git pull` de `main` antes de crear la rama |
| Filtrar un voto ajeno por un error de plantilla | El dato no llega a la vista (decisión 5) y la prueba de página idéntica lo detecta |
| Choque de nombres en `src/web` | Prefijo `Poker`/`poker_`; T003 hace `git grep -n -i poker` |
| Fixtures que no se pueden borrar (referencias sin cascada a `members`) | Orden de limpieza: historias → proyectos → integrantes (`data-model.md`) |
| El CTE del upsert no se comporta como se espera | `TestPostgresCastVote_Nuevo` y `TestPostgresCastVote_Reemplaza` lo verifican contra PostgreSQL antes del GREEN de la pantalla |

## Complexity Tracking

Sin violaciones de la constitución que justificar.

## Verificación posterior al diseño

- I–V: PASS (sin cambios respecto de la verificación inicial).
- VI: PASS. Dos interfaces de almacenamiento, ambas con adaptador PostgreSQL y prueba de integración.
- Ningún `NEEDS CLARIFICATION` pendiente (`research.md`, "Preguntas abiertas": ninguna).
