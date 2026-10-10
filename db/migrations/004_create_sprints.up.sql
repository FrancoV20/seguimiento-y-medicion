ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS next_sprint_number INTEGER NOT NULL DEFAULT 1 CHECK (next_sprint_number > 0);

CREATE TABLE sprints (
    id         BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number     INTEGER NOT NULL CHECK (number > 0),
    goal       TEXT NOT NULL CHECK (btrim(goal) <> ''),
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'Activo'
               CHECK (status IN ('Activo', 'Finalizado')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date > start_date),
    UNIQUE (project_id, number)
);

CREATE INDEX idx_sprints_project_id ON sprints(project_id);

CREATE TABLE sprint_story_assignments (
    sprint_id  BIGINT NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    story_id   BIGINT NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sprint_id, story_id),
    UNIQUE (story_id)
);

CREATE INDEX idx_sprint_story_assignments_sprint_id ON sprint_story_assignments(sprint_id);

ALTER TABLE user_stories
    ADD COLUMN IF NOT EXISTS active_sprint_id BIGINT REFERENCES sprints(id) ON DELETE SET NULL;

ALTER TABLE user_stories
    DROP CONSTRAINT IF EXISTS user_stories_status_check;

ALTER TABLE user_stories
    ADD CONSTRAINT chk_user_stories_status
    CHECK (status IN ('Pendiente', 'En Sprint', 'En progreso', 'Hecho'));

CREATE INDEX idx_user_stories_active_sprint_id ON user_stories(active_sprint_id);
