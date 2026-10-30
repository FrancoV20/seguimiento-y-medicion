# Specification Quality Checklist: Horas estimadas por historia (Estimaciones)

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-21

**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No contiene detalles de implementación en los escenarios de aceptación
- [x] Enfocada en el valor para quien estima y compara el esfuerzo
- [x] Escrita de forma que el Product Owner pueda revisarla
- [x] Todas las secciones obligatorias están completas

## Requirement Completeness

- [x] No quedan marcadores [NEEDS CLARIFICATION] ni "(a definir)"
- [x] Los requisitos son comprobables y no ambiguos
- [x] Los criterios de éxito son medibles
- [x] Todos los escenarios de aceptación están definidos y son simples (sin esquemas)
- [x] Los casos límite están identificados
- [x] El alcance está acotado y las dependencias están identificadas
- [x] La sección SDD (objetivo, entradas, salidas, reglas, restricciones, casos límite y errores) está completa

## Feature Readiness

- [x] Todos los requisitos funcionales tienen criterios de aceptación y pruebas en `tasks.md`
- [x] Las historias de usuario cubren los flujos principales
- [x] La trazabilidad Historia → Spec → Criterios → BDD → Tests → Código está completa

## Notes

- Revisión del 21/10/2026: se parte del texto original de HU-17. Agrega la pantalla "Estimaciones",
  porque las historias nuevas se entregan con su pantalla, y pasa de 6 a 11 escenarios simples (sin
  esquemas), uno por regla.
- Se resolvió el "(a definir)" de "Historia de un Sprint finalizado": se bloquea. Es una propuesta;
  confirmar con el equipo al refinar.
- Decidido en la spec: un valor vacío se rechaza como no numérico; no se puede quitar una estimación ya
  cargada.
- La regla de horas (mayor que 0, hasta dos decimales, coma o punto, máximo 9999,99) y "Sin estimar" son
  las que HU-22 deja en el dominio: HU-17 las reutiliza; por eso depende de HU-22.
- Mismo dato que HU-07: `user_stories.estimated_hours`. Solo se modifica esa columna; las horas reales
  se muestran para comparar y no se tocan.
- Depende también de HU-03 (solo lectura de las horas reales), HU-22, HU-20a y HU-20b. No depende de
  HU-14 ni de HU-23.
- Textos propuestos por el equipo (se cambian en `estimates_form.go`): "Horas estimadas registradas",
  "Horas estimadas actualizadas", el del Sprint finalizado y el motivo "Sprint finalizado". El resto es
  de HU-22.
- Sin identificación de integrante ni roles: autenticación y roles se resuelven después del Sprint 2
  (decisión del equipo).
- Verificado en T001 y T004 (28/10/2026): el paquete de historias es `src/backlog`; la regla de horas de
  HU-22 es `validEstimatedHours` con `ErrInvalidEstimatedHours`; `effort_logs` usa `story_id` y `hours`
  (ver `data-model.md`).
- Revisión del 30/10/2026: checklist verificado contra `spec.md`, `tasks.md` y la trazabilidad final.
- Estimación: 3 Story Points, como estaba sugerido; la regla de horas ya existe y falta el servicio y la
  pantalla.
- No incluido: quitar una estimación, el alta con horas (HU-22) y editar otros datos de la historia
  (HU-13).

All checklist items pass. The feature is ready for `/speckit-plan`.