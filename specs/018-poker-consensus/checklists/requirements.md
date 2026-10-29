# Checklist de calidad de la especificación: HU-18

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master (fijar los Story Points después de estimar).
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 12 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario".
- [x] Los datos de ejemplo son los del Issue y cierran (números de ronda, rangos, niveles y Story Points); lo que el
  Issue no da está en "Supuestos" y no cambia ninguna cuenta.
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-019).
- [x] Los criterios de éxito son medibles (SC-001 a SC-006).
- [x] Los casos límite están identificados (identificadores inválidos, otro proyecto, orden de rechazos, rondas
  simultáneas, estimación contra cierre de Sprint, reemplazo del valor, Sprint activo, historial, recarga, escape).
- [x] Alcance y fuera de alcance definidos (roles, automatismos, estimación por ronda, "Abrir ronda" sin cambios).
- [x] Dependencias explícitas: HU-05b, HU-05a, HU-02, HU-20a, HU-20b; HU-15 para la verificación manual.
- [x] Los códigos 303/422/409/404/500 están definidos por caso.
- [x] Los mensajes están definidos en un solo lugar (el archivo de mensajes de Planning Poker).

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas"), con 30
  tareas como máximo.
- [x] El plan respeta la constitución v1.1.0: los métodos nuevos del puerto `poker.Store` tienen adaptador PostgreSQL y
  pruebas de integración; RED → GREEN por bloque con `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [x] Solo se editan archivos de HU-05a y HU-05b; sus pruebas siguen en verde sin cambiar expectativas.
- [x] Los nombres de HU-05b que todavía no están en el código están marcados "(T001: HU-05b)" y T001 detiene todo si
  HU-05b no está en `main`.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Decisiones del equipo registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-clarify` (28/10): una pregunta (historias de un Sprint finalizado: acciones deshabilitadas con el motivo
  "Sprint finalizado" y 409 en las dos si se envían; las devueltas al Backlog se pueden volver a votar y estimar).
- `/speckit-analyze` (28/10): sin problemas críticos ni violaciones de la constitución; cobertura de FR y SC 100% (19 FR
  y 6 SC, 30 tareas); todos los nombres de prueba de la trazabilidad están en `tasks.md`. Se corrigieron:
  - I1, en `data-model.md`: `pokerPageData` usa `Series` (ya existía) en lugar de "sumarlo";
  - I2, en `plan.md` (riesgos) y T001: con divergencia, la última ronda cerrada tiene dos avisos `alert-warning` (alerta
    de HU-05b y recordatorio) y dos formularios más; T001 revisa si alguna prueba de HU-05b los cuenta;
  - U1, en "Supuestos": «Login», «Pagos» y «Reportes» se toman sin Sprint (el Issue solo da el de «Alta»);
  - L1, en T013: orden de etiquetas.
  - Queda documentado sin prueba propia: FR-016 se cubre con `TestHU05a_ErrorAlLeer` (la lectura ampliada usa la misma
    transacción).
- Revisión de cuentas: rondas «Login» 1 → 2 (→ 3 en la prueba de la regla); rangos 2–13 (niveles 2 y 6: 4, divergencia),
  3–8 (3 y 5: 2, sin), 3–13 (3 y 6: 3, divergencia), 5–5 (0, sin) y la ronda 2 de «Login» 5–8 (4 y 5: 1, sin); Story
  Points «Login» → 8 (escenario 3) o 5 (escenario 5), «Pagos» → 5, «Reportes» y «Alta» siguen en 3; 4 no es de la serie.
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
