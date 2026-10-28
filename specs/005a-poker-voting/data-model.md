# Modelo de datos: HU-05a – Emisión de voto oculto en Planning Poker

## Cambios de esquema

Una migración nueva: `db/migrations/011_create_poker.up.sql` y `011_create_poker.down.sql` (número elegido en
`research.md`, decisión 1). Crea `poker_rounds` y `poker_votes`. No modifica tablas existentes.

Tipos de las referencias (verificados en las migraciones 001 y 002): `projects.id` y `members.id` son `SERIAL`
(INTEGER), `user_stories.id` es `BIGSERIAL` (BIGINT).

### `011_create_poker.up.sql`

```sql
-- Rondas de Planning Poker (HU-05a). HU-05a solo crea rondas 'Abierta' con número 1;
-- HU-05b las cierra y HU-18 agrega rondas siguientes.
CREATE TABLE poker_rounds (
    id           BIGSERIAL PRIMARY KEY,
    project_id   INTEGER     NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    story_id     BIGINT      NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    round_number INTEGER     NOT NULL CHECK (round_number > 0),
    status       VARCHAR(20) NOT NULL DEFAULT 'Abierta'
                 CHECK (status IN ('Abierta', 'Cerrada')),
    opened_by    INTEGER     NOT NULL REFERENCES members(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at    TIMESTAMPTZ,
    -- Abierta <=> sin fecha de cierre
    CONSTRAINT chk_poker_rounds_closed_at CHECK ((status = 'Abierta') = (closed_at IS NULL)),
    CONSTRAINT uq_poker_rounds_story_number UNIQUE (story_id, round_number)
);

-- Una sola ronda abierta por historia (segunda defensa además del bloqueo de la historia).
CREATE UNIQUE INDEX uq_poker_rounds_one_open_per_story
    ON poker_rounds (story_id) WHERE status = 'Abierta';

CREATE INDEX idx_poker_rounds_project_status ON poker_rounds (project_id, status);

-- Votos: uno por integrante y ronda; votar de nuevo reemplaza el valor.
CREATE TABLE poker_votes (
    round_id   BIGINT      NOT NULL REFERENCES poker_rounds(id) ON DELETE CASCADE,
    member_id  INTEGER     NOT NULL REFERENCES members(id),
    value      INTEGER     NOT NULL CHECK (value IN (1, 2, 3, 5, 8, 13, 21, 34, 55, 89)),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (round_id, member_id)
);
```

### `011_create_poker.down.sql`

```sql
DROP TABLE IF EXISTS poker_votes;
DROP TABLE IF EXISTS poker_rounds;
```

### Notas del esquema

| Columna | Uso |
|---|---|
| `poker_rounds.project_id` | Proyecto de la historia; todas las consultas filtran por él |
| `poker_rounds.round_number` | 1 en HU-05a; `UNIQUE (story_id, round_number)` deja lista la numeración de HU-18 |
| `poker_rounds.status` | 'Abierta' o 'Cerrada'; HU-05b cierra con `status = 'Cerrada', closed_at = now()` en el mismo UPDATE |
| `poker_rounds.opened_by` | Integrante identificado que abrió la ronda |
| `poker_votes.value` | Valor de la serie; mismo CHECK que `user_stories.story_points` |
| `poker_votes.updated_at` | Fecha del último cambio del voto |

- `project_id` se copia de la historia ya verificada dentro de la transacción; no hay FK compuesta porque
  `user_stories` no tiene `UNIQUE (id, project_id)` y no se edita su migración.
- Cascadas: historia o proyecto → rondas → votos. Las referencias a `members` no tienen cascada (como
  `effort_logs`): los fixtures borran primero historias, después proyectos y por último integrantes.

## Estados de la ronda

| Desde | Acción | Hacia | Condición | Efecto |
|---|---|---|---|---|
| (sin rondas) | Abrir | Abierta | La historia es del proyecto y nunca tuvo rondas | Ronda 1, `opened_by` = integrante |
| Abierta | Votar | Abierta | Valor de la serie | Alta o reemplazo del voto del integrante |
| Abierta | Cerrar | Cerrada | — | **HU-05b** (fuera de alcance) |
| Cerrada | Votar | — | — | 409 `ErrRoundClosed` |
| Abierta (historia) | Abrir | — | — | 409 `ErrRoundAlreadyOpen` |
| Cerrada (historia sin abiertas) | Abrir | — | — | 409 `ErrStoryAlreadyRounded` |

