# Modelo de datos: HU-18 – Planning Poker: Nueva Ronda y Estimación Acordada

## Cambios de esquema

Ninguno. HU-18 usa tablas existentes:

| Tabla | Columnas usadas | Operación de HU-18 |
|---|---|---|
| `poker_rounds` (HU-05a) | `id`, `project_id`, `story_id`, `round_number`, `status`, `opened_by` | lee; inserta la ronda siguiente |
| `poker_votes` (HU-05a) | `round_id`, `member_id`, `value` | lee (votos revelados de HU-05b) |
| `user_stories` (HU-02) | `id`, `project_id`, `title`, `story_points` | bloquea; actualiza solo `story_points` |
| `sprint_story_assignments` (HU-04) | `sprint_id`, `story_id` | lee (Sprint de la historia; `UNIQUE (story_id)`) |
| `sprints` (HU-04) | `id`, `status` | lee con `FOR SHARE` (¿'Finalizado'?) |

`uq_poker_rounds_story_number UNIQUE (story_id, round_number)` y `uq_poker_rounds_one_open_per_story` (HU-05a) son la
segunda defensa de "Nueva ronda". El `CHECK (story_points IN (1, 2, 3, 5, 8, 13, 21, 34, 55, 89))` de `user_stories`
es la segunda defensa de la estimación.

## Estados

| Desde (ronda enviada) | Acción | Resultado | Condición |
|---|---|---|---|
| Última, cerrada | Nueva ronda | Ronda `número + 1` 'Abierta' | Sin rondas abiertas en la historia y sin Sprint finalizado |
| Cualquiera, con una ronda abierta en la historia | Nueva ronda | 409 `ErrRoundAlreadyOpen` | Gana sobre "no es la última" |
| Cerrada, no última (sin abiertas) | Nueva ronda / Estimación | 409 `ErrNotLatestRound` | — |
| Última, cerrada, Sprint finalizado | Nueva ronda / Estimación | 409 `ErrStoryInFinishedSprint` | — |
| Abierta | Estimación | 409 `ErrRoundStillOpen` | Antes que "no es la última" |
| Última, cerrada | Estimación | `story_points = valor` | Valor de la serie; sin Sprint o Sprint activo |
| Inexistente / otro proyecto | Nueva ronda / Estimación | 404 `ErrRoundNotFound` | — |

## Tipos de dominio nuevos (`src/poker`)

- `errors.go`:
  - `ErrNotLatestRound = errors.New("poker: hay una ronda más reciente de la historia")`
  - `ErrRoundStillOpen = errors.New("poker: la ronda sigue abierta")`
  - `ErrStoryInFinishedSprint = errors.New("poker: la historia pertenece a un Sprint finalizado")`
- `round.go` (junto a `CheckOpen`):

```go
// StoryRound es una ronda de la historia, para decidir la ronda siguiente.
type StoryRound struct {
    Number int
    Status RoundStatus
}

// CheckNextRound decide si se abre una ronda nueva desde la ronda número from.
// Orden: alguna abierta -> ErrRoundAlreadyOpen; from no es el mayor -> ErrNotLatestRound;
// Sprint finalizado -> ErrStoryInFinishedSprint. Si no, next = mayor + 1.
func CheckNextRound(from int, rounds []StoryRound, inFinishedSprint bool) (next int, err error)

// CheckAgree decide si se registra la estimación desde una ronda.
// Orden: abierta -> ErrRoundStillOpen; number != latest -> ErrNotLatestRound;
// Sprint finalizado -> ErrStoryInFinishedSprint.
func CheckAgree(status RoundStatus, number, latest int, inFinishedSprint bool) error

// AgreedEstimate es la estimación registrada.
type AgreedEstimate struct {
    StoryID     int64
    StoryTitle  string
    StoryPoints int
}
```

Casos de `TestCheckNextRound` y `TestCheckAgree` (datos del Issue):

| Caso | Entrada | Resultado |
|---|---|---|
| «Login», desde la 1 | from 1; {1 Cerrada}; sin Sprint | next 2 |
| «Login», desde la 2 (después del escenario 3) | from 2; {1 Cerrada, 2 Cerrada} | next 3 |
| «Login», desde la 1 con la 2 cerrada | from 1; {1 Cerrada, 2 Cerrada} | `ErrNotLatestRound` |
| «Reportes», desde la 1 (escenario 8) | from 1; {1 Cerrada, 2 Abierta} | `ErrRoundAlreadyOpen` |
| «Reportes», desde la 2 | from 2; {1 Cerrada, 2 Abierta} | `ErrRoundAlreadyOpen` |
| «Alta», desde la 1 | from 1; {1 Cerrada}; Sprint finalizado | `ErrStoryInFinishedSprint` |
| Estimación «Pagos» ronda 1 | Cerrada, 1, 1, no | nil |
| Estimación «Reportes» ronda 2 (escenario 7) | Abierta, 2, 2, no | `ErrRoundStillOpen` |
| Estimación «Reportes» ronda 1 (escenario 9) | Cerrada, 1, 2, no | `ErrNotLatestRound` |
| Estimación «Alta» ronda 1 (escenario 10) | Cerrada, 1, 1, sí | `ErrStoryInFinishedSprint` |
| Abierta y no última | Abierta, 1, 2, sí | `ErrRoundStillOpen` (primero) |

