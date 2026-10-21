ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_end_after_start;

ALTER TABLE projects
    ADD CONSTRAINT projects_end_not_before_start CHECK (end_date >= start_date);