## Transacciones (adaptador `poker.PostgreSQLStore`)

### Abrir ronda

```sql
BEGIN;
-- 1. Historia del proyecto, bloqueada. Sin fila => ErrStoryNotFound.
SELECT title FROM user_stories
 WHERE id = $1 AND project_id = $2
   FOR NO KEY UPDATE;

-- 2. Rondas existentes de la historia => CheckOpen(statuses)
--    alguna 'Abierta' => ErrRoundAlreadyOpen; alguna otra => ErrStoryAlreadyRounded.
SELECT status FROM poker_rounds WHERE story_id = $1;

-- 3. Alta de la ronda 1.
INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by)
VALUES ($2, $1, 1, 'Abierta', $3)
RETURNING id;
COMMIT;
```

- Un `23505` sobre `uq_poker_rounds_one_open_per_story` (no debería ocurrir con el bloqueo) se traduce a
  `ErrRoundAlreadyOpen`; uno sobre `uq_poker_rounds_story_number`, a `ErrStoryAlreadyRounded`.
- Orden de bloqueos: solo la historia. No forma ciclos con HU-14 (que actualiza `user_stories` sin leer rondas).

### Votar

```sql
BEGIN;
-- 1. Ronda del proyecto con bloqueo compartido (HU-05b la cerrará con FOR UPDATE).
--    Sin fila => ErrRoundNotFound. CheckVote(status): 'Cerrada' => ErrRoundClosed.
SELECT status FROM poker_rounds
 WHERE id = $1 AND project_id = $2
   FOR SHARE;

-- 2. Upsert del voto. "inserted" distingue "Voto registrado" de "Voto actualizado".
WITH previous AS (
    SELECT 1 FROM poker_votes WHERE round_id = $1 AND member_id = $3
)
INSERT INTO poker_votes (round_id, member_id, value)
VALUES ($1, $3, $4)
ON CONFLICT (round_id, member_id)
DO UPDATE SET value = EXCLUDED.value, updated_at = now()
RETURNING NOT EXISTS (SELECT 1 FROM previous) AS inserted;
COMMIT;
```

- El CTE ve el estado anterior a la sentencia, así que `inserted` es falso si el integrante ya había votado.
- Votos de distintos integrantes en la misma ronda no se bloquean entre sí (`FOR SHARE` es compatible consigo
  mismo).
- Cualquier error revierte la transacción: no queda nada guardado.

## Lecturas de pantalla (adaptador `web.PostgreSQLPokerCatalog`)

```sql
-- Historias del selector (todas las del proyecto, decisión del 27/10)
SELECT id, title, status FROM user_stories
 WHERE project_id = $1
 ORDER BY id;

-- Rondas abiertas del proyecto
SELECT r.id, r.story_id, us.title, r.round_number
  FROM poker_rounds r
  JOIN user_stories us ON us.id = r.story_id
 WHERE r.project_id = $1 AND r.status = 'Abierta'
 ORDER BY r.created_at, r.id;

-- Participantes de cada ronda abierta: NUNCA se lee poker_votes.value
SELECT r.id, m.id, m.display_name,
       EXISTS (SELECT 1 FROM poker_votes v
                WHERE v.round_id = r.id AND v.member_id = m.id) AS has_voted
  FROM poker_rounds r
  JOIN project_members pm ON pm.project_id = r.project_id
  JOIN members m ON m.id = pm.member_id
 WHERE r.project_id = $1 AND r.status = 'Abierta'
 ORDER BY r.id, m.display_name, m.id;

-- Voto propio en las rondas abiertas: solo el integrante identificado
SELECT v.round_id, v.value
  FROM poker_votes v
  JOIN poker_rounds r ON r.id = v.round_id
 WHERE r.project_id = $1 AND r.status = 'Abierta' AND v.member_id = $2;
```

