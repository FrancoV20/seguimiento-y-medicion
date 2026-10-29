CREATE TABLE defects (
    id                   BIGSERIAL PRIMARY KEY,
    project_id           INTEGER     NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    story_id             BIGINT      NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    description          TEXT        NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 1000),
    severity             VARCHAR(20) NOT NULL CHECK (severity IN ('Crítica', 'Alta', 'Media', 'Baja')),
    status               VARCHAR(20) NOT NULL DEFAULT 'Abierto' CHECK (status IN ('Abierto', 'Resuelto')),
    detection_sprint_id  BIGINT      NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    resolution_sprint_id BIGINT      REFERENCES sprints(id) ON DELETE CASCADE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((status = 'Abierto' AND resolution_sprint_id IS NULL)
        OR (status = 'Resuelto' AND resolution_sprint_id IS NOT NULL))
);

CREATE INDEX idx_defects_project ON defects (project_id, id DESC);
CREATE INDEX idx_defects_story ON defects (story_id);
CREATE INDEX idx_defects_detection ON defects (detection_sprint_id);
CREATE INDEX idx_defects_resolution ON defects (resolution_sprint_id);
