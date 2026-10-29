# Modelo de datos: HU-05b – Revelación de Votos y Detección de Divergencia

## Cambios de esquema

Ninguno. HU-05b usa las tablas de `db/migrations/011_create_poker.up.sql` (HU-05a) tal como están:

| Columna | Uso en HU-05b |
|---|---|
| `poker_rounds.status` | Pasa de `'Abierta'` a `'Cerrada'` al cerrar; las lecturas de valores filtran por `'Cerrada'` |
| `poker_rounds.closed_at` | Se completa con `now()` en el mismo UPDATE (lo exige `chk_poker_rounds_closed_at`); ordena "Rondas cerradas" |
| `poker_rounds.project_id` | Toda escritura y lectura filtra por el proyecto activo |
| `poker_votes.value` | Se lee solo para rondas cerradas |
| `project_members` | Integrantes actuales que figuran en la lista revelada |

## Estados de la ronda

| Desde | Acción | Hacia | Condición | Efecto |
|---|---|---|---|---|
| Abierta | Cerrar | Cerrada | Ronda del proyecto con al menos un voto | `status = 'Cerrada'`, `closed_at = now()`; los votos se revelan |
| Abierta (sin votos) | Cerrar | — | — | 409 `ErrNoVotes`; sigue abierta |
| Cerrada | Cerrar | — | — | 409 `ErrRoundClosed`; `closed_at` no cambia |
| Cerrada | Votar | — | — | 409 `ErrRoundClosed` (HU-05a) |
| (inexistente / otro proyecto) | Cerrar | — | — | 404 `ErrRoundNotFound` |
| Cerrada | Abrir otra ronda de la historia | — | — | 409 `ErrStoryAlreadyRounded` (HU-05a) hasta HU-18 |

## Regla de divergencia (`src/poker/spread.go`) — contrato para HU-18

```go
// VoteSpread es el rango de los votos emitidos en una ronda.
type VoteSpread struct {
    Low, High int  // menor y mayor valor emitido
    Divergent bool // level(High) - level(Low) > 2
}

// Spread calcula el rango y la divergencia de los valores emitidos.
// Sin valores: ErrNoVotes. Un valor fuera de Series: ErrValueNotInSeries.
// No modifica values.
func Spread(values []int) (VoteSpread, error)
```

- `level(v)` (no exportada) = posición de `v` en `Series` + 1: 1→1, 2→2, 3→3, 5→4, 8→5, 13→6, 21→7, 34→8, 55→9, 89→10.
- `maxLevelGap = 2` (no exportada). Un solo valor, o todos iguales: `Divergent == false`.
- HU-18 la usa con los valores de la última ronda cerrada; los nombres `poker.Spread`, `poker.VoteSpread` y
  `poker.ErrNoVotes` son el contrato.

Casos de la prueba de tabla `TestSpread` (datos del Issue y bordes):

| Valores | Low | High | Niveles | Diferencia | Divergent |
|---|---|---|---|---|---|
| 2, 13, 5 («Login») | 2 | 13 | 2 y 6 | 4 | true |
| 3, 5, 8 («Pagos») | 3 | 8 | 3 y 5 | 2 | false |
| 3, 13, 3 («Reportes») | 3 | 13 | 3 y 6 | 3 | true |
| 5, 5 («Perfil») | 5 | 5 | 4 y 4 | 0 | false |
| 8 | 8 | 8 | 5 y 5 | 0 | false |
| 1, 3 | 1 | 3 | 1 y 3 | 2 | false |
| 1, 5 | 1 | 5 | 1 y 4 | 3 | true |
| 55, 89 | 55 | 89 | 9 y 10 | 1 | false |
| 1, 89 | 1 | 89 | 1 y 10 | 9 | true |
| (vacío) | — | — | — | — | `ErrNoVotes` |
| 4 / 2, 0 | — | — | — | — | `ErrValueNotInSeries` |

## Tipos de dominio nuevos (`src/poker`)

