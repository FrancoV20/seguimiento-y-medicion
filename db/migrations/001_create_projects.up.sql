-- Integrantes (alta gestionada fuera de HU-01; esta tabla solo habilita la referencia)
CREATE TABLE members (
    id           SERIAL PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL
);

-- Proyectos
CREATE TABLE projects (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    status     VARCHAR(50) NOT NULL DEFAULT 'Activo',
    CONSTRAINT chk_project_dates CHECK (end_date >= start_date)
);

-- Relación proyecto <-> integrantes (sin duplicados por diseño de la PK compuesta)
CREATE TABLE project_members (
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    member_id  INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, member_id)
);