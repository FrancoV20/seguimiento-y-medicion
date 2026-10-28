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