- `errors.go`: `ErrNoVotes = errors.New("poker: la ronda no tiene votos")`.
- `round.go`:
  - `CheckClose(status RoundStatus, votes int) error`: `status != StatusOpen` → `ErrRoundClosed` (antes que contar);
    `votes == 0` → `ErrNoVotes`; si no, nil.
  - `ClosedRound{RoundID, StoryID int64; StoryTitle string; Number int}`.
- Puerto `Store` (se agrega un método):
  - `CloseRound(ctx context.Context, projectID, roundID int64) (ClosedRound, error)`
- `Service`:
  - `CloseRound(ctx context.Context, projectID, roundID, memberID int64) (ClosedRound, error)`: `roundID == 0` →
    `ErrRoundNotFound` y `memberID == 0` → `ErrMemberRequired`, sin llamar al almacén; si no, `store.CloseRound(ctx,
    projectID, roundID)` y devuelve sus errores sin envolver.

## Transacción de cierre (adaptador `poker.PostgreSQLStore.CloseRound`)

```sql
BEGIN;
-- 1. Ronda del proyecto con bloqueo exclusivo (los votos la toman FOR SHARE y esperan).
--    Sin fila => ErrRoundNotFound (404, antes que cualquier 409).
SELECT r.status, r.story_id, r.round_number, us.title
  FROM poker_rounds r
  JOIN user_stories us ON us.id = r.story_id
 WHERE r.id = $1 AND r.project_id = $2
   FOR UPDATE OF r;

-- 2. Votos de la ronda.
SELECT COUNT(*) FROM poker_votes WHERE round_id = $1;

-- 3. CheckClose(status, count): 'Cerrada' => ErrRoundClosed; 0 => ErrNoVotes.

-- 4. Cierre en una sola sentencia.
UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1;
COMMIT;
```

- `FOR UPDATE OF r` bloquea solo la fila de la ronda, no la historia.
- Cualquier error (incluido un contexto vencido) revierte la transacción: la ronda sigue `'Abierta'`.
- Orden de bloqueos: solo la ronda; el voto de HU-05a también toma solo la ronda. No hay ciclos.

## Lecturas de pantalla (adaptador `web.PostgreSQLPokerCatalog`)

Las cuatro lecturas de HU-05a no cambian. Se agregan dos, en la misma transacción `READ ONLY ISOLATION LEVEL
REPEATABLE READ`:

```sql
-- Rondas cerradas del proyecto, de la más recientemente cerrada a la más antigua.
SELECT r.id, r.story_id, us.title, r.round_number
  FROM poker_rounds r
  JOIN user_stories us ON us.id = r.story_id
 WHERE r.project_id = $1 AND r.status = 'Cerrada'
 ORDER BY r.closed_at DESC, r.id DESC;

-- Votos revelados: integrantes actuales del proyecto más quien votó en la ronda.
-- poker_votes.value se lee SOLO para rondas del CTE closed (status = 'Cerrada').
WITH closed AS (
    SELECT id, project_id FROM poker_rounds
     WHERE project_id = $1 AND status = 'Cerrada'
), people AS (
    SELECT c.id AS round_id, pm.member_id
      FROM closed c JOIN project_members pm ON pm.project_id = c.project_id
    UNION
    SELECT v.round_id, v.member_id
      FROM poker_votes v JOIN closed c ON c.id = v.round_id
)
SELECT p.round_id, m.id, m.display_name, v.value
  FROM people p
  JOIN members m ON m.id = p.member_id
  LEFT JOIN poker_votes v ON v.round_id = p.round_id AND v.member_id = p.member_id
 ORDER BY p.round_id, m.display_name, m.id;
```

`v.value` NULL → "No votó". Las filas se asignan a su ronda con el mismo índice por identificador que usa HU-05a.

## Modelo de vista (`src/web`)

- `PokerRevealedVoteView{MemberID int64; Name string; Value *int}` — `Value` nil = "No votó".
- `PokerClosedRoundView{ID, StoryID int64; StoryTitle string; Number int; Votes []PokerRevealedVoteView}`.
- `PokerBoard{Stories []PokerStoryOption; Rounds []PokerRoundView; ClosedRounds []PokerClosedRoundView}` —
  `ClosedRounds` nunca es nil (como `Stories` y `Rounds`).
