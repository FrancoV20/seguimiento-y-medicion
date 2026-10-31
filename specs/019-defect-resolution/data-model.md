# Modelo de datos: HU-19 – Resolver y reabrir defectos

## Cambios de esquema

Ninguno. HU-19 usa las columnas creadas por HU-06:

| Columna (`defects`) | Uso en HU-19 |
|---|---|
| `id` | Identificador recibido en el campo `defecto` |
| `project_id` | Comprueba que el defecto es del proyecto activo |
| `status` | 'Abierto' ↔ 'Resuelto' |
| `detection_sprint_id` | Sprint de detección (no se modifica) |
| `resolution_sprint_id` | Se completa al resolver y se borra al reabrir |

CHECK de HU-06 que HU-19 respeta: `status = 'Abierto'` ⇒ `resolution_sprint_id IS NULL`;
`status = 'Resuelto'` ⇒ `resolution_sprint_id IS NOT NULL`.

## Transiciones

| Desde | Acción | Hacia | Condición | Efecto |
|---|---|---|---|---|
| Abierto | Resolver | Resuelto | Hay Sprint activo y su número ≥ el de detección | `resolution_sprint_id` = Sprint activo |
| Resuelto | Reabrir | Abierto | Ninguna adicional | `resolution_sprint_id` = NULL |
| Resuelto | Resolver | — | — | 409 "ya resuelto" |
| Abierto | Reabrir | — | — | 409 "ya abierto" |

## Consultas

Resolver (una transacción):

```sql
BEGIN;
SELECT d.status, ds.number
  FROM defects d JOIN sprints ds ON ds.id = d.detection_sprint_id
 WHERE d.id = $1 AND d.project_id = $2
   FOR UPDATE OF d;                       -- sin fila => ErrDefectNotFound

SELECT id, number FROM sprints
 WHERE project_id = $2 AND status = 'Activo'
 ORDER BY number DESC LIMIT 1
   FOR SHARE;                             -- sin fila => ErrNoActiveSprint

-- reglas: ErrAlreadyResolved, ErrResolutionBeforeDetection
UPDATE defects SET status = 'Resuelto', resolution_sprint_id = $3 WHERE id = $1;
COMMIT;
```

Reabrir (una transacción):

```sql
BEGIN;
SELECT status FROM defects WHERE id = $1 AND project_id = $2 FOR UPDATE;
                                          -- sin fila => ErrDefectNotFound
-- regla: ErrAlreadyOpen
UPDATE defects SET status = 'Abierto', resolution_sprint_id = NULL WHERE id = $1;
COMMIT;
```

Cualquier error revierte la transacción: no queda nada guardado.

## Tipos de dominio

- `Resolved{DefectID int64, SprintNumber int64}`: resultado de resolver.
- Errores nuevos: `ErrDefectNotFound`, `ErrResolutionBeforeDetection`,
  `ErrAlreadyResolved`, `ErrAlreadyOpen`. Se reutiliza `ErrNoActiveSprint` de HU-06.
- Puerto `ResolutionStore`: `Resolve(ctx, projectID, defectID)` y `Reopen(ctx, projectID, defectID)`.

## Notas de pruebas

- Escenario 4: el fixture crea el Sprint 3 "Finalizado" (donde se detectó el
  defecto) y el Sprint 2 "Activo" directamente en la base.
- Concurrencia: dos `Resolve` simultáneos sobre el mismo defecto; uno termina bien
  y el otro con `ErrAlreadyResolved`.
- Limpieza de fixtures: borrar primero las historias (los defectos caen en
  cascada) y después el proyecto, porque `user_stories.project_id` no tiene
  borrado en cascada.