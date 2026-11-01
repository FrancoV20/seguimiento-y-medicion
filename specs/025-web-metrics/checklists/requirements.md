# Specification Quality Checklist: Interfaz web de métricas del Sprint

**Purpose**: Validar que la especificación está completa antes de planificar la implementación.

**Created**: 2026-10-23

**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No contiene detalles de implementación en los escenarios de aceptación
- [x] Enfocada en el valor para el Scrum Master
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

- Revisión del 23/10/2026: se parte del texto original de HU-25 (4 escenarios). Se quita "Cierre bloqueado"
  porque la lista de historias a corregir ya se muestra en la pantalla "Cierre de Sprint" (HU-15); acá solo se
  muestran Sprints cerrados, que no reciben horas nuevas (HU-03), ni cambios de estimación (HU-17), ni ediciones
  (HU-13). Se agregan "Velocidad cero", "Elegir otro Sprint", "Sin Sprints finalizados" y "Sprint en curso"
  (la pantalla se entrega completa): 7 escenarios simples (sin esquemas).
- La web no recalcula las métricas: velocidad y desviación las calcula HU-07; HU-25 las pide, las formatea y las
  muestra. "No disponible" y "Cálculo parcial: basado en X de Y historias" salen de HU-07 (T001).
- Una sola pantalla de solo lectura: lista de Sprints finalizados y métricas del elegido (`?sprint=ID`); por
  defecto, el más reciente. Un `sprint` inválido, activo o de otro proyecto se ignora.
- Formato de la desviación propuesto por el equipo: dos decimales, coma y signo; positivo significa que se trabajó
  más de lo estimado.
- Textos propuestos por el equipo (se cambian en `metrics_view.go`): "Aún no hay Sprints finalizados" y "El Sprint N
  sigue en curso; sus métricas se calculan al cerrarlo".
- Sin migraciones. Pendiente de verificación (no bloquea la spec): servicio, tipos y textos de HU-07, estado de
  Sprint finalizado y cómo trata HU-07 las historias devueltas por HU-15 (T001 y T004), y los contratos de HU-20a
  y HU-20b (T002).
- Sin autenticación ni roles: se resuelven después del Sprint 2 (decisión del equipo). La sección no requiere
  integrante.
- Depende de HU-07, HU-15, HU-17, HU-20a y HU-20b (sin cambios respecto del PDF). Estimación: 3 Story Points (sin
  cambios), porque es de solo lectura.
- Revisión del 31/10/2026 (antes de integrar a `main`): checklist verificado contra `spec.md`, `tasks.md`,
  la trazabilidad final y las pruebas; sin marcadores pendientes y con los 7 escenarios simples (sin esquemas).
- No incluido: Story Points planificados y porcentaje de historias completadas (HU-10), detalle de historias y
  Sprints anteriores (HU-16), gráficos (HU-08) y exportar (HU-09).

All checklist items pass. The feature is ready for `/speckit-plan`.