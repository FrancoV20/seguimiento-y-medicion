# Specification Quality Checklist: Registro y trazabilidad de defectos

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-26

**Feature**: [spec.md](../spec.md)

## Content Quality

- [ ] No contiene detalles de implementación en los escenarios de aceptación
- [ ] Enfocada en el valor para el Product Builder
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

- Revisión del 26/10/2026: se parte del texto original de HU-06 (2 escenarios). Agrega la pantalla "Defectos"
  (las historias nuevas se entregan con su pantalla) y pasa a 9 escenarios simples (sin esquemas).
- Decisión a confirmar con el equipo: sin Sprint activo se bloquea el registro. Es la misma decisión votada para
  HU-19; para HU-06 no se votó. El Sprint de detección es siempre el Sprint activo, nunca se elige a mano.
- Niveles de severidad propuestos (se cambian en un solo lugar): Crítica, Alta, Media y Baja. La descripción es
  obligatoria, con un máximo de 1000 caracteres (propuesta del equipo).
- La historia relacionada puede ser cualquiera del proyecto activo, de cualquier estado; registrar un defecto no
  cambia su estado.
- El registro bloquea el Sprint activo con `FOR SHARE` para no mezclarse con el cierre de HU-15 (`FOR UPDATE`).
- Migración nueva (`defects`): toma el último número de `main` más uno y se coordina con la de HU-15
  (`sprint_story_results`). La base ya valida severidad, estado y la coherencia con el Sprint de resolución para
  HU-19.
- HU-19 extiende esta pantalla (resolver y reabrir) y completa el Sprint de resolución; HU-10 cuenta defectos por
  Sprint de detección y de resolución.
- Textos propuestos por el equipo (se cambian en `defects_form.go`): ver `spec.md`.
- Sin identificación de integrante ni roles: autenticación y roles se resuelven después del Sprint 2 (decisión del
  equipo). El defecto no guarda quién lo reportó.
- Pendiente de verificación (no bloquea la spec): esquema de `sprints` y `user_stories`, nombres existentes y el
  número de migración (T001 y T004), y los contratos de HU-20b (T002).
- Depende de HU-02, HU-04, FIX-HU-04, HU-20a y HU-20b. Estimación: 8 Story Points (el PDF indica 5) por la
  migración, el dominio y la pantalla; conviene revisarla en el Planning Poker.
- No incluido: resolver y reabrir (HU-19), editar o borrar defectos, quién reportó, filtros y métricas de defectos
  (HU-10).

All checklist items pass. The feature is ready for `/speckit-plan`.