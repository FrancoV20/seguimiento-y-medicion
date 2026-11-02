ALTER TABLE project_members
    ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'Integrante'
    CHECK (role IN ('Scrum Master', 'Integrante'));
