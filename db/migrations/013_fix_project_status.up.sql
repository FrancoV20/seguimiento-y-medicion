-- FIX-HU-21: los proyectos creados antes del fix quedaron con estado 'Active'
-- y el Product Backlog los rechazaba. Se pasan al estado que usa todo el sistema.
UPDATE projects SET status = 'Activo' WHERE status = 'Active';