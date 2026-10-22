# Specification Quality Checklist: Cierre de Sprint

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-21

**Feature**: [spec.md](../spec.md)

## Content Quality

- [ ] No contiene detalles de implementación en los escenarios de aceptación
- [ ] Enfocada en el valor para el Scrum Master
- [ ] Escrita de forma que el Product Owner pueda revisarla
- [ ] Todas las secciones obligatorias están completas

## Requirement Completeness

- [ ] No quedan marcadores [NEEDS CLARIFICATION] ni "(a definir)"
- [ ] Los requisitos son comprobables y no ambiguos
- [ ] Los criterios de éxito son medibles
- [ ] Todos los escenarios de aceptación están definidos y son simples (sin esquemas)
- [ ] Los casos límite están identificados
- [ ] El alcance está acotado y las dependencias están identificadas
- [ ] La sección SDD (objetivo, entradas, salidas, reglas, restricciones, casos límite y errores) está completa

## Feature Readiness

- [ ] Todos los requisitos funcionales tienen criterios de aceptación y pruebas en `tasks.md`
- [ ] Las historias de usuario cubren los flujos principales
- [ ] La trazabilidad Historia → Spec → Criterios → BDD → Tests → Código está completa

## Notes

- Revisión del 22/10/2026: se parte del texto original de HU-15. Agrega la pantalla "Cierre de Sprint"
  con resumen previo, porque las historias nuevas se entregan con su pantalla, y pasa de 6 a 11
  escenarios simples (sin esquemas).
- Se resolvió el "(a definir)" de las historias no completadas por voto del equipo (21/10/2026): vuelven
  al Product Backlog para poder asignarse a otro Sprint, con los 4 estados: pasan a "Pendiente" y
  pierden la asignación.
- Decisión de diseño aceptada por el equipo (22/10/2026): al devolver las historias el Sprint cerrado
  perdería lo que planificó, y HU-10, HU-16 y la Sprint Review quedarían mal. Por eso el cierre guarda
  un registro por historia (`sprint_story_results`) con una migración nueva. Sin esto no se puede
  reconstruir después.
- El formulario envía el identificador del Sprint para no cerrar por error un Sprint nuevo con una
  pantalla vieja.
- HU-15 solo invoca la validación de HU-07; la lista de historias a corregir se muestra acá y HU-25 no la
  repite.
- La regla "carga de esfuerzo posterior" es de HU-03; T001 verifica si existe y, si no, T016 la agrega.
- Depende de HU-14 (estados y tablero), HU-07, HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b.
- Textos propuestos por el equipo (se cambian en `close_form.go`); el de "ya está cerrado" sigue el texto
  original.
- Sin identificación de integrante ni roles: autenticación y roles se resuelven después del Sprint 2
  (decisión del equipo).
- Pendiente de verificación (no bloquea la spec): validación previa y velocidad de HU-07, regla de HU-03
  y `BoardCatalog` de HU-14 (T001), y el número de migración (T004). En `src/sprint/errors.go` y
  `model.go` no hay nombres que choquen con los de HU-15 (verificado el 22/10/2026).
- Estimación: 8 Story Points (antes 5 sugeridos) por la pantalla, la transacción y la tabla nueva;
  conviene revisarla en el Planning Poker.
- No incluido: reabrir un Sprint, validar la fecha de fin al cerrar y consultar Sprints anteriores
  (HU-16).

All checklist items pass. The feature is ready for `/speckit-plan`.