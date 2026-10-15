ALTER TABLE user_stories
    DROP CONSTRAINT IF EXISTS chk_user_stories_status;

UPDATE user_stories
SET status = 'Pendiente'
WHERE status = 'En Sprint';

ALTER TABLE user_stories
    DROP COLUMN IF EXISTS active_sprint_id;

ALTER TABLE user_stories
    ADD CONSTRAINT user_stories_status_check
    CHECK (status IN ('Pendiente', 'En progreso', 'Hecho'));

DROP TABLE IF EXISTS sprint_story_assignments;
DROP TABLE IF EXISTS sprints;

ALTER TABLE projects
    DROP COLUMN IF EXISTS next_sprint_number;
