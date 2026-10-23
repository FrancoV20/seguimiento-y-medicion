# Specification Quality Checklist: Interfaz web de registro de esfuerzo

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-22

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

- Revisión del 22/10/2026: se parte del texto original de HU-24 (5 escenarios). La pantalla se entrega completa:
  se agregan "Historia sin registros", "Sin Sprint activo", "Sin integrante identificado" e "Historia no
  disponible", y pasa a 9 escenarios simples (sin esquemas).
- El integrante sale de la identificación de HU-20b y no del formulario; por eso la sección exige proyecto e
  integrante. Quien solo quiera consultar igual debe identificarse (sin contraseña).
- Las reglas de HU-03 (horas, actividad, fecha, integrante) no se copian: "los mensajes exactos de HU-03" salen de
  sus constantes. Para mostrar todos los errores juntos HU-03 debe devolver todas las violaciones; T001 lo verifica
  y T010 lo agrega si falta, sin cambiar lo que hace al registrar.
- El selector ofrece todas las historias del Sprint activo, incluidas las "Hecho", como HU-03 (T001 lo confirma).
  Las devueltas por HU-15 y las de Sprints finalizados no aparecen.
- Una sola pantalla: lista, formulario e historial (se abre con `?historia=ID`). No hay JavaScript.
- Textos propuestos por el equipo (se cambian en `effort_form.go`): "Seleccione una historia", "La historia no
  pertenece al Sprint activo", "La fecha no es válida", "No hay un Sprint activo" y "Todavía no hay horas
  registradas en esta historia"; "Esfuerzo registrado correctamente" es el texto original.
- Sin migraciones. Pendiente de verificación (no bloquea la spec): firma, errores y reglas de HU-03, columnas de
  `effort_logs` (T001 y T004) y los contratos de HU-20b (T002).
- Sin autenticación ni roles: se resuelven después del Sprint 2 (decisión del equipo).
- Depende de HU-03, HU-04, FIX-HU-04, HU-20a y HU-20b. Estimación: 3 Story Points (sin cambios), siempre que HU-03 ya
  devuelva todas las violaciones; si hace falta T010, suma trabajo.
- No incluido: editar o borrar registros, validar la fecha contra el Sprint o el futuro, avisar de cargas duplicadas
  y métricas (HU-07, HU-25).

All checklist items pass. The feature is ready for `/speckit-plan`.