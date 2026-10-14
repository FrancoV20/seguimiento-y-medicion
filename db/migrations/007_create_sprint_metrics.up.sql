CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'Activo',
    next_sprint_number INT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS backlog_stories (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority TEXT NOT NULL DEFAULT 'Media',
    story_points INT,
    estimated_hours NUMERIC(10,2),
    actual_hours NUMERIC(10,2),
    status TEXT NOT NULL DEFAULT 'Pendiente',
    estimation_alert TEXT
);

CREATE TABLE IF NOT EXISTS sprints (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    number INT NOT NULL DEFAULT 1,
    display_id TEXT NOT NULL DEFAULT 'Sprint 1',
    goal TEXT NOT NULL DEFAULT '',
    start_date DATE,
    end_date DATE,
    status TEXT NOT NULL DEFAULT 'Activo'
);

CREATE TABLE IF NOT EXISTS sprint_stories (
    sprint_id TEXT NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    story_id TEXT NOT NULL REFERENCES backlog_stories(id) ON DELETE CASCADE,
    PRIMARY KEY (sprint_id, story_id)
);

CREATE TABLE IF NOT EXISTS sprint_metrics (
    sprint_id TEXT PRIMARY KEY REFERENCES sprints(id) ON DELETE CASCADE,
    velocity INT NOT NULL,
    estimated_hours_total NUMERIC(12,2) NOT NULL,
    actual_hours_total NUMERIC(12,2) NOT NULL,
    deviation_percentage TEXT NOT NULL,
    calculation_used_count INT NOT NULL,
    calculation_total_count INT NOT NULL,
    is_partial BOOLEAN NOT NULL DEFAULT FALSE,
    warning TEXT NOT NULL DEFAULT '',
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