- Puerto `Store` (se agregan dos métodos):
  - `OpenNextRound(ctx context.Context, projectID, roundID, memberID int64) (OpenedRound, error)`
  - `AgreeEstimate(ctx context.Context, projectID, roundID int64, value int) (AgreedEstimate, error)`
- `Service`:
  - `OpenNextRound(ctx, projectID, roundID, memberID int64) (OpenedRound, error)`: `roundID == 0` →
    `ErrRoundNotFound`; `memberID == 0` → `ErrMemberRequired`; sin llamar al almacén.
  - `AgreeEstimate(ctx, projectID, roundID, memberID int64, rawValue string) (AgreedEstimate, error)`:
    `ParseValue(rawValue)` primero (422 antes que 404/409); `roundID == 0` → `ErrRoundNotFound`; `memberID == 0` →
    `ErrMemberRequired`; luego `store.AgreeEstimate(ctx, projectID, roundID, value)`. Errores sin envolver.

## Transacción "Nueva ronda" (`PostgreSQLStore.OpenNextRound`)

```sql
BEGIN;
-- 1. Ronda base del proyecto (bloqueo compartido: se serializa con la estimación y el cierre, que usan FOR UPDATE).
--    Sin fila => ErrRoundNotFound.
SELECT story_id, round_number FROM poker_rounds
 WHERE id = $1 AND project_id = $2
   FOR SHARE;

-- 2. Sprint de la historia, sin bloquear (para saber qué Sprint bloquear).
SELECT sprint_id FROM sprint_story_assignments WHERE story_id = $1;

-- 3. Solo si hay Sprint: bloquearlo antes que la historia (orden de HU-13 y HU-15).
SELECT status = 'Finalizado' FROM sprints WHERE id = $1 FOR SHARE;

-- 4. Historia bloqueada y relectura de la asignación.
SELECT us.title, a.sprint_id
  FROM user_stories us
  LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
 WHERE us.id = $1
   FOR NO KEY UPDATE OF us;
--    sprint_id distinto del paso 2 => ROLLBACK y repetir desde el paso 1 una vez; otra vez distinto => error (500).

-- 5. Rondas de la historia => CheckNextRound(base.round_number, rondas, inFinishedSprint).
SELECT round_number, status FROM poker_rounds WHERE story_id = $1;

-- 6. Ronda siguiente.
INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by)
VALUES ($2, $1, $3, 'Abierta', $4)
RETURNING id;
COMMIT;
```

- `23505` sobre `uq_poker_rounds_one_open_per_story` → `ErrRoundAlreadyOpen`; sobre `uq_poker_rounds_story_number` →
  `ErrNotLatestRound`.
- Devuelve `OpenedRound{RoundID: nueva, StoryID, StoryTitle, Number: next}`.

## Transacción "Estimación acordada" (`PostgreSQLStore.AgreeEstimate`)

```sql
BEGIN;
-- 1. Ronda del proyecto con bloqueo exclusivo. Sin fila => ErrRoundNotFound.
SELECT story_id, round_number, status FROM poker_rounds
 WHERE id = $1 AND project_id = $2
   FOR UPDATE;

-- 2. Última ronda de la historia.
SELECT max(round_number) FROM poker_rounds WHERE story_id = $1;

-- 3, 4 y 5: asignación, Sprint FOR SHARE y relectura con la historia FOR NO KEY UPDATE OF us (igual que "Nueva ronda").

-- 6. CheckAgree(status, round_number, max, inFinishedSprint).

-- 7. Story Points.
UPDATE user_stories SET story_points = $2 WHERE id = $1 RETURNING title;
COMMIT;
```

- Solo cambia `story_points`: ni `status`, ni `active_sprint_id`, ni rondas.
- Cualquier error revierte todo.

## Lectura de pantalla (adaptador `web.PostgreSQLPokerCatalog`)

Se amplía la consulta de rondas cerradas de HU-05b (`pokerClosedRoundsQuery`, T001: HU-05b) con dos columnas, en la
misma transacción `READ ONLY REPEATABLE READ`:

```sql
SELECT r.id, r.story_id, us.title, r.round_number,
       NOT EXISTS (SELECT 1 FROM poker_rounds n
                    WHERE n.story_id = r.story_id AND n.round_number > r.round_number) AS latest,
       EXISTS (SELECT 1 FROM sprint_story_assignments a
                 JOIN sprints s ON s.id = a.sprint_id
                WHERE a.story_id = r.story_id AND s.status = 'Finalizado') AS in_finished_sprint
  FROM poker_rounds r
  JOIN user_stories us ON us.id = r.story_id
 WHERE r.project_id = $1 AND r.status = 'Cerrada'
 ORDER BY r.closed_at DESC, r.id DESC;
```

La consulta de rondas abiertas de HU-05a ya trae `round_number` ("Ronda 2"); no cambia. La de votos revelados de
HU-05b tampoco.

