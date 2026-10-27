# Coordinación de HU-24 con otras historias (T048)

Revisado el 2026-10-26 contra `main` (750b2b7) y las ramas `HU_14`, `HU_22` y `HU-21`.

| Historia | Qué se verificó | Resultado |
|---|---|---|
| HU-07 y HU-25 (métricas) | `src/metrics/postgres_metrics_store.go` suma `effort_logs.hours` de las historias de `sprint_story_assignments`. | Sin cambios: HU-24 no escribe en `effort_logs`; registra con el servicio de HU-03, así que las métricas leen lo mismo que antes. |
| HU-23 (Sprints web) | `sprint.RemoveStoryFromSprint` solo exige estado "En Sprint"; no mira las horas. | **Hoy `main` no bloquea quitar una historia con horas.** Una historia con horas cargadas desde HU-24 sigue en "En Sprint" (registrar horas no cambia el estado). La validación la agrega HU-23 en su T047: hay que mantenerla antes de integrar HU-23. |
| HU-14 (Tablero) | La rama `HU_14` comprueba si existen horas con `EXISTS (SELECT 1 FROM effort_logs ...)`. | Compatible: con HU-24 se pueden cargar horas desde la web antes de completar una historia. |
| HU-15 (Cerrar Sprint) | Su spec (escenario 10) delega en HU-03 el rechazo de cargas después del cierre. | Compatible: al cerrar el Sprint la historia deja de estar en el Sprint activo, desaparece del selector de HU-24 y HU-03 rechaza la carga (prueba `TestEffort_SprintCerradoEntreAbrirYEnviar`). Las historias devueltas al Backlog dejan de aparecer si HU-15 limpia `active_sprint_id`. |
| HU-03 (registro de esfuerzo) | `effort.ValidateInput` ya devuelve todas las violaciones. | T010 no modificó HU-03; solo se agregó la prueba `effort_actions_test.go`. |
| HU-21 y HU-14 (`main.go`) | Las dos ramas también registran su sección en `src/cmd/server/main.go`. | Conflicto de merge simple: conservar los bloques de registro de cada historia. No hay identificadores `Effort*` repetidos en otras ramas. |
| HU-21 (manual de usuario) | `docs/manual-usuario.md` llega con HU-21. | La sección "Registro de esfuerzo" está en `quickstart.md`; copiarla al manual cuando se integre HU-21. |

## Mensaje para el equipo

> HU-24 (registro de esfuerzo web) está lista para revisión. Para coordinar:
> - **HU-07 / HU-25**: no cambia nada; las horas se siguen guardando en `effort_logs` con el servicio de HU-03.
> - **HU-23**: hoy `RemoveStoryFromSprint` deja quitar una historia "En Sprint" aunque tenga horas. Registrar horas
>   no cambia el estado, así que la validación de su T047 es necesaria antes de integrar HU-23.
> - **HU-14**: ya se pueden cargar horas desde la web en `/esfuerzo` antes de completar una historia.
> - **HU-15**: cuando se cierra el Sprint, sus historias dejan de aparecer en el selector y HU-03 rechaza cargas.
> - **HU-03**: no se modificó.
> - **HU-21 y HU-14**: al mergear vamos a chocar en `src/cmd/server/main.go`; se resuelve dejando los bloques de
>   registro de las tres secciones.
