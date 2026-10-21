# Specification Quality Checklist: Actualización de estado de historias (Tablero)

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-21

**Feature**: [spec.md](../spec.md)

## Content Quality

- [ ] No contiene detalles de implementación en los escenarios de aceptación
- [ ] Enfocada en el valor para quien trabaja en el Sprint
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

- Revisión del 21/10/2026: se parte del texto original de HU-14. Pasa a los 4 estados (decisión del
  equipo del 20/10), parte de "En Sprint" y agrega la pantalla "Tablero", porque las historias nuevas se
  entregan con su pantalla y no había forma de cambiar el estado desde la web.
- Se resolvió el "(a definir)" de "Completar sin horas": se permite con advertencia. Es una propuesta;
  confirmar con el equipo al refinar.
- Se agregaron "Transición no permitida" y "Estado desactualizado", y los escenarios de la pantalla
  (13 en total, todos simples).
- Los Story Points completados se calculan (suma de las historias "Hecho"), no se guardan: evita
  duplicados al reabrir o enviar dos veces.
- Depende también de FIX-HU-04 y HU-03 (horas registradas), HU-20a y HU-20b. No depende de HU-23: las
  pruebas crean sus propios Sprints.
- Textos propuestos por el equipo (se cambian en `board_form.go`); "La historia no está asignada a un
  Sprint activo" es el de HU-14 original.
- Sin identificación de integrante ni roles: autenticación y roles se resuelven después del Sprint 2
  (decisión del equipo).
- Pendiente de verificación (no bloquea la spec): ruta del paquete de Sprint, constantes de estado,
  columna de historia en `effort_logs` y la definición de "completada" de HU-07 (T001 y T004).
- Estimación: 5 Story Points sugeridos (antes 3) por incluir la regla de dominio y la pantalla; conviene
  revisarla en el Planning Poker.
- No incluido: volver de "En progreso" a "En Sprint", límite de una historia en curso por persona y
  registrar quién hizo el cambio.

All checklist items pass. The feature is ready for `/speckit-plan`.