Las cuatro lecturas corren en una transacción de solo lectura (`BEGIN READ ONLY ISOLATION LEVEL REPEATABLE
READ`) para que la pantalla sea coherente aunque alguien vote mientras se arma.

## Tipos de dominio (`src/poker`)

- `Series []int` = 1, 2, 3, 5, 8, 13, 21, 34, 55, 89; `ParseValue(raw string) (int, error)`.
- `RoundStatus` (`StatusOpen = "Abierta"`, `StatusClosed = "Cerrada"`).
- `CheckOpen(existing []RoundStatus) error` y `CheckVote(status RoundStatus) error`.
- `OpenedRound{RoundID, StoryID int64; StoryTitle string; Number int}`.
- `VoteOutcome{RoundID int64; Value int; Updated bool}`.
- Errores: `ErrValueRequired`, `ErrValueNotInSeries`, `ErrRoundClosed`, `ErrRoundNotFound`, `ErrStoryNotFound`,
  `ErrRoundAlreadyOpen`, `ErrStoryAlreadyRounded`, `ErrMemberRequired`.
- Puerto `Store`:
  - `OpenRound(ctx, projectID, storyID, memberID int64) (OpenedRound, error)`
  - `CastVote(ctx, projectID, roundID, memberID int64, value int) (VoteOutcome, error)`
- `Service` (`NewService(store Store) *Service`):
  - `OpenRound(ctx, projectID, storyID, memberID int64) (OpenedRound, error)`: `storyID == 0` →
    `ErrStoryNotFound` sin llamar al almacén; `memberID == 0` → `ErrMemberRequired`.
  - `Vote(ctx, projectID, roundID, memberID int64, rawValue string) (VoteOutcome, error)`: `ParseValue` primero;
    `roundID == 0` → `ErrRoundNotFound`; `memberID == 0` → `ErrMemberRequired`.

Orden de validación al votar: valor (422) → ronda (404/409). Un voto con valor vacío sobre una ronda inexistente
responde 422; ninguno de los dos casos modifica datos.

## Modelo de vista (`src/web`)

- `PokerStoryOption{ID int64; Title, Status string}`.
- `PokerParticipantView{MemberID int64; Name string; HasVoted, IsMe bool}` — **sin campo de valor**. La
  plantilla marca cada fila con `data-integrante="{{.MemberID}}"` para que las pruebas la ubiquen.
- `PokerRoundView{ID, StoryID int64; StoryTitle string; Number int; Participants []PokerParticipantView;
  OwnVote *int}` — `OwnVote` es el único valor y solo se llena con el voto del integrante identificado.
- `PokerBoard{Stories []PokerStoryOption; Rounds []PokerRoundView}`.
- `PokerCatalog`: `PokerBoard(ctx, projectID, memberID int64) (PokerBoard, error)`.

## Notas de pruebas

- Escenarios 7 y 10: el fixture inserta una ronda `'Cerrada'` con `closed_at = now()` directamente en la base.
- Concurrencia: dos `OpenRound` simultáneos sobre la misma historia → uno OK y otro `ErrRoundAlreadyOpen`;
  votos simultáneos de dos integrantes → ambos guardados.
- Falla de guardado: un `poker.Store` de prueba que devuelve error, detrás del `Service` real.
- Ocultamiento (`TestHU05a_VotosAjenosOcultos`): Ana vota 13 y Beto 3 (valores distintos).
  1. En la página de Beto, la fila `data-integrante` de Ana dice "Votó" y no contiene "13"; la fila de Beto
     muestra 3.
  2. El único `checked` del formulario de voto es el valor 3.
  3. Ana cambia su voto a 21 y la página de Beto es idéntica byte a byte a la anterior: ningún dato de la
     respuesta depende del valor ajeno (cubre atributos, campos ocultos y comentarios).
- `TestPokerViewModel_SinCampoDeValorAjeno` recorre con `reflect` `PokerParticipantView` y falla si aparece un
  campo numérico distinto de `MemberID` o un campo llamado `Value`/`Vote`.
