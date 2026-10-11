CREATE TABLE effort_logs (
    id         BIGSERIAL PRIMARY KEY,
    story_id   BIGINT        NOT NULL REFERENCES user_stories(id),
    member_id  INTEGER       NOT NULL REFERENCES members(id),
    work_date  DATE          NOT NULL,
    activity   TEXT          NOT NULL CHECK (btrim(activity) <> ''),
    hours      NUMERIC(5,2)  NOT NULL CHECK (hours > 0 AND hours <= 24),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_effort_logs_story_id ON effort_logs(story_id);
CREATE INDEX idx_effort_logs_member_id ON effort_logs(member_id);