## Modelo de vista (`src/web`)

- `PokerClosedRoundView` (T001: HU-05b) suma `Latest bool` y `InFinishedSprint bool`.
- `pokerClosedRoundPage` (T001: HU-05b) suma:
  - `Actions bool` = `Latest`;
  - `Locked bool` = `InFinishedSprint`;
  - `Reminder *Flash` = `&Flash{Kind: FlashWarning, Text: MsgPokerDivergenceReminder}` si `Latest` y `poker.Spread`
    de sus votos dice `Divergent`; nil en otro caso.
- `pokerPageData` usa `Series` (ya existe desde HU-05a) para los radios y suma `FinishedSprint string`
  (`LabelPokerFinishedSprint`).

## Pantalla (`templates/poker.html`)

Dentro de cada `data-ronda-cerrada` (T001: HU-05b), después del rango o la alerta, solo si `.Actions`:

```html
<div class="poker-actions">
  <form method="post" action="/planning-poker/nueva-ronda">
    <input type="hidden" name="ronda" value="{{.ID}}">
    <button type="submit"{{if .Locked}} disabled{{end}}>Nueva ronda</button>
  </form>
  <form method="post" action="/planning-poker/estimacion">
    <input type="hidden" name="ronda" value="{{.ID}}">
    {{with .Reminder}}{{template "alert" .}}{{end}}
    <fieldset class="poker-estimate"{{if .Locked}} disabled{{end}}>
      <legend>Estimación acordada</legend>
      {{range $.Data.Series}}<label><input type="radio" name="valor" value="{{.}}"> {{.}}</label>{{end}}
    </fieldset>
    <button type="submit"{{if .Locked}} disabled{{end}}>Registrar estimación</button>
  </form>
  {{if .Locked}}<p class="poker-locked">{{$.Data.FinishedSprint}}</p>{{end}}
</div>
```

Ningún radio de la estimación sale marcado. Salida del recordatorio (parcial de HU-20b):

```html
<div class="alert alert-warning" role="status"><strong class="alert-label">Advertencia:</strong> <span class="alert-text">La última ronda tuvo divergencia: registrá el valor que acordaron después de debatir</span></div>
```

## Textos (`poker_messages.go`)

| Nombre | Texto |
|---|---|
| `MsgPokerNotLatestRound` | "Hay una ronda más reciente de esta historia" |
| `MsgPokerRoundStillOpen` | "La ronda sigue abierta: cerrala antes de registrar la estimación" |
| `MsgPokerStoryInFinishedSprint` | "La historia pertenece a un Sprint finalizado; sus Story Points no se pueden modificar" |
| `MsgPokerDivergenceReminder` | "La última ronda tuvo divergencia: registrá el valor que acordaron después de debatir" |
| `LabelPokerFinishedSprint` | "Sprint finalizado" |
| `pokerNextRoundOpenedMessage(number, title)` | "Ronda N abierta para la historia «title»" |
| `pokerEstimateRegisteredMessage(title, points)` | "Estimación registrada: «title» queda con N Story Points" |

`pokerOutcomeFor` suma `ErrNotLatestRound`, `ErrRoundStillOpen` y `ErrStoryInFinishedSprint` → 409 con su texto.

## Datos de prueba (los mismos en `src/poker`, `src/web` y `test`)

| Elemento | Datos |
|---|---|
| Proyecto «Alfa» | Integrantes Ana, Beto y Caro |
| «Login» | Sin Story Points, "Pendiente", sin Sprint; ronda 1 cerrada (Ana 2, Beto 13, Caro 5) |
| «Pagos» | Sin Story Points, "Pendiente", sin Sprint; ronda 1 cerrada (Ana 3, Beto 5, Caro 8) |
| «Reportes» | 3 Story Points, "Pendiente", sin Sprint; ronda 1 cerrada (Ana 3, Beto 13, Caro 3); ronda 2 abierta con el voto de Ana (5) |
| «Alta» | 3 Story Points, "Hecho"; Sprint 1 "Finalizado" (asignada); ronda 1 cerrada (Ana 5, Beto 5, Caro 5) |
| Sprint 1 de «Alfa» | "Finalizado", objetivo "Alta de usuarios", del 01/09/2026 al 14/09/2026 |
| Proyecto «Beta» | Integrante Dani; historia «Tablero» con su ronda 1 cerrada (Dani 8), para el escenario 11 |

- Las rondas cerradas se cargan directo en la base con `status = 'Cerrada'` y `closed_at` explícito, y los votos en
  `poker_votes`.
- Escenario 3: la prueba abre la ronda 2 de «Login» con "Nueva ronda", Ana, Beto y Caro votan 8, 8 y 5, y se cierra con
  `CloseRound` de HU-05b (T001: HU-05b) o directo en la base.
- Solo para las pruebas de concurrencia con el cierre de Sprint: Sprint 2 de «Alfa» "Activo" (del 15/09/2026 al
  30/09/2026, objetivo "Pagos online") con «Pagos» asignada, en "Hecho" o "En progreso" según el caso.
- Limpieza: asignaciones → historias → Sprints → proyectos → integrantes.
