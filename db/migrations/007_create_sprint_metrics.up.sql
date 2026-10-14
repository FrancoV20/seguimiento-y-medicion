CREATE TABLE IF NOT EXISTS sprint_metrics (
    sprint_id INT PRIMARY KEY REFERENCES sprints(id) ON DELETE CASCADE,
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