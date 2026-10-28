# Specification Quality Checklist: Actualización de estado de historias (Tablero)

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-21

**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No contiene detalles de implementación en los escenarios de aceptación
- [x] Enfocada en el valor para quien trabaja en el Sprint
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

- Revisión del 21/10/2026: se parte del texto original de HU-14. Pasa a los 4 estados (decisión del
  equipo del 20/10), parte de "En Sprint" y agrega la pantalla "Tablero", porque las historias nuevas se
  entregan con su pantalla y no había forma de cambiar el estado desde la web.
- Se resolvió el “(a definir)” de “Completar sin horas”: el Issue original lo dejaba como “permitir o bloquear: a definir” y el equipo decidió permitir con advertencia.
- Se agregaron "Transición no permitida" y "Estado desactualizado", y los escenarios de la pantalla
  (13 en total, todos simples).
- Los Story Points completados se calculan (suma de las historias "Hecho"), no se guardan: evita
  duplicados al reabrir o enviar dos veces.
- Depende también de FIX-HU-04 y HU-03 (horas registradas), HU-20a y HU-20b. No depende de HU-23: las
  pruebas crean sus propios Sprints.
- Textos definidos por el equipo (se cambian en board_form.go); “La historia no está asignada a un Sprint activo” es el de HU-14 original.
- Sin identificación de integrante ni roles: autenticación y roles se resuelven después del Sprint 2
  (decisión del equipo).
- Verificado en T001 (24/10/2026): paquete src/sprint; constantes Pendiente, EnSprint, InProgress y Done; columna effort_logs.story_id; HU-07 cuenta como completada la historia en “Hecho”.
- Estimación: 5 Story Points sugeridos (antes 3) por incluir la regla de dominio y la pantalla; conviene
  revisarla en el Planning Poker.
- No incluido: volver de "En progreso" a "En Sprint", límite de una historia en curso por persona y
  registrar quién hizo el cambio.
- La acción define el estado de origen (comenzar: En Sprint, completar: En progreso, reabrir: Hecho); si el estado real no coincide se rechaza con el mensaje de “Estado desactualizado”. Decisión del 24/10/2026.

All checklist items pass. The feature is ready for `/speckit-plan`.