- Sin cambios: `PokerStoryOption`, `PokerParticipantView` (sigue sin campo de valor), `PokerRoundView`, `PokerCatalog`.
- Solo para la plantilla (`poker_handlers.go`): `pokerClosedRoundPage{PokerClosedRoundView; Alert *Flash; Range
  string}` y `pokerPageData` suma `ClosedRounds []pokerClosedRoundPage`, `NoClosedRounds` y `DidNotVote`.
  - Con `Spread` OK y `Divergent`: `Alert = &Flash{Kind: FlashWarning, Text: pokerDivergenceMessage(Low, High)}`,
    `Range = ""`.
  - Con `Spread` OK sin divergencia: `Alert = nil`, `Range = pokerNoDivergenceMessage(Low, High)`.
  - Con error de `Spread` (ronda cerrada sin votos, solo con datos externos): `Alert = nil`, `Range = ""`.

## Pantalla (`templates/poker.html`)

- En cada `<section class="poker-round" data-ronda="…">` abierta, después del formulario de voto, un formulario
  `method="post" action="/planning-poker/cerrar"` con `<input type="hidden" name="ronda" value="{{.ID}}">` y el botón
  "Cerrar ronda".
- Al final de la página, después de todas las rondas abiertas (así `participantRow` de HU-05a, que toma la primera
  fila `data-integrante`, sigue encontrando las filas de rondas abiertas), sección
  `<section aria-labelledby="poker-closed-title">` con `<h2 id="poker-closed-title">Rondas cerradas</h2>`; sin
  rondas, `<p class="empty">Aún no hay rondas cerradas</p>`; por ronda, `<section class="poker-closed-round"
  data-ronda-cerrada="{{.ID}}">` con título, "Ronda N", lista `<li data-integrante="{{.MemberID}}">Nombre: valor</li>`
  (o "No votó") y debajo `{{with .Alert}}{{template "alert" .}}{{end}}` o `<p class="poker-range">{{.Range}}</p>`.

Salida esperada de la alerta (parcial de HU-20b):

```html
<div class="alert alert-warning" role="status"><strong class="alert-label">Advertencia:</strong> <span class="alert-text">Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación</span></div>
```

## Datos de prueba (los mismos en `src/poker`, `src/web` y `test`)

| Elemento | Datos |
|---|---|
| Proyecto «Alfa» | Integrantes Ana, Beto y Caro |
| «Login», ronda 1 | Abierta (la abre Ana); votos Ana 2, Beto 13, Caro 5 |
| «Pagos», ronda 1 | Abierta; votos Ana 3, Beto 5, Caro 8 |
| «Reportes», ronda 1 | Abierta; votos Ana 3, Beto 13, Caro 3 |
| «Perfil», ronda 1 | Abierta; votos Ana 5, Beto 5; Caro no votó |
| «Ayuda», ronda 1 | Abierta; sin votos |
| Proyecto «Beta» | Integrante Dani; historia «Tablero» con una ronda abierta (Dani 8) y otra historia «Agenda» con una ronda cerrada sin votos (para el 404) |

Cuando una prueba necesita una ronda ya cerrada (escenarios 2, 3, 4, 7, 8 y 9), la cierra con
`UPDATE poker_rounds SET status = 'Cerrada', closed_at = $2` (fecha explícita, para el orden) o con `CloseRound`.
Integrante quitado: Ana vota en «Login» y se borra su fila de `project_members` antes de cerrar.

## Notas de pruebas

- Revelación: con «Login» y «Pagos» cerradas (Pagos después), `ClosedRounds` = [Pagos, Login]; en «Login», Ana 2, Beto
  13, Caro 5; en «Perfil» cerrada, Caro con `Value` nil.
- Seguridad: con «Login» cerrada y «Pagos» todavía abierta (Ana 3, Beto 5, Caro 8), «Pagos» no está en
  `ClosedRounds`, en la página de Beto la fila de Ana en «Pagos» dice "Votó" sin valor, y la página de Beto es idéntica
  byte a byte cuando Ana cambia su voto de «Pagos» a 21 (un valor que no aparece en ninguna ronda cerrada).
- Concurrencia y falla: ver `research.md`, decisión 4.
