CREATE TABLE user_stories (
    id           BIGSERIAL PRIMARY KEY,
    project_id   BIGINT NOT NULL REFERENCES projects(id),
    title        TEXT NOT NULL CHECK (btrim(title) <> ''),
    description  TEXT NOT NULL CHECK (btrim(description) <> ''),
    priority     TEXT NOT NULL CHECK (priority IN ('Alta', 'Media', 'Baja')),
    story_points INTEGER CHECK (story_points IN (1,2,3,5,8,13,21,34,55,89)),
    status       TEXT NOT NULL DEFAULT 'Pendiente'
                 CHECK (status IN ('Pendiente', 'En progreso', 'Hecho')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
    -- , estimated_hours NUMERIC(6,2) CHECK (estimated_hours >= 0)  -- si el equipo lo aprueba
);

CREATE INDEX idx_user_stories_project_id ON user_stories(project_id);

CREATE TABLE acceptance_criteria (
    id       BIGSERIAL PRIMARY KEY,
    story_id BIGINT NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    content  TEXT NOT NULL CHECK (btrim(content) <> ''),
    UNIQUE (story_id, position)
);