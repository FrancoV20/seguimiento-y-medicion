-- Integrantes de prueba para el formulario de HU-21.
-- No es una migración: no consume número y se puede correr más de una vez sin duplicar.
INSERT INTO members (display_name)
SELECT v.name
FROM (VALUES
    ('Franco Valentin Velazco'),
    ('Adriel Alonso Gonzalez'),
    ('Alejandro Ruiz Andreola'),
    ('Giuliana Betitol Rivadeo'),
    ('Juan Ignacio Sanchez')
) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM members m WHERE m.display_name = v.name);
